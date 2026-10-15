//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Edge coverage for the preview.9-only /v1/catalog/products routes. Their forge counterparts are on the items routes.

// covCatalogProductsListContainsID reports whether id appears in a list response's data.
func covCatalogProductsListContainsID(list *ListResponse, id string) bool {
	if list == nil {
		return false
	}
	for _, raw := range list.Data {
		if DataItemField(raw, "id") == id {
			return true
		}
	}
	return false
}

const covCatalogProductsNonexistentID = "pd_00000000000000000000000000"

func TestCovCatalogProductsPreview9_DuplicateSKUConflicts(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	skuA := uniqueName("e2e-cov-p9-prod-dupA")
	createPreview9Product(t, productsPath, validProductBody(skuA))
	idB := jsonField(createPreview9Product(t, productsPath, validProductBody(uniqueName("e2e-cov-p9-prod-dupB"))), "id")

	status, body, err := preview.Post(productsPath, validProductBody(skuA), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	assertErrorParam(t, requirePreview9Error(t, body, "resource_conflict"), "sku")

	status, body, err = preview.Patch(productsPath+"/"+idB, map[string]any{"sku": skuA}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	assertErrorParam(t, requirePreview9Error(t, body, "resource_conflict"), "sku")
}

func TestCovCatalogProductsPreview9_UpdateValidation(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	productID := jsonField(createPreview9Product(t, productsPath, validProductBody(uniqueName("e2e-cov-p9-prod-updval"))), "id")

	cases := []struct {
		name  string
		body  map[string]any
		code  string
		param string
	}{
		{"null sku", map[string]any{"sku": nil}, "invalid_format", "sku"},
		{"unknown portal visibility", map[string]any{"portal_visibility": "public"}, "parameter_invalid", "portal_visibility"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := preview.Patch(productsPath+"/"+productID, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			assertErrorParam(t, requirePreview9Error(t, body, tc.code), tc.param)
		})
	}
}

// A null clears description (and notes, which read null regardless); a blank string is a value, not a clear.
func TestCovCatalogProductsPreview9_ClearAndBlankDescription(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	body := validProductBody(uniqueName("e2e-cov-p9-prod-clear"))
	body["description"] = "has a description"
	productID := jsonField(createPreview9Product(t, productsPath, body), "id")

	status, respBody, err := preview.Patch(productsPath+"/"+productID+"?include=item", map[string]any{"description": ""}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	desc, ok := jsonObject(parseJSON(respBody), "item")["description"]
	require.True(t, ok, "description key should be present after a blank-string update")
	assert.Equal(t, "", desc, "a blank string sets the description")

	status, respBody, err = preview.Patch(productsPath+"/"+productID+"?include=item", map[string]any{
		"description": nil,
		"notes":       nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	assertNilField(t, jsonObject(parseJSON(respBody), "item"), "description")

	item := jsonObject(getPreview9Product(t, productID, "item"), "item")
	assertNilField(t, item, "description")
	assertNilField(t, item, "notes")
}

func TestCovCatalogProductsPreview9_NotFound(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	path := productsPath + "/" + covCatalogProductsNonexistentID

	status, body, err := preview.GetListRaw(path, nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)

	status, body, err = preview.Patch(path, map[string]any{"sku": uniqueName("e2e-cov-p9-prod-nf")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)

	status, body, err = preview.Delete(path)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// The preview.9 list keeps its filters: category_ids, attribute_ids, customer_ids, portal_visibility and
// starts_at/ends_at.
func TestCovCatalogProductsPreview9_ListFilters(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)

	visibleBody := validProductBody(uniqueName("e2e-cov-p9-prod-list-vis"))
	visibleBody["portal_visibility"] = "visible"
	visibleBody["product_line_id"] = SeedProductLineID
	visibleID := jsonField(createPreview9Product(t, productsPath, visibleBody), "id")

	hiddenBody := validProductBody(uniqueName("e2e-cov-p9-prod-list-hid"))
	hiddenBody["portal_visibility"] = "hidden"
	hiddenID := jsonField(createPreview9Product(t, productsPath, hiddenBody), "id")

	list := func(params url.Values) *ListResponse {
		t.Helper()
		params.Set("limit", "100")
		got, status, err := preview.GetList(productsPath, params)
		require.NoError(t, err)
		require.Equal(t, 200, status)
		return got
	}

	cases := []struct {
		name     string
		params   url.Values
		has, not []string
	}{
		{"visible", url.Values{"portal_visibility": {"visible"}}, []string{visibleID}, []string{hiddenID}},
		{"hidden", url.Values{"portal_visibility": {"hidden"}}, []string{hiddenID}, []string{visibleID}},
		{"category", url.Values{"category_ids": {SeedItemCategoryID}}, []string{visibleID, hiddenID}, nil},
		{"customer with access to the line", url.Values{"customer_ids": {SeedCustomerAccountID}}, []string{visibleID}, nil},
		{"created in range", url.Values{"starts_at": {"2020-01-01T00:00:00Z"}, "ends_at": {"2030-01-01T00:00:00Z"}}, []string{visibleID}, nil},
		{"created out of range", url.Values{"starts_at": {"2020-01-01T00:00:00Z"}, "ends_at": {"2020-06-01T00:00:00Z"}}, nil, []string{visibleID}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := list(tc.params)
			for _, want := range tc.has {
				assert.True(t, covCatalogProductsListContainsID(got, want), "%s should list %s", tc.name, want)
			}
			for _, unwanted := range tc.not {
				assert.False(t, covCatalogProductsListContainsID(got, unwanted), "%s should not list %s", tc.name, unwanted)
			}
		})
	}

	for name, params := range map[string]url.Values{
		"unknown category":   {"category_ids": {"itcg_00000000000000000000000000"}},
		"unlinked attribute": {"attribute_ids": {"at_prod9noattr7filter0match55"}},
		"unknown customer":   {"customer_ids": {"ac_00000000000000000000000000"}},
	} {
		t.Run(name, func(t *testing.T) {
			assertEmptyListData(t, list(params).Data, name+" matches nothing")
		})
	}
}

func TestCovCatalogProductsPreview9_ListRejectsBadFilters(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	for param, value := range map[string]string{"portal_visibility": "bogus", "starts_at": "not-a-date"} {
		t.Run(param, func(t *testing.T) {
			status, body, err := preview.GetListRaw(productsPath, url.Values{param: {value}})
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			assertErrorParam(t, requirePreview9Error(t, body, "parameter_invalid"), param)
		})
	}
}
