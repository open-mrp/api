//go:build e2e

package api_test

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// The dashboard's invoice list, invoice detail, settle and receivables pages move onto these
// endpoints. Each test raises the invoices it reads by shipping orders to customers of its own, so
// filters and totals are exact and parallel tests never see each other's rows.

const (
	invoiceEmailRecordPath = "/v1/core/actions/email-record"
	invoicePaymentTermPath = "/v1/finance/payment-terms"
)

func customerInvoicesPathFor(customerID string) string {
	return "/v1/finance/accounts/" + customerID + "/invoices"
}

func receivablesPathFor(customerID string) string {
	return receivablesPath + "/accounts/" + customerID
}

func emailReceivablesPathFor(customerID string) string {
	return "/v1/finance/accounts/" + customerID + "/actions/email-receivables"
}

type parityInvoice struct {
	customerID  string
	orderID     string
	orderNumber string
	invoiceID   string
	number      string
	billToID    string
}

// parityInvoiceFor issues an order of lines (one seed sock when none) to customerID, ships all of it,
// and returns the invoice that raised. extra is merged into the order body.
func parityInvoiceFor(t *testing.T, customerID string, extra map[string]any, lines ...map[string]any) parityInvoice {
	t.Helper()
	if len(lines) == 0 {
		lines = []map[string]any{parityLine(SeedProductID, "4", SeedUnitID, "3.00")}
	}
	body := map[string]any{"lines": lines}
	for k, v := range extra {
		body[k] = v
	}
	order := issueOrderForCustomer(t, customerID, body)
	orderID := jsonField(order, "id")
	shipWholeOrder(t, orderID)

	status, raw, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, url.Values{"include": {"related.invoices", "bill_to_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	got := parseJSON(raw)
	invoices := jsonArray(jsonObject(jsonObject(got, "related"), "invoices"), "data")
	require.Len(t, invoices, 1, "shipping the whole order raises one invoice: %s", string(raw))
	invoiceID := jsonField(invoices[0].(map[string]any), "id")

	invoice := getInvoice(t, invoiceID)
	return parityInvoice{
		customerID:  customerID,
		orderID:     orderID,
		orderNumber: jsonField(got, "number"),
		invoiceID:   invoiceID,
		number:      jsonField(invoice, "number"),
		billToID:    jsonField(jsonObject(got, "bill_to_address"), "id"),
	}
}

// invoiceListIDs lists invoices as client sees them, reading every page.
func invoiceListIDs(t *testing.T, client *Client, path string, params url.Values) []string {
	t.Helper()
	var ids []string
	status, body, err := client.GetListRaw(path, params)
	for page := 0; ; page++ {
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		got := parseJSON(body)
		for _, raw := range jsonArray(got, "data") {
			row := raw.(map[string]any)
			if inv := jsonObject(row, "invoice"); inv != nil {
				ids = append(ids, jsonField(inv, "id"))
				continue
			}
			ids = append(ids, jsonField(row, "id"))
		}
		next := jsonField(jsonObject(got, "page_info"), "next_page_url")
		if next == "" || page >= maxListScanPages {
			return ids
		}
		status, body, err = client.GetListRawFromPageURL(&next)
	}
}

func invoiceRow(t *testing.T, client *Client, path string, params url.Values, id string) map[string]any {
	t.Helper()
	status, body, err := client.GetListRaw(path, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	for _, raw := range jsonArray(parseJSON(body), "data") {
		if row := raw.(map[string]any); jsonField(row, "id") == id {
			return row
		}
	}
	require.Failf(t, "invoice not listed", "%s %v has no %s: %s", path, params, id, string(body))
	return nil
}

func patchInvoice(t *testing.T, id string, body map[string]any) map[string]any {
	t.Helper()
	status, raw, err := apiClient.Patch(invoicesPath+"/"+id, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	return parseJSON(raw)
}

func withCustomers(params url.Values, customerIDs ...string) url.Values {
	out := url.Values{"customer_ids": customerIDs}
	for k, v := range params {
		out[k] = v
	}
	return out
}

// --- Access ---

// A buyer that also keeps the seller as one of its own suppliers holds a relation to the seller. Its
// users still read none of the seller's invoices or receivables: those are the seller's ledger.
func TestInvoiceParity_PortalUserCannotReachTheSellersInvoices(t *testing.T) {
	t.Parallel()
	customerID := parityCustomer(t, "")
	inv := parityInvoiceFor(t, customerID, nil)

	merchant := apiClient.WithAccountID(customerID)
	username, password := uniqueName("e2e-inv-portal"), "PortalPass123!"
	status, body, err := merchant.Post(accountUsersPath, map[string]any{"username": username, "password": password}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	accountUserID := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { _, _, _ = merchant.Put(accountUsersPath+"/"+accountUserID+"/actions/remove", nil) })

	db := authDB(t)
	reverseID := "acre_e2einv_" + uuid.New().String()[:12]
	_, err = db.Exec(`INSERT INTO account_relation (id, owner_account_id, counterparty_account_id, account_relation_role_code, external_number, priority_code, created_at, updated_at)
		VALUES (?, ?, ?, 'supplier', ?, 'normal', NOW(3), NOW(3))`, reverseID, customerID, SeedAccountID, uniqueName("E2E-REV"))
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM account_relation WHERE id = ?`, reverseID) })

	portalUser := loginAsUser(t, username, password, SeedAccountID)
	status, _, err = portalUser.GetListRaw(salesOrdersPath, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status, "the portal user reaches the seller, so every refusal below is the invoice gate")

	for name, client := range map[string]*Client{
		"portal user":        portalUser,
		"customer's API key": NewClient(envOr("E2E_BASE_URL", defaultBaseURL), SeedCustomerAPIKey, SeedAccountID),
	} {
		assertInvoiceAccessRefused(t, name, client, inv, http.StatusForbidden)
	}

	assert.Nil(t, getInvoice(t, inv.invoiceID)["note"], "the refused PATCH wrote nothing")
}

// assertInvoiceAccessRefused checks every invoice and receivable endpoint refuses client with want.
func assertInvoiceAccessRefused(t *testing.T, name string, client *Client, inv parityInvoice, want int) {
	t.Helper()
	reads := map[string]string{
		"list invoices":          invoicesPath,
		"retrieve invoice":       invoicesPath + "/" + inv.invoiceID,
		"list customer invoices": customerInvoicesPathFor(inv.customerID),
		"list receivables":       receivablesPath,
		"customer receivables":   receivablesPathFor(inv.customerID),
		"export receivables":     receivablesPathFor(inv.customerID) + "/actions/export",
	}
	for what, path := range reads {
		status, body, err := client.GetListRaw(path, nil)
		require.NoError(t, err)
		assert.Equal(t, want, status, "%s: %s: %s", name, what, string(body))
	}
	writes := map[string]func() (int, []byte, error){
		"update invoice": func() (int, []byte, error) {
			return client.Patch(invoicesPath+"/"+inv.invoiceID, map[string]any{"note": "not yours"}, newIdempotencyKey())
		},
		"email invoice": func() (int, []byte, error) {
			return client.Post(invoiceEmailRecordPath, map[string]any{"type": "invoice", "id": inv.invoiceID}, newIdempotencyKey())
		},
		"email statement": func() (int, []byte, error) {
			return client.Post(emailReceivablesPathFor(inv.customerID), map[string]any{"recipient_emails": []string{"ap@example.com"}}, newIdempotencyKey())
		},
	}
	for what, call := range writes {
		status, body, err := call()
		require.NoError(t, err)
		assert.Equal(t, want, status, "%s: %s: %s", name, what, string(body))
	}
}

// Another tenant reading in its own account finds none of the seller's invoices: a direct read is a
// 404, the lists leave them out, and the statement and invoice emails refuse them.
func TestInvoiceParity_OtherTenantFindsNoneOfTheSellersInvoices(t *testing.T) {
	t.Parallel()
	tenantB := getTenantBClient()

	for what, call := range map[string]func() (int, []byte, error){
		"retrieve": func() (int, []byte, error) { return tenantB.GetListRaw(invoicesPath+"/"+SeedInvoiceID, nil) },
		"update": func() (int, []byte, error) {
			return tenantB.Patch(invoicesPath+"/"+SeedInvoiceID, map[string]any{"note": "not yours"}, newIdempotencyKey())
		},
		"email invoice": func() (int, []byte, error) {
			return tenantB.Post(invoiceEmailRecordPath, map[string]any{"type": "invoice", "id": SeedInvoiceID}, newIdempotencyKey())
		},
		"export statement": func() (int, []byte, error) {
			return tenantB.GetListRaw(receivablesPathFor(SeedCustomerAccountID)+"/actions/export", nil)
		},
		"email statement": func() (int, []byte, error) {
			return tenantB.Post(emailReceivablesPathFor(SeedCustomerAccountID), map[string]any{"recipient_emails": []string{"ap@example.com"}}, newIdempotencyKey())
		},
	} {
		status, body, err := call()
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, status, "%s: %s", what, string(body))
	}

	for what, path := range map[string]string{
		"invoices":             invoicesPath,
		"customer invoices":    customerInvoicesPathFor(SeedCustomerAccountID),
		"receivables":          receivablesPath,
		"customer receivables": receivablesPathFor(SeedCustomerAccountID),
	} {
		assert.NotContains(t, invoiceListIDs(t, tenantB, path, nil), SeedInvoiceID, what)
	}
	assert.Empty(t, invoiceListIDs(t, tenantB, customerInvoicesPathFor(SeedCustomerAccountID), nil),
		"the seller's customer is not tenant B's, so it has nothing payable there")

	status, body, err := tenantB.WithAccountID(SeedAccountID).GetListRaw(invoicesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, status, "targeting the seller is refused outright: %s", string(body))
}

// --- Expandable fields ---

// The billing address and payment term arrive with the invoice, so a role that may read invoices
// expands them without the address or payment-term permissions, as a sales order's own do.
func TestInvoiceParity_InvoicesReadRoleExpandsBillingAddressAndPaymentTerm(t *testing.T) {
	t.Parallel()
	inv := parityInvoiceFor(t, parityCustomer(t, ""), nil)
	reader := customRoleClient(t, "invoices:read")

	status, body, err := reader.GetListRaw(invoicePaymentTermPath+"/"+SeedPaymentTermID, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, status, "the role cannot read payment terms on their own: %s", string(body))

	wantAddress := parseJSON(mustGet(t, addressesPath+"/"+inv.billToID))
	wantTerm := parseJSON(mustGet(t, invoicePaymentTermPath+"/"+SeedPaymentTermID))
	includes := url.Values{"include": {"billing_address", "payment_term"}}

	retrieved := parseJSON(mustGetAs(t, reader, invoicesPath+"/"+inv.invoiceID, includes))
	assert.Equal(t, wantAddress, jsonObject(retrieved, "billing_address"), "the invoice bills to its order's address")
	assert.Equal(t, wantTerm, jsonObject(retrieved, "payment_term"))

	listed := invoiceRow(t, reader, invoicesPath, withCustomers(includes, inv.customerID), inv.invoiceID)
	assert.Equal(t, wantAddress, jsonObject(listed, "billing_address"))
	assert.Equal(t, wantTerm, jsonObject(listed, "payment_term"))

	updated := parseJSON(mustPatchWithIncludes(t, invoicesPath+"/"+inv.invoiceID, includes, map[string]any{"has_been_sent": true}))
	assert.Equal(t, wantAddress, jsonObject(updated, "billing_address"), "a PATCH answers ?include= like a GET")
	assert.Equal(t, wantTerm, jsonObject(updated, "payment_term"))

	payable := invoiceRow(t, reader, customerInvoicesPathFor(inv.customerID), url.Values{"include": {"billing_address"}}, inv.invoiceID)
	assert.Equal(t, wantAddress, jsonObject(payable, "billing_address"))

	bare := parseJSON(mustGetAs(t, reader, invoicesPath+"/"+inv.invoiceID, nil))
	assertNilField(t, bare, "billing_address")
	assertNilField(t, bare, "payment_term")
	assertNilField(t, invoiceRow(t, reader, customerInvoicesPathFor(inv.customerID), nil, inv.invoiceID), "billing_address")
}

func mustGetAs(t *testing.T, client *Client, path string, params url.Values) []byte {
	t.Helper()
	status, body, err := client.GetListRaw(path, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return body
}

func mustPatchWithIncludes(t *testing.T, path string, params url.Values, body map[string]any) []byte {
	t.Helper()
	status, raw, err := apiClient.Patch(path+"?"+params.Encode(), body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	return raw
}

// Each invoice line's order line carries its number and the product description as sold.
func TestInvoiceParity_LinesCarryOrderLineNumberAndDescription(t *testing.T) {
	t.Parallel()
	first, second := parityLine(SeedProductID, "2", SeedUnitID, "3.00"), parityLine(SeedProductID, "3", SeedUnitID, "4.00")
	first["product_description"], second["product_description"] = uniqueName("e2e first line"), uniqueName("e2e second line")
	inv := parityInvoiceFor(t, parityCustomer(t, ""), nil, first, second)

	order := parseJSON(mustGetAs(t, apiClient, salesOrdersPath+"/"+inv.orderID, url.Values{"include": {"lines"}}))
	type sold struct{ number, sku, description string }
	soldLines := map[string]sold{}
	for _, raw := range jsonListData(order, "lines") {
		line := raw.(map[string]any)
		soldLines[jsonField(line, "id")] = sold{jsonField(line, "line_item_number"), jsonField(line, "product_sku"), jsonField(line, "product_description")}
	}

	invoice := parseJSON(mustGetAs(t, apiClient, invoicesPath+"/"+inv.invoiceID, url.Values{"include": {"lines", "lines.order_line"}}))
	lines := jsonListData(invoice, "lines")
	require.Len(t, lines, len(soldLines), "the two sold lines and the order's freight line")
	var descriptions []string
	for _, raw := range lines {
		orderLine := jsonObject(raw.(map[string]any), "order_line")
		require.NotNil(t, orderLine)
		want, ok := soldLines[jsonField(orderLine, "id")]
		require.True(t, ok, "the invoice line bills one of the order's lines")
		assert.Equal(t, want.number, jsonField(orderLine, "line_item_number"))
		assert.NotEqual(t, "0", jsonField(orderLine, "line_item_number"))
		assert.Equal(t, want.sku, jsonField(orderLine, "product_sku"))
		assert.NotEmpty(t, jsonField(orderLine, "product_sku"), "the SKU as sold, so the page needs no product include")
		assert.Equal(t, want.description, jsonField(orderLine, "product_description"))
		if d := jsonField(orderLine, "product_description"); d != "" {
			descriptions = append(descriptions, d)
		}
	}
	assert.ElementsMatch(t, []string{first["product_description"].(string), second["product_description"].(string)}, descriptions)
}

// An allocation names the settlement that recorded it, on the invoice and on the settle list.
func TestInvoiceParity_AllocationsNameTheirSettlement(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	funds := time.Now().UTC()
	payment := createPayment(t, inv.customerID, "10.00", &funds, nil)
	settlement := settle(t, map[string]any{"allocations": []any{allocation(jsonField(payment, "id"), inv.invoiceID, "10.00")}})
	want := map[string]any{"id": jsonField(settlement, "id"), "object": "settlement", "number": jsonField(settlement, "number")}

	bare := getInvoice(t, inv.invoiceID)
	assertNilField(t, bare, "allocations")

	invoice := parseJSON(mustGetAs(t, apiClient, invoicesPath+"/"+inv.invoiceID, url.Values{"include": {"allocations"}}))
	allocations := jsonListData(invoice, "allocations")
	require.Len(t, allocations, 1)
	assert.Equal(t, want, jsonObject(allocations[0].(map[string]any), "settlement"))

	payable := invoiceRow(t, apiClient, customerInvoicesPathFor(inv.customerID), url.Values{"include": {"allocations"}}, inv.invoiceID)
	allocations = jsonListData(payable, "allocations")
	require.Len(t, allocations, 1)
	assert.Equal(t, want, jsonObject(allocations[0].(map[string]any), "settlement"))
}

// The transactions an invoice's allocations draw on are part of its ledger, so a role that may read
// invoices expands them, and their amounts' currency, without transactions:read or units:read — on
// the invoice, on a PATCH, and on the settle list.
func TestInvoiceParity_InvoicesReadRoleExpandsAllocationTransactions(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	funds := time.Now().UTC()
	payment := createPayment(t, inv.customerID, "10.00", &funds, nil)
	settle(t, map[string]any{"allocations": []any{allocation(jsonField(payment, "id"), inv.invoiceID, "10.00")}})
	// Settling marks the payment fully allocated afterwards; the two reads below must see the same payment.
	awaitFullyAllocated(t, jsonField(payment, "id"), true)
	reader := customRoleClient(t, "invoices:read")

	status, body, err := reader.GetListRaw(transactionsPath+"/"+jsonField(payment, "id"), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, status, "the role cannot read the transaction on its own: %s", string(body))

	includes := url.Values{"include": {
		"allocations", "allocations.amount", "allocations.amount.unit",
		"allocations.transaction", "allocations.transaction.amount", "allocations.transaction.amount.unit",
	}}
	assertDrawsOnThePayment := func(name string, allocations []any) {
		t.Helper()
		require.Len(t, allocations, 1, name)
		transaction := jsonObject(allocations[0].(map[string]any), "transaction")
		require.NotNil(t, transaction, "%s: the allocation's transaction is expanded", name)
		assert.Equal(t, jsonField(payment, "id"), jsonField(transaction, "id"), name)
		assert.Equal(t, jsonField(payment, "number"), jsonField(transaction, "number"), name)
		amount := jsonObject(transaction, "amount")
		require.NotNil(t, amount, name)
		assert.NotNil(t, jsonObject(amount, "unit"), "%s: the transaction amount's currency is expanded", name)
	}

	got := parseJSON(mustGetAs(t, reader, invoicesPath+"/"+inv.invoiceID, includes))
	want := parseJSON(mustGetAs(t, apiClient, invoicesPath+"/"+inv.invoiceID, includes))
	assert.Equal(t, jsonListData(want, "allocations"), jsonListData(got, "allocations"), "the role reads the allocations exactly as an admin does")
	assertDrawsOnThePayment("retrieve", jsonListData(got, "allocations"))

	updated := parseJSON(mustPatchWithIncludes(t, invoicesPath+"/"+inv.invoiceID, includes, map[string]any{"has_been_sent": true}))
	assertDrawsOnThePayment("update", jsonListData(updated, "allocations"))

	payable := invoiceRow(t, reader, customerInvoicesPathFor(inv.customerID), includes, inv.invoiceID)
	assertDrawsOnThePayment("settle list", jsonListData(payable, "allocations"))

	bare := parseJSON(mustGetAs(t, reader, invoicesPath+"/"+inv.invoiceID, url.Values{"include": {"allocations"}}))
	allocations := jsonListData(bare, "allocations")
	require.Len(t, allocations, 1)
	assertNilField(t, allocations[0].(map[string]any), "transaction")
}

// --- Payment state ---

// An overpaid invoice is paid in full, which payment_status alone cannot show once the mark is cleared.
func TestInvoiceParity_IsPaidInFullRidesBesideOverpaid(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	fresh := getInvoice(t, inv.invoiceID)
	assert.Equal(t, "false", jsonField(fresh, "is_paid_in_full"))
	assert.Equal(t, "unpaid", jsonField(fresh, "payment_status"))
	assert.Contains(t, invoiceListIDs(t, apiClient, customerInvoicesPathFor(inv.customerID), nil), inv.invoiceID)

	funds := time.Now().UTC()
	over := inv.total.Add(decimal.NewFromInt(5)).StringFixed(2)
	payment := createPayment(t, inv.customerID, over, &funds, nil)
	settle(t, map[string]any{"allocations": []any{allocation(jsonField(payment, "id"), inv.invoiceID, over)}})
	awaitInvoiceStatus(t, inv.invoiceID, "overpaid")
	assert.Equal(t, "true", jsonField(getInvoice(t, inv.invoiceID), "is_paid_in_full"))

	byStatus := func(status string) []string {
		return invoiceListIDs(t, apiClient, invoicesPath, withCustomers(url.Values{"status": {status}}, inv.customerID))
	}
	assert.Equal(t, []string{inv.invoiceID}, byStatus("overpaid"))
	assert.Equal(t, []string{inv.invoiceID}, byStatus("paid"), "an overpaid invoice is paid in full")
	assert.Empty(t, byStatus("unpaid"))
	assert.NotContains(t, invoiceListIDs(t, apiClient, customerInvoicesPathFor(inv.customerID), nil), inv.invoiceID,
		"an overpaid invoice owes nothing, so it is not offered for payment")

	cleared := patchInvoice(t, inv.invoiceID, map[string]any{"is_paid_in_full": false})
	assert.Equal(t, "false", jsonField(cleared, "is_paid_in_full"))
	assert.Equal(t, "overpaid", jsonField(cleared, "payment_status"), "the overpayment is still recorded")
	assert.Equal(t, []string{inv.invoiceID}, byStatus("unpaid"))
	assert.Contains(t, invoiceListIDs(t, apiClient, customerInvoicesPathFor(inv.customerID), nil), inv.invoiceID)
}

// --- Update ---

func TestInvoiceParity_UpdateEachField(t *testing.T) {
	t.Parallel()
	inv := parityInvoiceFor(t, parityCustomer(t, ""), nil)
	path := invoicesPath + "/" + inv.invoiceID
	original := getInvoice(t, inv.invoiceID)
	require.Nil(t, original["note"])

	preserved := func(got map[string]any, except ...string) {
		t.Helper()
		for _, key := range []string{"number", "total_invoiced", "line_count", "priority", "created_at"} {
			assert.Equal(t, original[key], got[key], key)
		}
		for _, key := range []string{"note", "has_been_sent", "is_paid_in_full"} {
			if !slices.Contains(except, key) {
				assert.Equal(t, getInvoice(t, inv.invoiceID)[key], got[key], key)
			}
		}
	}

	note := uniqueName("e2e invoice note")
	got := patchInvoice(t, inv.invoiceID, map[string]any{"note": note})
	assert.Equal(t, note, jsonField(got, "note"))
	assert.Equal(t, "false", jsonField(got, "has_been_sent"))
	preserved(got, "note")

	got = patchInvoice(t, inv.invoiceID, map[string]any{"has_been_sent": true})
	assert.Equal(t, "true", jsonField(got, "has_been_sent"))
	assert.Equal(t, note, jsonField(got, "note"), "omitting the note keeps it")
	preserved(got)

	got = patchInvoice(t, inv.invoiceID, map[string]any{"is_paid_in_full": true})
	assert.Equal(t, "true", jsonField(got, "is_paid_in_full"))
	assert.Equal(t, "paid", jsonField(got, "payment_status"))
	preserved(got)

	got = patchInvoice(t, inv.invoiceID, map[string]any{"note": nil})
	assert.Nil(t, got["note"], "null clears the note")
	assert.Equal(t, "true", jsonField(got, "is_paid_in_full"))

	got = patchInvoice(t, inv.invoiceID, map[string]any{"is_paid_in_full": false, "has_been_sent": false})
	assert.Equal(t, "unpaid", jsonField(got, "payment_status"))
	assert.Equal(t, "false", jsonField(got, "is_paid_in_full"))
	assert.Equal(t, "false", jsonField(got, "has_been_sent"))
	assert.Nil(t, got["note"], "a cleared note stays cleared")

	status, body, err := apiClient.Patch(path, map[string]any{bogusE2EJSONField: true}, newIdempotencyKey())
	require.NoError(t, err)
	assertJSONUnknownFieldRejected(t, http.MethodPatch, path, status, body)

	status, body, err = apiClient.Patch(path, map[string]any{"is_paid_in_full": nil}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "a flag cannot be null: %s", string(body))
}

// --- List ---

// Every filter the invoices page sends, each with an invoice it must keep and one it must drop.
func TestInvoiceParity_ListFilters(t *testing.T) {
	t.Parallel()
	groupA := leadTimeAccountGroup(t, "e2e-inv-grp-a", nil)
	groupB := leadTimeAccountGroup(t, "e2e-inv-grp-b", nil)
	lineB := parityProductLine(t)
	productB, itemB := parityProduct(t, lineB)
	_, repB := paritySalesRep(t)

	invA := parityInvoiceFor(t, parityCustomer(t, groupA), map[string]any{"sales_rep_id": SeedAccountUserID})
	invB := parityInvoiceFor(t, parityCustomer(t, groupB, lineB), map[string]any{"sales_rep_id": repB},
		parityLine(productB, "2", SeedUnitID, "5.00"))
	both := []string{invA.customerID, invB.customerID}
	list := func(params url.Values) []string {
		t.Helper()
		return invoiceListIDs(t, apiClient, invoicesPath, withCustomers(params, both...))
	}
	require.ElementsMatch(t, []string{invA.invoiceID, invB.invoiceID}, list(nil))

	assert.Equal(t, []string{invA.invoiceID}, invoiceListIDs(t, apiClient, invoicesPath, url.Values{"customer_ids": {invA.customerID}}))
	assert.Equal(t, []string{invA.invoiceID}, list(url.Values{"customer_group_ids": {groupA}}))
	assert.Equal(t, []string{invB.invoiceID}, list(url.Values{"customer_group_ids": {groupB}}))
	// Sales reps are named by account user, as on the order.
	assert.Equal(t, []string{invA.invoiceID}, list(url.Values{"sales_rep_ids": {SeedAccountUserID}}))
	assert.Equal(t, []string{invB.invoiceID}, list(url.Values{"sales_rep_ids": {repB}}))
	assert.Equal(t, []string{invB.invoiceID}, list(url.Values{"item_ids": {itemB}}))
	assert.Equal(t, []string{invA.invoiceID}, list(url.Values{"item_ids": {SeedItemID}}))
	assert.Equal(t, []string{invB.invoiceID}, list(url.Values{"product_line_ids": {lineB}}))
	assert.Equal(t, []string{invA.invoiceID}, list(url.Values{"product_line_ids": {SeedProductLineID}}))

	assert.Contains(t, invoiceListIDs(t, apiClient, salesOrdersPath, url.Values{"sales_rep_ids": {repB}, "customer_ids": {invB.customerID}}), invB.orderID,
		"sales orders filter by the same account-user ids")

	patchInvoice(t, invA.invoiceID, map[string]any{"is_paid_in_full": true})
	assert.Equal(t, []string{invA.invoiceID}, list(url.Values{"status": {"paid"}}))
	assert.Equal(t, []string{invB.invoiceID}, list(url.Values{"status": {"unpaid"}}))
	assert.Empty(t, list(url.Values{"status": {"overpaid"}}))
	assert.ElementsMatch(t, []string{invA.invoiceID, invB.invoiceID}, list(url.Values{"status": {"all"}}))

	today := time.Now().UTC().Format("2006-01-02")
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	assert.Len(t, list(url.Values{"starts_at": {today}, "ends_at": {today}}), 2, "an end date covers its whole day")
	assert.Empty(t, list(url.Values{"ends_at": {yesterday}}))
	assert.Empty(t, list(url.Values{"starts_at": {tomorrow}}))
}

// A date window that does not parse is refused on the parameter rather than read as no filter; the
// sales-order list holds its dates to the same rule.
func TestInvoiceParity_ListDatesMustParse(t *testing.T) {
	t.Parallel()
	for path, params := range map[string][]string{
		invoicesPath:    {"starts_at", "ends_at"},
		salesOrdersPath: {"starts_at", "ends_at", "ship_by_after", "ship_by_before"},
	} {
		for _, param := range params {
			for _, bad := range []string{"not-a-date", "2026-13-01", "10/05/2026"} {
				status, body, err := apiClient.GetListRaw(path, url.Values{param: {bad}})
				require.NoError(t, err)
				requireStatus(t, 400, status, body)
				assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), param)
			}
			for _, good := range []string{"2026-10-05", "2026-10-05T04:00:00Z", "2026-10-05T00:00:00-04:00"} {
				status, body, err := apiClient.GetListRaw(path, url.Values{param: {good}, "limit": {"1"}})
				require.NoError(t, err)
				requireStatus(t, 200, status, body)
			}
		}
	}
}

// A contains search reaches anywhere in the invoice, its order, and the customer relation's number, alias
// and notes.
func TestInvoiceParity_ListSearch(t *testing.T) {
	t.Parallel()
	customerA, customerB := parityCustomer(t, ""), parityCustomer(t, "")
	po := uniqueName("E2E-PO")
	invA := parityInvoiceFor(t, customerA, map[string]any{"customer_purchase_order_number": po})
	invB := parityInvoiceFor(t, customerB, nil)

	alias, notes := uniqueName("e2e-alias"), uniqueName("e2e-relation-notes")
	db := authDB(t)
	_, err := db.Exec(`UPDATE account_relation SET alias = ?, notes = ? WHERE owner_account_id = ? AND counterparty_account_id = ? AND account_relation_role_code = 'customer'`,
		alias, notes, SeedAccountID, customerA)
	require.NoError(t, err)
	invoiceNote := uniqueName("e2e-invoice-note")
	patchInvoice(t, invA.invoiceID, map[string]any{"note": invoiceNote})

	search := func(q string) []string {
		t.Helper()
		return invoiceListIDs(t, apiClient, invoicesPath, withCustomers(url.Values{"q": {q}, "q_match": {"contains"}}, customerA, customerB))
	}
	for what, q := range map[string]string{
		"customer alias":  alias,
		"customer notes":  notes,
		"invoice note":    invoiceNote,
		"customer PO":     po,
		"order number":    invA.orderNumber,
		"invoice number":  invA.number,
		"alias substring": alias[4:],
	} {
		got := search(q)
		assert.Contains(t, got, invA.invoiceID, "%s %q", what, q)
		if !bytes.Contains([]byte(invB.number+invB.orderNumber), []byte(q)) {
			assert.NotContains(t, got, invB.invoiceID, "%s %q", what, q)
		}
	}
	assert.Empty(t, search(uniqueName("e2e-no-such-invoice")))
}

// A search matches the start of the invoice number, the order number, the customer PO, and the customer's
// name, number and alias. It does not reach into the middle of them, nor into any notes, and a wildcard in
// the term is a literal character.
func TestInvoiceParity_ListSearchPrefix(t *testing.T) {
	t.Parallel()
	customerA, customerB := parityCustomer(t, ""), parityCustomer(t, "")
	po := uniqueName("E2E-PO")
	invA := parityInvoiceFor(t, customerA, map[string]any{"customer_purchase_order_number": po})
	invB := parityInvoiceFor(t, customerB, nil)

	alias, notes, number := uniqueName("e2e-alias"), uniqueName("e2e-relation-notes"), uniqueName("E2E-CUSTNO")
	db := authDB(t)
	_, err := db.Exec(`UPDATE account_relation SET alias = ?, notes = ?, external_number = ? WHERE owner_account_id = ? AND counterparty_account_id = ? AND account_relation_role_code = 'customer'`,
		alias, notes, number, SeedAccountID, customerA)
	require.NoError(t, err)
	invoiceNote := uniqueName("e2e-invoice-note")
	patchInvoice(t, invA.invoiceID, map[string]any{"note": invoiceNote})
	name := getCustomerName(t, customerA)

	search := func(q string) []string {
		t.Helper()
		return invoiceListIDs(t, apiClient, invoicesPath, withCustomers(url.Values{"q": {q}}, customerA, customerB))
	}
	for what, q := range map[string]string{
		"invoice number":         invA.number,
		"order number":           invA.orderNumber,
		"customer PO":            po,
		"customer PO start":      po[:len(po)-3],
		"customer name":          name,
		"customer name start":    name[:len(name)-3],
		"customer name any case": strings.ToUpper(name),
		"customer number":        number,
		"customer number start":  number[:len(number)-3],
		"customer alias":         alias,
		"customer alias start":   alias[:len(alias)-3],
	} {
		got := search(q)
		assert.Contains(t, got, invA.invoiceID, "%s %q", what, q)
		if !strings.HasPrefix(invB.number, q) && !strings.HasPrefix(invB.orderNumber, q) {
			assert.NotContains(t, got, invB.invoiceID, "%s %q", what, q)
		}
	}
	for what, q := range map[string]string{
		"invoice note":        invoiceNote,
		"customer notes":      notes,
		"middle of the PO":    po[4:],
		"middle of the name":  name[4:],
		"middle of the alias": alias[4:],
		"underscore wildcard": "_" + alias[1:],
		"percent wildcard":    "%" + alias[1:],
		"backslash":           `\` + alias,
		"nothing":             uniqueName("e2e-no-such-invoice"),
	} {
		assert.NotContains(t, search(q), invA.invoiceID, "%s %q", what, q)
	}
}

// q_match is checked against the values it takes.
func TestInvoiceParity_ListSearchMatchIsValidated(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(invoicesPath, url.Values{"q": {"INV"}, "q_match": {"fuzzy"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), "q_match")
}

// --- Customer invoices (settle) ---

// The settle list holds a customer's and its children's unpaid invoices and searches their orders.
func TestInvoiceParity_CustomerInvoicesRollUpChildrenAndSearchOrders(t *testing.T) {
	t.Parallel()
	parent, child := parityCustomer(t, ""), parityCustomer(t, "")
	makeChildCustomer(t, parent, child)
	po := uniqueName("E2E-SETTLE-PO")
	own := parityInvoiceFor(t, parent, nil)
	paid := parityInvoiceFor(t, parent, nil)
	childs := parityInvoiceFor(t, child, map[string]any{"customer_purchase_order_number": po})
	patchInvoice(t, paid.invoiceID, map[string]any{"is_paid_in_full": true})

	path := customerInvoicesPathFor(parent)
	assert.ElementsMatch(t, []string{own.invoiceID, childs.invoiceID}, invoiceListIDs(t, apiClient, path, nil),
		"unpaid invoices of the customer and its child, not the paid one")
	assert.Equal(t, []string{childs.invoiceID}, invoiceListIDs(t, apiClient, path, url.Values{"q": {childs.orderNumber}}), "by order number")
	assert.Equal(t, []string{childs.invoiceID}, invoiceListIDs(t, apiClient, path, url.Values{"q": {po}}), "by customer PO")
	assert.Contains(t, invoiceListIDs(t, apiClient, path, url.Values{"q": {own.number}}), own.invoiceID, "by invoice number")
	assert.Empty(t, invoiceListIDs(t, apiClient, path, url.Values{"q": {uniqueName("e2e-no-such")}}))

	row := invoiceRow(t, apiClient, path, url.Values{"include": {"billing_address", "parent_account"}}, childs.invoiceID)
	assert.Equal(t, childs.billToID, jsonField(jsonObject(row, "billing_address"), "id"), "the order's billing address")
	assert.Equal(t, parent, jsonField(jsonObject(row, "parent_account"), "id"))
	assert.Equal(t, "false", jsonField(row, "is_paid_in_full"))
}

// --- Receivables ---

// A customer's receivables page both ways and drop what was invoiced after the cutoff.
func TestInvoiceParity_CustomerReceivablesPageAndCutOff(t *testing.T) {
	t.Parallel()
	customerID := parityCustomer(t, "")
	var want []string
	for range 3 {
		want = append(want, parityInvoiceFor(t, customerID, nil).invoiceID)
	}
	path := receivablesPathFor(customerID)

	status, body, err := apiClient.GetListRaw(path, url.Values{"limit": {"2"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	first := parseJSON(body)
	require.Len(t, jsonArray(first, "data"), 2)
	assert.ElementsMatch(t, want, invoiceListIDs(t, apiClient, path, url.Values{"limit": {"2"}}), "pages cover every receivable once")

	tomorrow := time.Now().UTC().AddDate(0, 0, 1).Format(time.RFC3339)
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format(time.RFC3339)
	assert.ElementsMatch(t, want, invoiceListIDs(t, apiClient, path, url.Values{"cutoff_at": {tomorrow}}))
	assert.Empty(t, invoiceListIDs(t, apiClient, path, url.Values{"cutoff_at": {yesterday}}), "nothing was invoiced before the cutoff")

	all := invoiceListIDs(t, apiClient, receivablesPath, url.Values{"limit": {"2"}, "q": {getCustomerName(t, customerID)}})
	assert.ElementsMatch(t, want, all, "the account-wide list finds the customer's by name, paging two at a time")
}

// The statement of account goes only to one of the account's customers; it lists received credits
// dated by when their funds arrived, and prints numbers as the dashboard pads them.
func TestInvoiceParity_StatementOfAccountEmail(t *testing.T) {
	t.Parallel()
	for _, stranger := range []string{SeedTenantBAccountID, "ac_01nosuchaccount0000"} {
		status, body, err := apiClient.Post(emailReceivablesPathFor(stranger), map[string]any{"recipient_emails": []string{"ap@example.com"}}, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, status, "%s is not a customer: %s", stranger, string(body))
	}

	inv := parityInvoiceFor(t, parityCustomer(t, ""), nil)
	received := time.Now().UTC().AddDate(0, 0, -45)
	credit := createPayment(t, inv.customerID, "7.50", &received, nil)
	pending := createPayment(t, inv.customerID, "3.25", nil, nil)

	recipient := "e2e-soa-" + uuid.New().String()[:12] + "@example.com"
	status, body, err := apiClient.Post(emailReceivablesPathFor(inv.customerID), map[string]any{"recipient_emails": []string{recipient}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, body)

	email := queuedSendEmail(t, recipient)
	attachment, ok := email["attachment_data"].(string)
	require.True(t, ok, "the statement is attached: %v", email["attachment_filename"])
	raw, err := base64.StdEncoding.DecodeString(attachment)
	require.NoError(t, err)
	book, err := excelize.OpenReader(bytes.NewReader(raw))
	require.NoError(t, err)
	defer func() { _ = book.Close() }()
	rows, err := book.GetRows("Statement of Account")
	require.NoError(t, err)

	labels := map[string][]string{}
	for _, row := range rows[1:] {
		if len(row) > 0 && row[0] != "" {
			labels[row[0]] = row
		}
	}
	assert.Contains(t, labels, paddedNumber(inv.number), "invoice numbers are zero-padded: %v", rows)
	creditRow, ok := labels["Credit: "+paddedNumber(jsonField(credit, "number"))]
	if assert.True(t, ok, "the received credit is listed, padded: %v", rows) {
		assert.Equal(t, received.Format("1/2/2006"), creditRow[2], "dated by when its funds arrived")
	}
	assert.NotContains(t, labels, "Credit: "+paddedNumber(jsonField(pending, "number")), "a credit not yet received is left off")
}
