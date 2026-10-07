//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dashboardItemIncludes is what the dashboard's item list and category change ask for.
var dashboardItemIncludes = []string{
	"category", "category.unit_group", "category.unit_group.base_unit", "category.unit_group.associated_units",
	"category.unit_group.associated_units.unit", "category.properties", "unit_value", "unit_cost", "burn_rate", "attributes",
}

// itemWithAttribute creates a sock in the seeded socks category carrying the seeded attribute, and returns its SKU and item.
func itemWithAttribute(t *testing.T) (sku, itemID string) {
	t.Helper()
	sku = uniqueName("e2e-embed-sku")
	created := createAndCleanup(t, productsPath, map[string]any{"sku": sku, "type": "sale", "category_id": SeedItemCategoryID})
	productID := jsonField(created, "id")
	itemID = jsonField(jsonObject(parseJSON(mustGetAs(t, apiClient, productsPath+"/"+productID, url.Values{"include": {"item"}})), "item"), "id")
	require.NotEmpty(t, itemID)
	status, body, err := apiClient.Put(itemsPath+"/"+itemID+"/attributes/"+SeedAttributeID, nil)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	return sku, itemID
}

// An items reader sees an item's category and attribute properties as an admin does; on their own they stay gated.
func TestIncludes_ItemCategoryAndAttributesShownToAnItemsReader(t *testing.T) {
	t.Parallel()
	reader := customRoleClient(t, "items:read")
	sku, itemID := itemWithAttribute(t)

	for _, path := range []string{itemCategoriesPath, itemCategoriesPath + "/" + SeedItemCategoryID, propertiesPath, propertiesPath + "/" + SeedPropertyID} {
		status, body, err := reader.GetListRaw(path, nil)
		require.NoError(t, err)
		requireStatus(t, http.StatusForbidden, status, body)
		requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
	}

	// Unit cost is the one field the role does not share with the admin: it needs costs:read.
	withoutCost := func(item map[string]any) map[string]any {
		item["unit_cost"] = nil
		return item
	}

	listParams := url.Values{"q": {sku}, "include": dashboardItemIncludes}
	gotList := parseJSON(mustGetAs(t, reader, itemsPath, listParams))
	wantList := parseJSON(mustGetAs(t, apiClient, itemsPath, listParams))
	for _, row := range jsonArray(wantList, "data") {
		withoutCost(row.(map[string]any))
	}
	assert.Equal(t, wantList, gotList, "the role lists the item as an admin does, less its cost")
	rows := jsonArray(gotList, "data")
	require.Len(t, rows, 1)

	retrieveParams := url.Values{"include": dashboardItemIncludes}
	got := parseJSON(mustGetAs(t, reader, itemsPath+"/"+itemID, retrieveParams))
	assert.Equal(t, withoutCost(parseJSON(mustGetAs(t, apiClient, itemsPath+"/"+itemID, retrieveParams))), got, "the role retrieves the item as an admin does, less its cost")
	assert.Equal(t, rows[0], any(got), "the list row and the retrieved item agree")

	category := jsonObject(got, "category")
	require.NotNil(t, category, "the category is expanded")
	assert.Equal(t, SeedItemCategoryID, jsonField(category, "id"))
	assert.NotEmpty(t, jsonField(category, "name"))
	properties := jsonArray(jsonObject(category, "properties"), "data")
	require.NotEmpty(t, properties, "the socks category has properties")
	for _, raw := range properties {
		assert.NotEmpty(t, jsonField(raw.(map[string]any), "name"))
	}
	unitGroup := jsonObject(category, "unit_group")
	require.NotNil(t, unitGroup, "the category's unit group is expanded")
	require.NotNil(t, jsonObject(unitGroup, "base_unit"), "the unit group's base unit is expanded")
	associated := jsonArray(jsonObject(unitGroup, "associated_units"), "data")
	require.NotEmpty(t, associated)
	for _, raw := range associated {
		assert.NotNil(t, jsonObject(raw.(map[string]any), "unit"), "each associated unit carries its unit")
	}

	attributes := jsonArray(jsonObject(got, "attributes"), "data")
	require.Len(t, attributes, 1)
	attribute := attributes[0].(map[string]any)
	assert.Equal(t, SeedAttributeID, jsonField(attribute, "id"))
	property := jsonObject(attribute, "property")
	require.NotNil(t, property, "the attribute names its property")
	assert.Equal(t, SeedPropertyID, jsonField(property, "id"))
	assert.NotEmpty(t, jsonField(property, "name"))
}

// The category comes with the item wherever the item is embedded, here a product's.
func TestIncludes_EmbeddedItemCarriesItsCategoryForAnItemsReader(t *testing.T) {
	t.Parallel()
	reader := customRoleClient(t, "items:read")
	sku, itemID := itemWithAttribute(t)
	list, _, err := apiClient.GetList(productsPath, url.Values{"q": {sku}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	path := productsPath + "/" + DataItemField(list.Data[0], "id")

	params := url.Values{"include": {"item", "item.category", "item.category.properties", "item.category.unit_group", "item.attributes"}}
	got := parseJSON(mustGetAs(t, reader, path, params))
	assert.Equal(t, parseJSON(mustGetAs(t, apiClient, path, params)), got, "the role reads the product exactly as an admin does")
	item := jsonObject(got, "item")
	require.NotNil(t, item)
	assert.Equal(t, itemID, jsonField(item, "id"))
	require.NotNil(t, jsonObject(item, "category"))
	assert.Equal(t, SeedItemCategoryID, jsonField(jsonObject(item, "category"), "id"))
	assert.NotNil(t, jsonObject(jsonObject(item, "category"), "unit_group"))
}

// Read with the item does not mean read without asking: the category stays null until it is included.
func TestIncludes_ItemCategoryStaysNullUnlessIncluded(t *testing.T) {
	t.Parallel()
	reader := customRoleClient(t, "items:read")
	_, itemID := itemWithAttribute(t)

	got := parseJSON(mustGetAs(t, reader, itemsPath+"/"+itemID, nil))
	assert.Nil(t, got["category"], "category is null without include=category")
	assert.Nil(t, got["attributes"], "attributes are null without include=attributes")

	got = parseJSON(mustGetAs(t, reader, itemsPath+"/"+itemID, url.Values{"include": {"category"}}))
	category := jsonObject(got, "category")
	require.NotNil(t, category)
	assert.Nil(t, category["properties"], "the category's properties stay null until included")
	assert.Nil(t, category["unit_group"], "the category's unit group stays null until included")
}
