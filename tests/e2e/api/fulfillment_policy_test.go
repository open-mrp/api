//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fulfillment policy: whether a SKU is built to a forecast or only against orders
// already on the book, resolved item -> product line -> account default.

const itemSettingsPath = "/v1/operations/production-schedule-settings/items"

// setItemPolicy writes an item's planning override and removes it afterwards, so a
// policy set by one test cannot change another test's plan.
func setItemPolicy(t *testing.T, itemID, policy string) map[string]any {
	t.Helper()

	body := map[string]any{"participation_status": "included"}
	if policy != "" {
		body["fulfillment_policy"] = policy
	}
	status, respBody, err := apiClient.Put(itemSettingsPath+"/"+itemID, body)
	require.NoError(t, err)
	require.Less(t, status, 500, "item setting write must not 5xx: %s", string(respBody))
	requireStatus(t, 200, status, respBody)

	t.Cleanup(func() { _, _, _ = apiClient.Delete(itemSettingsPath + "/" + itemID) })
	return parseJSON(respBody)
}

func TestFulfillmentPolicy_ItemSettingRoundTrips(t *testing.T) {
	t.Parallel()

	got := setItemPolicy(t, SeedItemID, "make_to_order")

	assert.Equal(t, "production_schedule_item_setting", jsonField(got, "object"))
	assert.Equal(t, "make_to_order", jsonField(got, "fulfillment_policy"))
	assert.Equal(t, "included", jsonField(got, "participation_status"))

	item, ok := got["item"].(map[string]any)
	require.True(t, ok, "the setting must name the item it applies to: %s", got)
	assert.Equal(t, SeedItemID, jsonField(item, "id"))

	// It has to come back on the list too, not only from the write.
	listStatus, listBody, err := apiClient.GetListRaw(itemSettingsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, listStatus, listBody)

	data, _ := parseJSON(listBody)["data"].([]any)
	found := false
	for _, raw := range data {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if rowItem, ok := row["item"].(map[string]any); ok && jsonField(rowItem, "id") == SeedItemID {
			found = true
			assert.Equal(t, "make_to_order", jsonField(row, "fulfillment_policy"))
		}
	}
	assert.True(t, found, "the override should appear in the list")
}

// Clearing the policy returns the item to its product line, and deleting the whole
// override returns it to the defaults.
func TestFulfillmentPolicy_ClearingAndDeleting(t *testing.T) {
	t.Parallel()

	// Its own item: the seeded one is shared with tests that assert on how it is planned.
	itemID := createItemsViaMaterials(t, uniqueName("e2e-fulfil"), 1)[0]
	setItemPolicy(t, itemID, "make_to_order")

	cleared := setItemPolicy(t, itemID, "")
	assert.Empty(t, jsonField(cleared, "fulfillment_policy"),
		"clearing the policy should leave the item inheriting, not pin it")

	status, body, err := apiClient.Delete(itemSettingsPath + "/" + itemID)
	require.NoError(t, err)
	require.Less(t, status, 500, "delete must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	// A second delete has nothing to remove, and saying otherwise would hide a typo.
	repeatStatus, repeatBody, err := apiClient.Delete(itemSettingsPath + "/" + itemID)
	require.NoError(t, err)
	require.Less(t, repeatStatus, 500, "repeat delete must not 5xx: %s", string(repeatBody))
	assert.Equal(t, 404, repeatStatus, "deleting an override that does not exist should 404")
}

func TestFulfillmentPolicy_RejectsUnknownValues(t *testing.T) {
	t.Parallel()

	t.Run("unknown policy", func(t *testing.T) {
		status, body, err := apiClient.Put(itemSettingsPath+"/"+SeedItemID, map[string]any{
			"participation_status": "included",
			"fulfillment_policy":   "make_to_whatever",
		})
		require.NoError(t, err)
		require.Less(t, status, 500, "a bad policy must be a client error: %s", string(body))
		assert.Equal(t, 400, status, "body: %s", string(body))
	})

	t.Run("unknown item", func(t *testing.T) {
		status, body, err := apiClient.Put(itemSettingsPath+"/it_doesnotexist0000", map[string]any{
			"participation_status": "included",
			"fulfillment_policy":   "make_to_order",
		})
		require.NoError(t, err)
		require.Less(t, status, 500, "an unknown item must not 5xx: %s", string(body))
		assert.Contains(t, []int{400, 404}, status,
			"a setting for an item that does not exist should be rejected, got %d: %s", status, string(body))
	})
}

// The policy has to reach the solver and be recorded on what it produces, or the
// setting is a value nothing reads.
func TestFulfillmentPolicy_SolverReportsIt(t *testing.T) {
	t.Parallel()

	diagnostics := previewDiagnostics(t, previewPlan(t))
	require.Contains(t, diagnostics, "make_to_order_item_count",
		"the solve must report how many items it planned to order")
}

// A make-to-order item holds no safety stock: it is not built until the demand
// exists, so there is nothing to buffer against.
//
// Set on the greige item the constraint department actually plans, not on a finished
// SKU: policy is resolved per planned item, and asserting on a SKU the solver never
// reaches would pass without exercising anything.
func TestFulfillmentPolicy_MakeToOrderPolicyHoldsNoBuffer(t *testing.T) {
	// Not parallel: it changes how a planned item is built, and a concurrent solve
	// would see the change mid-flight.
	setItemPolicy(t, SeedGreigeItemID, "make_to_order")

	defer lockPlanningRead()()
	status, body, err := apiClient.Put(productionSchedulePreviewPath, map[string]any{})
	require.NoError(t, err)
	require.Less(t, status, 500, "preview must not 5xx: %s", string(body))
	if status == 400 {
		t.Skip("no constraint department configured in this environment")
	}
	requireStatus(t, 200, status, body)

	policies, ok := parseJSON(body)["policies"].(map[string]any)
	require.True(t, ok, "the preview should carry item policies")
	data, _ := policies["data"].([]any)

	checked := 0
	for _, raw := range data {
		policy, ok := raw.(map[string]any)
		if !ok || jsonField(policy, "fulfillment_policy") != "make_to_order" {
			continue
		}
		checked++
		assert.Equal(t, "0", jsonField(policy, "safety_stock_primary"),
			"a make-to-order item holds no primary safety stock")
		assert.Equal(t, "0", jsonField(policy, "safety_stock_downstream"),
			"a make-to-order item holds no downstream safety stock")
		assert.Equal(t, "0", jsonField(policy, "reorder_point"),
			"a make-to-order item has no statistical reorder point")
	}

	require.Positive(t, checked,
		"no make-to-order policy appeared in the plan, so this asserted nothing")
}
