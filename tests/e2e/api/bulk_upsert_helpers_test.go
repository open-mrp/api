//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// bulkCurrencyRate builds a unit_price/unit_cost rate input with a currency
// numerator and a non-currency denominator (the valid shape).
func bulkCurrencyRate(value string) map[string]any {
	return map[string]any{
		"value":               value,
		"numerator_unit_id":   currencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}
}

// bulkProperty builds a {name, value} property pair for a bulk upsert row.
func bulkProperty(name, value string) map[string]any {
	return map[string]any{"name": name, "value": value}
}

// catalogItem fetches a part/product by id and returns its expanded item object.
func catalogItem(t *testing.T, basePath, id string) map[string]any {
	t.Helper()
	status, body, err := apiClient.GetListRaw(basePath+"/"+id, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	item := jsonObject(parseJSON(body), "item")
	require.NotNil(t, item, "item should be populated with ?include=item")
	return item
}

// catalogItemID fetches a part/product by id and returns its wrapped item id.
func catalogItemID(t *testing.T, basePath, id string) string {
	t.Helper()
	itemID := jsonField(catalogItem(t, basePath, id), "id")
	require.NotEmpty(t, itemID)
	return itemID
}

// catalogRateValue fetches a part/product and returns the decimal value of a nested
// rate. rateInclude is e.g. "item.unit_value" or "item.unit_cost".
func catalogRateValue(t *testing.T, basePath, id, rateInclude string) string {
	t.Helper()
	status, body, err := apiClient.GetListRaw(basePath+"/"+id, url.Values{"include": {rateInclude}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	rateKey := strings.TrimPrefix(rateInclude, "item.")
	rate := jsonObject(jsonObject(parseJSON(body), "item"), rateKey)
	require.NotNil(t, rate, "%s should be populated with ?include=%s", rateKey, rateInclude)
	return jsonField(rate, "value")
}

// catalogItemAttributeValues returns the `value` of every attribute attached to an item.
func catalogItemAttributeValues(t *testing.T, itemID string) []string {
	t.Helper()
	status, body, err := apiClient.GetListRaw("/v1/catalog/items/"+itemID, url.Values{"include": {"attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	var values []string
	for _, raw := range jsonListData(parseJSON(body), "attributes") {
		if obj, ok := raw.(map[string]any); ok {
			values = append(values, jsonField(obj, "value"))
		}
	}
	return values
}
