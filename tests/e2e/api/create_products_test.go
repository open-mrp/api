//go:build e2e

package api_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validProductBody returns a minimal valid preview.9 CreateProductRequest body. Note the
// JSON field for product type code is `type`, not `product_type_code`.
func validProductBody(sku string) map[string]any {
	return map[string]any{
		"sku":         sku,
		"type":        "sale",
		"category_id": SeedItemCategoryID,
	}
}

// The preview.9 create product route still validates its body; its refusals keep preview.9's 400 and error codes.
func TestProductsPreview9_CreateValidation(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)

	with := func(set map[string]any) map[string]any {
		body := validProductBody(uniqueName("e2e-p9-prod-val"))
		for k, v := range set {
			body[k] = v
		}
		return body
	}

	cases := []struct {
		name   string
		body   map[string]any
		status int
		code   string
		param  string
	}{
		{"material category", with(map[string]any{"category_id": SeedMaterialCategoryID}), 400, "invalid_format", "category_id"},
		{"unsupported currency on unit_cost", with(map[string]any{
			"unit_cost": map[string]any{"amount": "1.00", "currency": "eur", "per_unit_id": nonCurrencyUnitID},
		}), 400, "currency_not_supported", "unit_cost.currency"},
		{"currency per unit on unit_price", with(map[string]any{
			"unit_price": map[string]any{"amount": "1.00", "currency": "usd", "per_unit_id": currencyUnitID},
		}), 400, "invalid_format", "unit_price.per_unit_id"},
		{"unknown type", with(map[string]any{"type": "bogus_type"}), 400, "parameter_invalid", "type"},
		{"unknown portal visibility", with(map[string]any{"portal_visibility": "public"}), 400, "parameter_invalid", "portal_visibility"},
		{"sku over max length", with(map[string]any{"sku": strings.Repeat("a", 256)}), 400, "invalid_format", "sku"},
		{"missing sku", map[string]any{"type": "sale", "category_id": SeedItemCategoryID}, 400, "missing_field", "sku"},
		{"attribute outside the category", with(map[string]any{
			"category_id": SeedPropertylessItemCategoryID, "attribute_ids": []string{SeedAttributeID},
		}), 400, "invalid_format", "attribute_ids"},
		{"unknown category", with(map[string]any{"category_id": "itcg_00000000000000000000000000"}), 404, "resource_not_found", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := preview.Post(productsPath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			if status == 201 {
				productID := jsonField(parseJSON(body), "id")
				t.Cleanup(func() { preview.Delete(productsPath + "/" + productID) })
			}
			requireStatus(t, tc.status, status, body)
			errObj := requirePreview9Error(t, body, tc.code)
			if tc.param != "" {
				assertErrorParam(t, errObj, tc.param)
			}
			assert.NotContains(t, errObj, "errors", "preview.9 errors carry no errors list")
		})
	}

	// Unknown references are refused rather than stored as orphan joins.
	refs := map[string]map[string]any{
		"unknown product line": with(map[string]any{"product_line_id": "pdln_00000000000000000000000000"}),
		"unknown attribute":    with(map[string]any{"attribute_ids": []string{"at_00000000000000000000000000"}}),
	}
	for name, body := range refs {
		t.Run(name, func(t *testing.T) {
			status, respBody, err := preview.Post(productsPath, body, newIdempotencyKey())
			require.NoError(t, err)
			if status == 201 {
				productID := jsonField(parseJSON(respBody), "id")
				t.Cleanup(func() { preview.Delete(productsPath + "/" + productID) })
			}
			assert.Contains(t, []int{400, 404}, status, "%s should be refused: %s", name, respBody)
		})
	}
}

// Products are items on forge, and these refusals have no forge test of their own elsewhere.
func TestItems_CreateProductRejectsUnknownReferencesAndCurrencies(t *testing.T) {
	t.Parallel()

	with := func(set map[string]any) map[string]any {
		body := forgeItemBody("product", uniqueName("e2e-item-prod-ref"))
		for k, v := range set {
			body[k] = v
		}
		return body
	}
	cases := []struct {
		name string
		body map[string]any
	}{
		{"unknown product line", with(map[string]any{"sales": map[string]any{"product_line_id": "pdln_00000000000000000000000000"}})},
		{"unknown attribute", with(map[string]any{"attribute_ids": []string{"at_00000000000000000000000000"}})},
		{"unsupported currency on unit_cost", with(map[string]any{
			"unit_cost": map[string]any{"amount": "1.00", "currency": "eur", "per_unit_id": nonCurrencyUnitID},
		})},
		{"currency per unit on sales.unit_price", with(map[string]any{
			"sales": map[string]any{"unit_price": map[string]any{"amount": "1.00", "currency": "usd", "per_unit_id": currencyUnitID}},
		})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := apiClient.Post(itemsPath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			if status == 201 {
				itemID := jsonField(parseJSON(body), "id")
				t.Cleanup(func() { apiClient.Delete(itemsPath + "/" + itemID) })
			}
			assert.Contains(t, []int{400, 404, 422}, status, "%s should be refused: %s", tc.name, body)
			requireErrorResponse(t, body, "", "invalid_request_error")
			assert.Empty(t, listItemIDs(t, map[string][]string{"skus": {jsonField(tc.body, "sku")}}), "a refused create writes nothing")
		})
	}
}
