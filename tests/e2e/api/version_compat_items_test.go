//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// preview9ItemFields are the forge item fields a preview.9 caller never sees.
var preview9ItemFields = []string{"status", "supply", "sales", "material", "default_manufacturing_method"}

func getPreview9Item(t *testing.T, itemID string) map[string]any {
	t.Helper()
	status, body, err := apiClient.WithAPIVersion(preview9Version).GetListRaw(itemsPath+"/"+itemID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func assertPreview9ItemShape(t *testing.T, item map[string]any) {
	t.Helper()
	assertObjectField(t, item, "item")
	for _, field := range preview9ItemFields {
		assert.NotContains(t, item, field, "preview.9 items have no %s", field)
	}
	notes, has := item["notes"]
	assert.True(t, has, "preview.9 items keep notes")
	assert.Nil(t, notes, "forge stores no notes")
	assert.Contains(t, item, "unit_value")
}

// A sold item's unit_value is its sales price as a preview.9 rate; an item that isn't sold has none.
func TestVersionCompatItems_ItemShapeOnPreview9(t *testing.T) {
	t.Parallel()

	t.Run("product", func(t *testing.T) {
		body := forgeItemBody("product", uniqueName("e2e-compat-item-prod"))
		body["sales"] = map[string]any{"unit_price": priceInputPer("4.50", SeedUnitID)}
		itemID := jsonField(createForgeItem(t, itemsPath, body), "id")

		item := getPreview9Item(t, itemID)
		assertPreview9ItemShape(t, item)
		assert.Equal(t, "product", jsonField(item, "type"))
		value := jsonObject(item, "unit_value")
		require.NotNil(t, value, "a product is sold")
		assert.Equal(t, "rate", jsonField(value, "object"))
		assertDecimalEqual(t, "4.50", jsonField(value, "value"), "unit_value.value")
	})

	t.Run("service reads as a product", func(t *testing.T) {
		itemID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("service", uniqueName("e2e-compat-item-svc"))), "id")
		item := getPreview9Item(t, itemID)
		assertPreview9ItemShape(t, item)
		assert.Equal(t, "product", jsonField(item, "type"))
		assert.NotNil(t, jsonObject(item, "unit_value"), "a service is sold")
	})

	t.Run("material not sold has no unit value", func(t *testing.T) {
		itemID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("material", uniqueName("e2e-compat-item-mat"))), "id")
		item := getPreview9Item(t, itemID)
		assertPreview9ItemShape(t, item)
		assert.Equal(t, "material", jsonField(item, "type"))
		assertNilField(t, item, "unit_value")
	})

	t.Run("list items take the same shape", func(t *testing.T) {
		list, _, err := apiClient.WithAPIVersion(preview9Version).GetList(itemsPath, url.Values{"limit": {"10"}})
		require.NoError(t, err)
		require.NotEmpty(t, list.Data)
		for _, raw := range list.Data {
			assertPreview9ItemShape(t, parseJSON(raw))
		}
	})
}

// The preview.9 list keeps starts_at / ends_at / supplier_id and hides archived items, as it hid deleted ones.
func TestVersionCompatItems_ListOnPreview9(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	prefix := uniqueName("e2e-compat-item-list")
	activeID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("part", prefix+"-a")), "id")
	archived := forgeItemBody("part", prefix+"-b")
	archived["status"] = "archived"
	archivedID := jsonField(createForgeItem(t, itemsPath, archived), "id")

	ids := func(c *Client, params url.Values) []string {
		list, _, err := c.GetList(itemsPath, params)
		require.NoError(t, err)
		out := make([]string, 0, len(list.Data))
		for _, raw := range list.Data {
			out = append(out, DataItemField(raw, "id"))
		}
		return out
	}

	assert.Equal(t, []string{activeID}, ids(preview, url.Values{"q": {prefix}}), "preview.9 hides archived items")
	assert.ElementsMatch(t, []string{activeID, archivedID}, ids(apiClient, url.Values{"q": {prefix}}), "forge lists them")

	assert.Equal(t, []string{activeID}, ids(preview, url.Values{"q": {prefix}, "starts_at": {"2000-01-01T00:00:00Z"}}))
	assert.Empty(t, ids(preview, url.Values{"q": {prefix}, "starts_at": {"2999-01-01T00:00:00Z"}}))
	assert.Empty(t, ids(preview, url.Values{"q": {prefix}, "ends_at": {"2000-01-01T00:00:00Z"}}))

	found := listFindByFieldWith(t, preview, url.Values{"supplier_id": {SeedSupplierAccountID}}, SeedMaterialItemID)
	assert.True(t, found, "preview.9 supplier_id still filters by supplier")
	assert.Empty(t, ids(preview, url.Values{"supplier_id": {"ac_00000000000000000000000000"}}))
}

