//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The dashboard's payments pages (transactions, settlements, the settle flow, open credits and payments
// data) run on these endpoints. Each test ships an order to a customer created for it, so the invoice
// it settles is its own and the flags it checks move only with its own payments.

const (
	financeTransactionsPath = "/v1/finance/transactions"
	financeSettlementsPath  = "/v1/finance/settlements"
	financeAllocationsPath  = "/v1/finance/transaction-allocations"
	financeOpenCreditsPath  = "/v1/finance/open-credits"
	financeInvoicesPath     = "/v1/finance/invoices"
)

type paymentsInvoice struct {
	customerID string
	invoiceID  string
	total      decimal.Decimal
}

// invoiceNewCustomer ships an order to a fresh customer and returns the invoice it raised.
func invoiceNewCustomer(t *testing.T) paymentsInvoice {
	t.Helper()
	sale := shipSaleToNewCustomer(t)

	status, body, err := apiClient.GetListRaw(shipmentsPath+"/"+sale.shipmentID, url.Values{"include": {"related.invoice"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	invoiceID := jsonField(jsonObject(jsonObject(parseJSON(body), "related"), "invoice"), "id")
	require.NotEmpty(t, invoiceID, "shipping must raise an invoice: %s", string(body))

	invoice := getInvoice(t, invoiceID)
	total := decimal.RequireFromString(jsonField(invoice, "total_invoiced"))
	require.True(t, total.IsPositive(), "invoice total: %s", total)
	return paymentsInvoice{customerID: sale.customerID, invoiceID: invoiceID, total: total}
}

func getInvoice(t *testing.T, id string) map[string]any {
	t.Helper()
	status, body, err := apiClient.GetListRaw(financeInvoicesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func invoicePaymentStatus(t *testing.T, id string) string {
	t.Helper()
	return jsonField(getInvoice(t, id), "payment_status")
}

// Payment flags are recomputed asynchronously after a settlement or allocation changes, so tests wait
// for them. A recompute sets a transaction's and its invoices' flags together, so waiting on a flag
// that changes proves the flags that did not change were recomputed too.
func awaitInvoiceStatus(t *testing.T, id, want string) {
	t.Helper()
	eventually(t, 15*time.Second, 200*time.Millisecond, func() error {
		if got := invoicePaymentStatus(t, id); got != want {
			return fmt.Errorf("invoice %s payment_status = %s, want %s", id, got, want)
		}
		return nil
	})
}

func awaitFullyAllocated(t *testing.T, transactionID string, want bool) {
	t.Helper()
	eventually(t, 15*time.Second, 200*time.Millisecond, func() error {
		if got := jsonField(getTransaction(t, transactionID), "is_fully_allocated"); got != fmt.Sprint(want) {
			return fmt.Errorf("transaction %s is_fully_allocated = %s, want %v", transactionID, got, want)
		}
		return nil
	})
}

// createPayment records a payment from the customer; funds is when the money arrived (nil: not yet).
func createPayment(t *testing.T, customerID, amount string, funds *time.Time, extra map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{"customer_id": customerID, "type": "payment", "amount": amount, "method": "check", "responsible_user_id": SeedAccountUserID}
	if funds != nil {
		body["funds_received_at"] = rfc3339(*funds)
	}
	for k, v := range extra {
		body[k] = v
	}
	status, resp, err := apiClient.Post(financeTransactionsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	tx := parseJSON(resp)
	id := jsonField(tx, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(financeTransactionsPath + "/" + id) })
	return tx
}

func getTransaction(t *testing.T, id string, include ...string) map[string]any {
	t.Helper()
	var params url.Values
	if len(include) > 0 {
		params = url.Values{"include": include}
	}
	status, body, err := apiClient.GetListRaw(financeTransactionsPath+"/"+id, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func settle(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	if _, ok := body["responsible_user_id"]; !ok {
		body["responsible_user_id"] = SeedUserID
	}
	status, resp, err := apiClient.Post(financeSettlementsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	settlement := parseJSON(resp)
	id := jsonField(settlement, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(financeSettlementsPath + "/" + id) })
	return settlement
}

func allocation(transactionID, invoiceID, amount string) map[string]any {
	return map[string]any{"transaction_id": transactionID, "invoice_id": invoiceID, "amount": amount}
}

func amountOf(t *testing.T, obj map[string]any) decimal.Decimal {
	t.Helper()
	return decimal.RequireFromString(jsonField(jsonObject(obj, "amount"), "value"))
}

func listPayments(t *testing.T, path string, params url.Values) ([]map[string]any, map[string]any) {
	t.Helper()
	status, body, err := apiClient.GetListRaw(path, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	page := parseJSON(body)
	var rows []map[string]any
	for _, r := range jsonArray(page, "data") {
		rows = append(rows, r.(map[string]any))
	}
	return rows, jsonObject(page, "page_info")
}

func rowIDs(rows []map[string]any) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = jsonField(r, "id")
	}
	return out
}

// --- Transactions ---

func TestPayments_TransactionKeepsItsDatesAndClearsThem(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	createdAt := time.Now().UTC().Add(-72 * time.Hour).Truncate(time.Second)
	funds := createdAt.Add(time.Hour)

	tx := createPayment(t, inv.customerID, "25.00", &funds, map[string]any{"occurred_at": rfc3339(createdAt), "note": "check 1042"})
	assert.Equal(t, rfc3339(createdAt), rfc3339(mustTime(t, jsonField(tx, "created_at"))))
	assert.Equal(t, rfc3339(funds), rfc3339(mustTime(t, jsonField(tx, "funds_received_at"))))
	responsibleID := jsonField(jsonObject(getTransaction(t, jsonField(tx, "id"), "responsible_user"), "responsible_user"), "id")
	require.NotEmpty(t, responsibleID)

	// The dashboard saves a transaction whole: re-sending its responsible user is not a change.
	status, body, err := apiClient.Patch(financeTransactionsPath+"/"+jsonField(tx, "id"), map[string]any{
		"note": nil, "funds_received_at": nil, "responsible_user_id": responsibleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	updated := parseJSON(body)
	assertNilField(t, updated, "note")
	assertNilField(t, updated, "funds_received_at")
}

func TestPayments_NumbersAreSearchedByWordPrefix(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	funds := time.Now().UTC()
	tx := createPayment(t, inv.customerID, "5.00", &funds, nil)
	number := jsonField(tx, "number")
	require.NotEmpty(t, number)

	rows, _ := listPayments(t, financeTransactionsPath, url.Values{"q": {number}, "customer_ids": {inv.customerID}})
	assert.Equal(t, []string{jsonField(tx, "id")}, rowIDs(rows))

	rows, _ = listPayments(t, financeTransactionsPath, url.Values{"q": {number + " nosuchword"}, "customer_ids": {inv.customerID}})
	assert.Empty(t, rows, "every word must match")
}

func TestPayments_TransactionsPageByCreatedAtNotID(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	funds := time.Now().UTC()
	now := time.Now().UTC().Truncate(time.Second)
	// Created in id order newest-first, so an id cursor would walk them in the wrong order.
	oldest := createPayment(t, inv.customerID, "1.00", &funds, map[string]any{"occurred_at": rfc3339(now.Add(-1 * time.Hour))})
	middle := createPayment(t, inv.customerID, "2.00", &funds, map[string]any{"occurred_at": rfc3339(now.Add(-2 * time.Hour))})
	newest := createPayment(t, inv.customerID, "3.00", &funds, map[string]any{"occurred_at": rfc3339(now.Add(-3 * time.Hour))})
	want := []string{jsonField(oldest, "id"), jsonField(middle, "id"), jsonField(newest, "id")}

	var got []string
	rows, pageInfo := listPayments(t, financeTransactionsPath, url.Values{"customer_ids": {inv.customerID}, "limit": {"1"}})
	got = append(got, rowIDs(rows)...)
	for pageInfo["has_next_page"] == true {
		next := jsonField(pageInfo, "next_page_url")
		status, body, err := apiClient.GetListRawFromPageURL(&next)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		page := parseJSON(body)
		for _, r := range jsonArray(page, "data") {
			got = append(got, jsonField(r.(map[string]any), "id"))
		}
		pageInfo = jsonObject(page, "page_info")
		require.LessOrEqual(t, len(got), 3, "paging must end")
	}
	assert.Equal(t, want, got, "newest created first, each exactly once")
}

func TestPayments_SettleableTransactionsHaveTheirFunds(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	funds := time.Now().UTC()
	received := createPayment(t, inv.customerID, "10.00", &funds, nil)
	createPayment(t, inv.customerID, "11.00", nil, nil)

	rows, _ := listPayments(t, "/v1/finance/accounts/"+inv.customerID+"/transactions", url.Values{"status": {"unallocated"}, "include": {"allocations"}})
	assert.Equal(t, []string{jsonField(received, "id")}, rowIDs(rows), "a transaction whose money has not arrived cannot be settled")
	require.Len(t, rows, 1)
	assert.NotNil(t, rows[0]["allocations"])
}

// --- Settlements ---

func TestPayments_SettlingRecomputesFlagsFromEveryAllocation(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	funds := time.Now().UTC()
	part := inv.total.Sub(decimal.NewFromInt(10))

	first := createPayment(t, inv.customerID, part.StringFixed(2), &funds, nil)
	settle(t, map[string]any{"allocations": []any{allocation(jsonField(first, "id"), inv.invoiceID, part.StringFixed(2))}})
	awaitFullyAllocated(t, jsonField(first, "id"), true)
	assert.Equal(t, "unpaid", invoicePaymentStatus(t, inv.invoiceID))

	// The second payment is larger than what is left: it pays the invoice and keeps a balance.
	second := createPayment(t, inv.customerID, "15.00", &funds, nil)
	secondSettlement := settle(t, map[string]any{"allocations": []any{allocation(jsonField(second, "id"), inv.invoiceID, "10.00")}})
	awaitInvoiceStatus(t, inv.invoiceID, "paid")
	assert.Equal(t, "false", jsonField(getTransaction(t, jsonField(second, "id")), "is_fully_allocated"), "$5 of it is still unapplied")

	// Deleting the second settlement leaves the first one's payment applied.
	status, body, err := apiClient.Delete(financeSettlementsPath + "/" + jsonField(secondSettlement, "id"))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	awaitInvoiceStatus(t, inv.invoiceID, "unpaid")
	assert.Equal(t, "true", jsonField(getTransaction(t, jsonField(first, "id")), "is_fully_allocated"), "recomputed, not reset")
}

func TestPayments_SettlementRecordsItsNewTransactions(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	appliedAt := time.Now().UTC().Add(-48 * time.Hour).Truncate(time.Second)

	settlement := settle(t, map[string]any{
		"new_transactions": []any{map[string]any{"key": "adj", "type": "adjustment", "adjustment_type": "discount", "customer_id": inv.customerID}},
		"allocations": []any{
			map[string]any{"transaction_key": "adj", "invoice_id": inv.invoiceID, "amount": "5.00", "applied_at": rfc3339(appliedAt)},
			map[string]any{"transaction_key": "adj", "invoice_id": inv.invoiceID, "amount": "2.50", "applied_at": rfc3339(appliedAt.Add(time.Hour))},
		},
	})

	status, body, err := apiClient.GetListRaw(financeSettlementsPath+"/"+jsonField(settlement, "id"), url.Values{"include": {"allocations", "allocations.transaction"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	allocations := jsonArray(jsonObject(parseJSON(body), "allocations"), "data")
	require.Len(t, allocations, 2)
	first := allocations[0].(map[string]any)
	assert.Equal(t, rfc3339(appliedAt), rfc3339(mustTime(t, jsonField(first, "created_at"))))
	assert.Equal(t, inv.invoiceID, jsonField(jsonObject(first, "invoice"), "id"))
	assert.Equal(t, jsonField(settlement, "id"), jsonField(jsonObject(first, "settlement"), "id"))

	tx := jsonObject(first, "transaction")
	txID := jsonField(tx, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(financeTransactionsPath + "/" + txID) })
	got := getTransaction(t, txID)
	assert.True(t, decimal.RequireFromString("7.50").Equal(amountOf(t, got)), "the sum of its allocations")
	assert.Equal(t, rfc3339(appliedAt), rfc3339(mustTime(t, jsonField(got, "created_at"))))
	assert.Equal(t, rfc3339(appliedAt), rfc3339(mustTime(t, jsonField(got, "funds_received_at"))))
	assert.NotEmpty(t, jsonField(got, "number"))
	awaitFullyAllocated(t, txID, true)
}

// A settlement owns the transactions it recorded: deleting it deletes them, whatever their type, or
// their funds would stay received and reappear as open credits to apply again. Money it drew from a
// transaction recorded on its own is released, not deleted.
func TestPayments_DeletingASettlementRemovesTheTransactionsItRecorded(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	funds := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	existing := createPayment(t, inv.customerID, "10.00", &funds, nil)

	types := []string{"payment", "credit_memo", "rebate", "adjustment"}
	var newTransactions, allocations []any
	for _, typ := range types {
		nt := map[string]any{"key": typ, "type": typ, "customer_id": inv.customerID}
		if typ == "payment" {
			nt["method"] = "check"
		}
		newTransactions = append(newTransactions, nt)
		allocations = append(allocations, map[string]any{"transaction_key": typ, "invoice_id": inv.invoiceID, "amount": "1.00"})
	}
	allocations = append(allocations, allocation(jsonField(existing, "id"), inv.invoiceID, "4.00"))
	settlement := settle(t, map[string]any{"new_transactions": newTransactions, "allocations": allocations})
	settlementID := jsonField(settlement, "id")

	status, body, err := apiClient.GetListRaw(financeSettlementsPath+"/"+settlementID, url.Values{"include": {"allocations", "allocations.transaction"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	var recorded []string
	for _, a := range jsonArray(jsonObject(parseJSON(body), "allocations"), "data") {
		if txID := jsonField(jsonObject(a.(map[string]any), "transaction"), "id"); txID != jsonField(existing, "id") {
			recorded = append(recorded, txID)
		}
	}
	require.Len(t, recorded, len(types), "one transaction recorded per new_transactions entry")
	awaitFullyAllocated(t, recorded[0], true)

	status, body, err = apiClient.Delete(financeSettlementsPath + "/" + settlementID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for _, txID := range recorded {
		status, body, err := apiClient.GetListRaw(financeTransactionsPath+"/"+txID, nil)
		require.NoError(t, err)
		assert.Equal(t, 404, status, "transaction %s recorded by the deleted settlement must go with it: %s", txID, string(body))
	}
	assert.Equal(t, jsonField(existing, "id"), jsonField(getTransaction(t, jsonField(existing, "id")), "id"), "a transaction recorded on its own is kept")
	awaitFullyAllocated(t, jsonField(existing, "id"), false)

	rows, _ := listPayments(t, financeOpenCreditsPath, url.Values{"customer_ids": {inv.customerID}})
	require.Equal(t, []string{jsonField(existing, "id")}, rowIDs(rows), "only the standalone payment is open again")
	assert.True(t, decimal.RequireFromString("10").Equal(decimal.RequireFromString(jsonField(rows[0], "leftover_amount"))))
}

func TestPayments_MoneyNotYetReceivedCannotBeSettled(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	pending := createPayment(t, inv.customerID, "10.00", nil, nil)

	status, body, err := apiClient.Post(financeSettlementsPath, map[string]any{
		"responsible_user_id": SeedUserID,
		"allocations":         []any{allocation(jsonField(pending, "id"), inv.invoiceID, "10.00")},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assert.Equal(t, "unpaid", invoicePaymentStatus(t, inv.invoiceID))
}

func TestPayments_SettlementListSummarizesAllItsAllocations(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	funds := time.Now().UTC()
	payment := createPayment(t, inv.customerID, "12.00", &funds, nil)
	settlement := settle(t, map[string]any{
		"new_transactions": []any{map[string]any{"key": "adj", "type": "adjustment", "adjustment_type": "discount", "customer_id": inv.customerID}},
		"allocations": []any{
			allocation(jsonField(payment, "id"), inv.invoiceID, "12.00"),
			map[string]any{"transaction_key": "adj", "invoice_id": inv.invoiceID, "amount": "3.00"},
		},
	})
	// Filtering by the payment still summarizes the adjustment drawn in the same settlement.
	rows, _ := listPayments(t, financeSettlementsPath, url.Values{"transaction_ids": {jsonField(payment, "id")}})
	require.Len(t, rows, 1)
	row := rows[0]
	assert.Equal(t, jsonField(settlement, "id"), jsonField(row, "id"))
	assert.Equal(t, "2", jsonField(row, "allocation_count"))
	assert.True(t, decimal.RequireFromString("12").Equal(decimal.RequireFromString(jsonField(row, "total_payments"))))
	assert.True(t, decimal.RequireFromString("3").Equal(decimal.RequireFromString(jsonField(row, "total_adjustments"))))
	assertNilField(t, row, "total_rebates")
	assertNilField(t, row, "total_credits")

	customer := getCustomerName(t, inv.customerID)
	assert.Equal(t, []any{customer}, row["customer_names"])
}

func getCustomerName(t *testing.T, id string) string {
	t.Helper()
	status, body, err := apiClient.GetListRaw("/v1/sales/customers/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return jsonField(parseJSON(body), "name")
}

// --- Allocations ---

func TestPayments_EditingAnAllocationRecomputesFlags(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	funds := time.Now().UTC()
	payment := createPayment(t, inv.customerID, inv.total.StringFixed(2), &funds, nil)
	settlement := settle(t, map[string]any{"allocations": []any{allocation(jsonField(payment, "id"), inv.invoiceID, inv.total.StringFixed(2))}})
	awaitInvoiceStatus(t, inv.invoiceID, "paid")

	detail := getTransaction(t, jsonField(payment, "id"), "allocations")
	allocs := jsonArray(jsonObject(detail, "allocations"), "data")
	require.Len(t, allocs, 1)
	alloc := allocs[0].(map[string]any)
	assert.Equal(t, jsonField(settlement, "id"), jsonField(jsonObject(alloc, "settlement"), "id"))
	assert.Equal(t, inv.invoiceID, jsonField(jsonObject(alloc, "invoice"), "id"))
	allocID := jsonField(alloc, "id")

	redated := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Second)
	status, body, err := apiClient.Patch(financeAllocationsPath+"/"+allocID, map[string]any{
		"amount": inv.total.Sub(decimal.NewFromInt(1)).StringFixed(2), "applied_at": rfc3339(redated),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, rfc3339(redated), rfc3339(mustTime(t, jsonField(parseJSON(body), "created_at"))))
	awaitFullyAllocated(t, jsonField(payment, "id"), false)
	assert.Equal(t, "unpaid", invoicePaymentStatus(t, inv.invoiceID))

	status, body, err = apiClient.Delete(financeAllocationsPath + "/" + allocID)
	require.NoError(t, err)
	require.Less(t, status, 300, string(body))
	awaitInvoiceStatus(t, inv.invoiceID, "unpaid")
}

func TestPayments_OpenCreditsAreReceivedAndUnapplied(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	funds := time.Now().UTC().Truncate(time.Second)
	credit := createPayment(t, inv.customerID, "20.00", &funds, nil)
	createPayment(t, inv.customerID, "30.00", nil, nil)
	spent := createPayment(t, inv.customerID, "4.00", &funds, nil)
	settle(t, map[string]any{"allocations": []any{
		allocation(jsonField(credit, "id"), inv.invoiceID, "5.00"),
		allocation(jsonField(spent, "id"), inv.invoiceID, "4.00"),
	}})

	rows, _ := listPayments(t, financeOpenCreditsPath, url.Values{"customer_ids": {inv.customerID}})
	require.Equal(t, []string{jsonField(credit, "id")}, rowIDs(rows), "received, and with money left to apply")
	assert.True(t, decimal.RequireFromString("15").Equal(decimal.RequireFromString(jsonField(rows[0], "leftover_amount"))))
	assert.Equal(t, rfc3339(funds), rfc3339(mustTime(t, jsonField(rows[0], "funds_received_at"))))
}

func TestPayments_AllocationEntriesFindTheirCustomer(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	funds := time.Now().UTC()
	payment := createPayment(t, inv.customerID, "6.00", &funds, nil)
	settle(t, map[string]any{"allocations": []any{allocation(jsonField(payment, "id"), inv.invoiceID, "6.00")}})
	name := getCustomerName(t, inv.customerID)

	rows, _ := listPayments(t, financeAllocationsPath, url.Values{"q": {name}, "limit": {"100"}})
	require.NotEmpty(t, rows)
	found := false
	for _, r := range rows {
		if jsonField(jsonObject(r, "transaction"), "id") == jsonField(payment, "id") {
			found = true
			assert.Equal(t, inv.customerID, jsonField(jsonObject(r, "customer"), "id"))
		}
	}
	assert.True(t, found, "searching the customer's name finds the entry")
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339Nano, s)
	require.NoError(t, err, "timestamp %q", s)
	return v
}

// Settlements against one invoice recompute its flags on separate deliveries; whichever runs last must
// see every allocation, so the invoice ends paid however the recomputes interleave.
func TestPayments_ConcurrentSettlementsLeaveTheInvoicePaid(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	funds := time.Now().UTC()
	const parts = 5
	share := inv.total.Div(decimal.NewFromInt(parts)).RoundDown(2)
	last := inv.total.Sub(share.Mul(decimal.NewFromInt(parts - 1)))

	payments := make([]string, parts)
	for i := range parts {
		amount := share
		if i == parts-1 {
			amount = last
		}
		payments[i] = jsonField(createPayment(t, inv.customerID, amount.StringFixed(2), &funds, nil), "id")
	}

	var wg sync.WaitGroup
	errs := make(chan error, parts)
	for i, id := range payments {
		amount := share
		if i == parts-1 {
			amount = last
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, body, err := apiClient.Post(financeSettlementsPath, map[string]any{
				"responsible_user_id": SeedUserID,
				"allocations":         []any{allocation(id, inv.invoiceID, amount.StringFixed(2))},
			}, newIdempotencyKey())
			if err == nil && status != 201 {
				err = fmt.Errorf("settle %s: %d %s", id, status, body)
			}
			if err == nil {
				settlementID := jsonField(parseJSON(body), "id")
				t.Cleanup(func() { _, _, _ = apiClient.Delete(financeSettlementsPath + "/" + settlementID) })
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	awaitInvoiceStatus(t, inv.invoiceID, "paid")
	for _, id := range payments {
		awaitFullyAllocated(t, id, true)
	}
}

func TestPayments_OpenCreditsPageBothWays(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	var want []string
	for i := range 3 {
		funds := base.Add(time.Duration(-i) * time.Minute) // newest first
		want = append(want, jsonField(createPayment(t, inv.customerID, "10.00", &funds, nil), "id"))
	}

	params := url.Values{"customer_ids": {inv.customerID}, "limit": {"1"}}
	var got []string
	rows, info := listPayments(t, financeOpenCreditsPath, params)
	got = append(got, rowIDs(rows)...)
	for i := 0; i < 5 && info["has_next_page"] == true; i++ {
		next := jsonField(info, "next_page_url")
		status, body, err := apiClient.GetListRawFromPageURL(&next)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		page := parseJSON(body)
		for _, r := range jsonArray(page, "data") {
			got = append(got, jsonField(r.(map[string]any), "id"))
		}
		info = jsonObject(page, "page_info")
	}
	require.Equal(t, want, got, "most recently received first")

	require.Equal(t, true, info["has_prev_page"], "the last page links back")
	prev := jsonField(info, "previous_page_url")
	status, body, err := apiClient.GetListRawFromPageURL(&prev)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, []string{want[1]}, func() []string {
		var out []string
		for _, r := range jsonArray(parseJSON(body), "data") {
			out = append(out, jsonField(r.(map[string]any), "id"))
		}
		return out
	}())
}
