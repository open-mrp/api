//go:build e2e

package api_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The per-item policies a schedule is solved with, pinned to 1.0.forge-preview.5, which typed unit_cost, setup_cost and holding_cost as numbers. A caller without costs:read reads them as null on preview.6 and as 0 on preview.5; a caller with costs:read reads the costs on both.

var schedulePolicyCostFields = []string{"unit_cost", "setup_cost", "holding_cost"}

// policiesByItem keys a list of policies by the item each one plans.
func policiesByItem(t *testing.T, rows []any) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	for _, raw := range rows {
		policy, ok := raw.(map[string]any)
		require.True(t, ok)
		out[jsonField(jsonObject(policy, "item"), "id")] = policy
	}
	require.NotEmpty(t, out, "the seeded demand chain must produce policies for this to test anything")
	return out
}

// assertCostsAcrossVersions checks every policy the latest version returned against the same item's policy on preview.5.
func assertCostsAcrossVersions(t *testing.T, latest, old map[string]map[string]any, canReadCosts bool) {
	t.Helper()
	compared := 0
	for itemID, policy := range latest {
		previous, ok := old[itemID]
		if !ok {
			continue
		}
		compared++
		assert.Equal(t, jsonField(policy, "sku"), jsonField(previous, "sku"))
		for _, field := range schedulePolicyCostFields {
			if canReadCosts {
				require.NotNil(t, policy[field], "%s %s: a costs:read holder reads it", itemID, field)
				assert.Equal(t, policy[field], previous[field], "%s %s", itemID, field)
				continue
			}
			assertNilField(t, policy, field)
			assert.Equal(t, float64(0), previous[field], "%s %s: preview.5 typed it as a number", itemID, field)
		}
	}
	require.Positive(t, compared, "both versions must plan at least one item in common")
}

func TestVersionCompat_SchedulePreview_Policies(t *testing.T) {
	t.Parallel()
	defer lockPlanningRead()()

	body := map[string]any{"planning_as_of": rfc3339(time.Now().UTC()), "horizon_weeks": 4}
	solve := func(client *Client) map[string]map[string]any {
		t.Helper()
		status, raw, err := client.Put(productionSchedulePreviewPath, body)
		require.NoError(t, err)
		requireStatus(t, 200, status, raw)
		return policiesByItem(t, jsonListData(parseJSON(raw), "policies"))
	}

	t.Run("without costs:read preview.5 reads 0", func(t *testing.T) {
		planner := customRoleClient(t, "production_schedules:read")
		assertCostsAcrossVersions(t, solve(planner), solve(planner.WithAPIVersion(preview5APIVersion)), false)
	})

	t.Run("with costs:read both versions read the costs", func(t *testing.T) {
		planner := customRoleClient(t, "production_schedules:read", costsRead)
		assertCostsAcrossVersions(t, solve(planner), solve(planner.WithAPIVersion(preview5APIVersion)), true)
	})
}

func TestVersionCompat_ScheduleItemPolicies_List(t *testing.T) {
	t.Parallel()
	created := generateSchedule(t, map[string]any{"horizon_weeks": 4})
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	cleanupSchedule(t, id)
	path := productionSchedulesPath + "/" + id + "/item-policies"

	list := func(client *Client) map[string]map[string]any {
		t.Helper()
		return policiesByItem(t, jsonArray(parseJSON(mustGetAs(t, client, path, nil)), "data"))
	}

	t.Run("without costs:read preview.5 reads 0", func(t *testing.T) {
		planner := customRoleClient(t, "production_schedules:read")
		assertCostsAcrossVersions(t, list(planner), list(planner.WithAPIVersion(preview5APIVersion)), false)
	})

	t.Run("with costs:read both versions read the costs", func(t *testing.T) {
		planner := customRoleClient(t, "production_schedules:read", costsRead)
		assertCostsAcrossVersions(t, list(planner), list(planner.WithAPIVersion(preview5APIVersion)), true)
	})
}
