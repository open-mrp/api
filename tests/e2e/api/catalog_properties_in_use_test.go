//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createAttributeForTest adds an attribute to propertyID and deletes it when the test ends.
func createAttributeForTest(t *testing.T, propertyID string, body map[string]any) map[string]any {
	t.Helper()
	status, raw, err := apiClient.Post(attributesPath(propertyID), body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, raw)
	created := parseJSON(raw)
	t.Cleanup(func() { apiClient.Delete(attributePath(propertyID, jsonField(created, "id"))) })
	return created
}

// productWithAttribute creates a product in the seed category, which carries the seed property, with attributeID
// assigned, and deletes it before the attribute when the test ends.
func productWithAttribute(t *testing.T, attributeID string) string {
	t.Helper()
	body := forgeItemBody("product", uniqueName("e2e-attr-inuse"))
	body["attribute_ids"] = []string{attributeID}
	return jsonField(createForgeItem(t, itemsPath, body), "id")
}

// An attribute an item uses can't be deleted; archived, it stays on the item but can't be assigned again.
func TestAttributes_InUseAndArchived(t *testing.T) {
	t.Parallel()
	attr := createAttributeForTest(t, SeedPropertyID, map[string]any{"value": uniqueName("e2e-attr-inuse")})
	attrID := jsonField(attr, "id")
	itemID := productWithAttribute(t, attrID)

	status, body, err := apiClient.Delete(attributePath(SeedPropertyID, attrID))
	require.NoError(t, err)
	requireStatus(t, http.StatusConflict, status, body)
	errObj := requireErrorResponse(t, body, "resource_in_use", "invalid_request_error")
	assert.Contains(t, jsonField(errObj, "message"), "items")
	assert.Contains(t, jsonField(errObj, "message"), "archived")

	status, body, err = apiClient.Patch(attributePath(SeedPropertyID, attrID), map[string]any{"status": "archived"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	assert.Equal(t, "archived", jsonField(parseJSON(body), "status"))

	status, body, err = apiClient.GetListRaw(itemsPath+"/"+itemID, url.Values{"include": {"attributes,attributes.property"}})
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	attrs := jsonListData(parseJSON(body), "attributes")
	require.Len(t, attrs, 1, "the archived attribute stays on the item")
	onItem := attrs[0].(map[string]any)
	assert.Equal(t, "archived", jsonField(onItem, "status"))
	assert.Equal(t, SeedPropertyID, jsonField(jsonObject(onItem, "property"), "id"))

	otherItemID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("product", uniqueName("e2e-attr-archived"))), "id")
	status, body, err = apiClient.Patch(itemsPath+"/"+otherItemID, map[string]any{"attribute_ids": []string{attrID}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusUnprocessableEntity, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "resource_archived", "invalid_request_error"), "attribute_ids")

	list, _, err := apiClient.GetList(attributesPath(SeedPropertyID), url.Values{"statuses[]": {"archived"}, "limit": {"100"}})
	require.NoError(t, err)
	found := false
	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Equal(t, "archived", jsonField(m, "status"))
		found = found || jsonField(m, "id") == attrID
	}
	assert.True(t, found, "statuses[]=archived lists the archived attribute")
}

