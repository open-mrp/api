//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func listFinishingLines(t *testing.T, scheduleID string, params url.Values) []map[string]any {
	t.Helper()

	status, body, err := apiClient.GetListRaw(schedulePath(scheduleID)+"/finishing-lines", params)
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

// The point of stage two: the plan says which finished good to make, not just how much greige to knit.
//
// Solved fresh rather than read off the seeded fixture, whose diagnostics blob was written before the second stage existed and would make every assertion here vacuous.
func TestFinishingLines_ShapeAndParentage(t *testing.T) {
	t.Parallel()

	scheduleID := jsonField(ownedSchedule(t, uniqueName("e2e-finishing-shape")), "id")

	lines := listFinishingLines(t, scheduleID, nil)
	require.NotEmpty(t, lines, "a solved plan must decide what to make from what it knits")

	for _, line := range lines {
		assert.Equal(t, "production_schedule_finishing_line", jsonField(line, "object"))

		// Every finishing line names both ends of the conversion, or the two stages cannot be reconciled.
		item := jsonObject(line, "item")
		require.NotNil(t, item, "a finishing line must name the finished good: %v", line)
		assert.NotEmpty(t, jsonField(item, "id"))
		assert.NotEmpty(t, jsonField(line, "sku"))

		greigeItem := jsonObject(line, "greige_item")
		require.NotNil(t, greigeItem, "a finishing line must name the greige it comes from: %v", line)
		assert.NotEmpty(t, jsonField(greigeItem, "id"))
		assert.NotEmpty(t, jsonField(line, "greige_sku"))

		quantity, ok := line["planned_quantity"].(float64)
		require.True(t, ok, "planned_quantity must be numeric: %v", line)
		assert.Positive(t, quantity, "a line that makes nothing is not a line")

		consumed, ok := line["greige_consumed"].(float64)
		require.True(t, ok)
		assert.Positive(t, consumed, "making a finished good has to draw greige down")

		hours, ok := line["planned_run_hours"].(float64)
		require.True(t, ok)
		assert.Positive(t, hours, "a levelled line has to cost the department hours, or capacity means nothing")
	}
}

// Levelled means the plan never asks the second stage for more hours than it has in a week.
func TestFinishingLines_NoWeekExceedsTheStagesCapacity(t *testing.T) {
	t.Parallel()

	scheduleID := jsonField(ownedSchedule(t, uniqueName("e2e-finishing-cap")), "id")

	lines := listFinishingLines(t, scheduleID, nil)
	require.NotEmpty(t, lines)

	capacity := finishingCapacityHours(t, scheduleID)
	require.Positive(t, capacity, "a levelled stage must report the capacity it was levelled against")

	hoursByWeek := map[float64]float64{}
	for _, line := range lines {
		week, _ := line["week_index"].(float64)
		hours, _ := line["planned_run_hours"].(float64)
		hoursByWeek[week] += hours
	}

	for week, hours := range hoursByWeek {
		assert.LessOrEqual(t, hours, capacity+1e-6,
			"week %.0f plans %.2f hours against a capacity of %.2f", week, hours, capacity)
	}
}

// The filters are what a supervisor reads their own week by, and what a planner reads one SKU's run by.
func TestFinishingLines_FiltersNarrowTheStage(t *testing.T) {
	t.Parallel()

	scheduleID := jsonField(ownedSchedule(t, uniqueName("e2e-finishing-filter")), "id")

	all := listFinishingLines(t, scheduleID, nil)
	require.NotEmpty(t, all)

	week, _ := all[0]["week_index"].(float64)
	byWeek := listFinishingLines(t, scheduleID,
		url.Values{"week_index": {strconv.Itoa(int(week))}})
	require.NotEmpty(t, byWeek, "filtering to a week that has lines must return them")
	for _, line := range byWeek {
		got, _ := line["week_index"].(float64)
		assert.InDelta(t, week, got, 1e-9)
	}

	item := jsonObject(all[0], "item")
	require.NotNil(t, item)
	itemID := jsonField(item, "id")
	byItem := listFinishingLines(t, scheduleID, url.Values{"item_id": {itemID}})
	require.NotEmpty(t, byItem)
	for _, line := range byItem {
		got := jsonObject(line, "item")
		require.NotNil(t, got)
		assert.Equal(t, itemID, jsonField(got, "id"))
	}
}

