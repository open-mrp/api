//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Saving an item with a new category is one request. On forge, PATCH /v1/catalog/items/{id} takes category_id under
// items:update and changes only the category. preview.9's PATCH on a part, material or product still takes category_id
// under that route's own permission and moves the item's prices onto the new category's base unit.

// productCategoryInEaches creates a product category whose unit group counts in eaches.
func productCategoryInEaches(t *testing.T) string {
	t.Helper()
	unitGroupID := dashItemsCreateAs(t, apiClient, unitGroupsPath, map[string]any{
		"name": uniqueName("e2e-moveitem-ug"), "type": "quantity", "base_unit_id": "each",
	})
	return dashItemsCreateAs(t, apiClient, itemCategoriesPath, map[string]any{
		"name": uniqueName("e2e-moveitem-cat"), "type": "product", "unit_group_id": unitGroupID,
	})
}

// partInSocks creates a part in the seeded socks category through the preview.9 parts route and returns the part and
// the item behind it.
func partInSocks(t *testing.T) (partID, itemID string) {
	t.Helper()
	sku := uniqueName("e2e-moveitem-part")
	partID = dashItemsCreateAs(t, apiClient, partsPath, validPartBody(sku))
	return partID, dashItemsItemBySKU(t, sku)
}

// unitCostPerUnit is the unit a forge item's cost is per; the cost itself is always in dollars.
func unitCostPerUnit(item map[string]any) string {
	return jsonField(jsonObject(jsonObject(item, "unit_cost"), "per_unit"), "id")
}

// requireItemCategoryMoveAudited waits for the item's own update event and checks it records the move to categoryID.
func requireItemCategoryMoveAudited(t *testing.T, itemID, categoryID string) {
	t.Helper()
	event := expectAuditEventWithChanges(t, itemID, "item", "update")
	change, ok := changeForField(jsonListData(event, "changes"), "item_category_id")
	require.True(t, ok, "the item's update event records the category it moved to")
	assert.Equal(t, categoryID, jsonField(change, "new_value"))
}

// requirePreview9Refused checks a preview.9 refusal: its status, and for a 400 the field it names.
func requirePreview9Refused(t *testing.T, status int, body []byte, wantStatus int, code, param string) {
	t.Helper()
	requireStatus(t, wantStatus, status, body)
	errObj := requirePreview9Error(t, body, code)
	if param != "" {
		assert.Equal(t, param, errObj["param"], "error.param (body: %s)", body)
	}
}

