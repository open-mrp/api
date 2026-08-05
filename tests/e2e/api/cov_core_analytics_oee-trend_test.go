//go:build e2e

package api_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const analyticsOeeTrendPath = "/v1/core/analytics/oee-trend"

// analyzeOeeTrend calls the OEE trend endpoint and fails on anything other than 200.
func analyzeOeeTrend(t *testing.T, body map[string]any) map[string]any {
	t.Helper()

	status, respBody, err := apiClient.Put(analyticsOeeTrendPath, body)
	require.NoError(t, err)
	require.Less(t, status, 500, "OEE trend analytics must not 5xx: %s", string(respBody))
	requireStatus(t, 200, status, respBody)
	return parseJSON(respBody)
}

// oeeTrendWindow is four whole production weeks ending now, which is long enough that the bucketing is visible and short enough to stay inside the seeded data.
func oeeTrendWindow() (time.Time, time.Time) {
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -28)
	return start, end
}

func TestAnalyticsOeeTrend_ReturnsOnePeriodPerWeek(t *testing.T) {
	t.Parallel()

	start, end := oeeTrendWindow()
	resp := analyzeOeeTrend(t, map[string]any{
		"starts_at": rfc3339(start),
		"ends_at":   rfc3339(end),
	})

	assert.Equal(t, "analyze_oee_trend_response", jsonField(resp, "object"))
	periods := jsonListData(resp, "periods")
	require.NotNil(t, periods, "periods must always be present, even when empty")
	// 28 days starting mid-week spans five partial-or-whole production weeks at most.
	assert.LessOrEqual(t, len(periods), 5)
	assert.GreaterOrEqual(t, len(periods), 4)
}

// Periods must be contiguous and ordered: a chart draws them in the order they arrive, and a gap or an overlap would silently misplace a week.
func TestAnalyticsOeeTrend_PeriodsAreContiguousAndOrdered(t *testing.T) {
	t.Parallel()

	start, end := oeeTrendWindow()
	resp := analyzeOeeTrend(t, map[string]any{
		"starts_at": rfc3339(start),
		"ends_at":   rfc3339(end),
	})

	var previousEnd time.Time
	for i, raw := range jsonListData(resp, "periods") {
		period, ok := raw.(map[string]any)
		require.True(t, ok)

		periodStart, err := time.Parse(time.RFC3339, jsonField(period, "starts_at"))
		require.NoError(t, err)
		periodEnd, err := time.Parse(time.RFC3339, jsonField(period, "ends_at"))
		require.NoError(t, err)

		assert.True(t, periodEnd.After(periodStart), "period %d must cover time", i)
		if i == 0 {
			assert.WithinDuration(t, start, periodStart, time.Second, "the first period starts where the window does")
		} else {
			assert.Equal(t, previousEnd, periodStart, "period %d must start where period %d ended", i, i-1)
		}
		previousEnd = periodEnd
	}

	if !previousEnd.IsZero() {
		assert.WithinDuration(t, end, previousEnd, time.Second, "the last period ends where the window does")
	}
}

// The trend answers with the same arithmetic as the per-department table: Performance is standard time earned over run time, and every ratio is null rather than zero when its denominator is unknown.
func TestAnalyticsOeeTrend_RatiosMatchTheOeeArithmetic(t *testing.T) {
	t.Parallel()

	start, end := oeeTrendWindow()
	resp := analyzeOeeTrend(t, map[string]any{
		"starts_at": rfc3339(start),
		"ends_at":   rfc3339(end),
	})

	for _, raw := range jsonListData(resp, "periods") {
		period, ok := raw.(map[string]any)
		require.True(t, ok)

		for _, field := range []string{
			"good_units", "waste_units", "seconds_units", "standard_seconds_earned",
			"scheduled_seconds", "run_time_seconds", "availability_loss_seconds", "not_scheduled_seconds",
		} {
			_, ok := period[field].(float64)
			assert.True(t, ok, "%q must be a number", field)
		}

		assert.Contains(t, []string{"measured", "estimated"}, jsonField(period, "measurement_status"))
		_, ok = period["downtime_event_count"].(float64)
		assert.True(t, ok, "downtime_event_count must be a number")

		scheduled := period["scheduled_seconds"].(float64)
		runTime := period["run_time_seconds"].(float64)
		earned := period["standard_seconds_earned"].(float64)

		if scheduled > 0 {
			require.NotNil(t, period["availability_pct"], "a scheduled week has availability")
			assert.InDelta(t, runTime/scheduled, period["availability_pct"].(float64), 0.0001)
		} else {
			assert.Nil(t, period["availability_pct"], "an unscheduled week has no availability, which is not 0%")
			assert.Nil(t, period["oee_pct"])
		}

		units := period["good_units"].(float64) + period["waste_units"].(float64) + period["seconds_units"].(float64)
		if units > 0 {
			require.NotNil(t, period["quality_pct"], "a period that produced units has quality")
			assert.InDelta(t, period["good_units"].(float64)/units, period["quality_pct"].(float64), 0.0001)
		} else {
			assert.Nil(t, period["quality_pct"], "a period that produced nothing has no quality, which is not 0%")
		}

		if runTime > 0 && earned > 0 {
			require.NotNil(t, period["performance_pct"])
			assert.InDelta(t, earned/runTime, period["performance_pct"].(float64), 0.0001,
				"performance_pct must equal standard_seconds_earned / run_time_seconds")
		} else {
			assert.Nil(t, period["performance_pct"])
		}

		if period["oee_pct"] != nil {
			expected := period["availability_pct"].(float64) * period["performance_pct"].(float64) * period["quality_pct"].(float64)
			assert.InDelta(t, expected, period["oee_pct"].(float64), 0.0001, "oee_pct must be the product of the three terms")
		}
	}
}

// A window with no time in it is an empty answer, not an error: the page asks for whatever range the user picked.
func TestAnalyticsOeeTrend_EmptyWindowReturnsNoPeriods(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	resp := analyzeOeeTrend(t, map[string]any{
		"starts_at": rfc3339(now),
		"ends_at":   rfc3339(now),
	})

	assert.Empty(t, jsonListData(resp, "periods"))
}

// Filtering to a department that produces nothing must not error, and must not invent periods with output in them.
func TestAnalyticsOeeTrend_UnknownDepartmentFilterYieldsNoOutput(t *testing.T) {
	t.Parallel()

	start, end := oeeTrendWindow()
	resp := analyzeOeeTrend(t, map[string]any{
		"starts_at":      rfc3339(start),
		"ends_at":        rfc3339(end),
		"department_ids": []string{"dp_does_not_exist"},
	})

	for _, raw := range jsonListData(resp, "periods") {
		period, ok := raw.(map[string]any)
		require.True(t, ok)
		assert.Zero(t, period["scheduled_seconds"])
		assert.Zero(t, period["good_units"])
		assert.Nil(t, period["oee_pct"])
	}
}
