//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// saleLineUnitPrices returns the unit_price value of every line on the order whose product
// is SeedProductID.
func saleLineUnitPrices(t *testing.T, orderID string) []float64 {
	t.Helper()
	got := getSalesOrder(t, orderID, url.Values{"include": {"lines", "lines.product", "lines.unit_price"}})
	lines := jsonObject(got, "lines")
	require.NotNil(t, lines, "lines present with ?include=lines")

	var prices []float64
	for _, raw := range jsonArray(lines, "data") {
		line, ok := raw.(map[string]any)
		require.True(t, ok)
		product := jsonObject(line, "product")
		if product == nil || jsonField(product, "id") != SeedProductID {
			continue
		}
		up := jsonObject(line, "unit_price")
		require.NotNil(t, up, "a sale line carries a unit price")
		v, err := strconv.ParseFloat(jsonField(up, "value"), 64)
		require.NoError(t, err)
		prices = append(prices, v)
	}
	return prices
}

// TestSalesOrder_AddLine_WithoutPrice_CalculatesFromProduct pins that adding a line without
// a unit price prices it server-side from the product — identically to a line created with
// the order — instead of failing or leaving the price at zero. This is the fix for the
// dashboard add-line flow, which no longer sends a price (the user overrides it afterward).
func TestSalesOrder_AddLine_WithoutPrice_CalculatesFromProduct(t *testing.T) {
	t.Parallel()
	// createLifecycleOrder makes an order with one sale line (SeedProductID, qty 1),
	// priced server-side at order-create time.
	orderID := createLifecycleOrder(t)

	before := saleLineUnitPrices(t, orderID)
	require.Len(t, before, 1, "the seed order has exactly one sale line")
	require.Greater(t, before[0], 0.0, "the order-created line is priced from the product")

	// Add a second line of the same product and quantity, WITHOUT a unit price.
	lineBody := map[string]any{
		"product_id":  SeedProductID,
		"product_sku": "E2E-CALC-PRICE",
		"quantity":    map[string]any{"value": "1", "unit_id": SeedUnitID},
	}
	status, body, err := apiClient.Post(salesOrdersPath+"/"+orderID+"/lines", lineBody, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	// The added line must carry the same server-calculated price as the order-created line.
	after := saleLineUnitPrices(t, orderID)
	require.Len(t, after, 2, "the order now has two sale lines")
	assert.Equal(t, before[0], after[0], "the original line's price is unchanged")
	assert.Equal(t, before[0], after[1], "the added line is priced identically to the order-created line")
}

// TestSalesOrder_AddLine_WithExplicitPrice_HonoredAsOverride pins that an internal user may
// still supply an explicit unit price on create, which is honored as an override.
func TestSalesOrder_AddLine_WithExplicitPrice_HonoredAsOverride(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)

	lineBody := map[string]any{
		"product_id":  SeedProductID,
		"product_sku": "E2E-OVERRIDE-PRICE",
		"quantity":    map[string]any{"value": "1", "unit_id": SeedUnitID},
		"unit_price": map[string]any{
			"value":               "3.33",
			"numerator_unit_id":   e2eCurrencyUnitID,
			"denominator_unit_id": SeedUnitID,
		},
	}
	status, body, err := apiClient.Post(salesOrdersPath+"/"+orderID+"/lines", lineBody, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	prices := saleLineUnitPrices(t, orderID)
	require.Len(t, prices, 2)
	assert.Contains(t, prices, 3.33, "the explicit override price is honored")
}
