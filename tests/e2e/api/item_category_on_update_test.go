//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Saving an item with a new category is one request: PATCH on a part, material or product takes category_id and moves the item as Change Item Category does, under that endpoint's own permission.

var movedItemIncludes = url.Values{"include": {"category,unit_value,unit_cost"}}

// productCategoryInEaches creates a product category whose unit group counts in eaches.
func productCategoryInEaches(t *testing.T) string {
	t.Helper()
	unitGroupID := dashItemsCreateAs(t, apiClient, unitGroupsPath, map[string]any{
		"name": uniqueName("e2e-moveitem-ug"), "type": "quantity", "base_unit_id": "each",
	})
	return dashItemsCreateAs(t, apiClient, itemCategoriesPath, map[string]any{
		"name": uniqueName("e2e-moveitem-cat"), "type": "product_category", "unit_group_id": unitGroupID,
	})
}

// partInSocks creates a part in the seeded socks category and returns the part and the item behind it.
func partInSocks(t *testing.T) (partID, itemID string) {
	t.Helper()
	sku := uniqueName("e2e-moveitem-part")
	partID = dashItemsCreateAs(t, apiClient, partsPath, validPartBody(sku))
	return partID, dashItemsItemBySKU(t, sku)
}

func readMovedItem(t *testing.T, itemID string) map[string]any {
	t.Helper()
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+itemID, movedItemIncludes)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	return parseJSON(body)
}

func rateDenominator(item map[string]any, rate string) string {
	return jsonField(jsonObject(jsonObject(item, rate), "denominator_unit"), "id")
}

// requireItemCategoryMoveAudited waits for the item's own update event and checks it records the move to categoryID.
func requireItemCategoryMoveAudited(t *testing.T, itemID, categoryID string) {
	t.Helper()
	event := expectAuditEventWithChanges(t, itemID, "item", "update")
	change, ok := changeForField(jsonListData(event, "changes"), "item_category_id")
	require.True(t, ok, "the item's update event records the category it moved to")
	assert.Equal(t, categoryID, jsonField(change, "new_value"))
}

