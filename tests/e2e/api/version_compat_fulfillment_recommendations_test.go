//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fulfillment recommendations pinned to 1.0.forge-preview.5, which typed annual_cogs as a number. A caller without costs:read reads it as null on preview.6 and as 0 on preview.5; a caller with costs:read reads the figure on both.

// recommendationsByItem keys recommendation rows by the item each one advises on.
func recommendationsByItem(t *testing.T, raw []byte) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	for _, row := range jsonArray(parseJSON(raw), "data") {
		rec, ok := row.(map[string]any)
		require.True(t, ok)
		out[jsonField(jsonObject(rec, "item"), "id")] = rec
	}
	require.NotEmpty(t, out, "recommendations: %s", string(raw))
	return out
}

// assertCOGSAcrossVersions checks every recommendation the latest version returned against the same item's on preview.5, and returns how many carried a non-zero figure.
func assertCOGSAcrossVersions(t *testing.T, latest, old map[string]map[string]any, canReadCosts bool) int {
	t.Helper()
	compared, nonZero := 0, 0
	for itemID, rec := range latest {
		previous, ok := old[itemID]
		if !ok {
			continue
		}
		compared++
		assert.Equal(t, rec["recommended_policy"], previous["recommended_policy"], "%s: the advice is the same on both", itemID)
		if !canReadCosts {
			assertNilField(t, rec, "annual_cogs")
			assert.Equal(t, float64(0), previous["annual_cogs"], "%s: preview.5 typed annual_cogs as a number", itemID)
			continue
		}
		require.NotNil(t, rec["annual_cogs"], "%s: a costs:read holder reads annual_cogs", itemID)
		// The figure is worked out from live sales, which parallel tests add to between the two reads, so each version is checked for carrying it rather than for the same number.
		require.NotNil(t, previous["annual_cogs"], "%s: preview.5 reads annual_cogs for a costs:read holder", itemID)
		assert.IsType(t, float64(0), previous["annual_cogs"], "%s: preview.5 types annual_cogs as a number", itemID)
		if rec["annual_cogs"] != float64(0) {
			nonZero++
		}
	}
	require.Positive(t, compared, "both versions must advise on at least one item in common")
	return nonZero
}

func TestVersionCompat_FulfillmentRecommendations_List(t *testing.T) {
	t.Parallel()
	list := func(client *Client) map[string]map[string]any {
		t.Helper()
		return recommendationsByItem(t, mustGetAs(t, client, recommendationsPath, nil))
	}

	t.Run("without costs:read preview.5 reads 0", func(t *testing.T) {
		planner := customRoleClient(t, "production_schedules:read")
		assertCOGSAcrossVersions(t, list(planner), list(planner.WithAPIVersion(preview5APIVersion)), false)
	})

	t.Run("with costs:read both versions read the figure", func(t *testing.T) {
		planner := customRoleClient(t, "production_schedules:read", costsRead)
		nonZero := assertCOGSAcrossVersions(t, list(planner), list(planner.WithAPIVersion(preview5APIVersion)), true)
		assert.Positive(t, nonZero, "the seeded items that sell carry a cost of goods")
	})
}

// Applying answers with the recommendations it applied, in the caller's version.
func TestVersionCompat_FulfillmentRecommendations_Apply(t *testing.T) {
	t.Parallel()
	itemID := createSellableItem(t, uniqueName("e2e-recommend-compat"))
	t.Cleanup(func() { _, _, _ = apiClient.Delete(itemSettingsPath + "/" + itemID) })
	apply := func(client *Client) map[string]map[string]any {
		t.Helper()
		status, raw, err := client.Post(recommendationsApplyPath, map[string]any{"item_ids": []string{itemID}}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, raw)
		applied := recommendationsByItem(t, raw)
		require.Contains(t, applied, itemID)
		return applied
	}

	t.Run("without costs:read preview.5 reads 0", func(t *testing.T) {
		planner := customRoleClient(t, "production_schedules:read", "production_schedules:update")
		assertCOGSAcrossVersions(t, apply(planner), apply(planner.WithAPIVersion(preview5APIVersion)), false)
	})

	t.Run("with costs:read both versions read the figure", func(t *testing.T) {
		planner := customRoleClient(t, "production_schedules:read", "production_schedules:update", costsRead)
		assertCOGSAcrossVersions(t, apply(planner), apply(planner.WithAPIVersion(preview5APIVersion)), true)
	})
}
