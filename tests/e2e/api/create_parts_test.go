//go:build e2e

package api_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// validPartBody returns a minimal valid CreatePartRequest body.
func validPartBody(sku string) map[string]any {
	return map[string]any{
		"sku":         sku,
		"category_id": SeedItemCategoryID,
	}
}

// preview.9 refuses a bad part with a 400 (404 for a missing reference) and the failing field's own code.
func TestParts_Preview9CreateValidation(t *testing.T) {
	t.Parallel()

	with := func(set map[string]any) map[string]any {
		body := validPartBody(uniqueName("e2e-part-val"))
		for k, v := range set {
			body[k] = v
		}
		return body
	}
	without := func(field string) map[string]any {
		body := validPartBody(uniqueName("e2e-part-val"))
		delete(body, field)
		return body
	}

	cases := []struct {
		name   string
		body   map[string]any
		status int
		code   string
		param  any
	}{
		{"material category", with(map[string]any{"category_id": SeedMaterialCategoryID}), 400, "invalid_format", "category_id"},
		{"missing sku", without("sku"), 400, "missing_field", "sku"},
		{"empty sku", with(map[string]any{"sku": ""}), 400, "missing_field", "sku"},
		{"sku over max length", with(map[string]any{"sku": strings.Repeat("a", 256)}), 400, "invalid_format", "sku"},
		{"missing category_id", without("category_id"), 400, "missing_field", "category_id"},
		{"unknown category", with(map[string]any{"category_id": "itcg_nonexistent0000000"}), 404, "resource_not_found", nil},
		{"unknown attribute", with(map[string]any{"attribute_ids": []string{"at_nonexistent00000000"}}), 404, "resource_not_found", nil},
		{"price per a currency", with(map[string]any{"unit_price": partPreview9Price("1.00", currencyUnitID)}), 400, "invalid_format", "unit_price.per_unit_id"},
		{"price per an unknown unit", with(map[string]any{"unit_price": partPreview9Price("1.00", "nonexistent_unit_id")}), 400, "invalid_format", "unit_price.per_unit_id"},
		{"cost per an unknown unit", with(map[string]any{"unit_cost": partPreview9Price("1.00", "nonexistent_unit_id")}), 400, "invalid_format", "unit_cost.per_unit_id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := preview9PartPost(t, partsPath, tc.body)
			requireStatus(t, tc.status, status, body)
			errObj := requirePreview9Error(t, body, tc.code)
			assert.Equal(t, tc.param, errObj["param"], "error.param (body: %s)", body)
			assert.NotContains(t, errObj, "errors", "preview.9 errors have no errors list")
		})
	}

	t.Run("include offered only on reads", func(t *testing.T) {
		status, body := preview9PartPost(t, partsPath+"?include=item.category.properties", with(nil))
		requireStatus(t, 400, status, body)
		assert.Equal(t, "include[]", requirePreview9Error(t, body, "parameter_invalid")["param"])
	})
}
