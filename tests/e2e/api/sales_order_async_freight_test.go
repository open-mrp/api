//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Order create prices freight only from the carrier rate cache, which the checkout's rate quote
// warms; an order placed without one is created at once and its freight line added out of band.
// The Shippo stub keys the cache miss on zipStubUncached (see stub.ZipStubUncached).
const zipStubUncached = "99915"

// stubGroundRate is what the Shippo stub quotes the seeded ground service on an ordinary lane.
const stubGroundRate = 12.5

// freightOrder creates an estimate on the rateable carrier's ground service, shipping to postalCode.
func freightOrder(t *testing.T, postalCode string) map[string]any {
	t.Helper()

	customerID := leadTimeCustomer(t, "e2e-async-freight", nil, "")
	body := minimalSalesOrderCreateBody(t, customerID)
	body["carrier_id"] = SeedTransitCarrierID
	body["service_level_id"] = SeedTransitGroundServiceLevelID
	body["ship_to_address_id"] = transitAddress(t, postalCode)

	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	order := parseJSON(respBody)
	deleteOrder(t, jsonField(order, "id"))
	return order
}

// freightLinePrices returns the unit price of every line on the order that is not a product the
// test ordered, which on these orders is only ever the synthesized freight line.
func freightLinePrices(t *testing.T, orderID string, orderedSKU string) []float64 {
	t.Helper()

	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, url.Values{"include": {"lines", "lines.unit_price"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	var prices []float64
	for _, raw := range jsonListData(parseJSON(body), "lines") {
		line := raw.(map[string]any)
		if jsonField(line, "product_sku") == orderedSKU {
			continue
		}
		price, err := strconv.ParseFloat(jsonField(jsonObject(line, "unit_price"), "value"), 64)
		require.NoError(t, err)
		prices = append(prices, price)
	}
	return prices
}

func orderedSKU(t *testing.T, orderID string) string {
	t.Helper()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return jsonField(jsonListData(parseJSON(body), "lines")[0].(map[string]any), "product_sku")
}

func TestSalesOrderFreight_CachedRateIsPricedAtCreate(t *testing.T) {
	t.Parallel()

	order := freightOrder(t, zipStubNormal)
	orderID := jsonField(order, "id")

	assert.EqualValues(t, 2, order["line_count"], "the freight line is written with the order")
	assert.Equal(t, []float64{stubGroundRate}, freightLinePrices(t, orderID, orderedSKU(t, orderID)))
}

func TestSalesOrderFreight_UncachedRateIsAddedAfterCreate(t *testing.T) {
	t.Parallel()

	order := freightOrder(t, zipStubUncached)
	orderID := jsonField(order, "id")
	sku := orderedSKU(t, orderID)

	assert.EqualValues(t, 1, order["line_count"], "create does not wait on the carrier for an uncached rate")

	var prices []float64
	require.Eventually(t, func() bool {
		prices = freightLinePrices(t, orderID, sku)
		return len(prices) > 0
	}, 15*time.Second, 200*time.Millisecond, "the freight consumer should add the freight line")
	assert.Equal(t, []float64{stubGroundRate}, prices, "the deferred quote prices the same service as create would have")
}
