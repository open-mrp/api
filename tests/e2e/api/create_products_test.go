//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validProductBody returns a minimal valid CreateProductRequest body. Note the
// JSON field for product type code is `type`, not `product_type_code`.
func validProductBody(sku string) map[string]any {
	return map[string]any{
		"sku":         sku,
		"type":        "sale",
		"category_id": SeedItemCategoryID,
	}
}

// TestProducts_Create_RejectsWrongCategoryType: products may only be created in
// product categories; a material category is rejected.
func TestProducts_Create_RejectsWrongCategoryType(t *testing.T) {
	t.Parallel()

	body := validProductBody(uniqueName("e2e-prod-wrongcat"))
	body["category_id"] = SeedMaterialCategoryID // Yarn — a material category

	status, respBody, err := apiClient.Post(productsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, respBody)

	errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "category_id")
}

func TestProducts_Create_Basic_DefaultsRates(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.PostFull(productsPath, validProductBody(uniqueName("e2e-prod")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + id) })

	assert.Equal(t, "product", jsonField(got, "object"))
}

func TestProducts_Create_WithValidRates(t *testing.T) {
	t.Parallel()

	body := validProductBody(uniqueName("e2e-prod-rates"))
	body["unit_price"] = map[string]any{
		"amount":      "9.99",
		"currency":    "usd",
		"per_unit_id": nonCurrencyUnitID,
	}
	body["unit_cost"] = map[string]any{
		"amount":      "4.50",
		"currency":    "usd",
		"per_unit_id": nonCurrencyUnitID,
	}

	resp, err := apiClient.PostFull(productsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + id) })
}

// Money is dollars only, so a cost in any other currency is refused.
func TestProducts_Create_RejectsAnUnsupportedCurrencyOnUnitCost(t *testing.T) {
	t.Parallel()

	body := validProductBody(uniqueName("e2e-prod-bad-num"))
	body["unit_cost"] = map[string]any{
		"amount":      "1.00",
		"currency":    "eur",
		"per_unit_id": nonCurrencyUnitID,
	}

	status, respBody, err := apiClient.Post(productsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, respBody)

	errObj := requireErrorResponse(t, respBody, "currency_not_supported", "invalid_request_error")
	assertErrorParam(t, errObj, "unit_cost.currency")
}

func TestProducts_Create_RejectsACurrencyPerUnitOnUnitPrice(t *testing.T) {
	t.Parallel()

	body := validProductBody(uniqueName("e2e-prod-bad-den"))
	body["unit_price"] = map[string]any{
		"amount":      "1.00",
		"currency":    "usd",
		"per_unit_id": currencyUnitID,
	}

	status, respBody, err := apiClient.Post(productsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, respBody)

	errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "unit_price.per_unit_id")
}

func TestProducts_Create_LinksAttributes(t *testing.T) {
	t.Parallel()

	body := validProductBody(uniqueName("e2e-prod-attrs"))
	body["attribute_ids"] = []string{SeedAttributeID}

	resp, err := apiClient.PostFull(productsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + id) })

	productStatus, productBody, err := apiClient.GetListRaw(productsPath+"/"+id, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, productStatus, productBody)
	itemID := jsonField(jsonObject(parseJSON(productBody), "item"), "id")
	require.NotEmpty(t, itemID)

	itemStatus, itemBody, err := apiClient.GetListRaw(
		"/v1/catalog/items/"+itemID,
		url.Values{"include": {"attributes"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, itemStatus, itemBody)

	attrs := jsonListData(parseJSON(itemBody), "attributes")
	require.NotEmpty(t, attrs)

	found := false
	for _, raw := range attrs {
		obj, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if jsonField(obj, "id") == SeedAttributeID {
			found = true
			break
		}
	}
	assert.True(t, found, "expected linked attribute %s in item.attributes", SeedAttributeID)
}
