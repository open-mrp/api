//go:build e2e

package api_test

import (
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Duplicate SKUs conflict on preview.9 whichever write brings them in.
func TestCovCatalogParts_Preview9DuplicateSKUConflicts(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-cov-part-dupsku")
	createPreview9Part(t, partsPath, validPartBody(sku))

	t.Run("create", func(t *testing.T) {
		status, body := preview9PartPost(t, partsPath, validPartBody(sku))
		requireStatus(t, 409, status, body)
		assert.Equal(t, "sku", requirePreview9Error(t, body, "resource_conflict")["param"])
	})

	t.Run("update", func(t *testing.T) {
		otherID := jsonField(createPreview9Part(t, partsPath, validPartBody(uniqueName("e2e-cov-part-dupsku-other"))), "id")
		status, body := preview9PartPatch(t, partsPath+"/"+otherID, map[string]any{"sku": sku})
		requireStatus(t, 409, status, body)
		assert.Equal(t, "sku", requirePreview9Error(t, body, "resource_conflict")["param"])
	})
}

func TestCovCatalogParts_Preview9UpdateRejections(t *testing.T) {
	t.Parallel()
	partID := jsonField(createPreview9Part(t, partsPath, validPartBody(uniqueName("e2e-cov-part-upd-rej"))), "id")

	t.Run("blank sku", func(t *testing.T) {
		status, body := preview9PartPatch(t, partsPath+"/"+partID, map[string]any{"sku": ""})
		requireStatus(t, 400, status, body)
		assert.Equal(t, "sku", requirePreview9Error(t, body, "invalid_format")["param"])
	})

	t.Run("include offered only on reads", func(t *testing.T) {
		status, body := preview9PartPatch(t, partsPath+"/"+partID+"?include=item.category.properties", map[string]any{"description": "x"})
		requireStatus(t, 400, status, body)
		assert.Equal(t, "include[]", requirePreview9Error(t, body, "parameter_invalid")["param"])
	})
}

// A part id that never existed is a plain 404 on every preview.9 route.
func TestCovCatalogParts_Preview9NeverExistedIDReturns404(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	ghost := partsPath + "/pt_covneverexisted0001"

	status, body := preview9PartPatch(t, ghost, map[string]any{"sku": uniqueName("e2e-cov-part-ghost")})
	requireStatus(t, 404, status, body)
	requirePreview9Error(t, body, "resource_not_found")

	status, body, err := preview.Delete(ghost)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requirePreview9Error(t, body, "resource_not_found")
}

// forge refuses the part pricing and attribute inputs the part routes refused, now on the item.
func TestCovCatalogParts_ForgePartRejections(t *testing.T) {
	t.Parallel()

	with := func(set map[string]any) map[string]any {
		body := forgeItemBody("part", uniqueName("e2e-cov-part-forge"))
		for k, v := range set {
			body[k] = v
		}
		return body
	}
	price := func(amount, currency, perUnitID string) map[string]any {
		return map[string]any{"amount": amount, "currency": currency, "per_unit_id": perUnitID}
	}

	cases := []struct {
		name  string
		body  map[string]any
		code  string
		param string
	}{
		{"cost in an unsupported currency", with(map[string]any{"unit_cost": price("2.50", "eur", nonCurrencyUnitID)}), "currency_not_supported", "unit_cost.currency"},
		{"cost per an unknown unit", with(map[string]any{"unit_cost": price("2.50", "usd", "nonexistent_unit_id")}), "invalid_format", "unit_cost.per_unit_id"},
		{"price per a currency", with(map[string]any{"sales": map[string]any{"unit_price": price("5.00", "usd", currencyUnitID)}}), "invalid_format", "sales.unit_price.per_unit_id"},
		{"price per an unknown unit", with(map[string]any{"sales": map[string]any{"unit_price": price("5.00", "usd", "nonexistent_unit_id")}}), "invalid_format", "sales.unit_price.per_unit_id"},
		{"unknown attribute", with(map[string]any{"attribute_ids": []string{mustGenID(t, id.AttributeIDPrefix)}}), "invalid_format", "attribute_ids"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := apiClient.Post(itemsPath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 422, status, body)
			assertErrorParam(t, requireErrorResponse(t, body, tc.code, "invalid_request_error"), tc.param)
		})
	}

	t.Run("update to a taken sku", func(t *testing.T) {
		taken := uniqueName("e2e-cov-part-forge-taken")
		createForgeItem(t, itemsPath, forgeItemBody("part", taken))
		otherID := jsonField(createForgeItem(t, itemsPath, with(nil)), "id")
		status, body, err := apiClient.Patch(itemsPath+"/"+otherID, map[string]any{"sku": taken}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 409, status, body)
		assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), "sku")
	})
}