// A filter that matches nothing returns an empty list rather than everything, which is the failure mode an unwired filter has.
func TestFinishingLines_UnmatchedFilterReturnsNothing(t *testing.T) {
	t.Parallel()

	lines := listFinishingLines(t, SeedProductionScheduleID,
		url.Values{"item_id": {"it_01definitelynotreal000"}})
	assert.Empty(t, lines)
}

// The two starvation lists are the only thing a two-stage plan knows that a one-stage plan does not, and they call for opposite responses: knit more, or find more hours.
func TestFinishingLines_DiagnosticsExplainTheStage(t *testing.T) {
	t.Parallel()

	scheduleID := jsonField(ownedSchedule(t, uniqueName("e2e-finishing-diag")), "id")

	diagnostics := finishingDiagnostics(t, scheduleID)
	require.NotNil(t, diagnostics, "every version must carry a finishing section in its diagnostics")

	for _, key := range []string{"greige_starved_skus", "capacity_starved_skus", "items_without_run_rate"} {
		_, ok := diagnostics[key]
		assert.True(t, ok, "%s must be present so an empty stage can explain itself: %v", key, diagnostics)
	}

	capacity, ok := diagnostics["weekly_capacity_hours"].(float64)
	require.True(t, ok, "the stage must report the capacity it was levelled against")
	assert.Positive(t, capacity)
}

// finishingDiagnostics reads the finishing section off a version's stored diagnostics.
func finishingDiagnostics(t *testing.T, scheduleID string) map[string]any {
	t.Helper()

	status, body, err := apiClient.GetListRaw(schedulePath(scheduleID), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	diagnostics := jsonObject(parseJSON(body), "diagnostics")
	require.NotNil(t, diagnostics, "a version must carry its diagnostics")
	return jsonObject(diagnostics, "finishing")
}

func finishingCapacityHours(t *testing.T, scheduleID string) float64 {
	t.Helper()
	capacity, _ := finishingDiagnostics(t, scheduleID)["weekly_capacity_hours"].(float64)
	return capacity
}

// The two stages have to reconcile: what stage two draws out of the greige buffer can never exceed what stage one puts in plus what was already there.
//
// This is the claim the whole two-stage model rests on. A finishing plan that consumed greige the knit plan never makes would be a plan the floor cannot work, and it would look perfectly reasonable on screen.
func TestFinishingLines_ConsumptionNeverExceedsTheKnitPlan(t *testing.T) {
	t.Parallel()

	scheduleID := jsonField(ownedSchedule(t, uniqueName("e2e-finishing-reconcile")), "id")

	consumedByGreige := map[string]float64{}
	for _, line := range listFinishingLines(t, scheduleID, nil) {
		greige := jsonObject(line, "greige_item")
		require.NotNil(t, greige)
		consumed, _ := line["greige_consumed"].(float64)
		consumedByGreige[jsonField(greige, "id")] += consumed
	}
	require.NotEmpty(t, consumedByGreige, "a solved plan must draw on the greige it knits")

	// Everything stage one plans, plus the greige already on hand when the horizon opened.
	plannedByGreige := map[string]float64{}
	for _, line := range listScheduleLines(t, scheduleID) {
		item := jsonObject(line, "item")
		require.NotNil(t, item)
		quantity, _ := line["planned_quantity"].(float64)
		plannedByGreige[jsonField(item, "id")] += quantity
	}

	onHandByGreige := map[string]float64{}
	for _, policy := range listItemPolicies(t, scheduleID) {
		item := jsonObject(policy, "item")
		require.NotNil(t, item)
		onHand, _ := policy["on_hand_greige"].(float64)
		onHandByGreige[jsonField(item, "id")] = onHand
	}

	for greigeID, consumed := range consumedByGreige {
		available := plannedByGreige[greigeID] + onHandByGreige[greigeID]
		assert.LessOrEqual(t, consumed, available+1e-6,
			"the finishing plan draws %.2f of %s against %.2f knitted and on hand",
			consumed, greigeID, available)
	}
}
