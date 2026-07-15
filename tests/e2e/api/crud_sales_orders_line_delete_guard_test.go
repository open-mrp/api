//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSalesOrder_DeleteLine_BlockedWhenShipmentAgainst pins that a line packed into a
// shipment cannot be deleted — even before the shipment ships. Packing a pick creates a
// shipment (status packed, not yet shipped) with a shipment line for the order line, and
// deleting that line would orphan the committed shipment/pick state. The rejection must be
// a client error (409), never a 5xx.
func TestSalesOrder_DeleteLine_BlockedWhenShipmentAgainst(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)
	lineID := orderSaleLineID(t, orderID)

	// Issue → pick → pack. Packing creates a not-yet-shipped shipment with a shipment line
	// for this order line.
	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)
	pickID := orderPickID(t, orderID)
	pickAllLines(t, pickID)
	packPick(t, pickID)

	beforeCount := getSalesOrder(t, orderID, nil)["line_count"]

	// The line now has a shipment against it → delete is rejected with a client error.
	dStatus, dBody, err := apiClient.Delete(salesOrdersPath + "/" + orderID + "/lines/" + lineID)
	require.NoError(t, err)
	requireClientError(t, dStatus, dBody)
	errObj := requireErrorResponse(t, dBody, "", "invalid_request_error")
	assert.Equal(t, "resource_conflict", errObj["code"],
		"deleting a line with a shipment against it must be a resource conflict: %s", string(dBody))

	// The line is still on the order.
	assert.EqualValues(t, beforeCount, getSalesOrder(t, orderID, nil)["line_count"],
		"the blocked delete left the line in place")
}