func TestPartUpdate_MovesTheCategoryUnderPartsUpdateAlone(t *testing.T) {
	t.Parallel()
	categoryID := productCategoryInEaches(t)
	partID, itemID := partInSocks(t)
	before := readMovedItem(t, itemID)
	require.NotEqual(t, "each", rateDenominator(before, "unit_value"), "the socks category does not count in eaches")

	writer := customRoleClient(t, "parts:update")
	notes := uniqueName("moved with its notes")
	status, body, err := writer.Patch(withQuery(partsPath+"/"+partID, url.Values{"include": {"item,item.category"}}), map[string]any{
		"category_id": categoryID,
		"notes":       notes,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	item := jsonObject(parseJSON(body), "item")
	require.NotNil(t, item)
	assert.Equal(t, categoryID, jsonField(jsonObject(item, "category"), "id"))
	assert.Equal(t, notes, jsonField(item, "notes"))

	after := readMovedItem(t, itemID)
	assert.Equal(t, categoryID, jsonField(jsonObject(after, "category"), "id"))
	for _, rate := range []string{"unit_value", "unit_cost"} {
		assert.Equal(t, "each", rateDenominator(after, rate), "%s moves onto the new base unit", rate)
		assertDecimalEqual(t, jsonField(jsonObject(before, rate), "value"), jsonField(jsonObject(after, rate), "value"), "%s keeps its figure", rate)
	}

	requireItemCategoryMoveAudited(t, itemID, categoryID)
	partEvent := expectAuditEventWithChanges(t, partID, "part", "update")
	_, movedOnPart := changeForField(jsonListData(partEvent, "changes"), "item_category_id")
	assert.False(t, movedOnPart, "the move is recorded on the item, the rest of the save on the part")
}

// The dashboard sends the cost in the new base unit alongside the move; it is written after the move, so it stays as sent.
func TestMaterialUpdate_MovesTheCategoryUnderMaterialsUpdateAlone(t *testing.T) {
	t.Parallel()
	categoryID := dashItemsMaterialCategory(t, apiClient)
	sku := uniqueName("e2e-moveitem-mat")
	materialID := dashItemsCreateAs(t, apiClient, materialsPath, validMaterialBody(sku))
	itemID := dashItemsItemBySKU(t, sku)

	writer := customRoleClient(t, "materials:update")
	status, body, err := writer.Patch(materialsPath+"/"+materialID, map[string]any{
		"category_id": categoryID,
		"unit_cost":   map[string]any{"value": "0.40", "numerator_unit_id": "dollar", "denominator_unit_id": "each"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)

	after := readMovedItem(t, itemID)
	assert.Equal(t, categoryID, jsonField(jsonObject(after, "category"), "id"))
	assertDecimalEqual(t, "0.40", jsonField(jsonObject(after, "unit_cost"), "value"), "the cost sent with the move")
	assert.Equal(t, "each", rateDenominator(after, "unit_cost"))
	assert.Equal(t, "each", rateDenominator(after, "unit_value"), "the price the request left alone moves onto the new base unit")
	assert.Equal(t, []string{itemID}, listIDs(t, itemsPath, url.Values{"q": {sku}, "category_ids": {categoryID}}))

	requireItemCategoryMoveAudited(t, itemID, categoryID)
	expectAuditEvent(t, materialID, "material", "update")
}

func TestProductUpdate_MovesTheCategoryUnderItemsUpdate(t *testing.T) {
	t.Parallel()
	categoryID := productCategoryInEaches(t)
	productID, itemID := parityProduct(t, "")

	writer := customRoleClient(t, "items:update")
	status, body, err := writer.Patch(productsPath+"/"+productID, map[string]any{
		"category_id": categoryID,
		"unit_price":  map[string]any{"value": "3.25", "numerator_unit_id": "dollar", "denominator_unit_id": "each"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)

	after := readMovedItem(t, itemID)
	assert.Equal(t, categoryID, jsonField(jsonObject(after, "category"), "id"))
	assertDecimalEqual(t, "3.25", jsonField(jsonObject(after, "unit_value"), "value"), "the price sent with the move")
	assert.Equal(t, "each", rateDenominator(after, "unit_value"))
	assert.Equal(t, "each", rateDenominator(after, "unit_cost"))

	requireItemCategoryMoveAudited(t, itemID, categoryID)
	expectAuditEvent(t, productID, "product", "update")
}

// A refused save moves nothing and writes none of the fields sent with the move.
func TestItemUpdates_CategoryMoveRefusals(t *testing.T) {
	t.Parallel()
	partID, partItemID := partInSocks(t)
	sku := uniqueName("e2e-moveitem-matref")
	materialID := dashItemsCreateAs(t, apiClient, materialsPath, validMaterialBody(sku))
	materialItemID := dashItemsItemBySKU(t, sku)
	attributeID := dashItemsColorAttribute(t)
	status, body, err := apiClient.Put(itemsPath+"/"+materialItemID+"/attributes/"+attributeID, nil)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(itemsPath + "/" + materialItemID + "/attributes/" + attributeID) })

	ownMaterialCategory := dashItemsMaterialCategory(t, apiClient)
	foreignMaterialCategory := dashItemsTenantBRefs(t).categoryID
	itemsWriter := customRoleClient(t, "items:update")
	notes := uniqueName("never written")

	cases := []struct {
		name   string
		client *Client
		path   string
		body   map[string]any
		check  func(t *testing.T, status int, body []byte)
	}{
		{"a part saved under items:update alone", itemsWriter, partsPath + "/" + partID,
			map[string]any{"category_id": ownMaterialCategory, "notes": notes},
			func(t *testing.T, status int, body []byte) { requirePermissionRefused(t, status, body, "parts:update") }},
		{"a material saved under items:update alone", itemsWriter, materialsPath + "/" + materialID,
			map[string]any{"category_id": ownMaterialCategory, "notes": notes},
			func(t *testing.T, status int, body []byte) { requirePermissionRefused(t, status, body, "materials:update") }},
		{"a part moved to a material category", apiClient, partsPath + "/" + partID,
			map[string]any{"category_id": SeedMaterialCategoryID, "notes": notes},
			func(t *testing.T, status int, body []byte) {
				requireStatus(t, http.StatusBadRequest, status, body)
				assertErrorParam(t, requireErrorResponse(t, body, "validation_failed", "invalid_request_error"), "category_id")
			}},
		{"a material moved to a category that does not carry its attribute", apiClient, materialsPath + "/" + materialID,
			map[string]any{"category_id": ownMaterialCategory, "notes": notes},
			func(t *testing.T, status int, body []byte) {
				requireStatus(t, http.StatusBadRequest, status, body)
				assertErrorParam(t, requireErrorResponse(t, body, "validation_failed", "invalid_request_error"), "category_id")
			}},
		{"a material moved to another tenant's category", apiClient, materialsPath + "/" + materialID,
			map[string]any{"category_id": foreignMaterialCategory, "notes": notes},
			func(t *testing.T, status int, body []byte) {
				requireStatus(t, http.StatusNotFound, status, body)
				requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
			}},
		{"a blank category", apiClient, partsPath + "/" + partID,
			map[string]any{"category_id": "", "notes": notes},
			func(t *testing.T, status int, body []byte) {
				requireStatus(t, http.StatusBadRequest, status, body)
				assertErrorParam(t, requireErrorResponse(t, body, "invalid_format", "invalid_request_error"), "category_id")
			}},
		{"a null category", apiClient, materialsPath + "/" + materialID,
			map[string]any{"category_id": nil, "notes": notes},
			func(t *testing.T, status int, body []byte) {
				requireStatus(t, http.StatusBadRequest, status, body)
				assertErrorParam(t, requireErrorResponse(t, body, "invalid_format", "invalid_request_error"), "category_id")
			}},
	}
	for _, tc := range cases {
		status, body, err := tc.client.Patch(tc.path, tc.body, newIdempotencyKey())
		require.NoError(t, err)
		t.Run(tc.name, func(t *testing.T) { tc.check(t, status, body) })
	}

	for itemID, categoryID := range map[string]string{partItemID: SeedItemCategoryID, materialItemID: SeedMaterialCategoryID} {
		item := readMovedItem(t, itemID)
		assert.Equal(t, categoryID, jsonField(jsonObject(item, "category"), "id"), "the item stays where it was")
		assert.NotEqual(t, notes, jsonField(item, "notes"), "nothing sent with a refused move is written")
	}
}
