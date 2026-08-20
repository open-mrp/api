//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Invoice & Receivable — Dashboard Parity
// ──────────────────────────────────────────────

const customerInvoicesPath = "/v1/finance/accounts/" + SeedCustomerAccountID + "/invoices"
const customerTransactionsPath = "/v1/finance/accounts/" + SeedCustomerAccountID + "/transactions"

// Collects the `id` of every entry in a list response.
func listIDs(t *testing.T, path string, params url.Values) []string {
	t.Helper()
	list, status, err := apiClient.GetList(path, params)
	require.NoError(t, err)
	require.Equal(t, 200, status, "GET %s", path)

	ids := make([]string, 0, len(list.Data))
	for _, raw := range list.Data {
		ids = append(ids, jsonField(parseJSON(raw), "id"))
	}
	return ids
}

// The search term reaches the order behind the invoice, not just the invoice's own columns.
func TestInvoices_SearchMatchesOrderNumber(t *testing.T) {
	t.Parallel()

	ids := listIDs(t, invoicesPath, url.Values{"q": {"ORD-001"}})
	assert.Contains(t, ids, "iv_01seedinvoice002000",
		"searching the sales order number should surface the invoice billed against it")
}

// The customer's external/relation number is part of the search surface.
func TestInvoices_SearchMatchesCustomerNumber(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(invoicesPath, url.Values{"q": {SeedCustomerNumber}, "include": {"customer"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.NotEmpty(t, list.Data, "seed customer %s should have invoices", SeedCustomerNumber)

	for _, raw := range list.Data {
		customer := jsonObject(parseJSON(raw), "customer")
		require.NotNil(t, customer)
		assert.Equal(t, SeedCustomerAccountID, jsonField(customer, "id"),
			"a customer-number search must only return that customer's invoices")
	}
}

// A customer PO number typed into the search box reaches the order it lives on.
func TestInvoices_SearchMatchesCustomerPONumber(t *testing.T) {
	t.Parallel()

	// The PO lives on the order, so every hit must be an invoice for an order carrying it.
	ids := listIDs(t, invoicesPath, url.Values{"q": {SeedSalesOrderPONumber}})
	all := listIDs(t, invoicesPath, nil)
	for _, id := range ids {
		assert.Contains(t, all, id, "PO search must not invent invoices outside the account")
	}
}

// A search term that matches nothing returns an empty page rather than the whole list.
func TestInvoices_SearchMissesReturnEmpty(t *testing.T) {
	t.Parallel()

	ids := listIDs(t, invoicesPath, url.Values{"q": {"zzz-no-such-invoice-zzz"}})
	assert.Empty(t, ids)
}

// `unpaid` keys off the paid-in-full flag alone, so partially paid and overpaid invoices stay in.
func TestInvoices_UnpaidStatusKeysOffPaidInFullOnly(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(invoicesPath, url.Values{"status": {"unpaid"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.NotEmpty(t, list.Data, "seed data should include unpaid invoices")

	for _, raw := range list.Data {
		m := parseJSON(raw)
		assert.NotEqual(t, "paid", jsonField(m, "payment_status"),
			"an invoice marked paid in full must not appear under status=unpaid")
	}

	// INV-001 is marked paid in full, so it is the negative control.
	ids := listIDs(t, invoicesPath, url.Values{"status": {"unpaid"}})
	assert.NotContains(t, ids, SeedInvoiceID)
}

// Item and product-line filters scope to the order's lines, so a partial invoice still matches.
func TestInvoices_LineFiltersScopeToOrderLines(t *testing.T) {
	t.Parallel()

	all := listIDs(t, invoicesPath, nil)

	byItem := listIDs(t, invoicesPath, url.Values{"item_ids": {SeedItemID}})
	require.NotEmpty(t, byItem, "seed item %s should match invoices through its order lines", SeedItemID)
	for _, id := range byItem {
		assert.Contains(t, all, id)
	}

	byProductLine := listIDs(t, invoicesPath, url.Values{"product_line_ids": {SeedProductLineID}})
	require.NotEmpty(t, byProductLine)
	for _, id := range byProductLine {
		assert.Contains(t, all, id)
	}
}

// An unknown item ID filters everything out instead of being ignored.
func TestInvoices_ItemFilterUnknownIDReturnsEmpty(t *testing.T) {
	t.Parallel()

	ids := listIDs(t, invoicesPath, url.Values{"item_ids": {"it_01nosuchitem0000000"}})
	assert.Empty(t, ids)
}

// The child-account roll-up is unconditional, matching legacy, which hard-codes allowChildAccounts:
// true. The old opt-out query parameter is gone, so sending it is rejected rather than ignored.
func TestCustomerInvoices_ChildAccountOptOutIsGone(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, listIDs(t, customerInvoicesPath, nil), "the customer has payable invoices")

	for _, path := range []string{customerInvoicesPath, customerTransactionsPath} {
		status, body, err := apiClient.GetListRaw(path, url.Values{"include_child_accounts": {"false"}})
		require.NoError(t, err)
		assert.Equal(t, 400, status, "GET %s must reject the removed parameter", path)
		assert.Contains(t, string(body), "parameter_unknown",
			"the parameter must be unknown, not merely ignored")
	}
}

// Clearing the note writes NULL rather than preserving the previous text.
func TestInvoices_NoteClearsToNull(t *testing.T) {
	// Not parallel: mutates the shared seed invoice.
	getStatus, getBody, err := apiClient.GetListRaw(invoicesPath+"/"+SeedInvoiceID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	original := parseJSON(getBody)["note"]

	t.Cleanup(func() {
		_, _, _ = apiClient.Patch(invoicesPath+"/"+SeedInvoiceID, map[string]any{"note": original}, newIdempotencyKey())
	})

	setStatus, setBody, err := apiClient.Patch(invoicesPath+"/"+SeedInvoiceID,
		map[string]any{"note": "parity note"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, setStatus, setBody)
	assert.Equal(t, "parity note", jsonField(parseJSON(setBody), "note"))

	clearStatus, clearBody, err := apiClient.Patch(invoicesPath+"/"+SeedInvoiceID,
		map[string]any{"note": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, clearStatus, clearBody)
	assert.Nil(t, parseJSON(clearBody)["note"], "an explicit null must clear the note")

	// An omitted note leaves the cleared value alone.
	keepStatus, keepBody, err := apiClient.Patch(invoicesPath+"/"+SeedInvoiceID,
		map[string]any{"has_been_sent": false}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, keepStatus, keepBody)
	assert.Nil(t, parseJSON(keepBody)["note"], "omitting the note must not resurrect it")
}

// An as-of run reports only entries that still owed money at the cutoff.
func TestReceivables_CutoffDropsSettledEntries(t *testing.T) {
	t.Parallel()

	cutoff := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	list, status, err := apiClient.GetList(receivablesPath, url.Values{"cutoff_at": {cutoff}})
	require.NoError(t, err)
	require.Equal(t, 200, status)

	for _, raw := range list.Data {
		m := parseJSON(raw)
		balance := jsonField(m, "remaining_balance")
		assert.NotEqual(t, "0.00", balance, "a cleared entry must drop out of an as-of run")
		assert.NotContains(t, balance, "-", "a credit balance must drop out of an as-of run")
	}
}

// The same cutoff pruning applies to the per-customer receivables listing.
func TestReceivablesByCustomer_CutoffDropsSettledEntries(t *testing.T) {
	t.Parallel()

	path := receivablesPath + "/accounts/" + SeedCustomerAccountID
	cutoff := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	list, status, err := apiClient.GetList(path, url.Values{"cutoff_at": {cutoff}})
	require.NoError(t, err)
	require.Equal(t, 200, status)

	for _, raw := range list.Data {
		m := parseJSON(raw)
		balance := jsonField(m, "remaining_balance")
		assert.NotEqual(t, "0.00", balance)
		assert.NotContains(t, balance, "-")
	}
}

// Without a cutoff every unpaid invoice is reported, settled-to-zero ones included.
func TestReceivables_NoCutoffKeepsZeroBalanceEntries(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(receivablesPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)

	for _, raw := range list.Data {
		assert.False(t, parseJSON(raw)["is_paid_in_full"].(bool),
			"receivables never report invoices already marked paid in full")
	}
}

// The invoices page sends customer, group, sales-rep and date filters alongside the ones above
// (invoice.api.ts fetchInvoices). A filter the server drops still returns 200 with a full page, so
// each case pairs a real match with a nonsense id that must empty the list.
func TestInvoices_FiltersByCustomerGroupAndSalesRep(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, listIDs(t, invoicesPath, url.Values{"customer_ids": {SeedCustomerAccountID}}),
		"the seeded customer has invoices")
	assert.Empty(t, listIDs(t, invoicesPath, url.Values{"customer_ids": {"ac_01nosuchcustomer000"}}))

	assert.NotEmpty(t, listIDs(t, invoicesPath, url.Values{"customer_group_ids": {SeedCustomerGroupID}}),
		"the seeded customer belongs to the seeded group")
	assert.Empty(t, listIDs(t, invoicesPath, url.Values{"customer_group_ids": {"acgp_01nosuchgroup0000"}}))

	assert.NotEmpty(t, listIDs(t, invoicesPath, url.Values{"sales_rep_ids": {SeedAccountUserID}}),
		"the seeded customer's default sales rep owns these invoices")
	assert.Empty(t, listIDs(t, invoicesPath, url.Values{"sales_rep_ids": {"acus_nosuchsalesrep00"}}))
}

func TestInvoices_FiltersByCreatedDateWindow(t *testing.T) {
	t.Parallel()

	today := time.Now().UTC().Format("2006-01-02")
	assert.NotEmpty(t, listIDs(t, invoicesPath, url.Values{"starts_at": {"2000-01-01"}, "ends_at": {today}}),
		"a window covering today must include the seeded invoices")
	assert.Empty(t, listIDs(t, invoicesPath, url.Values{"starts_at": {"2000-01-01"}, "ends_at": {"2000-01-02"}}),
		"a window that closed decades ago must exclude every invoice")
}

// The invoice detail page hydrates its lines and order in one request, so the deep chain must
// resolve: a line's order line, that line's product, and the order's own customer and payment term.
func TestInvoices_RetrieveHydratesDeepIncludes(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	for _, inc := range []string{
		"lines", "lines.order_line", "lines.order_line.product",
		"order",
	} {
		params.Add("include", inc)
	}

	status, body, err := apiClient.GetListRaw(invoicesPath+"/"+SeedInvoiceID, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)

	lines := jsonListData(got, "lines")
	require.NotEmpty(t, lines, "the seeded invoice has lines")
	line, ok := lines[0].(map[string]any)
	require.True(t, ok)

	orderLine := jsonObject(line, "order_line")
	require.NotNil(t, orderLine, "lines.order_line must expand")
	assert.NotEmpty(t, jsonField(orderLine, "id"))
	assert.NotEmpty(t, jsonField(orderLine, "id"), "the order line resolves to a real row")

	product := jsonObject(orderLine, "product")
	require.NotNil(t, product, "lines.order_line.product must expand")
	assert.Equal(t, "product", jsonField(product, "object"))
	assert.NotEmpty(t, jsonField(product, "id"))

	order := jsonObject(got, "order")
	require.NotNil(t, order, "order must expand")
	assert.Equal(t, "sales_order", jsonField(order, "object"))
}

// The settle flow works out each invoice's paid amount from its allocations, so the payment list
// must expand them — and, like every other expandable field, only when asked.
func TestCustomerInvoices_AllocationsExpandOnRequest(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(customerInvoicesPath, url.Values{"limit": {"25"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	for _, raw := range jsonArray(parseJSON(body), "data") {
		row, ok := raw.(map[string]any)
		require.True(t, ok)
		assert.Nil(t, row["allocations"], "allocations must stay unexpanded without ?include=")
	}

	status, body, err = apiClient.GetListRaw(customerInvoicesPath,
		url.Values{"limit": {"25"}, "include": {"allocations"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	sawAllocation := false
	for _, raw := range jsonArray(parseJSON(body), "data") {
		row, ok := raw.(map[string]any)
		require.True(t, ok)
		allocations := jsonObject(row, "allocations")
		require.NotNil(t, allocations, "allocations must expand on request")
		if entries := jsonArray(allocations, "data"); len(entries) > 0 {
			sawAllocation = true
			entry, ok := entries[0].(map[string]any)
			require.True(t, ok)
			assert.NotNil(t, jsonObject(entry, "amount"),
				"each allocation carries its amount, which is what the balance is worked out from")
		}
	}
	assert.True(t, sawAllocation, "a partially paid invoice is seeded, so at least one allocation must appear")
}
