//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// bulkDeleteOrders POSTs the atomic bulk-delete action and returns the raw response.
func bulkDeleteOrders(t *testing.T, orderIDs []string) (int, []byte) {
	t.Helper()
	status, body, err := apiClient.Post(salesOrdersPath+"/actions/bulk-delete",
		map[string]any{"sales_order_ids": orderIDs}, newIdempotencyKey())
	require.NoError(t, err)
	return status, body
}

func orderExists(t *testing.T, orderID string) bool {
	t.Helper()
	status, _, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, nil)
	require.NoError(t, err)
	return status == 200
}

// TestSalesOrder_BulkDelete_DeletesAll pins the happy path: several deletable orders are
// removed in a single request.
func TestSalesOrder_BulkDelete_DeletesAll(t *testing.T) {
	t.Parallel()
	a := createLifecycleOrder(t)
	b := createLifecycleOrder(t)

	status, body := bulkDeleteOrders(t, []string{a, b})
	requireStatus(t, 200, status, body)

	require.False(t, orderExists(t, a), "order a was bulk-deleted")
	require.False(t, orderExists(t, b), "order b was bulk-deleted")
}

// TestSalesOrder_BulkDelete_AtomicWhenOneUndeletable pins all-or-nothing: if any order in
// the batch cannot be deleted (a fulfilled order), none are deleted and the call fails with
// a client error (never a 5xx).
func TestSalesOrder_BulkDelete_AtomicWhenOneUndeletable(t *testing.T) {
	t.Parallel()
	deletable := createLifecycleOrder(t)

	fulfilled := createLifecycleOrder(t)
	status, body := salesOrderAction(t, fulfilled, "issue", false)
	requireStatus(t, 200, status, body)
	status, body = salesOrderAction(t, fulfilled, "close", false)
	requireStatus(t, 200, status, body)

	// The batch contains a fulfilled order → the whole delete is rejected.
	dStatus, dBody := bulkDeleteOrders(t, []string{deletable, fulfilled})
	requireClientError(t, dStatus, dBody)

	// Neither order was deleted — the operation is atomic.
	require.True(t, orderExists(t, deletable), "the deletable order survives an atomic bulk-delete failure")
	require.True(t, orderExists(t, fulfilled), "the fulfilled order is untouched")
}
