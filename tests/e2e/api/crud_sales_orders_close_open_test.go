//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSalesOrder_Close_PacksOpenPickLine_Open_ReopensIncomplete pins that closing an order
// packs its still-open pick lines (so the pick reads as complete with the order), and that
// reopening the order reopens the lines that are not complete (picked < ordered) so the work
// can continue.
func TestSalesOrder_Close_PacksOpenPickLine_Open_ReopensIncomplete(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)
	lineID := orderSaleLineID(t, orderID)
	patchSaleLineQuantity(t, orderID, lineID, "10")

	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)
	pickID := orderPickID(t, orderID)

	// Pick only 6 of the 10 (partial) — the line stays open and incomplete.
	setPickedQuantity(t, pickID, firstPickLineID(t, pickID), "6")
	rows := fetchPickLines(t, pickID)
	require.Len(t, rows, 1)
	require.False(t, rows[0].packed, "the pick line is open before the order is closed")

	// Close the order → the open pick line is packed, and the pick is finished.
	status, body = salesOrderAction(t, orderID, "close", false)
	requireStatus(t, 200, status, body)
	rows = fetchPickLines(t, pickID)
	require.Len(t, rows, 1)
	assert.True(t, rows[0].packed, "closing the order packs the open pick line")
	assert.Equal(t, 6.0, rows[0].picked, "the picked quantity is unchanged by closing")
	assert.True(t, pickIsFinished(t, pickID), "the pick is finished when the order is closed")

	// Reopen the order → the incomplete line (picked 6 < ordered 10) is reopened.
	status, body = salesOrderAction(t, orderID, "open", false)
	requireStatus(t, 200, status, body)
	rows = fetchPickLines(t, pickID)
	require.Len(t, rows, 1)
	assert.False(t, rows[0].packed, "reopening the order reopens the incomplete pick line")
	assert.False(t, pickIsFinished(t, pickID), "the pick is no longer finished after reopening")
}

// TestSalesOrder_Open_LeavesCompletePickLinePacked pins the complement: a fully-picked
// (complete) line stays packed when the order is reopened — only incomplete lines reopen.
func TestSalesOrder_Open_LeavesCompletePickLinePacked(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)
	lineID := orderSaleLineID(t, orderID)
	patchSaleLineQuantity(t, orderID, lineID, "10")

	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)
	pickID := orderPickID(t, orderID)

	// Fully pick the line (10 of 10) — it is complete.
	pickAllLines(t, pickID)

	status, body = salesOrderAction(t, orderID, "close", false)
	requireStatus(t, 200, status, body)
	require.True(t, fetchPickLines(t, pickID)[0].packed, "closing packs the fully-picked line")

	// Reopen → the complete line stays packed (only incomplete lines reopen).
	status, body = salesOrderAction(t, orderID, "open", false)
	requireStatus(t, 200, status, body)
	rows := fetchPickLines(t, pickID)
	require.Len(t, rows, 1)
	assert.True(t, rows[0].packed, "a fully-picked (complete) line stays packed after reopening")
	assert.Equal(t, 10.0, rows[0].picked)
}

// orderIssueTotals sums the order's inventory issues by status, in the seed unit.
func orderIssueTotals(t *testing.T, orderID string) map[string]float64 {
	t.Helper()
	rows, err := authDB(t).Query(`
		SELECT ii.status_code, SUM(q.value * (u.ratio_numerator / u.ratio_denominator)) / (su.ratio_numerator / su.ratio_denominator)
		FROM inventory_issue ii
		JOIN quantity q ON q.id = ii.quantity_id
		JOIN unit u ON u.id = q.unit_id
		JOIN unit su ON su.id = ?
		WHERE ii.order_id = ?
		GROUP BY ii.status_code, su.ratio_numerator, su.ratio_denominator`, SeedUnitID, orderID)
	require.NoError(t, err)
	defer rows.Close()

	totals := map[string]float64{}
	for rows.Next() {
		var status string
		var value float64
		require.NoError(t, rows.Scan(&status, &value))
		totals[status] = value
	}
	require.NoError(t, rows.Err())
	return totals
}

