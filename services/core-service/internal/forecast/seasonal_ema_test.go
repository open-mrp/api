package forecast

import (
	"math"
	"testing"
	"time"
)

func month(year int, m time.Month) time.Time {
	return time.Date(year, m, 1, 0, 0, 0, 0, time.UTC)
}

// flat builds n months of identical demand ending at the given month.
func flat(endYear int, endMonth time.Month, n int, value float64) []Observation {
	out := make([]Observation, n)
	for i := range n {
		out[i] = Observation{
			MonthStart: time.Date(endYear, endMonth-time.Month(n-1-i), 1, 0, 0, 0, 0, time.UTC),
			Value:      value,
		}
	}
	return out
}

func TestSeasonalEMA_EmptySeriesReturnsNoPoints(t *testing.T) {
	t.Parallel()

	got := SeasonalEMA(nil, month(2026, time.June), 12, 1)
	if len(got) != 0 {
		t.Errorf("len = %d, want 0; an empty series must not produce a guessed level", len(got))
	}
}

func TestSeasonalEMA_FlatSeriesForecastsTheSameLevel(t *testing.T) {
	t.Parallel()

	points := SeasonalEMA(flat(2026, time.May, 24, 100), month(2026, time.May), 3, 1)
	if len(points) != 3 {
		t.Fatalf("len = %d, want 3", len(points))
	}
	for i, p := range points {
		if math.Abs(p.Forecast-100) > 1e-9 {
			t.Errorf("point %d forecast = %v, want 100", i, p.Forecast)
		}
	}
}

// The engine deliberately has no trend extrapolation: a rising series raises the level but the forecast must not keep climbing month over month. This is the property Holt-Winters was dropped for.
func TestSeasonalEMA_DoesNotExtrapolateTrend(t *testing.T) {
	t.Parallel()

	observations := make([]Observation, 24)
	for i := range 24 {
		observations[i] = Observation{
			MonthStart: time.Date(2024, time.June+time.Month(i), 1, 0, 0, 0, 0, time.UTC),
			Value:      float64(100 + i*10), // steadily rising
		}
	}

	points := SeasonalEMA(observations, month(2026, time.May), 6, 1)

	// Seasonality can vary the points, but they must not march upward the way a trend model would. Compare the mean of the first half to the second half.
	var firstHalf, secondHalf float64
	for i := 0; i < 3; i++ {
		firstHalf += points[i].Forecast
	}
	for i := 3; i < 6; i++ {
		secondHalf += points[i].Forecast
	}
	growth := (secondHalf - firstHalf) / firstHalf
	if math.Abs(growth) > 0.25 {
		t.Errorf("second-half/first-half growth = %.3f, want ~0; the model must not extrapolate trend", growth)
	}
}

func TestSeasonalEMA_AppliesSeasonalFactors(t *testing.T) {
	t.Parallel()

	// Two years where December is triple every other month.
	observations := make([]Observation, 24)
	for i := range 24 {
		start := time.Date(2024, time.June+time.Month(i), 1, 0, 0, 0, 0, time.UTC)
		value := 100.0
		if start.Month() == time.December {
			value = 300
		}
		observations[i] = Observation{MonthStart: start, Value: value}
	}

	// Forecast from May 2026: point 6 covers December 2026.
	points := SeasonalEMA(observations, month(2026, time.May), 8, 1)
	december := points[6]

	if december.Forecast < 200 {
		t.Errorf("December forecast = %v, want the seasonal lift (>200)", december.Forecast)
	}
}

func TestSeasonalEMA_LowerBoundNeverNegative(t *testing.T) {
	t.Parallel()

	// Volatile series so the residual band is wide relative to the level.
	observations := make([]Observation, 24)
	for i := range 24 {
		value := 10.0
		if i%2 == 0 {
			value = 200
		}
		observations[i] = Observation{
			MonthStart: time.Date(2024, time.June+time.Month(i), 1, 0, 0, 0, 0, time.UTC),
			Value:      value,
		}
	}

	for i, p := range SeasonalEMA(observations, month(2026, time.May), 6, 3) {
		if p.LowerBound < 0 {
			t.Errorf("point %d lower bound = %v, want >= 0; negative demand is not meaningful", i, p.LowerBound)
		}
	}
}

// Point k covers baseForecastStart + (k+1) months and is stamped one further month out. Getting this off by one silently shifts an entire production plan.
func TestSeasonalEMA_PointDatesAreEndOfPeriod(t *testing.T) {
	t.Parallel()

	points := SeasonalEMA(flat(2026, time.May, 12, 50), month(2026, time.May), 2, 1)

	if got, want := points[0].Date, month(2026, time.July); !got.Equal(want) {
		t.Errorf("point 0 date = %v, want %v (June forecast stamped end-of-period)", got, want)
	}
	if got, want := points[1].Date, month(2026, time.August); !got.Equal(want) {
		t.Errorf("point 1 date = %v, want %v", got, want)
	}
}
