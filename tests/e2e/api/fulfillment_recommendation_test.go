//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fulfillment recommendations: which SKUs the engine thinks should be built to
// order, and why.

const (
	recommendationsPath      = "/v1/operations/fulfillment-recommendations"
	recommendationsApplyPath = "/v1/operations/fulfillment-recommendations/actions/apply"
)

func listRecommendations(t *testing.T) []any {
	t.Helper()

	status, body, err := apiClient.GetListRaw(recommendationsPath, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "recommendations must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	data, ok := parseJSON(body)["data"].([]any)
	require.True(t, ok, "recommendations must serialize as a list: %s", string(body))
	return data
}

// findRecommendation returns the advice for one item, or nil.
func findRecommendation(t *testing.T, itemID string) map[string]any {
	t.Helper()
	for _, raw := range listRecommendations(t) {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if item, ok := row["item"].(map[string]any); ok && jsonField(item, "id") == itemID {
			return row
		}
	}
	return nil
}

// Every recommendation must be a complete verdict: a policy, the rule that decided,
// and the measurement behind it. A bare answer is one nobody can argue with, which
// is what makes advice ignorable.
func TestFulfillmentRecommendations_EveryVerdictExplainsItself(t *testing.T) {
	t.Parallel()

	rows := listRecommendations(t)
	require.NotEmpty(t, rows, "the seeded account sells products, so it must produce recommendations")

	validPolicies := []string{"make_to_stock", "make_to_order"}
	validReasons := []string{
		"lead_time_infeasible", "no_recent_demand", "single_customer",
		"lumpy_demand", "slow_moving_high_value", "steady_demand",
	}

	for _, raw := range rows {
		row, ok := raw.(map[string]any)
		require.True(t, ok)

		assert.Equal(t, "fulfillment_recommendation", jsonField(row, "object"))
		assert.Contains(t, validPolicies, jsonField(row, "current_policy"))
		assert.Contains(t, validPolicies, jsonField(row, "recommended_policy"))
		assert.Contains(t, validReasons, jsonField(row, "reason"), "every verdict must name the rule that decided")

		item, ok := row["item"].(map[string]any)
		require.True(t, ok, "a recommendation must name its item: %v", row)
		assert.NotEmpty(t, jsonField(item, "id"))
		assert.NotEmpty(t, jsonField(row, "sku"))

		// `changes` has to agree with the two policies, or the review UI would offer
		// changes that change nothing.
		wantChanges := jsonField(row, "current_policy") != jsonField(row, "recommended_policy")
		assert.Equal(t, wantChanges, row["changes"],
			"changes must follow from current vs recommended: %v", row)

		for _, key := range []string{
			"average_demand_interval", "coefficient_of_variation", "top_customer_share_pct",
			"demand_weighted_lead_time_days", "annual_cogs", "months_since_last_sale",
			"mixed_stream_share_pct",
		} {
			_, ok := row[key].(float64)
			assert.True(t, ok, "%s must be present and numeric on every verdict: %v", key, row[key])
		}
	}
}

// Rows that would change something sort first: a merchant opens this looking for
// what to act on.
func TestFulfillmentRecommendations_ChangesSortFirst(t *testing.T) {
	t.Parallel()

	seenUnchanged := false
	for _, raw := range listRecommendations(t) {
		row, ok := raw.(map[string]any)
		require.True(t, ok)

		changes, _ := row["changes"].(bool)
		if !changes {
			seenUnchanged = true
			continue
		}
		assert.False(t, seenUnchanged,
			"a row that changes something appeared after one that does not; actionable rows must sort first")
	}
}

// A recommendation is advice. Reading it must not change how anything is planned —
// only applying it does.
//
// On its own item: the seeded one is shared with tests that deliberately change its
// policy, and a test asserting that nothing changed has to own the thing it is
// asserting about.
func TestFulfillmentRecommendations_ReadingChangesNothing(t *testing.T) {
	t.Parallel()

	itemID := createSellableItem(t, uniqueName("e2e-recommend-stable"))

	before := findRecommendation(t, itemID)
	require.NotNil(t, before, "a sellable item should have a recommendation")

	listRecommendations(t)
	listRecommendations(t)

	after := findRecommendation(t, itemID)
	require.NotNil(t, after)
	assert.Equal(t, jsonField(before, "current_policy"), jsonField(after, "current_policy"),
		"listing recommendations must not change how an item is planned")
	assert.Equal(t, jsonField(before, "recommended_policy"), jsonField(after, "recommended_policy"),
		"the same inputs must produce the same advice")
}

// Applying writes the advice as a per-item override, and what comes back is what was
// actually written.
func TestFulfillmentRecommendations_ApplyWritesTheOverride(t *testing.T) {
	t.Parallel()

	// A product, not a material: policy is resolved on what can be sold, so a raw
	// material is deliberately not a candidate for this advice.
	itemID := createSellableItem(t, uniqueName("e2e-recommend"))
	t.Cleanup(func() { _, _, _ = apiClient.Delete(itemSettingsPath + "/" + itemID) })

	rec := findRecommendation(t, itemID)
	require.NotNil(t, rec, "a newly created sellable item should be classified")
	recommended := jsonField(rec, "recommended_policy")
	require.NotEmpty(t, recommended)

	status, body, err := apiClient.Post(recommendationsApplyPath,
		map[string]any{"item_ids": []string{itemID}}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "apply must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	applied, ok := parseJSON(body)["data"].([]any)
	require.True(t, ok)
	require.Len(t, applied, 1, "exactly the named item should be applied")

	// The override is now readable through the item-settings endpoint, which is the
	// durable artifact — the recommendation itself is never stored.
	getStatus, getBody, err := apiClient.GetListRaw(itemSettingsPath+"/"+itemID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, recommended, jsonField(parseJSON(getBody), "fulfillment_policy"),
		"applying should write exactly the policy that was recommended")

	// And the item now reports itself as already on that policy, with nothing to change.
	afterRec := findRecommendation(t, itemID)
	require.NotNil(t, afterRec)
	assert.Equal(t, recommended, jsonField(afterRec, "current_policy"))
	assert.Equal(t, false, afterRec["changes"],
		"once applied, the recommendation should no longer be a change")
}

func TestFulfillmentRecommendations_ApplyValidatesItsInput(t *testing.T) {
	t.Parallel()

	t.Run("empty list", func(t *testing.T) {
		status, body, err := apiClient.Post(recommendationsApplyPath,
			map[string]any{"item_ids": []string{}}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "must not 5xx: %s", string(body))
		assert.Equal(t, 400, status, "applying nothing should be rejected, got %d: %s", status, string(body))
	})

	t.Run("unknown item", func(t *testing.T) {
		status, body, err := apiClient.Post(recommendationsApplyPath,
			map[string]any{"item_ids": []string{"it_doesnotexist0000"}}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "must not 5xx: %s", string(body))
		assert.Contains(t, []int{400, 404}, status,
			"an item with no recommendation cannot be applied, got %d: %s", status, string(body))
	})
}

// createSellableItem creates a product and returns the item behind it.
//
// Recommendations are only produced for items that carry a product: the question
// "should this be made to order" is asked of finished goods, and a raw material has
// no policy to resolve.
func createSellableItem(t *testing.T, sku string) string {
	t.Helper()

	status, body, err := apiClient.Post(productsPath, validProductBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	productID := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(productsPath + "/" + productID) })

	list, _, err := apiClient.GetList(itemsPath, url.Values{"q": {sku}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1, "exactly one item should match the unique SKU %q", sku)
	return DataItemField(list.Data[0], "id")
}

// Applying names its items explicitly, and everything else has to be left exactly
// as it was. Adopting advice in bulk without saying what is being adopted is how a
// plant changes what it builds by accident.
func TestFulfillmentRecommendations_ApplyTouchesOnlyNamedItems(t *testing.T) {
	t.Parallel()

	targetID := createSellableItem(t, uniqueName("e2e-recommend-target"))
	bystanderID := createSellableItem(t, uniqueName("e2e-recommend-bystander"))
	t.Cleanup(func() { _, _, _ = apiClient.Delete(itemSettingsPath + "/" + targetID) })
	t.Cleanup(func() { _, _, _ = apiClient.Delete(itemSettingsPath + "/" + bystanderID) })

	status, body, err := apiClient.Post(recommendationsApplyPath,
		map[string]any{"item_ids": []string{targetID}}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "apply must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	applied, ok := parseJSON(body)["data"].([]any)
	require.True(t, ok)
	require.Len(t, applied, 1, "only the named item should come back")
	row, ok := applied[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, targetID, jsonField(jsonObject(row, "item"), "id"))

	// The bystander has no override at all, which is the difference between "left
	// alone" and "written with the same value".
	bystanderStatus, _, err := apiClient.GetListRaw(itemSettingsPath+"/"+bystanderID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, bystanderStatus,
		"an item not named in the request must not have been given an override")
}

// Applying twice is the same as applying once: the advice is recomputed and the
// item keeps one override rather than accumulating a second.
func TestFulfillmentRecommendations_ApplyIsRepeatable(t *testing.T) {
	t.Parallel()

	itemID := createSellableItem(t, uniqueName("e2e-recommend-twice"))
	t.Cleanup(func() { _, _, _ = apiClient.Delete(itemSettingsPath + "/" + itemID) })

	apply := func() map[string]any {
		status, body, err := apiClient.Post(recommendationsApplyPath,
			map[string]any{"item_ids": []string{itemID}}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "apply must not 5xx: %s", string(body))
		requireStatus(t, 200, status, body)

		getStatus, getBody, err := apiClient.GetListRaw(itemSettingsPath+"/"+itemID, nil)
		require.NoError(t, err)
		requireStatus(t, 200, getStatus, getBody)
		return parseJSON(getBody)
	}

	first := apply()
	second := apply()

	assert.Equal(t, jsonField(first, "id"), jsonField(second, "id"),
		"the second apply must replace the override, not add another")
	assert.Equal(t, jsonField(first, "fulfillment_policy"), jsonField(second, "fulfillment_policy"),
		"the same demand must produce the same verdict")
}

// Applying is a write, so another tenant must not be able to aim it at an item it
// does not own.
func TestFulfillmentRecommendations_ApplyTenantIsolation(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	itemID := createSellableItem(t, uniqueName("e2e-recommend-tenant"))
	t.Cleanup(func() { _, _, _ = apiClient.Delete(itemSettingsPath + "/" + itemID) })

	status, body, err := clientB.Post(recommendationsApplyPath,
		map[string]any{"item_ids": []string{itemID}}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "cross-tenant apply must not 5xx: %s", string(body))
	assert.Contains(t, []int{400, 404}, status,
		"tenant B must not apply advice to tenant A's item, got %d: %s", status, string(body))

	getStatus, _, err := apiClient.GetListRaw(itemSettingsPath+"/"+itemID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus, "and must not have written an override in tenant A")
}

// The recommendation list is per-account: another tenant's SKUs must not be
// classified alongside this one's.
func TestFulfillmentRecommendations_ListIsScopedToTheTenant(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	itemID := createSellableItem(t, uniqueName("e2e-recommend-scope"))
	require.NotNil(t, findRecommendation(t, itemID), "precondition: tenant A classifies its own item")

	status, body, err := clientB.GetListRaw(recommendationsPath, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	for _, raw := range jsonArray(parseJSON(body), "data") {
		row, ok := raw.(map[string]any)
		require.True(t, ok)
		assert.NotEqual(t, itemID, jsonField(jsonObject(row, "item"), "id"),
			"tenant B must not be given advice about tenant A's SKU")
	}
}
