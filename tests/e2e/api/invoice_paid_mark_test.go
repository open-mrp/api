//go:build e2e

package api_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Recalculating an invoice's payments decides whether it is paid in full, over a flag someone set by hand.
// When it overturns that flag, the person who set it is told, with a notification that links to the
// invoice. A flag set by an API key, or one the recalculation agrees with, is nobody's to be told about.

const invoicePaymentStatusCategory = "invoice.payment_status_changed"

func awaitInvoicePaidInFull(t *testing.T, invoiceID string, want bool) {
	t.Helper()
	eventually(t, 15*time.Second, 200*time.Millisecond, func() error {
		if got := jsonField(getInvoice(t, invoiceID), "is_paid_in_full"); got != fmt.Sprint(want) {
			return fmt.Errorf("invoice %s is_paid_in_full = %s, want %v", invoiceID, got, want)
		}
		return nil
	})
}

// invoiceAlertsQueued counts the notification fan-outs queued about an invoice. The recalculation queues
// its alert in the transaction that writes the flag, so once the flag reads recalculated the count is final.
func invoiceAlertsQueued(t *testing.T, invoiceID string) int {
	t.Helper()
	var n int
	require.NoError(t, authDB(t).QueryRow(`
		SELECT COUNT(*) FROM message_outbox
		WHERE id > ? AND routing_key = 'notification.cmd.fanout'
		AND CAST(FROM_BASE64(JSON_UNQUOTE(JSON_EXTRACT(payload, '$.data'))) AS CHAR) LIKE ?`,
		outboxFloor(t), "%"+invoiceID+"%").Scan(&n))
	return n
}

func payInvoice(t *testing.T, customerID, invoiceID, amount string) {
	t.Helper()
	funds := time.Now().UTC()
	payment := createPayment(t, customerID, amount, &funds, nil)
	settle(t, map[string]any{"allocations": []any{allocation(jsonField(payment, "id"), invoiceID, amount)}})
	awaitFullyAllocated(t, jsonField(payment, "id"), true)
}

func TestInvoicePaidMark_RecalculationWinsAndTellsWhoSetIt(t *testing.T) {
	t.Parallel()
	markOutbox(t)
	person := notifUserClient(t)
	customerID := parityCustomer(t, "")
	inv := parityInvoiceFor(t, customerID, nil)

	status, raw, err := person.Patch(invoicesPath+"/"+inv.invoiceID, map[string]any{"is_paid_in_full": true}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)
	assert.Equal(t, "true", jsonField(parseJSON(raw), "is_paid_in_full"), "the mark is taken")

	// $5 applied: recalculating finds money still owed, and that wins over the mark.
	payInvoice(t, customerID, inv.invoiceID, "5.00")
	awaitInvoicePaidInFull(t, inv.invoiceID, false)

	alert := findNotif(t, person, "Invoice "+inv.number+" is no longer marked paid", url.Values{"category": {invoicePaymentStatusCategory}})
	assert.Equal(t, invoicePaymentStatusCategory, jsonField(alert, "category"))
	assert.Contains(t, jsonField(alert, "body"), "still owed")
	assert.Equal(t, 1, invoiceAlertsQueued(t, inv.invoiceID), "told once")
}

func TestInvoicePaidMark_AnAPIKeysMarkIsOverturnedWithoutAnAlert(t *testing.T) {
	t.Parallel()
	markOutbox(t)
	customerID := parityCustomer(t, "")
	inv := parityInvoiceFor(t, customerID, nil)

	patchInvoice(t, inv.invoiceID, map[string]any{"is_paid_in_full": true})
	payInvoice(t, customerID, inv.invoiceID, "5.00")
	awaitInvoicePaidInFull(t, inv.invoiceID, false)

	assert.Zero(t, invoiceAlertsQueued(t, inv.invoiceID), "an API key has no one to tell")
}

func TestInvoicePaidMark_AMarkTheRecalculationAgreesWithStands(t *testing.T) {
	t.Parallel()
	markOutbox(t)
	person := notifUserClient(t)
	customerID := parityCustomer(t, "")
	inv := parityInvoiceFor(t, customerID, nil)
	total := jsonField(getInvoice(t, inv.invoiceID), "total_invoiced")

	status, raw, err := person.Patch(invoicesPath+"/"+inv.invoiceID, map[string]any{"is_paid_in_full": true}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)

	payInvoice(t, customerID, inv.invoiceID, total)
	awaitInvoicePaidInFull(t, inv.invoiceID, true)

	assert.Zero(t, invoiceAlertsQueued(t, inv.invoiceID), "paid in full either way: nothing to tell")
}
