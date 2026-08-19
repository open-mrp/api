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

// The engine now carries a trend, but a damped one: a rising series keeps rising, and the increments shrink toward zero instead of compounding. Runaway growth is the property Holt-Winters was dropped for.
func TestSeasonalEMA_TrendIsDampedNotCompounding(t *testing.T) {
	t.Parallel()

	observations := make([]Observation, 24)
	for i := range 24 {
		observations[i] = Observation{
			MonthStart: time.Date(2024, time.June+time.Month(i), 1, 0, 0, 0, 0, time.UTC),
			Value:      float64(100 + i*10), // steadily rising
		}
	}

	points := SeasonalEMA(observations, month(2026, time.May), 12, 1)

	var previousStep float64
	for i := 1; i < len(points); i++ {
		step := points[i].Forecast - points[i-1].Forecast
		if step < 0 {
			t.Errorf("point %d fell below its predecessor; a rising series should keep rising", i)
		}
		if i > 1 && step > previousStep+1e-9 {
			t.Errorf("point %d step = %.4f, previous = %.4f; steps must shrink, not compound", i, step, previousStep)
		}
		previousStep = step
	}

	// Damping bounds the whole projection at level + slope*phi/(1-phi), so twelve months out cannot run away from the last observed month.
	last := observations[len(observations)-1].Value
	if points[len(points)-1].Forecast > last*2 {
		t.Errorf("12-month forecast = %.1f against a last observation of %.1f; damping should bound this", points[len(points)-1].Forecast, last)
	}
}

// The reason the trend term exists: with no trend the level lagged a shrinking book and the forecast sat above every actual month, indefinitely.
func TestSeasonalEMA_FollowsADecliningSeries(t *testing.T) {
	t.Parallel()

	observations := make([]Observation, 24)
	for i := range 24 {
		observations[i] = Observation{
			MonthStart: time.Date(2024, time.June+time.Month(i), 1, 0, 0, 0, 0, time.UTC),
			Value:      float64(1000 - i*20), // steadily falling, 1000 -> 540
		}
	}

	points := SeasonalEMA(observations, month(2026, time.May), 6, 1)

	// The smoothed level alone sits near the trailing average, which on a falling series is well above where demand actually is. The trend term has to pull the forecast below it.
	var trailingTotal float64
	for _, o := range observations[len(observations)-12:] {
		trailingTotal += o.Value
	}
	trailingAverage := trailingTotal / 12

	if points[0].Forecast >= trailingAverage {
		t.Errorf("first forecast = %.1f, want below the trailing average (%.1f); this is the lag the trend term exists to remove", points[0].Forecast, trailingAverage)
	}
	if points[5].Forecast >= points[0].Forecast {
		t.Errorf("forecast is not declining: point 5 = %.1f, point 0 = %.1f", points[5].Forecast, points[0].Forecast)
	}
	// Damping means the forecast still lags a perfectly linear decline rather than extrapolating it exactly; it must not overshoot into collapse either.
	if points[5].Forecast < observations[len(observations)-1].Value/2 {
		t.Errorf("point 5 = %.1f, want no worse than half the last observed month; damping should prevent a collapse", points[5].Forecast)
	}
}

// Seasonal factors are shrunk toward 1 before use: most items sell in a handful of months a year, so a raw factor is mostly memorised noise. The lift must survive, at a fraction of its raw amplitude.
func TestSeasonalEMA_AppliesShrunkSeasonalFactors(t *testing.T) {
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

	// Forecast from May 2026: point 6 covers December 2026, point 5 covers November.
	points := SeasonalEMA(observations, month(2026, time.May), 8, 1)
	december, november := points[6].Forecast, points[5].Forecast

	if december <= november {
		t.Errorf("December forecast = %.1f, November = %.1f; the seasonal lift must survive shrinkage", december, november)
	}
	// Raw factors would put December near 3x a normal month. Shrinkage deliberately keeps only part of that.
	if ratio := december / november; ratio > 2.5 {
		t.Errorf("December/November = %.2f, want well under the raw 3x; factors should be shrunk", ratio)
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
