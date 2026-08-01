//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func derivedLines(t *testing.T, scheduleID string, params url.Values) []map[string]any {
	t.Helper()

	status, body, err := apiClient.GetListRaw(schedulePath(scheduleID)+"/derived-lines", params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	out := []map[string]any{}
	for _, raw := range jsonArray(parseJSON(body), "data") {
		if row, ok := raw.(map[string]any); ok {
			out = append(out, row)
		}
	}
	return out
}

// The solver only schedules the constraint; every other department's work has to follow from it, or the plan tells most of the factory nothing.
func TestDerivedLines_GeneratedFromTheConstraintPlan(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-derived"))
	scheduleID := jsonField(schedule, "id")

	rows := derivedLines(t, scheduleID, nil)
	require.NotEmpty(t, rows, "a plan with constraint campaigns must derive downstream work")

	sourceLineIDs := map[string]bool{}
	for _, raw := range jsonArray(parseJSON(mustGetBody(t, schedulePath(scheduleID)+"/lines")), "data") {
		if line, ok := raw.(map[string]any); ok {
			sourceLineIDs[jsonField(line, "id")] = true
		}
	}

	for _, row := range rows {
		assert.Equal(t, "production_schedule_derived_line", jsonField(row, "object"))
		schedule := jsonObject(row, "production_schedule")
		require.NotNil(t, schedule, "derived work must name its schedule version: %v", row)
		assert.Equal(t, scheduleID, jsonField(schedule, "id"))
		step := jsonObject(row, "production_step")
		require.NotNil(t, step, "derived work must name the step that does it: %v", row)
		assert.NotEmpty(t, jsonField(step, "id"))

		// Every derived row must trace back to a real constraint campaign; work that followed from nothing would be work nobody asked for.
		sourceLine := jsonObject(row, "source_line")
		require.NotNil(t, sourceLine, "derived work must carry its source line: %v", row)
		assert.True(t, sourceLineIDs[jsonField(sourceLine, "id")],
			"derived work must trace back to a constraint campaign on this version")

		depth, ok := row["explosion_depth"].(float64)
		require.True(t, ok)
		assert.GreaterOrEqual(t, depth, float64(1),
			"the constraint itself is depth 0; derived work starts at 1")
	}
}

func TestDerivedLines_FiltersByDepartment(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-derived-dept"))
	scheduleID := jsonField(schedule, "id")

	all := derivedLines(t, scheduleID, nil)
	require.NotEmpty(t, all)

	var target string
	for _, row := range all {
		if id := jsonField(jsonObject(row, "department"), "id"); id != "" {
			target = id
			break
		}
	}
	require.NotEmpty(t, target, "at least one derived row should carry a department")

	params := url.Values{}
	params.Set("department_ids[]", target)

	filtered := derivedLines(t, scheduleID, params)
	require.NotEmpty(t, filtered, "the department filter must not empty the work list")
	for _, row := range filtered {
		assert.Equal(t, target, jsonField(jsonObject(row, "department"), "id"),
			"the department filter must exclude other departments")
	}
	assert.LessOrEqual(t, len(filtered), len(all))
}

func TestDerivedLines_FiltersByWeek(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-derived-week"))
	scheduleID := jsonField(schedule, "id")

	all := derivedLines(t, scheduleID, nil)
	require.NotEmpty(t, all)

	week, ok := all[0]["week_index"].(float64)
	require.True(t, ok)

	params := url.Values{}
	params.Set("week_index", jsonNumber(week))

	filtered := derivedLines(t, scheduleID, params)
	require.NotEmpty(t, filtered)
	for _, row := range filtered {
		assert.Equal(t, week, row["week_index"], "the week filter must exclude other weeks")
	}
}

// Derived work is a pure function of the constraint plan, so regenerating must not accumulate rows from previous runs.
func TestDerivedLines_RegeneratedNotAccumulated(t *testing.T) {
	t.Parallel()

	first := ownedSchedule(t, uniqueName("e2e-derived-regen-1"))
	firstRows := derivedLines(t, jsonField(first, "id"), nil)
	require.NotEmpty(t, firstRows)

	second := ownedSchedule(t, uniqueName("e2e-derived-regen-2"))
	secondRows := derivedLines(t, jsonField(second, "id"), nil)

	// Two versions of the same plan derive the same amount of work; a version that accumulated rows would grow without bound across regenerations.
	assert.Equal(t, len(firstRows), len(secondRows),
		"two versions of the same plan must derive the same amount of work")
}

// mustGetBody fetches a path and fails the test on anything but 200.
func mustGetBody(t *testing.T, path string) []byte {
	t.Helper()
	status, body, err := apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return body
}

func jsonNumber(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
