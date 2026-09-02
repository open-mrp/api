//go:build e2e

package api_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const analyticsOeePath = "/v1/core/analytics/oee"

// analyzeOee calls the OEE endpoint and fails on anything other than 200.
func analyzeOee(t *testing.T, body map[string]any) map[string]any {
	t.Helper()

	status, respBody, err := apiClient.Put(analyticsOeePath, body)
	require.NoError(t, err)
	require.Less(t, status, 500, "OEE analytics must not 5xx: %s", string(respBody))
	requireStatus(t, 200, status, respBody)
	return parseJSON(respBody)
}

// findOeeDepartment returns the department entry the seeded machine rolls up to, or nil when the window contains no production for it.
func findOeeDepartment(resp map[string]any, departmentID string) map[string]any {
	for _, raw := range jsonListData(resp, "departments") {
		dept, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if jsonField(jsonObject(dept, "department"), "id") == departmentID {
			return dept
		}
	}
	return nil
}

func oeeWindow() (time.Time, time.Time) {
	end := time.Now().UTC()
	return end.Add(-24 * time.Hour), end
}

// ──────────────────────────────────────────────
// Shape and backward compatibility
// ──────────────────────────────────────────────

// The pre-existing fields must keep working: the dashboard consumes them today and the downtime work was meant to be purely additive.
func TestAnalyticsOee_PreservesLegacyShape(t *testing.T) {
	t.Parallel()

	start, end := oeeWindow()
	resp := analyzeOee(t, map[string]any{
		"starts_at": rfc3339(start),
		"ends_at":   rfc3339(end),
	})

	assert.Equal(t, "analyze_oee_response", jsonField(resp, "object"))
	departments := jsonListData(resp, "departments")
	require.NotNil(t, departments, "departments must always be present, even when empty")

	for _, raw := range departments {
		dept, ok := raw.(map[string]any)
		require.True(t, ok)

		department := jsonObject(dept, "department")
		require.NotNil(t, department, "every row must name its department: %v", dept)
		assert.NotEmpty(t, jsonField(department, "id"))
		assert.NotEmpty(t, jsonField(department, "name"))
		for _, field := range []string{"good_units", "waste_units", "seconds_units", "estimated_runtime_hours"} {
			_, ok := dept[field].(float64)
			assert.True(t, ok, "legacy field %q must remain a number", field)
		}
	}
}

func TestAnalyticsOee_ExposesDowntimeFields(t *testing.T) {
	t.Parallel()

	start, end := oeeWindow()
	resp := analyzeOee(t, map[string]any{
		"starts_at": rfc3339(start),
		"ends_at":   rfc3339(end),
	})

	for _, raw := range jsonListData(resp, "departments") {
		dept, ok := raw.(map[string]any)
		require.True(t, ok)

		for _, field := range []string{
			"availability_loss_seconds", "performance_loss_seconds", "quality_loss_seconds",
			"not_scheduled_seconds", "changeover_seconds", "scheduled_seconds", "run_time_seconds",
		} {
			_, ok := dept[field].(float64)
			assert.True(t, ok, "downtime field %q must be a number", field)
		}

		// An estimate must be labelled as one: a department with no logged downtime computes as perfectly available.
		assert.Contains(t, []string{"measured", "estimated"}, jsonField(dept, "measurement_status"),
			"measurement_status must say whether availability was measured or estimated")
		assert.NotNil(t, dept["anomalies"], "anomalies must serialize as [] rather than null")

		assert.NotNil(t, dept["downtime_breakdown"], "downtime_breakdown must serialize as [] rather than null")
	}
}

// Performance is the ideal cycle time of the period's output over the time the department was running. Both terms are on the response, so a caller can check the ratio rather than trust it.
func TestAnalyticsOee_PerformanceIsStandardTimeOverRunTime(t *testing.T) {
	t.Parallel()

	start, end := oeeWindow()
	resp := analyzeOee(t, map[string]any{
		"starts_at": rfc3339(start),
		"ends_at":   rfc3339(end),
	})

	for _, raw := range jsonListData(resp, "departments") {
		dept, ok := raw.(map[string]any)
		require.True(t, ok)

		earned, ok := dept["standard_seconds_earned"].(float64)
		require.True(t, ok, "standard_seconds_earned must be a number")
		runTime, ok := dept["run_time_seconds"].(float64)
		require.True(t, ok, "run_time_seconds must be a number")

		performance := dept["performance_pct"]
		if runTime <= 0 || earned <= 0 {
			assert.Nil(t, performance, "performance_pct must be null without a run time to be fast or slow in")
			continue
		}

		require.NotNil(t, performance, "performance_pct must be present when the department ran and earned standard time")
		assert.InDelta(t, earned/runTime, performance.(float64), 0.0001,
			"performance_pct must equal standard_seconds_earned / run_time_seconds")
	}
}

