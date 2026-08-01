//go:build e2e

package api_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scheduleAttainmentPath = "/v1/core/analytics/schedule-attainment"

func analyzeAttainment(t *testing.T, body map[string]any) map[string]any {
	t.Helper()

	status, respBody, err := apiClient.Put(scheduleAttainmentPath, body)
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	return parseJSON(respBody)
}

// A window nobody published over has no plan, which is not the same as a missed one. Reporting 0% would put a phantom failure on the analytics page.
func TestScheduleAttainment_NoBaselineReportsNullNotZero(t *testing.T) {
	t.Parallel()

	result := analyzeAttainment(t, map[string]any{
		"start_date": "2019-01-07T00:00:00Z",
		"end_date":   "2019-02-04T00:00:00Z",
	})

	assert.Equal(t, "no_baseline", jsonField(result, "baseline_status"),
		"nothing was published over 2019, so there is no baseline")
	totals := jsonObject(result, "totals")
	require.NotNil(t, totals, "totals must always be present: %v", result)
	assert.Nil(t, totals["attainment_pct"], "attainment must be null with no plan, never 0")
	assertNilField(t, totals, "output_ratio_pct")

	// Empty collections must still hold mappable data arrays so a client can iterate them.
	assert.NotNil(t, jsonListData(result, "baseline_schedules"))
	assert.NotNil(t, jsonListData(result, "buckets"))
	assert.NotNil(t, jsonListData(result, "frozen_adherence"))
}

func TestScheduleAttainment_RejectsInvertedPeriod(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Put(scheduleAttainmentPath, map[string]any{
		"start_date": "2026-08-01T00:00:00Z",
		"end_date":   "2026-07-01T00:00:00Z",
	})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assert.Contains(t, string(body), "end_date")
}

func TestScheduleAttainment_RejectsUnknownGroupBy(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Put(scheduleAttainmentPath, map[string]any{
		"start_date": "2026-07-01T00:00:00Z",
		"end_date":   "2026-08-01T00:00:00Z",
		"group_by":   "shift",
	})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func TestScheduleAttainment_GroupByDimensionsAllRespond(t *testing.T) {
	t.Parallel()

	for _, groupBy := range []string{"week", "machine", "department", "item"} {
		t.Run(groupBy, func(t *testing.T) {
			t.Parallel()

			result := analyzeAttainment(t, map[string]any{
				"start_date": time.Now().UTC().AddDate(0, -2, 0).Format(time.RFC3339),
				"end_date":   time.Now().UTC().Format(time.RFC3339),
				"group_by":   groupBy,
			})

			assert.Equal(t, groupBy, jsonField(result, "group_by"),
				"the response must echo the grouping it was computed with")
			assert.NotNil(t, result["totals"])
		})
	}
}

// The whole point of publishing: what the floor was asked to build becomes measurable.
func TestScheduleAttainment_MeasuresAgainstPublishedPlan(t *testing.T) {
	t.Parallel()
	lockPublishing(t)
	schedule := ownedSchedule(t, uniqueName("e2e-attainment"))
	scheduleID := jsonField(schedule, "id")

	addLine(t, scheduleID, map[string]any{"week_index": 0, "quantity": 500})

	status, body, err := apiClient.Put(schedulePath(scheduleID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	published := parseJSON(body)
	require.Equal(t, "published", jsonField(published, "status"))

	// The window has to reach back before this version's horizon began. A schedule is only a baseline for weeks that started AFTER it was published, so the week it was published in is deliberately not measured against it.
	result := analyzeAttainment(t, map[string]any{
		"start_date": time.Now().UTC().AddDate(0, 0, -21).Format(time.RFC3339),
		"end_date":   time.Now().UTC().AddDate(0, 0, 21).Format(time.RFC3339),
		"group_by":   "week",
	})

	assert.Equal(t, "measured", jsonField(result, "baseline_status"),
		"a published version covering the window is a baseline")

	// Frozen adherence is reported per baseline, sourced from the counts snapshotted at publish rather than recomputed.
	adherence := jsonListData(result, "frozen_adherence")
	require.NotNil(t, adherence, "frozen_adherence must always carry a data array: %v", result)
	if len(adherence) > 0 {
		entry, ok := adherence[0].(map[string]any)
		require.True(t, ok)
		assert.NotNil(t, entry["frozen_line_count"])
		assert.NotNil(t, entry["frozen_planned_quantity"])
	}
}

// Frozen adherence is measured from the deviation log, so breaking a commitment has to move the number.
func TestScheduleAttainment_FrozenEditReducesAdherence(t *testing.T) {
	t.Parallel()
	lockPublishing(t)
	schedule := ownedSchedule(t, uniqueName("e2e-adherence"))
	scheduleID := jsonField(schedule, "id")

	line := addLine(t, scheduleID, map[string]any{"week_index": 0, "quantity": 500})
	lineID := jsonField(line, "id")

	status, body, err := apiClient.Put(schedulePath(scheduleID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	frozenLineCount, ok := parseJSON(body)["frozen_line_count"].(float64)
	require.True(t, ok)
	require.Greater(t, frozenLineCount, float64(0), "the week-0 line must have frozen")

	// Break the commitment.
	status, body, err = apiClient.Patch(schedulePath(scheduleID)+"/lines/"+lineID,
		map[string]any{"quantity": 300, "reason": "material_shortage"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	result := analyzeAttainment(t, map[string]any{
		"start_date": time.Now().UTC().AddDate(0, 0, -21).Format(time.RFC3339),
		"end_date":   time.Now().UTC().AddDate(0, 0, 21).Format(time.RFC3339),
	})

	adherence := jsonListData(result, "frozen_adherence")
	require.NotNil(t, adherence, "frozen_adherence must always carry a data array: %v", result)

	var found map[string]any
	for _, raw := range adherence {
		entry, ok := raw.(map[string]any)
		if ok && jsonField(jsonObject(entry, "schedule"), "id") == scheduleID {
			found = entry
		}
	}
	require.NotNil(t, found, "the published version must appear in frozen adherence")

	deviated, ok := found["deviated_lines"].(float64)
	require.True(t, ok)
	assert.GreaterOrEqual(t, deviated, float64(1), "the frozen edit must count against adherence")

	absDelta, ok := found["abs_delta_units"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 200, absDelta, 0.001, "200 units were pulled out of the frozen week")

	if lineAdherence, ok := found["line_adherence_pct"].(float64); ok {
		assert.Less(t, lineAdherence, float64(100),
			"a broken commitment cannot be 100% adherent")
	}
}
