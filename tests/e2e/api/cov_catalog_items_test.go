//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file closes e2e gaps for /v1/catalog/items not covered by crud_items_test.go and list_items_test.go: the
// refusals of a PATCH that names a category or attributes that can't apply, and the subassembly_filter
// invalid-value path.

// A PATCH naming a category or attributes that can't apply is refused whole, and the item is left as it was.
func TestCovCatalogItems_UpdateRefusesUnusableCategoryAndAttributes(t *testing.T) {
	t.Parallel()
	body := forgeItemBody("product", uniqueName("e2e-covitems-upd-ref"))
	body["category_id"] = SeedPropertylessItemCategoryID
	itemID := jsonField(createForgeItem(t, itemsPath, body), "id")

	cases := []struct {
		name  string
		body  map[string]any
		param string
	}{
		{"unknown category", map[string]any{"category_id": "itcg_01zzzzzzzzzzzzzzzzzzzzzzz"}, "category_id"},
		{"attribute that doesn't exist", map[string]any{"attribute_ids": []string{"at_01zzzzzzzzzzzzzzzzzzzzzzz"}}, "attribute_ids"},
		// SeedAttributeID is a Color; this item's category carries no properties.
		{"attribute of a property the category lacks", map[string]any{"attribute_ids": []string{SeedAttributeID}}, "attribute_ids"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, resp, err := apiClient.Patch(itemsPath+"/"+itemID, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 422, status, resp)
			assertErrorParam(t, requireErrorResponse(t, resp, "invalid_format", "invalid_request_error"), tc.param)
		})
	}

	got := getItem(t, itemID, "category", "attributes")
	assert.Equal(t, SeedPropertylessItemCategoryID, jsonField(jsonObject(got, "category"), "id"), "a refused update moves nothing")
	assert.Empty(t, attributeIDsOf(got), "a refused update assigns nothing")
}

// An invalid subassembly_filter names the query parameter the caller sent, not the Go field behind it.
func TestCovCatalogItems_ListItems_SubassemblyFilterInvalidValue(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath, url.Values{"subassembly_filter": {"bogus_value"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "subassembly_filter")
}
