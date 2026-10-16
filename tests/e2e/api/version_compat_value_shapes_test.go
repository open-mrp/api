//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const preview9Version = "1.0.forge-preview.9"

// A preview.9 caller still sends a credit limit as a quantity in the dollar unit, asks for its unit by include, and
// reads it back as a quantity; forge presents the same amount as Money.
func TestVersionCompatValueShapes_CreditLimitIsAQuantityOnPreview9(t *testing.T) {
	t.Parallel()

	preview := apiClient.WithAPIVersion(preview9Version)
	body := validCustomerBody(uniqueName("e2e-compat-credit"))
	body["credit_limit"] = map[string]any{"value": "5000", "unit_id": currencyUnitID}
	status, raw, err := preview.Post(customersPath+"?include=credit_limit,credit_limit.unit", body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	created := parseJSON(raw)
	id := jsonField(created, "id")
	t.Cleanup(func() { apiClient.Delete(customersPath + "/" + id) })

	limit := jsonObject(created, "credit_limit")
	require.NotNil(t, limit)
	assert.Equal(t, "quantity", limit["object"])
	assert.Equal(t, "5000.00", limit["value"])
	assert.Equal(t, "$5,000.00", limit["display_value"])
	assert.Nil(t, limit["unit"], "the currency unit is not part of forge, so preview.9 reads it as not included")
	assert.NotContains(t, limit, "amount")

	status, raw, err = apiClient.GetListRaw(customersPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assert.Equal(t, map[string]any{"amount": "5000.00", "currency": "usd"}, parseJSON(raw)["credit_limit"])
}

// A preview.9 caller prices a material with a currency numerator and reads the item's unit value back as a rate with a
// denominator unit. On forge only an explicitly sellable item has a sales section, so the material shows none.
func TestVersionCompatValueShapes_UnitValueIsARateOnPreview9(t *testing.T) {
	t.Parallel()

	preview := apiClient.WithAPIVersion(preview9Version)
	body := validMaterialBody(uniqueName("e2e-compat-mat"))
	body["unit_price"] = map[string]any{"value": "12.50", "numerator_unit_id": currencyUnitID, "denominator_unit_id": nonCurrencyUnitID}
	body["unit_cost"] = map[string]any{"value": "8.25", "numerator_unit_id": currencyUnitID, "denominator_unit_id": nonCurrencyUnitID}
	status, raw, err := preview.Post(materialsPath+"?include=item", body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	material := parseJSON(raw)
	t.Cleanup(func() { apiClient.Delete(materialsPath + "/" + jsonField(material, "id")) })
	itemID := jsonField(jsonObject(material, "item"), "id")
	require.NotEmpty(t, itemID)

	status, raw, err = preview.GetListRaw(itemsPath+"/"+itemID, url.Values{"include": {"unit_value,unit_value.numerator_unit,unit_value.denominator_unit,unit_cost"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	item := parseJSON(raw)
	value := jsonObject(item, "unit_value")
	require.NotNil(t, value)
	assert.Equal(t, "rate", value["object"])
	assert.Equal(t, "12.50", value["value"])
	assert.Nil(t, value["numerator_unit"])
	assert.Equal(t, nonCurrencyUnitID, jsonField(jsonObject(value, "denominator_unit"), "id"))
	assert.True(t, strings.HasPrefix(jsonField(value, "display_value"), "$12.50 / "), "display_value = %v", value["display_value"])
	assert.Equal(t, "8.25", jsonObject(item, "unit_cost")["value"])

	forgeItem := getItem(t, itemID)
	assert.NotContains(t, forgeItem, "unit_value", "forge holds a selling price only in sales.unit_price")
	assert.Nil(t, forgeItem["sales"], "a material with a price but no product row is not sellable on forge: %v", forgeItem)
	assertItemPrice(t, jsonObject(forgeItem, "unit_cost"), "8.25", nonCurrencyUnitID, "unit_cost")
}

// Include keys that reached into a value's units are cut back for preview.9 rather than refused, and a value that is no
// longer includable is dropped.
func TestVersionCompatValueShapes_UnitIncludesAreCutBack(t *testing.T) {
	t.Parallel()

	preview := apiClient.WithAPIVersion(preview9Version)
	status, raw, err := preview.GetListRaw(customersPath, url.Values{"include": {"credit_limit.unit"}, "limit": {"1"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)

	status, raw, err = apiClient.GetListRaw(customersPath, url.Values{"include": {"credit_limit.unit"}, "limit": {"1"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "forge has no unit include on a value: %s", raw)
}