func TestItemUpdate_MovesTheCategoryUnderItemsUpdateAlone(t *testing.T) {
	t.Parallel()
	categoryID := productCategoryInEaches(t)
	body := forgeItemBody("part", uniqueName("e2e-moveitem-forge"))
	body["unit_cost"] = priceInputPer("2.00", SeedUnitID)
	itemID := jsonField(createForgeItem(t, itemsPath, body), "id")

	status, resp, err := customRoleClient(t, "items:update").Patch(withQuery(itemsPath+"/"+itemID, url.Values{"include": {"category"}}),
		map[string]any{"category_id": categoryID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, resp)
	assert.Equal(t, categoryID, jsonField(jsonObject(parseJSON(resp), "category"), "id"))

	after := getItem(t, itemID, "category")
	assert.Equal(t, categoryID, jsonField(jsonObject(after, "category"), "id"))
	assertItemPrice(t, jsonObject(after, "unit_cost"), "2.00", SeedUnitID, "unit_cost stays as it was")

	requireItemCategoryMoveAudited(t, itemID, categoryID)
}

func TestPartUpdatePreview9_MovesTheCategoryUnderPartsUpdateAlone(t *testing.T) {
	t.Parallel()
	categoryID := productCategoryInEaches(t)
	partID, itemID := partInSocks(t)
	before := getItem(t, itemID)
	require.NotEqual(t, "each", unitCostPerUnit(before), "the socks category does not count in eaches")

	writer := customRoleClient(t, "parts:update").WithAPIVersion(preview9Version)
	status, body, err := writer.Patch(withQuery(partsPath+"/"+partID, url.Values{"include": {"item,item.category"}}), map[string]any{
		"category_id": categoryID,
		"notes":       uniqueName("notes are no longer stored"),
		"description": uniqueName("moved with its category"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	item := jsonObject(parseJSON(body), "item")
	require.NotNil(t, item)
	assert.Equal(t, categoryID, jsonField(jsonObject(item, "category"), "id"))
	assertNilField(t, item, "notes")

	after := getItem(t, itemID, "category")
	assert.Equal(t, categoryID, jsonField(jsonObject(after, "category"), "id"))
	assert.Equal(t, "each", unitCostPerUnit(after), "unit_cost moves onto the new base unit")
	assertDecimalEqual(t, jsonField(jsonObject(before, "unit_cost"), "amount"), jsonField(jsonObject(after, "unit_cost"), "amount"), "unit_cost keeps its figure")

	requireItemCategoryMoveAudited(t, itemID, categoryID)
	partEvent := expectAuditEventWithChanges(t, partID, "part", "update")
	_, movedOnPart := changeForField(jsonListData(partEvent, "changes"), "item_category_id")
	assert.False(t, movedOnPart, "the move is recorded on the item, the rest of the save on the part")
}

// The dashboard sent the cost in the new base unit alongside the move; it is written after the move, so it stays as sent.
func TestMaterialUpdatePreview9_MovesTheCategoryUnderMaterialsUpdateAlone(t *testing.T) {
	t.Parallel()
	categoryID := dashItemsMaterialCategory(t, apiClient)
	sku := uniqueName("e2e-moveitem-mat")
	materialID := dashItemsCreateAs(t, apiClient, materialsPath, validMaterialBody(sku))
	itemID := dashItemsItemBySKU(t, sku)

	writer := customRoleClient(t, "materials:update").WithAPIVersion(preview9Version)
	status, body, err := writer.Patch(materialsPath+"/"+materialID, map[string]any{
		"category_id": categoryID,
		"unit_cost":   map[string]any{"amount": "0.40", "currency": "usd", "per_unit_id": "each"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)

	after := getItem(t, itemID, "category")
	assert.Equal(t, categoryID, jsonField(jsonObject(after, "category"), "id"))
	assertItemPrice(t, jsonObject(after, "unit_cost"), "0.40", "each", "the cost sent with the move")
	assert.Equal(t, []string{itemID}, listIDs(t, itemsPath, url.Values{"q": {sku}, "category_ids": {categoryID}}))

	requireItemCategoryMoveAudited(t, itemID, categoryID)
	expectAuditEvent(t, materialID, "material", "update")
}

func TestProductUpdatePreview9_MovesTheCategoryUnderItemsUpdate(t *testing.T) {
	t.Parallel()
	categoryID := productCategoryInEaches(t)
	productID, itemID := parityProduct(t, "")

	writer := customRoleClient(t, "items:update").WithAPIVersion(preview9Version)
	status, body, err := writer.Patch(productsPath+"/"+productID, map[string]any{
		"category_id": categoryID,
		"unit_price":  map[string]any{"amount": "3.25", "currency": "usd", "per_unit_id": "each"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)

	after := getItem(t, itemID, "category")
	assert.Equal(t, categoryID, jsonField(jsonObject(after, "category"), "id"))
	sales := jsonObject(after, "sales")
	require.NotNil(t, sales, "a product is sold")
	assertItemPrice(t, jsonObject(sales, "unit_price"), "3.25", "each", "the price sent with the move")
	assert.Equal(t, "each", unitCostPerUnit(after), "unit_cost moves onto the new base unit")

	// A product's price is its item's, so the move and the price are both audited on the item; a product update event
	// is published only when a product field changes.
	requireItemCategoryMoveAudited(t, itemID, categoryID)
}

// A refused preview.9 save moves nothing and writes none of the fields sent with the move.
func TestItemUpdatesPreview9_CategoryMoveRefusals(t *testing.T) {
	t.Parallel()
	partID, partItemID := partInSocks(t)
	sku := uniqueName("e2e-moveitem-matref")
	materialID := dashItemsCreateAs(t, apiClient, materialsPath, validMaterialBody(sku))
	materialItemID := dashItemsItemBySKU(t, sku)
	attributeID := dashItemsColorAttribute(t)
	patchItem(t, materialItemID, map[string]any{"attribute_ids": []string{attributeID}})

	ownMaterialCategory := dashItemsMaterialCategory(t, apiClient)
	foreignMaterialCategory := dashItemsTenantBRefs(t).categoryID
	preview := apiClient.WithAPIVersion(preview9Version)
	itemsWriter := customRoleClient(t, "items:update").WithAPIVersion(preview9Version)
	description := uniqueName("never written")

	cases := []struct {
		name       string
		client     *Client
		path       string
		body       map[string]any
		wantStatus int
		code       string
		param      string
	}{
		{"a part saved under items:update alone", itemsWriter, partsPath + "/" + partID,
			map[string]any{"category_id": ownMaterialCategory, "description": description}, http.StatusForbidden, "insufficient_permissions", ""},
		{"a material saved under items:update alone", itemsWriter, materialsPath + "/" + materialID,
			map[string]any{"category_id": ownMaterialCategory, "description": description}, http.StatusForbidden, "insufficient_permissions", ""},
		{"a part moved to a material category", preview, partsPath + "/" + partID,
			map[string]any{"category_id": SeedMaterialCategoryID, "description": description}, http.StatusBadRequest, "", "category_id"},
		{"a material moved to a category that does not carry its attribute", preview, materialsPath + "/" + materialID,
			map[string]any{"category_id": ownMaterialCategory, "description": description}, http.StatusBadRequest, "", "category_id"},
		{"a material moved to another tenant's category", preview, materialsPath + "/" + materialID,
			map[string]any{"category_id": foreignMaterialCategory, "description": description}, http.StatusNotFound, "resource_not_found", ""},
		{"a blank category", preview, partsPath + "/" + partID,
			map[string]any{"category_id": "", "description": description}, http.StatusBadRequest, "invalid_format", "category_id"},
		{"a null category", preview, materialsPath + "/" + materialID,
			map[string]any{"category_id": nil, "description": description}, http.StatusBadRequest, "invalid_format", "category_id"},
	}
	for _, tc := range cases {
		status, body, err := tc.client.Patch(tc.path, tc.body, newIdempotencyKey())
		require.NoError(t, err)
		t.Run(tc.name, func(t *testing.T) {
			if tc.code == "" {
				requireStatus(t, tc.wantStatus, status, body)
				errObj, _ := parseJSON(body)["error"].(map[string]any)
				require.NotNil(t, errObj, "%s", body)
				assert.Equal(t, tc.param, errObj["param"], "error.param (body: %s)", body)
				return
			}
			requirePreview9Refused(t, status, body, tc.wantStatus, tc.code, tc.param)
		})
	}

	for itemID, categoryID := range map[string]string{partItemID: SeedItemCategoryID, materialItemID: SeedMaterialCategoryID} {
		item := getItem(t, itemID, "category")
		assert.Equal(t, categoryID, jsonField(jsonObject(item, "category"), "id"), "the item stays where it was")
		assert.NotEqual(t, description, jsonField(item, "description"), "nothing sent with a refused move is written")
	}
}
