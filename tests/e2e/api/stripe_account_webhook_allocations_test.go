//go:build e2e

package api_test

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// postAccountPaymentIntentEvent delivers a payment intent event to the seed account's Stripe webhook.
func postAccountPaymentIntentEvent(t *testing.T, eventType, paymentIntentID string, amountCents int64, metadata map[string]string) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"id":   "evt_" + uniqueName("e2e"),
		"type": eventType,
		"data": map[string]any{"object": map[string]any{
			"id":                   paymentIntentID,
			"amount":               amountCents,
			"payment_method_types": []string{"card"},
			"metadata":             metadata,
		}},
	})
	require.NoError(t, err)
	status, body, err := apiClient.PostSigned(stripeWebhookPath+"/accounts/"+SeedAccountID, StubStripeSignature, payload)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

func connectSeedAccountStripe(t *testing.T) {
	t.Helper()
	lockStripeConnected(t)
	status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
		"name":        uniqueName("e2e-stripe-webhook"),
		"provider":    "stripe",
		"credentials": covSettingsIntegrationsStripeCredentials("webhook"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(covSettingsIntegrationsPath + "/" + id) })
}

func stripeTransactionID(t *testing.T, paymentIntentID string) string {
	t.Helper()
	var id string
	require.NoError(t, authDB(t).QueryRow("SELECT id FROM transaction WHERE stripe_payment_id = ?", paymentIntentID).Scan(&id))
	return id
}

// A failed or canceled Stripe payment releases its allocations the way deleting them by hand does: the invoice owes the money again and the deleted rows are remembered.
func TestStripeAccountWebhook_FailedAndCanceledReleaseAllocations(t *testing.T) {
	t.Parallel()
	connectSeedAccountStripe(t)

	inv := invoiceNewCustomer(t)
	status, body, err := apiClient.GetListRaw(invoicesPath+"/"+inv.invoiceID, url.Values{"include": {"order"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	orderID := jsonField(jsonObject(parseJSON(body), "order"), "id")
	require.NotEmpty(t, orderID)

	paymentIntentID := "pi_" + uniqueName("e2e")
	cents := inv.total.Shift(2).IntPart()
	postAccountPaymentIntentEvent(t, "payment_intent.succeeded", paymentIntentID, cents, map[string]string{"orderID": orderID, "customerID": inv.customerID})
	transactionID := stripeTransactionID(t, paymentIntentID)
	_, err = authDB(t).Exec("UPDATE transaction SET funds_received_at = NOW(3) WHERE id = ?", transactionID)
	require.NoError(t, err)

	allocate := func() string {
		settlement := settle(t, map[string]any{"allocations": []any{allocation(transactionID, inv.invoiceID, inv.total.StringFixed(2))}})
		awaitInvoiceStatus(t, inv.invoiceID, "paid")
		status, body, err := apiClient.GetListRaw(financeSettlementsPath+"/"+jsonField(settlement, "id"), url.Values{"include": {"allocations"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		allocations := jsonListData(parseJSON(body), "allocations")
		require.Len(t, allocations, 1)
		return jsonField(allocations[0].(map[string]any), "id")
	}

	failedAllocationID := allocate()
	since := time.Now().Add(-5 * time.Second).UTC().Format(time.RFC3339)
	postAccountPaymentIntentEvent(t, "payment_intent.payment_failed", paymentIntentID, cents, nil)
	awaitInvoiceStatus(t, inv.invoiceID, "unpaid")
	awaitFullyAllocated(t, transactionID, false)
	findAuditEvent(t, apiClient, since, failedAllocationID, "transaction_allocation", "delete", anyEvent)
	status, body, err = apiClient.Delete(financeAllocationsPath + "/" + failedAllocationID)
	require.NoError(t, err)
	assert.Equal(t, 410, status, "the released allocation is remembered as deleted: %s", body)

	canceledAllocationID := allocate()
	postAccountPaymentIntentEvent(t, "payment_intent.canceled", paymentIntentID, cents, nil)
	awaitInvoiceStatus(t, inv.invoiceID, "unpaid")
	findAuditEvent(t, apiClient, since, canceledAllocationID, "transaction_allocation", "delete", anyEvent)
	findAuditEvent(t, apiClient, since, transactionID, "transaction", "delete", anyEvent)
	status, body, err = apiClient.Delete(financeAllocationsPath + "/" + canceledAllocationID)
	require.NoError(t, err)
	assert.Equal(t, 410, status, "the released allocation is remembered as deleted: %s", body)
	status, body, err = apiClient.Delete(financeTransactionsPath + "/" + transactionID)
	require.NoError(t, err)
	assert.Equal(t, 410, status, "the canceled transaction is remembered as deleted: %s", body)
}
