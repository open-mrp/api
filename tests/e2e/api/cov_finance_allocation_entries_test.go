//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// settledPaymentAndDiscount settles a payment of 12.00 and a new 3.00 discount against a fresh
// customer's invoice, and returns the invoice and the payment.
func settledPaymentAndDiscount(t *testing.T) (paymentsInvoice, map[string]any) {
	t.Helper()
	inv := invoiceNewCustomer(t)
	funds := time.Now().UTC()
	payment := createPayment(t, inv.customerID, "12.00", &funds, nil)
	settle(t, map[string]any{
		"new_transactions": []any{map[string]any{"key": "adj", "type": "adjustment", "adjustment_type": "discount", "customer_id": inv.customerID}},
		"allocations": []any{
			allocation(jsonField(payment, "id"), inv.invoiceID, "12.00"),
			map[string]any{"transaction_key": "adj", "invoice_id": inv.invoiceID, "amount": "3.00"},
		},
	})
	return inv, payment
}

// A search for a transaction or invoice number is answered from the allocations of just that
// transaction or invoice; the entries it returns are the ones a settlement recorded through the API.
func TestPayments_AllocationEntriesFoundByNumber(t *testing.T) {
	t.Parallel()
	inv, payment := settledPaymentAndDiscount(t)
	invoiceNumber := jsonField(getInvoice(t, inv.invoiceID), "number")
	customerName := getCustomerName(t, inv.customerID)

	rows, _ := listPayments(t, financeAllocationsPath, url.Values{"q": {jsonField(payment, "number")}, "limit": {"100"}})
	require.Len(t, rows, 1, "the payment's number finds its one entry")
	entry := rows[0]
	assert.NotEmpty(t, jsonField(entry, "id"))
	assert.Equal(t, "allocation_entry", jsonField(entry, "object"))
	assert.True(t, decimal.RequireFromString("12").Equal(decimal.RequireFromString(jsonField(entry, "amount"))), "amount %s", jsonField(entry, "amount"))
	assert.NotEmpty(t, jsonField(entry, "display_amount"))
	assert.Equal(t, inv.customerID, jsonField(jsonObject(entry, "customer"), "id"))
	assert.Equal(t, customerName, jsonField(jsonObject(entry, "customer"), "name"))
	tx := jsonObject(entry, "transaction")
	assert.Equal(t, jsonField(payment, "id"), jsonField(tx, "id"))
	assert.Equal(t, "payment", jsonField(tx, "type"))
	assert.Equal(t, "check", jsonField(tx, "method"))
	assertNilField(t, tx, "adjustment_type")
	assert.Equal(t, inv.invoiceID, jsonField(jsonObject(entry, "invoice"), "id"))
	assert.Equal(t, invoiceNumber, jsonField(jsonObject(entry, "invoice"), "number"))
	assertNilField(t, entry, "note")
	assert.WithinDuration(t, time.Now(), mustTime(t, jsonField(entry, "created_at")), 5*time.Minute)

	rows, _ = listPayments(t, financeAllocationsPath, url.Values{"q": {invoiceNumber}, "limit": {"100"}})
	types := map[string]bool{}
	for _, r := range rows {
		if jsonField(jsonObject(r, "invoice"), "id") == inv.invoiceID {
			types[jsonField(jsonObject(r, "transaction"), "type")] = true
		}
	}
	assert.Equal(t, map[string]bool{"payment": true, "adjustment": true}, types, "the invoice's number finds both its entries")

	rows, _ = listPayments(t, financeAllocationsPath, url.Values{"q": {"no-such-entry-" + inv.invoiceID}, "limit": {"100"}})
	assert.Empty(t, rows)
}

