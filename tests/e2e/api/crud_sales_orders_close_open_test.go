//go:build e2e

package api_test

import (
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