// Deleting an attribute nothing uses returns the deleted stub.
func TestAttributes_DeleteUnusedReturnsStub(t *testing.T) {
	t.Parallel()
	prop := createAndCleanup(t, propertiesPath, map[string]any{"name": uniqueName("e2e-attr-del")})
	propID := jsonField(prop, "id")
	status, body, err := apiClient.Post(attributesPath(propID), map[string]any{"value": "Large"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, body)
	attrID := jsonField(parseJSON(body), "id")

	status, body, err = apiClient.Delete(attributePath(propID, attrID))
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	assert.Equal(t, map[string]any{"id": attrID, "object": "attribute", "deleted": true}, parseJSON(body))
}

// A property a category lists, or whose attribute an item uses, can't be deleted.
func TestProperties_DeleteInUse(t *testing.T) {
	t.Parallel()

	onCategory := createPropertyForTest(t, uniqueName("e2e-prop-oncat"))
	category := createAndCleanup(t, itemCategoriesPath, map[string]any{"name": uniqueName("e2e-prop-oncat-c"), "type": "product", "property_ids": []string{onCategory}})
	status, body, err := apiClient.Delete(propertiesPath + "/" + onCategory)
	require.NoError(t, err)
	requireStatus(t, http.StatusConflict, status, body)
	assert.Contains(t, jsonField(requireErrorResponse(t, body, "resource_in_use", "invalid_request_error"), "message"), "item categories")

	status, body, err = apiClient.Patch(itemCategoriesPath+"/"+jsonField(category, "id"), map[string]any{"property_ids": []string{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	status, body, err = apiClient.Delete(propertiesPath + "/" + onCategory)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	assert.Equal(t, map[string]any{"id": onCategory, "object": "property", "deleted": true}, parseJSON(body))

	// SeedPropertyID is listed by the seed category and its attributes are on seed items.
	status, body, err = apiClient.Delete(propertiesPath + "/" + SeedPropertyID)
	require.NoError(t, err)
	requireStatus(t, http.StatusConflict, status, body)
	requireErrorResponse(t, body, "resource_in_use", "invalid_request_error")
}

// The properties list narrows to the categories that list them.
func TestProperties_ListByItemCategory(t *testing.T) {
	t.Parallel()
	listed := createPropertyForTest(t, uniqueName("e2e-prop-bycat"))
	createPropertyForTest(t, uniqueName("e2e-prop-notcat"))
	category := createAndCleanup(t, itemCategoriesPath, map[string]any{"name": uniqueName("e2e-prop-bycat-c"), "type": "material", "property_ids": []string{listed}})

	list, _, err := apiClient.GetList(propertiesPath, url.Values{"item_category_ids[]": {jsonField(category, "id")}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	assert.Equal(t, listed, jsonField(parseJSON(list.Data[0]), "id"))
}

// A property's embedded attributes stop at ten and link to the attributes list.
func TestProperties_IncludeAttributesPagesAfterTen(t *testing.T) {
	t.Parallel()
	prop := createAndCleanup(t, propertiesPath, map[string]any{"name": uniqueName("e2e-prop-page")})
	propID := jsonField(prop, "id")
	for i := 0; i < 11; i++ {
		createAttributeForTest(t, propID, map[string]any{"value": uniqueName("e2e-attr-page")})
	}
	status, body, err := apiClient.GetListRaw(propertiesPath+"/"+propID, url.Values{"include": {"attributes"}})
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	attrs := jsonObject(parseJSON(body), "attributes")
	require.NotNil(t, attrs)
	assert.Len(t, attrs["data"], 10)
	assert.Equal(t, propertiesPath+"/"+propID+"/attributes", jsonField(jsonObject(attrs, "page_info"), "next_page_url"))
}

// preview.9 callers keep what they relied on: an item's included attributes carry their property, a new attribute
// without a color gets a named one, and categories read with their old type names.
func TestVersionCompatCatalog_Preview9(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)

	status, body, err := preview.GetListRaw(itemsPath+"/"+SeedItemID, url.Values{"include": {"attributes,category"}})
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	item := parseJSON(body)
	for _, a := range jsonListData(item, "attributes") {
		assert.NotNil(t, jsonObject(a.(map[string]any), "property"), "preview.9 reads each attribute's property")
	}
	assert.Contains(t, []string{"material_category", "product_category"}, jsonField(jsonObject(item, "category"), "type"))

	status, body, err = apiClient.GetListRaw(itemsPath+"/"+SeedItemID, url.Values{"include": {"attributes"}})
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	for _, a := range jsonListData(parseJSON(body), "attributes") {
		assert.Nil(t, a.(map[string]any)["property"], "forge leaves the property null unless included")
	}

	prop := createAndCleanup(t, propertiesPath, map[string]any{"name": uniqueName("e2e-p9-color")})
	propID := jsonField(prop, "id")
	status, body, err = preview.Post(attributesPath(propID), map[string]any{"value": "Large"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, body)
	created := parseJSON(body)
	t.Cleanup(func() { apiClient.Delete(attributePath(propID, jsonField(created, "id"))) })
	assert.NotEqual(t, "default", jsonField(created, "color"))
	assert.NotEmpty(t, jsonField(created, "color"))

	status, body, err = preview.GetListRaw(itemCategoriesPath+"/"+SeedMaterialCategoryID, nil)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	assert.Equal(t, "material_category", jsonField(parseJSON(body), "type"))
}