// shipPartialOrder issues a 10-unit order, picks and packs only 6, and ships that shipment, leaving 4
// still reserved on the order.
func shipPartialOrder(t *testing.T) string {
	t.Helper()
	orderID := createLifecycleOrder(t)
	patchSaleLineQuantity(t, orderID, orderSaleLineID(t, orderID), "10")

	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)
	pickID := orderPickID(t, orderID)
	setPickedQuantity(t, pickID, firstPickLineID(t, pickID), "6")
	packPick(t, pickID)

	numbers := pickShipmentNumbers(t, pickID)
	require.NotEmpty(t, numbers, "packing must produce a shipment")
	listStatus, listBody, err := apiClient.GetListRaw(shipmentsPath, url.Values{"q": {numbers[0]}})
	require.NoError(t, err)
	requireStatus(t, 200, listStatus, listBody)
	shipments := jsonArray(parseJSON(listBody), "data")
	require.NotEmpty(t, shipments)
	shipmentID := jsonField(shipments[0].(map[string]any), "id")

	shipStatus, shipBody, err := apiClient.Post(shipmentsPath+"/"+shipmentID+"/actions/ship", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, shipStatus, shipBody)
	t.Cleanup(func() {
		_, _, _ = apiClient.Post(shipmentsPath+"/"+shipmentID+"/actions/void", map[string]any{}, newIdempotencyKey())
	})

	totals := orderIssueTotals(t, orderID)
	require.InDelta(t, 4.0, totals["reserved"], 1e-9, "a partial shipment leaves its balance reserved: %v", totals)
	require.InDelta(t, 6.0, totals["open"]+totals["closed"], 1e-9, "the shipped quantity is issued: %v", totals)
	return orderID
}

// Closing a short-shipped order releases the balance it still had reserved — left behind, it counts
// as reserved against the item forever — and reopening reserves that balance again so the next
// shipment has something to draw on.
func TestSalesOrder_Close_ReleasesShortShippedBalance_Open_ReservesItAgain(t *testing.T) {
	t.Parallel()
	orderID := shipPartialOrder(t)

	status, body := salesOrderAction(t, orderID, "close", false)
	requireStatus(t, 200, status, body)
	totals := orderIssueTotals(t, orderID)
	assert.Zero(t, totals["reserved"], "closing releases the unshipped balance: %v", totals)
	assert.InDelta(t, 6.0, totals["open"]+totals["closed"], 1e-9, "closing leaves the shipped issues alone: %v", totals)

	status, body = salesOrderAction(t, orderID, "open", false)
	requireStatus(t, 200, status, body)
	totals = orderIssueTotals(t, orderID)
	assert.InDelta(t, 4.0, totals["reserved"], 1e-9, "reopening reserves exactly the unshipped balance: %v", totals)
	assert.InDelta(t, 6.0, totals["open"]+totals["closed"], 1e-9, "reopening leaves the shipped issues alone: %v", totals)

	// A second round trip lands in the same place.
	status, body = salesOrderAction(t, orderID, "close", false)
	requireStatus(t, 200, status, body)
	status, body = salesOrderAction(t, orderID, "open", false)
	requireStatus(t, 200, status, body)
	assert.InDelta(t, 4.0, orderIssueTotals(t, orderID)["reserved"], 1e-9)
}

// Orders closed before close released anything are still carrying their balance. Reopening one must
// count that reservation rather than reserve the balance a second time.
func TestSalesOrder_Open_DoesNotDoubleReserveBalanceLeftByOldClose(t *testing.T) {
	t.Parallel()
	orderID := shipPartialOrder(t)

	// The state the old close left behind: fulfilled, with the balance still reserved.
	_, err := authDB(t).Exec(
		"UPDATE sales_order SET sales_order_status_code = 'fulfilled', completed_at = NOW(3) WHERE id = ?", orderID)
	require.NoError(t, err)

	status, body := salesOrderAction(t, orderID, "open", false)
	requireStatus(t, 200, status, body)
	totals := orderIssueTotals(t, orderID)
	assert.InDelta(t, 4.0, totals["reserved"], 1e-9, "the surviving reservation already covers the balance: %v", totals)
}
