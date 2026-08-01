//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func listFinishedPolicies(t *testing.T, scheduleID string) []map[string]any {
	t.Helper()

	status, body, err := apiClient.GetListRaw(schedulePath(scheduleID)+"/finished-policies", nil)
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

func listItemPolicies(t *testing.T, scheduleID string) []map[string]any {
	t.Helper()

	status, body, err := apiClient.GetListRaw(schedulePath(scheduleID)+"/item-policies", nil)
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

func TestFinishedPolicies_ShapeAndParentage(t *testing.T) {
	t.Parallel()

	policies := listFinishedPolicies(t, SeedProductionScheduleID)
	require.NotEmpty(t, policies, "the seeded schedule must carry finished-goods targets")

	for _, policy := range policies {
		assert.Equal(t, "production_schedule_finished_policy", jsonField(policy, "object"))
		// Every finished target has to name the greige it decomposes, or the two stages cannot be reconciled.
		greigeItem := jsonObject(policy, "greige_item")
		require.NotNil(t, greigeItem, "a finished policy must carry its greige item: %v", policy)
		assert.NotEmpty(t, jsonField(greigeItem, "id"))
		assert.NotEmpty(t, jsonField(policy, "greige_sku"))
		assert.NotEmpty(t, jsonField(policy, "sku"))

		for _, key := range []string{"weekly_demand", "sigma_weekly", "safety_stock", "reorder_point", "on_hand"} {
			_, ok := policy[key].(float64)
			assert.True(t, ok, "%s must be present and numeric", key)
		}
	}
}

// The greige stage is reported on its own as well as rolled into the echelon figure. Once summed the echelon total cannot be decomposed back, and "how much greige is there" is exactly the question it hides.
func TestItemPolicies_ReportTheGreigeStageSeparately(t *testing.T) {
	t.Parallel()

	policies := listItemPolicies(t, SeedProductionScheduleID)
	require.NotEmpty(t, policies)

	for _, policy := range policies {
		greige, ok := policy["on_hand_greige"].(float64)
		require.True(t, ok, "on_hand_greige must be present")
		echelon, ok := policy["on_hand_echelon"].(float64)
		require.True(t, ok, "on_hand_echelon must be present")

		// Echelon is the greige stage plus everything downstream of it, so it can never be the smaller of the two.
		assert.LessOrEqual(t, greige, echelon,
			"greige on hand must be contained by the echelon figure, not exceed it")

		average, ok := policy["average_greige_inventory"].(float64)
		require.True(t, ok)
		max, ok := policy["max_greige_inventory"].(float64)
		require.True(t, ok)
		buffer, ok := policy["safety_stock_primary"].(float64)
		require.True(t, ok)

		// The stage holds its buffer plus a campaign that lands and drains: half of one on average, a whole one at the peak.
		assert.GreaterOrEqual(t, average, buffer,
			"average greige holding must sit at or above the buffer it always keeps")
		assert.GreaterOrEqual(t, max, average,
			"peak greige holding must be at least the average")
	}
}

func TestFinishedPolicies_RejectsUnknownSchedule(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(
		productionSchedulesPath+"/pnsc_01definitelynotreal/finished-policies", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	// A version that does not exist has no targets; an empty page is the honest answer rather than an error, matching how the sibling policy list behaves.
	assert.Contains(t, []int{200, 404}, status)
}

// A run of weeks with nothing planned is the (s,S) policy waiting while stock covers demand. The curve is what makes that legible; without it an empty plan grid is indistinguishable from a solver that did nothing.
func TestItemPolicies_CarryTheProjectedStockCurve(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-projection"))
	scheduleID := jsonField(schedule, "id")

	horizonWeeks, ok := schedule["horizon_weeks"].(float64)
	require.True(t, ok)

	policies := listItemPolicies(t, scheduleID)
	require.NotEmpty(t, policies, "a generated version must carry item policies")

	sawCurve := false
	for _, policy := range policies {
		raw, present := policy["projected_on_hand"]
		require.True(t, present, "projected_on_hand must always be serialized, even when null")
		curve, ok := raw.([]any)
		if !ok {
			continue
		}
		sawCurve = true

		assert.Len(t, curve, int(horizonWeeks),
			"the curve must cover every horizon week, or the grid cannot line up with it")
		for _, value := range curve {
			_, numeric := value.(float64)
			assert.True(t, numeric, "every projected position must be numeric")
		}
	}

	assert.True(t, sawCurve, "at least one policy must carry a projection")
}