// listFindByFieldWith pages a client's item list looking for an id.
func listFindByFieldWith(t *testing.T, c *Client, params url.Values, itemID string) bool {
	t.Helper()
	list, _, err := c.GetList(itemsPath, params)
	require.NoError(t, err)
	for page := 0; ; page++ {
		for _, raw := range list.Data {
			if DataItemField(raw, "id") == itemID {
				return true
			}
		}
		if !list.PageInfo.HasNextPage() || page >= 50 {
			return false
		}
		list, _, err = c.GetListFromPageURL(list.PageInfo.NextPageURL)
		require.NoError(t, err)
	}
}

// The attribute and category sub-routes were replaced by PATCH; they answer preview.9 callers only.
func TestVersionCompatItems_SubRoutesArePreview9Only(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	itemID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("product", uniqueName("e2e-compat-item-sub"))), "id")
	attributeRoute := itemsPath + "/" + itemID + "/attributes/" + SeedAttributeID
	categoryRoute := itemsPath + "/" + itemID + "/category/" + SeedItemCategoryID

	t.Run("forge 404s", func(t *testing.T) {
		status, body, err := apiClient.Put(attributeRoute, nil)
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
		status, body, err = apiClient.Delete(attributeRoute)
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
		status, body, err = apiClient.Put(categoryRoute, nil)
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
		assert.Empty(t, attributeIDsOf(getItem(t, itemID, "attributes")), "a refused route writes nothing")
	})

	t.Run("preview.9 still works", func(t *testing.T) {
		status, body, err := preview.Put(attributeRoute+"?include=attributes", nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		got := parseJSON(body)
		assertPreview9ItemShape(t, got)
		assert.Equal(t, []string{SeedAttributeID}, attributeIDsOf(got))

		status, body, err = preview.Put(categoryRoute+"?include=category", nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		assert.Equal(t, SeedItemCategoryID, jsonField(jsonObject(parseJSON(body), "category"), "id"))

		status, body, err = preview.Delete(attributeRoute)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		assert.Empty(t, attributeIDsOf(getItem(t, itemID, "attributes")))
	})
}

// preview.9 creates items through the per-type routes, and forge reads what they made.
func TestVersionCompatItems_Preview9ProductReadsOnForge(t *testing.T) {
	t.Parallel()
	_, itemID := newProductItemIDs(t, "e2e-compat-item-pd")
	item := getItem(t, itemID)
	assertItemEnvelope(t, item, "product", jsonField(item, "sku"))
	assert.Equal(t, "active", jsonField(item, "status"))
	assert.NotNil(t, jsonObject(item, "sales"), "a product made on preview.9 has a sales section on forge")
}

// The preview.9 list's active-only default is the list's alone: a preview.9 create on the same route still succeeds.
func TestVersionCompatItems_CreateOnPreview9(t *testing.T) {
	t.Parallel()
	resp, err := apiClient.WithAPIVersion(preview9Version).PostFull(itemsPath, forgeItemBody("part", uniqueName("e2e-compat-item-create")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	created := parseJSON(resp.Body)
	itemID := jsonField(created, "id")
	require.NotEmpty(t, itemID)
	t.Cleanup(func() { apiClient.Delete(itemsPath + "/" + itemID) })
	assertPreview9ItemShape(t, created)
}