// ──────────────────────────────────────────────
// The nil-vs-zero rule
// ──────────────────────────────────────────────

// Scheduled time is derived from the account's actual production schedule, not from a shift pattern multiplied out over the window — a formula that counted every calendar week and every machine row whether or not the plant scheduled them, and so reported hours it never planned.
//
// A department has availability exactly when the published plan scheduled it in the window, and none when it did not: a department with no plan has null availability, not a fabricated 100%. The two must never disagree, whatever the seed's schedule state.
func TestAnalyticsOee_DerivesScheduledTimeFromThePublishedPlan(t *testing.T) {
	t.Parallel()

	start, end := oeeWindow()
	resp := analyzeOee(t, map[string]any{
		"starts_at": rfc3339(start),
		"ends_at":   rfc3339(end),
	})

	departments := jsonListData(resp, "departments")
	require.NotEmpty(t, departments, "the seeded account produces in at least one department")

	for _, raw := range departments {
		dept, ok := raw.(map[string]any)
		require.True(t, ok)

		scheduled, _ := dept["scheduled_seconds"].(float64)
		_, hasAvailability := dept["availability_pct"].(float64)

		if scheduled > 0 {
			availability := dept["availability_pct"].(float64)
			assert.GreaterOrEqual(t, availability, float64(0))
			assert.LessOrEqual(t, availability, float64(1),
				"this endpoint reports ratios as fractions, unlike attainment which reports percentages")
		} else {
			assert.False(t, hasAvailability,
				"a department the plan never scheduled has no availability, not a fabricated one: %v", dept["availability_pct"])
		}
	}
}

// Quality needs no planned time — it is good over total produced — so it must still be measured when a department produced anything.
func TestAnalyticsOee_QualityComputedWithoutPlannedTime(t *testing.T) {
	t.Parallel()

	start, end := oeeWindow()
	resp := analyzeOee(t, map[string]any{
		"starts_at": rfc3339(start),
		"ends_at":   rfc3339(end),
	})

	for _, raw := range jsonListData(resp, "departments") {
		dept, ok := raw.(map[string]any)
		require.True(t, ok)

		good, _ := dept["good_units"].(float64)
		waste, _ := dept["waste_units"].(float64)
		if good+waste <= 0 {
			continue // nothing produced, so quality is genuinely unmeasurable
		}

		quality, ok := dept["quality_pct"].(float64)
		require.True(t, ok, "quality must be measured when units were produced")
		assert.InDelta(t, good/(good+waste), quality, 1e-6, "quality must be good / (good + waste)")
	}
}

// ──────────────────────────────────────────────
// Real Availability, driven by logged downtime
// ──────────────────────────────────────────────

// The headline behaviour of B3: logged downtime reduces Availability, and the arithmetic must tie out exactly.
func TestAnalyticsOee_LoggedDowntimeReducesAvailability(t *testing.T) {
	// Not parallel: it asserts on aggregate downtime for the seeded department, which other tests in this package also write to.
	start := time.Now().UTC().Add(-4 * time.Hour)
	end := time.Now().UTC()

	baseline := analyzeOee(t, map[string]any{
		"starts_at":    rfc3339(start),
		"ends_at":      rfc3339(end),
		"planned_time": []map[string]any{{"department_id": SeedDepartmentID, "planned_hours": 8}},
	})
	baselineDept := findOeeDepartment(baseline, SeedDepartmentID)
	var baselineLoss float64
	if baselineDept != nil {
		baselineLoss, _ = baselineDept["availability_loss_seconds"].(float64)
	}

	// One hour of breakdown, fully inside the window.
	downStart := start.Add(time.Hour)
	downEnd := downStart.Add(time.Hour)
	created := logDowntime(t, "breakdown", downStart, &downEnd, nil)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer deleteDowntime(t, id)

	after := analyzeOee(t, map[string]any{
		"starts_at":    rfc3339(start),
		"ends_at":      rfc3339(end),
		"planned_time": []map[string]any{{"department_id": SeedDepartmentID, "planned_hours": 8}},
	})

	dept := findOeeDepartment(after, SeedDepartmentID)
	require.NotNil(t, dept, "the seeded department must appear once it has downtime")

	loss, ok := dept["availability_loss_seconds"].(float64)
	require.True(t, ok)
	assert.InDelta(t, baselineLoss+3600, loss, 5,
		"one hour of breakdown must add 3600s of availability loss")

	assert.Equal(t, "measured", jsonField(dept, "measurement_status"),
		"availability becomes a measurement once downtime is logged")

	// scheduled = planned - not_scheduled; run_time = scheduled - availability_loss.
	scheduled, ok := dept["scheduled_seconds"].(float64)
	require.True(t, ok)
	notScheduled, _ := dept["not_scheduled_seconds"].(float64)
	assert.InDelta(t, 8*3600-notScheduled, scheduled, 1,
		"scheduled time must be planned time net of not-scheduled downtime")

	runTime, ok := dept["run_time_seconds"].(float64)
	require.True(t, ok)
	assert.InDelta(t, scheduled-loss, runTime, 1, "run time must be scheduled minus availability loss")

	availability, ok := dept["availability_pct"].(float64)
	require.True(t, ok, "availability must be computed once planned time is supplied")
	assert.InDelta(t, runTime/scheduled, availability, 1e-6, "availability must be run time over scheduled time")
	assert.Less(t, availability, 1.0, "an hour of downtime must pull availability below 100%")
}

