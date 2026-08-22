// Package forecast holds OpenMRP's standard monthly demand forecaster.
//
// Seasonal EMA is used by both the analytics demand-forecast endpoint and the production scheduler, so it lives here rather than in either caller. It mirrors dashboard/packages/utils/src/seasonal-ema.ts; the two must stay in step or the schedule and the dashboard will disagree about the same demand.
//
// Method:
//  1. Seasonal factors — each calendar month's average divided by the overall average, then shrunk toward 1. Most items sell in only a handful of months a year, so the raw factors fit twelve parameters to a few observations and largely memorise noise; shrinking keeps whatever shape is real and discards the amplitude.
//  2. Deseasonalize, then smooth to a current base run rate (alpha = 2 / (min(observations, 12) + 1), so recent months weigh more) carrying a damped additive trend.
//  3. Forecast = (level + damped slope) x seasonal factor.
//  4. Fixed-width confidence band from the last 12 residuals x seasonal factor x z.
//
// The trend term is additive on a deseasonalized series and damped, so it converges to level + slope*phi/(1-phi) instead of compounding. This is not the Holt-Winters that was removed: that trend was multiplicative and produced runaway exponential forecasts. Without any trend at all the forecaster over-forecast a declining book by roughly a fifth, month after month, because nothing in the model could lower the level fast enough to keep up.
package forecast

import (
	"math"
	"time"
)

const (
	// seasonalShrink raises each factor to this power. These three constants were fitted together against 25 rolling backtests; the surface around them is flat, so none is load-bearing to a decimal place.
	seasonalShrink = 0.35
	// trendDamping is how much of the slope survives each month projected forward.
	trendDamping = 0.90
	// trendSmoothing is the slope's own learning rate, kept well below the level's so a single odd month reads as noise rather than a change of direction.
	trendSmoothing = 0.20
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

	seasonalFactors := shrunkSeasonalFactors(completeMonths)

	// Deseasonalize.
	deseasonalized := make([]float64, observations)
	for i, cm := range completeMonths {
		deseasonalized[i] = cm.Value / factorFor(seasonalFactors, cm.MonthStart.Month())
	}

	// Smooth to a base level and slope. The slope is what lets the forecast follow a book that is growing or shrinking; residuals are measured against the prediction the model would have made, so they capture the trend's error too.
	smoothingPeriod := min(observations, 12)
	emaAlpha := 2.0 / (float64(smoothingPeriod) + 1.0)
	emaLevel := deseasonalized[0]
	var emaSlope float64
	residuals := make([]float64, observations)
	for i := range observations {
		prediction := emaLevel + trendDamping*emaSlope
		if i == 0 {
			prediction = deseasonalized[0]
		}
		residuals[i] = deseasonalized[i] - prediction

		previousLevel := emaLevel
		emaLevel = emaAlpha*deseasonalized[i] + (1-emaAlpha)*(previousLevel+trendDamping*emaSlope)
		emaSlope = trendSmoothing*(emaLevel-previousLevel) + (1-trendSmoothing)*trendDamping*emaSlope
	}

	// Residual std dev, last 12 only so the warm-up does not inflate the band.
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

	// Forecast points.
	forecast := make([]Point, numMonths)
	dampedSlope := 0.0
	damping := 1.0
	for idx := range numMonths {
		fMonth := time.Date(baseForecastStart.Year(), baseForecastStart.Month()+time.Month(idx+1), 1, 0, 0, 0, 0, time.UTC)
		displayDate := time.Date(fMonth.Year(), fMonth.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		factor := factorFor(seasonalFactors, fMonth.Month())

		damping *= trendDamping
		dampedSlope += damping * emaSlope

		baseLevel := emaLevel + dampedSlope
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

// shrunkSeasonalFactors is each calendar month's average over the overall average, pulled toward 1 by seasonalShrink.
//
// A month whose average is zero, or a series whose overall average is zero, yields a factor of 1: there is nothing to say about that month's seasonality, and a factor of 0 would erase the level rather than shape it.
func shrunkSeasonalFactors(completeMonths []Observation) map[time.Month]float64 {
	type seasonalAgg struct {
		total float64
		count int
	}
	seasonalTotals := make(map[time.Month]*seasonalAgg)
	var overallTotal float64
	for _, cm := range completeMonths {
		agg, ok := seasonalTotals[cm.MonthStart.Month()]
		if !ok {
			agg = &seasonalAgg{}
			seasonalTotals[cm.MonthStart.Month()] = agg
		}
		agg.total += cm.Value
		agg.count++
		overallTotal += cm.Value
	}

	overallAverage := 0.0
	if overallTotal > 0 {
		overallAverage = overallTotal / float64(len(completeMonths))
	}

	factors := make(map[time.Month]float64, len(seasonalTotals))
	for season, agg := range seasonalTotals {
		raw := 1.0
		if overallAverage > 0 {
			raw = (agg.total / float64(agg.count)) / overallAverage
		}
		if raw <= 0 {
			factors[season] = 1
			continue
		}
		factors[season] = math.Pow(raw, seasonalShrink)
	}
	return factors
}

func factorFor(factors map[time.Month]float64, season time.Month) float64 {
	f, ok := factors[season]
	if !ok || f <= 0 {
		return 1
	}
	return f
}