func TestPayments_AllocationEntriesFilterByTransactionType(t *testing.T) {
	t.Parallel()
	inv, payment := settledPaymentAndDiscount(t)
	invoiceNumber := jsonField(getInvoice(t, inv.invoiceID), "number")

	for _, tc := range []struct{ typ, want string }{{"adjustment", "adjustment"}, {"payment", "payment"}} {
		all, _ := listPayments(t, financeAllocationsPath, url.Values{"q": {invoiceNumber}, "transaction_type": {tc.typ}, "limit": {"100"}})
		var rows []map[string]any
		for _, r := range all {
			assert.Equal(t, tc.want, jsonField(jsonObject(r, "transaction"), "type"))
			if jsonField(jsonObject(r, "invoice"), "id") == inv.invoiceID {
				rows = append(rows, r)
			}
		}
		require.Len(t, rows, 1, "type %s", tc.typ)
		assert.Equal(t, tc.want, jsonField(jsonObject(rows[0], "transaction"), "type"))
		if tc.typ == "adjustment" {
			assert.Equal(t, "discount", jsonField(jsonObject(rows[0], "transaction"), "adjustment_type"))
		} else {
			assert.Equal(t, jsonField(payment, "id"), jsonField(jsonObject(rows[0], "transaction"), "id"))
		}
	}

	// Without a search, the type filter pages the account's entries of that type.
	rows, _ := listPayments(t, financeAllocationsPath, url.Values{"transaction_type": {"adjustment"}, "limit": {"5"}})
	require.NotEmpty(t, rows)
	for _, r := range rows {
		assert.Equal(t, "adjustment", jsonField(jsonObject(r, "transaction"), "type"))
	}
}

func TestPayments_AllocationEntriesPageBothWays(t *testing.T) {
	t.Parallel()
	inv, _ := settledPaymentAndDiscount(t)
	invoiceNumber := jsonField(getInvoice(t, inv.invoiceID), "number")

	params := url.Values{"q": {invoiceNumber}, "limit": {"1"}}
	first, info := listPayments(t, financeAllocationsPath, params)
	require.Len(t, first, 1)
	require.Equal(t, true, info["has_next_page"])

	next := jsonField(info, "next_page_url")
	status, body, err := apiClient.GetListRawFromPageURL(&next)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	page := parseJSON(body)
	second := jsonArray(page, "data")
	require.Len(t, second, 1)
	assert.NotEqual(t, jsonField(first[0], "id"), jsonField(second[0].(map[string]any), "id"))
	info = jsonObject(page, "page_info")
	assert.Equal(t, false, info["has_next_page"])

	require.Equal(t, true, info["has_prev_page"])
	prev := jsonField(info, "previous_page_url")
	status, body, err = apiClient.GetListRawFromPageURL(&prev)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	back := jsonArray(parseJSON(body), "data")
	require.Len(t, back, 1)
	assert.Equal(t, jsonField(first[0], "id"), jsonField(back[0].(map[string]any), "id"))
}

// The transaction and invoice filters are resolved to the settlements holding a matching allocation;
// both given, the same allocation must match both.
func TestPayments_SettlementsFilterByInvoiceAndTransaction(t *testing.T) {
	t.Parallel()
	inv, payment := settledPaymentAndDiscount(t)
	other := invoiceNewCustomer(t)

	rows, _ := listPayments(t, financeSettlementsPath, url.Values{"invoice_ids": {inv.invoiceID}})
	require.Len(t, rows, 1)
	settlementID := jsonField(rows[0], "id")
	assert.Equal(t, "2", jsonField(rows[0], "allocation_count"))

	rows, _ = listPayments(t, financeSettlementsPath, url.Values{"invoice_ids": {inv.invoiceID}, "transaction_ids": {jsonField(payment, "id")}})
	assert.Equal(t, []string{settlementID}, rowIDs(rows))

	rows, _ = listPayments(t, financeSettlementsPath, url.Values{"invoice_ids": {other.invoiceID}, "transaction_ids": {jsonField(payment, "id")}})
	assert.Empty(t, rows, "no allocation of the payment to the other invoice")

	rows, _ = listPayments(t, financeSettlementsPath, url.Values{"invoice_ids": {other.invoiceID}})
	assert.Empty(t, rows, "an invoice never settled")
}
