//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// SalesOrder — payment_status parity
// ──────────────────────────────────────────────
//
// payment_status is a derived, server-computed scalar that mirrors the legacy
// dashboard "is paid" rule: an order is PAID if it has a Stripe payment intent,
// OR it is fulfilled and every one of its invoices is paid in full
// (invoice.is_paid_in_full). "partially_paid" surfaces settlement-allocation
// activity that has not yet fully paid the order; everything else is "unpaid".
//
// These tests pin that classification on both the detail (GET /{id}) and list
// (GET /) endpoints so the customer-facing list and the invoice never disagree
// (the reported bug: an order showed "unpaid" while its invoice was paid).

// salesOrderPaymentStatus fetches one order's detail and returns its
// payment_status field.
func salesOrderPaymentStatus(t *testing.T, orderID string) string {
	t.Helper()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return jsonField(parseJSON(body), "payment_status")
}

func TestSalesOrders_PaymentStatus_Detail(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		orderID string
		want    string
	}{
		// Fulfilled order, invoice marked paid in full -> paid. This is the
		// reported bug scenario; it must not read "unpaid".
		{"fulfilled with paid invoice", SeedFulfilledPaidOrderID, "paid"},
		// Fulfilled + paid invoice but no settlement allocation -> still paid.
		// The earlier allocation-vs-invoiced derivation wrongly marked this
		// "unpaid"; the legacy rule reads invoice.is_paid_in_full directly.
		{"fulfilled paid invoice, no allocation", SeedPaidNoAllocOrderID, "paid"},
		// Issued order with a partial settlement allocation -> partially_paid.
		{"issued with partial allocation", SeedSalesOrderID, "partially_paid"},
		// Estimate with no invoice or payment -> unpaid.
		{"estimate, no payment", SeedEstimateOrderID, "unpaid"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, salesOrderPaymentStatus(t, tc.orderID),
				"order %s should be %q", tc.orderID, tc.want)
		})
	}
}

// TestSalesOrders_PaymentStatus_List pins that the list endpoint reports the
// same derived payment_status as detail — the list is where the bug surfaced.
func TestSalesOrders_PaymentStatus_List(t *testing.T) {
	t.Parallel()

	params := url.Values{"limit": {"100"}}
	status, body, err := apiClient.GetListRaw(salesOrdersPath, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	statuses := map[string]string{}
	for _, item := range jsonArray(got, "data") {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		statuses[jsonField(row, "id")] = jsonField(row, "payment_status")
	}

	want := map[string]string{
		SeedFulfilledPaidOrderID: "paid",
		SeedPaidNoAllocOrderID:   "paid",
		SeedSalesOrderID:         "partially_paid",
		SeedEstimateOrderID:      "unpaid",
	}
	for id, expected := range want {
		actual, present := statuses[id]
		require.Truef(t, present, "order %s missing from list page", id)
		assert.Equalf(t, expected, actual,
			"list payment_status for %s should match detail (%s)", id, expected)
	}
}
