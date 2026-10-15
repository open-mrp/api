//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The fulfillment policy chain above the item: a product line's default supply, and
// the account default underneath it. A line that supplies from stock is planned make
// to stock; one that makes for the order is planned make to order.
//
// The chain is read through the recommendation list, which reports each SKU's
// current policy — the same resolution a solve applies. Asserting on the stored
// value alone would prove the field round-trips without proving anything reads it.

// createProductLineWithSupply creates a product line, optionally with a default
// supply, and removes it afterwards.
func createProductLineWithSupply(t *testing.T, prefix, supply string) string {
	t.Helper()

	body := map[string]any{
		"name":              uniqueName(prefix),
		"commission_policy": "applied",
		"freight_policy":    "billed",
	}
	if supply != "" {
		body["supply"] = supply
	}

	created := createAndCleanup(t, productLinesPath, body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	return id
}

// createSellableItemInLine creates a product inside a product line and returns the
// item behind it.
func createSellableItemInLine(t *testing.T, sku, productLineID string) string {
	t.Helper()

	body := validProductBody(sku)
	body["product_line_id"] = productLineID

	// The item is expandable, so it is asked for explicitly rather than looked up by
	// SKU afterwards.
	status, respBody, err := apiClient.Post(productsPath+"?include=item", body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)

	created := parseJSON(respBody)
	productID := jsonField(created, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(productsPath + "/" + productID) })

	itemID := jsonField(jsonObject(created, "item"), "id")
	require.NotEmpty(t, itemID, "a created product must carry the item behind it: %s", string(respBody))
	return itemID
}

// A line's supply has to survive create, update and clearing, since it is the rule
// every SKU in the line falls back to.
func TestProductLineSupply_RoundTripsAndClears(t *testing.T) {
	t.Parallel()

	lineID := createProductLineWithSupply(t, "e2e-pdln-policy", "make")

	status, body, err := apiClient.GetListRaw(productLinesPath+"/"+lineID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "make", jsonField(parseJSON(body), "supply"),
		"a supply set at creation must be readable back")

	patchStatus, patchBody, err := apiClient.Patch(productLinesPath+"/"+lineID,
		map[string]any{"supply": "stock"}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, patchStatus, 500, "updating the policy must not 5xx: %s", string(patchBody))
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, "stock", jsonField(parseJSON(patchBody), "supply"))

	// An unrelated edit must not disturb it, or every rename would reset how a whole
	// line is built.
	renameStatus, renameBody, err := apiClient.Patch(productLinesPath+"/"+lineID,
		map[string]any{"name": uniqueName("e2e-pdln-policy-renamed")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, renameStatus, renameBody)
	assert.Equal(t, "stock", jsonField(parseJSON(renameBody), "supply"),
		"omitting the supply on an update must leave it alone")

	clearStatus, clearBody, err := apiClient.Patch(productLinesPath+"/"+lineID,
		map[string]any{"supply": nil}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, clearStatus, 500, "clearing the policy must not 5xx: %s", string(clearBody))
	requireStatus(t, 200, clearStatus, clearBody)
	assert.Nil(t, parseJSON(clearBody)["supply"],
		"a cleared supply returns the line's products to the account default")
}

// A line with no policy is the normal case, and it must read as null rather than as
// a guess at what the account default happens to be today.
func TestProductLineSupply_DefaultsToNull(t *testing.T) {
	t.Parallel()

	lineID := createProductLineWithSupply(t, "e2e-pdln-nopolicy", "")

	status, body, err := apiClient.GetListRaw(productLinesPath+"/"+lineID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Nil(t, parseJSON(body)["supply"])
}

func TestProductLineSupply_RejectsUnknownValues(t *testing.T) {
	t.Parallel()

	t.Run("on create", func(t *testing.T) {
		status, body, err := apiClient.Post(productLinesPath, map[string]any{
			"name":              uniqueName("e2e-pdln-badpolicy"),
			"commission_policy": "applied",
			"freight_policy":    "billed",
			"supply":            "make_to_vibes",
		}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "must not 5xx: %s", string(body))
		assert.Equal(t, 422, status, "body: %s", string(body))
	})

	t.Run("on update", func(t *testing.T) {
		lineID := createProductLineWithSupply(t, "e2e-pdln-badupdate", "")

		status, body, err := apiClient.Patch(productLinesPath+"/"+lineID,
			map[string]any{"supply": "make_to_vibes"}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "must not 5xx: %s", string(body))
		assert.Equal(t, 422, status, "body: %s", string(body))
	})
}

// The chain, most specific first: an item override beats its product line, and
// removing the override hands the item back to the line rather than to the account
// default.
func TestFulfillmentPolicy_ItemOverrideBeatsProductLine(t *testing.T) {
	t.Parallel()

	lineID := createProductLineWithSupply(t, "e2e-chain-line", "make")
	itemID := createSellableItemInLine(t, uniqueName("e2e-chain"), lineID)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(itemSettingsPath + "/" + itemID) })

	inherited := findRecommendation(t, itemID)
	require.NotNil(t, inherited, "a sellable item should be classified")
	require.Equal(t, "make_to_order", jsonField(inherited, "current_policy"),
		"an item with no override of its own is planned on its product line's policy")

	status, body := putItemSetting(t, itemID, map[string]any{
		"participation_status": "included",
		"fulfillment_policy":   "make_to_stock",
	})
	requireStatus(t, 200, status, body)

	overridden := findRecommendation(t, itemID)
	require.NotNil(t, overridden)
	assert.Equal(t, "make_to_stock", jsonField(overridden, "current_policy"),
		"an explicit item override must beat the line it sells under")

	delStatus, delBody, err := apiClient.Delete(itemSettingsPath + "/" + itemID)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	restored := findRecommendation(t, itemID)
	require.NotNil(t, restored)
	assert.Equal(t, "make_to_order", jsonField(restored, "current_policy"),
		"removing the override returns the item to its line, not to the account default")
}

// Clearing the supply on the line propagates to the SKUs in it, which is what makes
// the line a useful place to set one at all.
func TestFulfillmentPolicy_ClearingTheLineFallsThroughToTheAccount(t *testing.T) {
	t.Parallel()

	lineID := createProductLineWithSupply(t, "e2e-chain-clear-line", "make")
	itemID := createSellableItemInLine(t, uniqueName("e2e-chain-clear"), lineID)

	before := findRecommendation(t, itemID)
	require.NotNil(t, before)
	require.Equal(t, "make_to_order", jsonField(before, "current_policy"))

	status, body, err := apiClient.Patch(productLinesPath+"/"+lineID,
		map[string]any{"supply": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	accountDefault := jsonField(readScheduleSettings(t), "default_fulfillment_policy")
	require.NotEmpty(t, accountDefault, "the account must always advertise a default policy")

	after := findRecommendation(t, itemID)
	require.NotNil(t, after)
	assert.Equal(t, accountDefault, jsonField(after, "current_policy"),
		"with the line's policy cleared the item falls through to the account default")
}
