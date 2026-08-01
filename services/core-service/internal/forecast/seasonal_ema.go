// Package forecast holds Augno's standard monthly demand forecaster.
//
// Seasonal EMA is used by both the analytics demand-forecast endpoint and the production scheduler, so it lives here rather than in either caller. It mirrors dashboard/packages/utils/src/seasonal-ema.ts; the two must stay in step or the schedule and the dashboard will disagree about the same demand.
//
// Method:
//  1. Seasonal factors — each calendar month's average divided by the overall average.
//  2. Deseasonalize, then EMA-smooth to a current base run rate (alpha = 2 / (min(observations, 12) + 1), so recent months weigh more).
//  3. Forecast = base level x seasonal factor. There is NO trend extrapolation: recent growth raises or lowers the level, but the forecast is not projected to keep moving in that direction. Holt-Winters was dropped deliberately — its multiplicative trend produced runaway exponential forecasts.
//  4. Fixed-width confidence band from the last 12 residuals x seasonal factor x z.
package forecast

import (
	"math"
	"time"
)

// Observation is one complete month of history. The series must be sorted ascending and must exclude the current partial month, which would otherwise read as a collapse in demand.
type Observation struct {
	MonthStart time.Time
	Value      float64
}

// Point is one forecast month. Date is stamped end-of-period (one month past the month being forecast) to match the dashboard's display convention.
type Point struct {
	Date       time.Time
	Forecast   float64
	LowerBound float64
	UpperBound float64
}

// SeasonalEMA forecasts numMonths forward from baseForecastStart, which is the start of the last complete month. Point k covers baseForecastStart + (k + 1) months.
//
// Returns an empty slice for an empty series rather than guessing a level.
func SeasonalEMA(completeMonths []Observation, baseForecastStart time.Time, numMonths int, zScore float64) []Point {
	observations := len(completeMonths)
	if observations == 0 {
		return []Point{}
	}

	// 1. Seasonal factors.
	type seasonalAgg struct {
		total float64
		count int
	}
	seasonalTotals := make(map[time.Month]*seasonalAgg)
	var overallTotal float64
	for _, cm := range completeMonths {
		season := cm.MonthStart.Month()
		agg, ok := seasonalTotals[season]
		if !ok {
			agg = &seasonalAgg{}
			seasonalTotals[season] = agg
		}
		agg.total += cm.Value
		agg.count++
		overallTotal += cm.Value
	}

	overallAverage := 0.0
	if overallTotal > 0 {
		overallAverage = overallTotal / float64(observations)
	}

	seasonalFactors := make(map[time.Month]float64)
	for season, agg := range seasonalTotals {
		if overallAverage > 0 {
			seasonalFactors[season] = (agg.total / float64(agg.count)) / overallAverage
		} else {
			seasonalFactors[season] = 1
		}
	}

	// 2. Deseasonalize.
	deseasonalized := make([]float64, observations)
	for i, cm := range completeMonths {
		factor := seasonalFactors[cm.MonthStart.Month()]
		if factor <= 0 {
			factor = 1
		}
		deseasonalized[i] = cm.Value / factor
	}

	// 3. EMA to a base level.
	smoothingPeriod := min(observations, 12)
	emaAlpha := 2.0 / (float64(smoothingPeriod) + 1.0)
	emaLevel := deseasonalized[0]
	residuals := make([]float64, observations)
	for i := range observations {
		prediction := emaLevel
		if i == 0 {
			prediction = deseasonalized[0]
		}
		residuals[i] = deseasonalized[i] - prediction
		emaLevel = emaAlpha*deseasonalized[i] + (1-emaAlpha)*emaLevel
	}

	// 4. Residual std dev, last 12 only so the EMA warm-up does not inflate the band.
	recentResiduals := residuals
	if len(recentResiduals) > 12 {
		recentResiduals = recentResiduals[len(recentResiduals)-12:]
	}
	var residualStdDev float64
	if len(recentResiduals) > 1 {
		var sumSq float64
		for _, r := range recentResiduals {
			sumSq += r * r
		}
		variance := sumSq / float64(len(recentResiduals)-1)
		if variance > 0 {
			residualStdDev = math.Sqrt(variance)
		}
	}

	// 5. Forecast points.
	forecast := make([]Point, numMonths)
	for idx := range numMonths {
		fMonth := time.Date(baseForecastStart.Year(), baseForecastStart.Month()+time.Month(idx+1), 1, 0, 0, 0, 0, time.UTC)
		displayDate := time.Date(fMonth.Year(), fMonth.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		factor := seasonalFactors[fMonth.Month()]
		if factor <= 0 {
			factor = 1
		}
		baseLevel := emaLevel
		if baseLevel < 0 {
			baseLevel = 0
		}
		forecastMean := baseLevel * factor

		band := residualStdDev * factor * zScore
		lowerBound := forecastMean - band
		if lowerBound < 0 {
			lowerBound = 0
		}

		forecast[idx] = Point{
			Date:       displayDate,
			Forecast:   forecastMean,
			LowerBound: lowerBound,
			UpperBound: forecastMean + band,
		}
	}

	return forecast
}