// not_scheduled is removed from the denominator rather than charged as a loss: a machine nobody planned to run has no OEE, which is not the same as bad OEE.
func TestAnalyticsOee_NotScheduledShrinksDenominatorNotAvailability(t *testing.T) {
	start := time.Now().UTC().Add(-5 * time.Hour)
	end := time.Now().UTC()
	planned := []map[string]any{{"department_id": SeedDepartmentID, "planned_hours": 8}}

	downStart := start.Add(time.Hour)
	downEnd := downStart.Add(2 * time.Hour)
	created := logDowntime(t, "no_schedule", downStart, &downEnd, nil)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer deleteDowntime(t, id)

	resp := analyzeOee(t, map[string]any{
		"starts_at":    rfc3339(start),
		"ends_at":      rfc3339(end),
		"planned_time": planned,
	})

	dept := findOeeDepartment(resp, SeedDepartmentID)
	require.NotNil(t, dept)

	notScheduled, ok := dept["not_scheduled_seconds"].(float64)
	require.True(t, ok)
	assert.GreaterOrEqual(t, notScheduled, 7200.0, "two not-scheduled hours must be recorded")

	scheduled, _ := dept["scheduled_seconds"].(float64)
	assert.InDelta(t, 8*3600-notScheduled, scheduled, 1,
		"not-scheduled time must come out of the denominator")
}

// Changeover is an availability reason, so it must charge availability AND be reported separately for the changeover KPI.
func TestAnalyticsOee_ChangeoverCountsInBothPlaces(t *testing.T) {
	start := time.Now().UTC().Add(-3 * time.Hour)
	end := time.Now().UTC()

	downStart := start.Add(30 * time.Minute)
	downEnd := downStart.Add(45 * time.Minute)
	created := logDowntime(t, "changeover", downStart, &downEnd, nil)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer deleteDowntime(t, id)

	resp := analyzeOee(t, map[string]any{
		"starts_at":    rfc3339(start),
		"ends_at":      rfc3339(end),
		"planned_time": []map[string]any{{"department_id": SeedDepartmentID, "planned_hours": 8}},
	})

	dept := findOeeDepartment(resp, SeedDepartmentID)
	require.NotNil(t, dept)

	changeover, ok := dept["changeover_seconds"].(float64)
	require.True(t, ok)
	assert.GreaterOrEqual(t, changeover, 2700.0, "45 minutes of changeover must be reported")

	availabilityLoss, _ := dept["availability_loss_seconds"].(float64)
	assert.GreaterOrEqual(t, availabilityLoss, changeover,
		"changeover is an availability reason, so it must also be inside the availability loss")

	// It must appear in the reason breakdown too, which is what drives the Pareto.
	var found bool
	for _, raw := range jsonListData(dept, "downtime_breakdown") {
		reason, ok := raw.(map[string]any)
		require.True(t, ok)
		if jsonField(reason, "reason") == "changeover" {
			found = true
			assert.Equal(t, "availability", jsonField(reason, "oee_bucket"))
		}
	}
	assert.True(t, found, "changeover must appear in the downtime breakdown")
}

