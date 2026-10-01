//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Issuing reserves each line once. A line added, resized or removed while the order is issued moves the
// reservation with it, or the item reads over- or under-reserved until the order closes: order 24441
// shipped a line added after issue with nothing reserved, and order 24371 kept reserving a removed one.
func TestSalesOrder_IssuedLineEdits_KeepReservationInStep(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)
	firstLineID := orderSaleLineID(t, orderID)
	patchSaleLineQuantity(t, orderID, firstLineID, "10")

	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)
	require.InDelta(t, 10.0, orderIssueTotals(t, orderID)["reserved"], 1e-9, "issuing reserves the line")

	// Added after issue: reserved like the lines issuing reserved.
	status, body, err := apiClient.Post(salesOrdersPath+"/"+orderID+"/lines", map[string]any{
		"product_id":  SeedProductID,
		"product_sku": "E2E-RESERVE-ADD",
		"quantity":    map[string]any{"value": "5", "unit_id": SeedUnitID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	addedLineID := jsonField(parseJSON(body), "id")
	assert.InDelta(t, 15.0, orderIssueTotals(t, orderID)["reserved"], 1e-9, "the added line is reserved")

	// Cut below its reservation: the excess goes back.
	patchSaleLineQuantity(t, orderID, firstLineID, "6")
	assert.InDelta(t, 11.0, orderIssueTotals(t, orderID)["reserved"], 1e-9, "cutting a line releases the difference")

	// Removed: its share of the reservation goes with it.
	status, body, err = apiClient.Delete(salesOrdersPath + "/" + orderID + "/lines/" + addedLineID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, status, 200, "delete line: %s", string(body))
	require.Less(t, status, 300, "delete line: %s", string(body))
	assert.InDelta(t, 6.0, orderIssueTotals(t, orderID)["reserved"], 1e-9, "removing a line releases its reservation")
}

// A price edit asks for nothing different, so the reservation is left exactly as it was.
func TestSalesOrder_IssuedLinePriceEdit_LeavesReservationAlone(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)
	lineID := orderSaleLineID(t, orderID)
	patchSaleLineQuantity(t, orderID, lineID, "10")

	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)

	status, body, err := apiClient.Patch(salesOrdersPath+"/"+orderID+"/lines/"+lineID, map[string]any{
		"unit_price": map[string]any{
			"value":               "7.50",
			"numerator_unit_id":   e2eCurrencyUnitID,
			"denominator_unit_id": SeedUnitID,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.InDelta(t, 10.0, orderIssueTotals(t, orderID)["reserved"], 1e-9)
}
