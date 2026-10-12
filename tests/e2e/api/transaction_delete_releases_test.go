//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Deleting a transaction releases what it paid: each invoice it was applied to owes the money again, and
// its allocations are audited and remembered as deleted.
func TestDeleteTransaction_ReleasesItsAllocations(t *testing.T) {
	t.Parallel()

	inv := invoiceNewCustomer(t)
	funds := time.Now().Add(-time.Hour)
	tx := createPayment(t, inv.customerID, inv.total.StringFixed(2), &funds, nil)
	transactionID := jsonField(tx, "id")

	settlement := settle(t, map[string]any{"allocations": []any{allocation(transactionID, inv.invoiceID, inv.total.StringFixed(2))}})
	awaitInvoiceStatus(t, inv.invoiceID, "paid")
	status, body, err := apiClient.GetListRaw(financeSettlementsPath+"/"+jsonField(settlement, "id"), url.Values{"include": {"allocations"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	allocations := jsonListData(parseJSON(body), "allocations")
	require.Len(t, allocations, 1)
	allocationID := jsonField(allocations[0].(map[string]any), "id")

	since := time.Now().Add(-5 * time.Second).UTC().Format(time.RFC3339)
	status, body, err = apiClient.Delete(financeTransactionsPath + "/" + transactionID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	awaitInvoiceStatus(t, inv.invoiceID, "unpaid")
	findAuditEvent(t, apiClient, since, allocationID, "transaction_allocation", "delete", anyEvent)
	findAuditEvent(t, apiClient, since, transactionID, "transaction", "delete", anyEvent)

	status, body, err = apiClient.Delete(financeAllocationsPath + "/" + allocationID)
	require.NoError(t, err)
	assert.Equal(t, 410, status, "the released allocation is remembered as deleted: %s", body)
}