// An event that begins before the window and is still running must contribute only its in-window seconds — not zero, and not its whole length.
func TestAnalyticsOee_ClipsDowntimeToWindow(t *testing.T) {
	start := time.Now().UTC().Add(-2 * time.Hour)
	end := time.Now().UTC()
	planned := []map[string]any{{"department_id": SeedDepartmentID, "planned_hours": 8}}

	// Measure the delta this one event contributes rather than the absolute total, so unrelated downtime in the same window cannot make this pass or fail spuriously.
	before := analyzeOee(t, map[string]any{
		"starts_at": rfc3339(start), "ends_at": rfc3339(end), "planned_time": planned,
	})
	var lossBefore float64
	if dept := findOeeDepartment(before, SeedDepartmentID); dept != nil {
		lossBefore, _ = dept["availability_loss_seconds"].(float64)
	}

	// Starts 6h ago and is still open; the window only covers the last 2h.
	created := logDowntime(t, "breakdown", time.Now().UTC().Add(-6*time.Hour), nil, nil)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer deleteDowntime(t, id)

	after := analyzeOee(t, map[string]any{
		"starts_at": rfc3339(start), "ends_at": rfc3339(end), "planned_time": planned,
	})

	dept := findOeeDepartment(after, SeedDepartmentID)
	require.NotNil(t, dept, "an open event straddling the window must still register")

	lossAfter, ok := dept["availability_loss_seconds"].(float64)
	require.True(t, ok)

	contribution := lossAfter - lossBefore
	assert.Greater(t, contribution, 0.0, "an event overlapping the window must contribute, not be dropped")
	assert.LessOrEqual(t, contribution, 2*3600.0+30,
		"a 6-hour event must be clipped to the 2-hour window, not counted in full")
	assert.InDelta(t, 2*3600.0, contribution, 60,
		"the clipped contribution should be the window length")
}

// ──────────────────────────────────────────────
// Filtering and validation
// ──────────────────────────────────────────────

func TestAnalyticsOee_DepartmentFilter(t *testing.T) {
	t.Parallel()

	start, end := oeeWindow()
	resp := analyzeOee(t, map[string]any{
		"starts_at":      rfc3339(start),
		"ends_at":        rfc3339(end),
		"department_ids": []string{SeedDepartmentID},
	})

	for _, raw := range jsonListData(resp, "departments") {
		dept, ok := raw.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, SeedDepartmentID, jsonField(jsonObject(dept, "department"), "id"),
			"the department filter must exclude everything else")
	}
}

func TestAnalyticsOee_UnknownDepartmentReturnsEmpty(t *testing.T) {
	t.Parallel()

	start, end := oeeWindow()
	resp := analyzeOee(t, map[string]any{
		"starts_at":      rfc3339(start),
		"ends_at":        rfc3339(end),
		"department_ids": []string{"dp_00000000000000000000000000"},
	})

	assert.Empty(t, jsonListData(resp, "departments"),
		"a department with no production must return an empty list, not an error")
}

func TestAnalyticsOee_RejectsMissingDates(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Put(analyticsOeePath, map[string]any{})
	require.NoError(t, err)
	assert.Less(t, status, 500, "a missing date must be a client error, not a 5xx: %s", string(body))
	assert.Equal(t, 400, status, "starts_at and ends_at are required, got %d: %s", status, string(body))
}

// planned_time for a department that produced nothing must not invent a row or crash.
func TestAnalyticsOee_PlannedTimeForUnknownDepartmentIsHarmless(t *testing.T) {
	t.Parallel()

	start, end := oeeWindow()
	resp := analyzeOee(t, map[string]any{
		"starts_at":    rfc3339(start),
		"ends_at":      rfc3339(end),
		"planned_time": []map[string]any{{"department_id": "dp_00000000000000000000000000", "planned_hours": 40}},
	})

	for _, raw := range jsonListData(resp, "departments") {
		dept, ok := raw.(map[string]any)
		require.True(t, ok)
		assert.NotEqual(t, "dp_00000000000000000000000000", jsonField(jsonObject(dept, "department"), "id"),
			"planned time alone must not conjure a department into the results")
	}
}

// Zero planned hours is not a denominator; it must leave the ratios null rather than dividing by zero.
func TestAnalyticsOee_ZeroPlannedHoursLeavesRatiosNull(t *testing.T) {
	t.Parallel()

	start, end := oeeWindow()
	resp := analyzeOee(t, map[string]any{
		"starts_at":    rfc3339(start),
		"ends_at":      rfc3339(end),
		"planned_time": []map[string]any{{"department_id": SeedDepartmentID, "planned_hours": 0}},
	})

	if dept := findOeeDepartment(resp, SeedDepartmentID); dept != nil {
		assertNilField(t, dept, "availability_pct")
		assertNilField(t, dept, "oee_pct")
	}
}
