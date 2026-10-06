//go:build e2e

package api_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The dashboard's invoice list and detail, the settle flow, the receivables reports, and the
// invoice and statement emails. Each test raises its own invoices for a customer of its own, so
// what it reads and sends is exact however many rows other tests leave behind.

var (
	dashInvoicesListIncludes   = []string{"customer", "billing_address", "payment_term", "related.sales_order", "related.shipment"}
	dashInvoicesDetailIncludes = append(append([]string{}, dashInvoicesListIncludes...),
		"lines", "lines.order_line", "lines.quantity", "lines.quantity.unit",
		"lines.unit_price", "lines.unit_price.numerator_unit", "lines.unit_price.denominator_unit",
		"allocations", "allocations.amount", "allocations.amount.unit", "allocations.transaction")
	dashInvoicesSettleIncludes = []string{"customer", "parent_account", "billing_address",
		"allocations", "allocations.amount", "allocations.amount.unit", "allocations.transaction"}
)

func dashInvoicesTransactionsPathFor(customerID string) string {
	return "/v1/finance/accounts/" + customerID + "/transactions"
}

func dashInvoicesAddress(token string) string {
	return "e2e-inv-" + token + "@example.com"
}

// dashInvoicesSoldLine is one seed sock line whose description marks every email that prints it.
func dashInvoicesSoldLine(description string) map[string]any {
	line := parityLine(SeedProductID, "4", SeedUnitID, "3.00")
	line["product_description"] = description
	return line
}

// dashInvoicesContact adds a user with an email address to the customer's account, as an invoice recipient must be.
func dashInvoicesContact(t *testing.T, customerID string) (accountUserID, email string) {
	t.Helper()
	email = dashInvoicesAddress(strings.ReplaceAll(uuid.New().String()[:13], "-", ""))
	merchant := apiClient.WithAccountID(customerID)
	status, body, err := merchant.Post(accountUsersPath, map[string]any{"email": email, "name": "E2E invoice contact"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	accountUserID = jsonField(parseJSON(body), "id")
	require.NotEmpty(t, accountUserID)
	t.Cleanup(func() { _, _, _ = merchant.Put(accountUsersPath+"/"+accountUserID+"/actions/remove", nil) })
	return accountUserID, email
}

// dashInvoicesQueuedEmails is every send-email command in the outbox whose payload contains marker.
func dashInvoicesQueuedEmails(t *testing.T, marker string) []map[string]any {
	t.Helper()
	rows, err := authDB(t).Query(`
		SELECT CAST(FROM_BASE64(JSON_UNQUOTE(JSON_EXTRACT(payload, '$.data'))) AS CHAR)
		FROM message_outbox
		WHERE routing_key = 'notification.cmd.send_email'
		AND CAST(FROM_BASE64(JSON_UNQUOTE(JSON_EXTRACT(payload, '$.data'))) AS CHAR) LIKE ?`,
		"%"+marker+"%")
	require.NoError(t, err)
	defer rows.Close()
	var emails []map[string]any
	for rows.Next() {
		var data string
		require.NoError(t, rows.Scan(&data))
		emails = append(emails, parseJSON([]byte(data)))
	}
	require.NoError(t, rows.Err())
	return emails
}

// dashInvoicesAuditedUpdates counts the update audit events the outbox holds for an invoice; they are written in the PATCH's own transaction.
func dashInvoicesAuditedUpdates(t *testing.T, invoiceID string) int {
	t.Helper()
	var n int
	require.NoError(t, authDB(t).QueryRow(`
		SELECT COUNT(*) FROM message_outbox
		WHERE routing_key = 'platform.event.audit_logged'
		AND CAST(FROM_BASE64(JSON_UNQUOTE(JSON_EXTRACT(payload, '$.data'))) AS CHAR) LIKE ?`,
		`%"action":"update","resource_type":"invoice","resource_id":"`+invoiceID+`"%`).Scan(&n))
	return n
}

func dashInvoicesWithSubject(emails []map[string]any, subject string) []map[string]any {
	var out []map[string]any
	for _, e := range emails {
		if jsonField(e, "subject") == subject {
			out = append(out, e)
		}
	}
	return out
}

// dashInvoicesAwaitEmailLog waits for the notification service to log a send to recipient under subject.
func dashInvoicesAwaitEmailLog(t *testing.T, recipient, subject string) map[string]any {
	t.Helper()
	var found map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		status, body, err := apiClient.GetListRaw(emailLogsPath, url.Values{"q": {recipient}, "include": {"sent_by"}, "limit": {"100"}})
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("email logs: %d %s", status, string(body))
		}
		for _, raw := range jsonArray(parseJSON(body), "data") {
			if row := raw.(map[string]any); jsonField(row, "subject") == subject {
				found = row
				return nil
			}
		}
		return fmt.Errorf("no email log to %s with subject %q yet", recipient, subject)
	})
	return found
}

func dashInvoicesRetrieve(t *testing.T, client *Client, invoiceID string, includes ...string) map[string]any {
	t.Helper()
	var params url.Values
	if len(includes) > 0 {
		params = url.Values{"include": includes}
	}
	return parseJSON(mustGetAs(t, client, invoicesPath+"/"+invoiceID, params))
}

func dashInvoicesEmailRecord(t *testing.T, client *Client, body map[string]any, key string) (int, []byte) {
	t.Helper()
	status, raw, err := client.Post(invoiceEmailRecordPath, body, key)
	require.NoError(t, err)
	return status, raw
}

func dashInvoicesRequireBadRequest(t *testing.T, status int, body []byte, code, param string) {
	t.Helper()
	requireStatus(t, http.StatusBadRequest, status, body)
	errObj := requireErrorResponse(t, body, code, "invalid_request_error")
	if param != "" {
		assertErrorParam(t, errObj, param)
	}
}

// dashInvoicesTransaction records a transaction of typ from the customer; funds is when its money arrived (nil: not yet).
func dashInvoicesTransaction(t *testing.T, customerID, typ, amount string, funds *time.Time) map[string]any {
	t.Helper()
	body := map[string]any{"customer_id": customerID, "type": typ, "amount": amount, "responsible_user_id": SeedAccountUserID}
	if typ == "payment" {
		body["method"] = "check"
	}
	if funds != nil {
		body["funds_received_at"] = rfc3339(*funds)
	}
	status, resp, err := apiClient.Post(financeTransactionsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	tx := parseJSON(resp)
	id := jsonField(tx, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(financeTransactionsPath + "/" + id) })
	return tx
}

// --- List & retrieve ---

// The invoices page and the invoice detail page each read one request's worth of includes; every
// one resolves to the invoice's own customer, order, shipment and billing, and none appears unasked.
func TestDashInvoices_ListAndDetailExpandWhatTheDashboardRequests(t *testing.T) {
	t.Parallel()
	customerID := parityCustomer(t, "")
	po := uniqueName("E2E-DASHINV-PO")
	inv := parityInvoiceFor(t, customerID, map[string]any{"customer_purchase_order_number": po})

	assertRelated := func(name string, got map[string]any) {
		t.Helper()
		customer := jsonObject(got, "customer")
		require.NotNil(t, customer, "%s: customer", name)
		assert.Equal(t, customerID, jsonField(customer, "id"), name)
		assert.Equal(t, "customer", jsonField(customer, "object"), name)
		assert.Equal(t, getCustomerName(t, customerID), jsonField(customer, "name"), name)

		assert.Equal(t, inv.billToID, jsonField(jsonObject(got, "billing_address"), "id"), "%s: the order's billing address", name)
		assert.Equal(t, SeedPaymentTermID, jsonField(jsonObject(got, "payment_term"), "id"), "%s: payment term", name)

		related := jsonObject(got, "related")
		require.NotNil(t, related, "%s: related", name)
		assert.Equal(t, "invoice_related", jsonField(related, "object"), name)
		order := jsonObject(related, "sales_order")
		require.NotNil(t, order, "%s: related.sales_order", name)
		assert.Equal(t, inv.orderID, jsonField(order, "id"), name)
		assert.Equal(t, "sales_order", jsonField(order, "type"), name)
		assert.Equal(t, inv.orderNumber, jsonField(order, "number"), name)
		shipment := jsonObject(related, "shipment")
		require.NotNil(t, shipment, "%s: related.shipment", name)
		assert.True(t, strings.HasPrefix(jsonField(shipment, "id"), "sh_"), "%s: %v", name, shipment)
		assert.Equal(t, "shipment", jsonField(shipment, "type"), name)
		assert.Equal(t, "shipped", jsonField(shipment, "status"), name)
	}

	listed := invoiceRow(t, apiClient, invoicesPath, withCustomers(url.Values{"include": dashInvoicesListIncludes, "limit": {"10"}}, customerID), inv.invoiceID)
	assertRelated("list", listed)
	assertNilField(t, listed, "lines")
	assertNilField(t, listed, "allocations")

	detail := dashInvoicesRetrieve(t, apiClient, inv.invoiceID, dashInvoicesDetailIncludes...)
	assertRelated("detail", detail)
	assert.Equal(t, jsonField(jsonObject(jsonObject(listed, "related"), "shipment"), "id"),
		jsonField(jsonObject(jsonObject(detail, "related"), "shipment"), "id"), "list and detail name the same shipment")

	assert.Equal(t, "invoice", jsonField(detail, "object"))
	assert.Equal(t, inv.number, jsonField(detail, "number"))
	assert.Nil(t, detail["note"])
	assert.Equal(t, "normal", jsonField(detail, "priority"))
	assert.Equal(t, "unpaid", jsonField(detail, "payment_status"))
	for _, flag := range []string{"is_paid_in_full", "has_been_sent", "is_edi_sent", "accepts_invoice_emails", "customer_is_edi_enabled"} {
		assert.Equal(t, "false", jsonField(detail, flag), flag)
	}
	assert.True(t, decimal.RequireFromString(jsonField(detail, "total_invoiced")).IsPositive())
	assertValidTimestamp(t, jsonField(detail, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(detail, "updated_at"), "updated_at")

	lines := jsonListData(detail, "lines")
	require.NotEmpty(t, lines)
	assert.Equal(t, fmt.Sprint(len(lines)), jsonField(detail, "line_count"))
	for i, raw := range lines {
		line := raw.(map[string]any)
		assert.Equal(t, "invoice_line", jsonField(line, "object"), "lines[%d]", i)
		assert.NotNil(t, jsonObject(line, "order_line"), "lines[%d].order_line", i)
		assert.NotEmpty(t, jsonField(jsonObject(jsonObject(line, "quantity"), "unit"), "id"), "lines[%d].quantity.unit", i)
		price := jsonObject(line, "unit_price")
		assert.NotEmpty(t, jsonField(jsonObject(price, "numerator_unit"), "id"), "lines[%d].unit_price.numerator_unit", i)
		assert.NotEmpty(t, jsonField(jsonObject(price, "denominator_unit"), "id"), "lines[%d].unit_price.denominator_unit", i)
	}
	allocations := jsonObject(detail, "allocations")
	require.NotNil(t, allocations, "an unpaid invoice expands to an empty allocation list, not null")
	assert.Empty(t, jsonArray(allocations, "data"))

	bare := dashInvoicesRetrieve(t, apiClient, inv.invoiceID)
	for _, key := range []string{"customer", "order", "shipment", "billing_address", "payment_term", "related", "lines", "allocations"} {
		assertNilField(t, bare, key)
	}
}

// The dashboard sends its date window as JavaScript ISO timestamps (milliseconds, UTC), which bound
// the list exactly; a status or cutoff it does not know is refused rather than ignored.
func TestDashInvoices_ListTakesTheDashboardsTimestampsAndRefusesBadFilters(t *testing.T) {
	t.Parallel()
	customerID := parityCustomer(t, "")
	inv := parityInvoiceFor(t, customerID, nil)
	created, err := time.Parse(time.RFC3339, jsonField(getInvoice(t, inv.invoiceID), "created_at"))
	require.NoError(t, err)
	iso := func(at time.Time) string { return at.UTC().Format("2006-01-02T15:04:05.000Z") }
	window := func(from, to time.Time) []string {
		t.Helper()
		return invoiceListIDs(t, apiClient, invoicesPath, withCustomers(url.Values{"starts_at": {iso(from)}, "ends_at": {iso(to)}}, customerID))
	}

	assert.Equal(t, []string{inv.invoiceID}, window(created.Add(-time.Second), created.Add(time.Second)))
	assert.Empty(t, window(created.Add(-time.Hour), created.Add(-time.Second)), "a timestamp end is the exact bound, not the end of its day")
	assert.Empty(t, window(created.Add(time.Second), created.Add(time.Hour)))

	status, body, err := apiClient.GetListRaw(invoicesPath, url.Values{"status": {"settled"}})
	require.NoError(t, err)
	dashInvoicesRequireBadRequest(t, status, body, "parameter_invalid", "status")

	cutoff := url.Values{"cutoff_at": {iso(time.Now().Add(time.Hour))}}
	assert.Equal(t, []string{inv.invoiceID}, invoiceListIDs(t, apiClient, receivablesPathFor(customerID), cutoff))
	cutoff.Set("q", inv.number)
	assert.Contains(t, invoiceListIDs(t, apiClient, receivablesPath, cutoff), inv.invoiceID)
	for _, path := range []string{receivablesPath, receivablesPathFor(customerID)} {
		status, body, err := apiClient.GetListRaw(path, url.Values{"cutoff_at": {"not-a-date"}})
		require.NoError(t, err)
		dashInvoicesRequireBadRequest(t, status, body, "parameter_invalid", "cutoff_at")
	}
}

// --- Update ---

// A PATCH that names nothing, or names a field with the wrong type, is refused and changes nothing.
func TestDashInvoices_UpdateRefusesMalformedBodies(t *testing.T) {
	t.Parallel()
	inv := parityInvoiceFor(t, parityCustomer(t, ""), nil)
	path := invoicesPath + "/" + inv.invoiceID
	before := getInvoice(t, inv.invoiceID)

	status, body, err := apiClient.Patch(path, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	dashInvoicesRequireBadRequest(t, status, body, "validation_failed", "")

	for field, value := range map[string]any{
		"has_been_sent":   "yes",
		"is_edi_sent":     1,
		"is_paid_in_full": "true",
		"note":            123,
	} {
		status, body, err := apiClient.Patch(path, map[string]any{field: value}, newIdempotencyKey())
		require.NoError(t, err)
		dashInvoicesRequireBadRequest(t, status, body, "invalid_format", field)
	}
	for _, flag := range []string{"has_been_sent", "is_edi_sent"} {
		status, body, err := apiClient.Patch(path, map[string]any{flag: nil}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, http.StatusBadRequest, status, body)
	}

	status, body, err = apiClient.Patch(invoicesPath+"/iv_01nosuchinvoice0000", map[string]any{"note": "nobody's"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusNotFound, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")

	after := getInvoice(t, inv.invoiceID)
	for _, key := range []string{"note", "has_been_sent", "is_edi_sent", "is_paid_in_full", "payment_status", "updated_at"} {
		assert.Equal(t, before[key], after[key], key)
	}
	assert.Zero(t, dashInvoicesAuditedUpdates(t, inv.invoiceID), "no refused PATCH was audited")
}

// Replaying a PATCH answers with the first response and does not write it again: a later change
// made under another key survives the replay. The same key with another body is refused.
func TestDashInvoices_UpdateReplayReturnsTheFirstResultWithoutReapplying(t *testing.T) {
	t.Parallel()
	inv := parityInvoiceFor(t, parityCustomer(t, ""), nil)
	path := invoicesPath + "/" + inv.invoiceID
	first, second := uniqueName("e2e first note"), uniqueName("e2e second note")
	key := newIdempotencyKey()

	status, firstBody, err := apiClient.Patch(path, map[string]any{"note": first}, key)
	require.NoError(t, err)
	requireStatus(t, 200, status, firstBody)
	patchInvoice(t, inv.invoiceID, map[string]any{"note": second, "has_been_sent": true})

	status, replayBody, err := apiClient.Patch(path, map[string]any{"note": first}, key)
	require.NoError(t, err)
	requireStatus(t, 200, status, replayBody)
	assert.Equal(t, parseJSON(firstBody), parseJSON(replayBody), "the replay is the first response")

	stored := getInvoice(t, inv.invoiceID)
	assert.Equal(t, second, jsonField(stored, "note"), "the replay did not write its note again")
	assert.Equal(t, "true", jsonField(stored, "has_been_sent"))

	status, body, err := apiClient.Patch(path, map[string]any{"note": uniqueName("e2e other note")}, key)
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, body)
	requireErrorResponse(t, body, "validation_failed", "idempotency_error")
	assert.Equal(t, second, jsonField(getInvoice(t, inv.invoiceID), "note"))
	assert.Equal(t, 2, dashInvoicesAuditedUpdates(t, inv.invoiceID), "one audit event per applied PATCH, none for the replays")
}

// The invoices page hides editing from a role that may only read invoices; the API refuses it too.
func TestDashInvoices_ReadOnlyRoleCannotUpdate(t *testing.T) {
	t.Parallel()
	inv := parityInvoiceFor(t, parityCustomer(t, ""), nil)
	reader := customRoleClient(t, "invoices:read")

	assert.Equal(t, inv.invoiceID, jsonField(dashInvoicesRetrieve(t, reader, inv.invoiceID), "id"), "the role reads the invoice")
	for _, body := range []map[string]any{{"note": "read only"}, {"has_been_sent": true}, {"is_paid_in_full": true}} {
		status, raw, err := reader.Patch(invoicesPath+"/"+inv.invoiceID, body, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, status, "%v: %s", body, string(raw))
	}
	after := getInvoice(t, inv.invoiceID)
	assert.Nil(t, after["note"])
	assert.Equal(t, "false", jsonField(after, "has_been_sent"))
	assert.Equal(t, "false", jsonField(after, "is_paid_in_full"))
}

// --- Receivables ---

// dashInvoicesReceivable is invoiceID's entry on a receivables report, or nil when the report leaves it out.
func dashInvoicesReceivable(t *testing.T, path string, params url.Values, invoiceID string) map[string]any {
	t.Helper()
	status, body, err := apiClient.GetListRaw(path, params)
	for page := 0; ; page++ {
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		got := parseJSON(body)
		for _, raw := range jsonArray(got, "data") {
			if row := raw.(map[string]any); jsonField(jsonObject(row, "invoice"), "id") == invoiceID {
				return row
			}
		}
		next := jsonField(jsonObject(got, "page_info"), "next_page_url")
		if next == "" || page >= maxListScanPages {
			return nil
		}
		status, body, err = apiClient.GetListRawFromPageURL(&next)
	}
}

// A receivable entry owes the invoice total less what has been applied to it, and the paid-in-full
// mark takes the invoice off both receivables reports until it is cleared.
func TestDashInvoices_ReceivablesFollowPaymentsAndThePaidMark(t *testing.T) {
	t.Parallel()
	customerID := parityCustomer(t, "")
	po := uniqueName("E2E-DASHINV-AR")
	inv := parityInvoiceFor(t, customerID, map[string]any{"customer_purchase_order_number": po})
	invoice := getInvoice(t, inv.invoiceID)
	total := decimal.RequireFromString(jsonField(invoice, "total_invoiced"))
	reports := map[string]url.Values{
		receivablesPathFor(customerID): nil,
		receivablesPath:                {"q": {inv.number}},
	}

	for path, params := range reports {
		entry := dashInvoicesReceivable(t, path, params, inv.invoiceID)
		require.NotNil(t, entry, "%s lists the unpaid invoice", path)
		assert.Equal(t, "receivable_entry", jsonField(entry, "object"), path)
		assert.Equal(t, inv.number, jsonField(jsonObject(entry, "invoice"), "number"), path)
		assert.Equal(t, "invoice", jsonField(jsonObject(entry, "invoice"), "object"), path)
		customer := jsonObject(entry, "customer")
		assert.Equal(t, customerID, jsonField(customer, "id"), path)
		assert.Equal(t, getCustomerName(t, customerID), jsonField(customer, "name"), path)
		assert.Equal(t, po, jsonField(entry, "po_number"), path)
		invoicedAt, err := time.Parse(time.RFC3339, jsonField(entry, "invoiced_at"))
		require.NoError(t, err)
		createdAt, err := time.Parse(time.RFC3339, jsonField(invoice, "created_at"))
		require.NoError(t, err)
		assert.True(t, createdAt.Equal(invoicedAt), "%s: invoiced when the invoice was created", path)
		assert.True(t, total.Equal(decimal.RequireFromString(jsonField(entry, "remaining_balance"))), "%s: nothing applied yet", path)
		assert.Equal(t, "false", jsonField(entry, "is_paid_in_full"), path)
	}

	funds := time.Now().UTC()
	payment := createPayment(t, customerID, "5.00", &funds, nil)
	settle(t, map[string]any{"allocations": []any{allocation(jsonField(payment, "id"), inv.invoiceID, "5.00")}})
	owed := total.Sub(decimal.NewFromInt(5))
	for path, params := range reports {
		entry := dashInvoicesReceivable(t, path, params, inv.invoiceID)
		require.NotNil(t, entry, path)
		assert.True(t, owed.Equal(decimal.RequireFromString(jsonField(entry, "remaining_balance"))),
			"%s: %s owed after $5 applied, got %s", path, owed, jsonField(entry, "remaining_balance"))
	}

	patchInvoice(t, inv.invoiceID, map[string]any{"is_paid_in_full": true})
	for path, params := range reports {
		assert.Nil(t, dashInvoicesReceivable(t, path, params, inv.invoiceID), "%s: marked paid, it owes nothing", path)
	}
	patchInvoice(t, inv.invoiceID, map[string]any{"is_paid_in_full": false})
	for path, params := range reports {
		entry := dashInvoicesReceivable(t, path, params, inv.invoiceID)
		require.NotNil(t, entry, "%s: clearing the mark puts it back", path)
		assert.True(t, owed.Equal(decimal.RequireFromString(jsonField(entry, "remaining_balance"))), path)
	}
}

// --- Settle flow ---

// The settle page reads each payable invoice's total, billing, account and allocations from one
// list; each include resolves to the invoice's own and none appears unasked.
func TestDashInvoices_CustomerInvoicesCarryWhatTheSettleFlowReads(t *testing.T) {
	t.Parallel()
	customerID := parityCustomer(t, "")
	po := uniqueName("E2E-DASHINV-SETTLE")
	inv := parityInvoiceFor(t, customerID, map[string]any{"customer_purchase_order_number": po})
	invoice := getInvoice(t, inv.invoiceID)
	funds := time.Now().UTC()
	payment := createPayment(t, customerID, "5.00", &funds, nil)
	settle(t, map[string]any{"allocations": []any{allocation(jsonField(payment, "id"), inv.invoiceID, "5.00")}})
	path := customerInvoicesPathFor(customerID)

	row := invoiceRow(t, apiClient, path, url.Values{"include": dashInvoicesSettleIncludes}, inv.invoiceID)
	assert.Equal(t, "invoice_for_payment", jsonField(row, "object"))
	assert.Equal(t, inv.number, jsonField(row, "number"))
	assert.Equal(t, po, jsonField(row, "customer_po"))
	assert.Equal(t, jsonField(invoice, "total_invoiced"), jsonField(row, "invoice_total"))
	assert.Equal(t, "false", jsonField(row, "is_paid_in_full"))
	assert.Equal(t, "false", jsonField(row, "is_prepaid"))
	assert.Equal(t, "false", jsonField(row, "is_parent_account"), "the customer is no other customer's child")
	assertNilField(t, row, "parent_account")
	assert.Equal(t, customerID, jsonField(jsonObject(row, "customer"), "id"))
	assert.Equal(t, inv.billToID, jsonField(jsonObject(row, "billing_address"), "id"))
	assertValidTimestamp(t, jsonField(row, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(row, "updated_at"), "updated_at")

	allocations := jsonListData(row, "allocations")
	require.Len(t, allocations, 1)
	applied := allocations[0].(map[string]any)
	assert.True(t, decimal.NewFromInt(5).Equal(amountOf(t, applied)))
	assert.NotNil(t, jsonObject(jsonObject(applied, "amount"), "unit"), "allocations.amount.unit")
	assert.Equal(t, jsonField(payment, "id"), jsonField(jsonObject(applied, "transaction"), "id"))

	bare := invoiceRow(t, apiClient, path, nil, inv.invoiceID)
	for _, key := range []string{"customer", "parent_account", "billing_address", "allocations"} {
		assertNilField(t, bare, key)
	}
}

// The settle page lists a customer's transactions by allocation status and searches them by
// number; the type filter narrows the same list, and unknown values are refused.
func TestDashInvoices_AccountTransactionsFilter(t *testing.T) {
	t.Parallel()
	inv := invoiceNewCustomer(t)
	funds := time.Now().UTC()

	applied := dashInvoicesTransaction(t, inv.customerID, "payment", "10.00", &funds)
	settle(t, map[string]any{"allocations": []any{allocation(jsonField(applied, "id"), inv.invoiceID, "10.00")}})
	awaitFullyAllocated(t, jsonField(applied, "id"), true)
	open := dashInvoicesTransaction(t, inv.customerID, "payment", "7.00", &funds)
	credit := dashInvoicesTransaction(t, inv.customerID, "credit_memo", "3.00", &funds)
	pending := dashInvoicesTransaction(t, inv.customerID, "payment", "4.00", nil)
	id := func(tx map[string]any) string { return jsonField(tx, "id") }
	path := dashInvoicesTransactionsPathFor(inv.customerID)
	list := func(params url.Values) []string {
		t.Helper()
		if params == nil {
			params = url.Values{}
		}
		params.Set("limit", "100")
		rows, _ := listPayments(t, path, params)
		return rowIDs(rows)
	}

	assert.ElementsMatch(t, []string{id(applied), id(open), id(credit), id(pending)}, list(nil))
	assert.Equal(t, []string{id(applied)}, list(url.Values{"status": {"allocated"}}))
	assert.ElementsMatch(t, []string{id(open), id(credit)}, list(url.Values{"status": {"unallocated"}}),
		"received and not fully applied; the pending payment has no funds yet")
	assert.ElementsMatch(t, []string{id(applied), id(open), id(pending)}, list(url.Values{"type": {"payment"}}))
	assert.Equal(t, []string{id(credit)}, list(url.Values{"type": {"credit_memo"}}))
	assert.Contains(t, list(url.Values{"q": {jsonField(credit, "number")}}), id(credit), "by number")
	assert.Empty(t, list(url.Values{"q": {searchToken("e2enosuchtx")}}))

	rows, _ := listPayments(t, path, url.Values{"include": {"allocations", "customer"}, "status": {"allocated"}})
	require.Len(t, rows, 1)
	assert.Equal(t, inv.customerID, jsonField(jsonObject(rows[0], "customer"), "id"))
	assert.Len(t, jsonListData(rows[0], "allocations"), 1)
	rows, _ = listPayments(t, path, url.Values{"status": {"allocated"}})
	require.Len(t, rows, 1)
	assertNilField(t, rows[0], "customer")
	assertNilField(t, rows[0], "allocations")

	for param, bad := range map[string]string{"status": "settled", "type": "refund"} {
		status, body, err := apiClient.GetListRaw(path, url.Values{param: {bad}})
		require.NoError(t, err)
		dashInvoicesRequireBadRequest(t, status, body, "parameter_invalid", param)
	}
}

// --- Access ---

// Another tenant, the seller's customer, and the seller acting inside its customer's account each
// reach none of the seller's invoices or a customer's transactions, and their writes change nothing.
func TestDashInvoices_AccountPathsLeakNothingAcrossAccounts(t *testing.T) {
	t.Parallel()
	customerID := parityCustomer(t, "")
	inv := parityInvoiceFor(t, customerID, nil)
	funds := time.Now().UTC()
	payment := dashInvoicesTransaction(t, customerID, "payment", "6.00", &funds)
	invoicePath := invoicesPath + "/" + inv.invoiceID
	require.Contains(t, invoiceListIDs(t, apiClient, dashInvoicesTransactionsPathFor(customerID), nil), jsonField(payment, "id"))

	tenantB := getTenantBClient()
	assert.Empty(t, invoiceListIDs(t, tenantB, dashInvoicesTransactionsPathFor(customerID), nil), "tenant B has no such customer")
	assert.Empty(t, invoiceListIDs(t, tenantB, invoicesPath, url.Values{"customer_ids": {customerID}}))
	status, body, err := tenantB.Patch(invoicePath, map[string]any{"note": "not yours"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, status, string(body))
	status, body = dashInvoicesEmailRecord(t, tenantB, map[string]any{"type": "invoice", "id": inv.invoiceID}, newIdempotencyKey())
	assert.Equal(t, http.StatusNotFound, status, string(body))

	portal := getCustomerPortalClient()
	for _, account := range []string{SeedCustomerAccountID, customerID} {
		status, body, err := portal.GetListRaw(dashInvoicesTransactionsPathFor(account), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, status, "the seller's transactions are its ledger: %s", string(body))
	}

	// A customer's key in its own account looks among that account's invoices, where the seller's are not.
	portalHome := NewClient(envOr("E2E_BASE_URL", defaultBaseURL), SeedCustomerAPIKey, SeedCustomerAccountID)
	status, body, err = portalHome.GetListRaw(invoicePath, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, status, string(body))
	status, body, err = portalHome.Patch(invoicePath, map[string]any{"note": "not yours"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, status, string(body))
	assert.NotContains(t, invoiceListIDs(t, portalHome, invoicesPath, nil), inv.invoiceID)
	assert.NotContains(t, invoiceListIDs(t, portalHome, customerInvoicesPathFor(SeedAccountID), nil), inv.invoiceID)

	// Invoices belong to the seller's own users: acting inside the customer's account is refused.
	inCustomer := apiClient.WithAccountID(customerID)
	for what, call := range map[string]func() (int, []byte, error){
		"list":     func() (int, []byte, error) { return inCustomer.GetListRaw(invoicesPath, nil) },
		"retrieve": func() (int, []byte, error) { return inCustomer.GetListRaw(invoicePath, nil) },
		"transactions": func() (int, []byte, error) {
			return inCustomer.GetListRaw(dashInvoicesTransactionsPathFor(customerID), nil)
		},
		"receivables": func() (int, []byte, error) { return inCustomer.GetListRaw(receivablesPathFor(customerID), nil) },
		"payable":     func() (int, []byte, error) { return inCustomer.GetListRaw(customerInvoicesPathFor(customerID), nil) },
		"update": func() (int, []byte, error) {
			return inCustomer.Patch(invoicePath, map[string]any{"note": "not here"}, newIdempotencyKey())
		},
		"email": func() (int, []byte, error) {
			return inCustomer.Post(invoiceEmailRecordPath, map[string]any{"type": "invoice", "id": inv.invoiceID}, newIdempotencyKey())
		},
	} {
		status, body, err := call()
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, status, "%s: %s", what, string(body))
	}

	after := getInvoice(t, inv.invoiceID)
	assert.Nil(t, after["note"], "no refused PATCH wrote the note")
	assert.Equal(t, "false", jsonField(after, "has_been_sent"), "no refused email marked it sent")
}

// --- Statement of account ---

// The statement goes to addresses typed into the dashboard; one that is blank or not an address is
// refused before anything is queued.
func TestDashInvoices_StatementEmailRefusesBadRecipients(t *testing.T) {
	t.Parallel()
	customerID := parityCustomer(t, "")
	path := emailReceivablesPathFor(customerID)
	post := func(body map[string]any) (int, []byte) {
		t.Helper()
		status, raw, err := apiClient.Post(path, body, newIdempotencyKey())
		require.NoError(t, err)
		return status, raw
	}

	status, body := post(map[string]any{})
	dashInvoicesRequireBadRequest(t, status, body, "missing_field", "recipient_emails")
	status, body = post(map[string]any{"recipient_emails": []string{}})
	dashInvoicesRequireBadRequest(t, status, body, "", "recipient_emails")
	status, body = post(map[string]any{"recipient_emails": "ap@example.com"})
	dashInvoicesRequireBadRequest(t, status, body, "invalid_format", "recipient_emails")

	token := searchToken("e2edashinvbad")
	for _, bad := range [][]string{{""}, {"not-an-address-" + token}, {dashInvoicesAddress(token), "also-not-" + token}} {
		status, body := post(map[string]any{"recipient_emails": bad})
		assert.Equal(t, http.StatusBadRequest, status, "%q must be refused: %s", bad, string(body))
	}
	assert.Zero(t, len(dashInvoicesQueuedEmails(t, token)), "nothing was queued for a refused request")
}

// Replaying the statement request answers the same and queues no second email; a new request
// sends again. The send reaches the email log, attributed to whoever sent it.
func TestDashInvoices_StatementEmailReplaySendsOnce(t *testing.T) {
	t.Parallel()
	customerID := parityCustomer(t, "")
	path := emailReceivablesPathFor(customerID)
	token := searchToken("e2edashinvsoa")
	first, second := dashInvoicesAddress(token+"a"), dashInvoicesAddress(token+"b")
	subject := "Statement of Account for " + getCustomerName(t, customerID)
	key := newIdempotencyKey()

	status, body, err := apiClient.Post(path, map[string]any{"recipient_emails": []string{first, second}}, key)
	require.NoError(t, err)
	requireStatus(t, http.StatusAccepted, status, body)
	status, replay, err := apiClient.Post(path, map[string]any{"recipient_emails": []string{first, second}}, key)
	require.NoError(t, err)
	requireStatus(t, http.StatusAccepted, status, replay)
	assert.JSONEq(t, string(body), string(replay))

	queued := dashInvoicesQueuedEmails(t, token)
	require.Equal(t, 1, len(queued), "the replay queued nothing")
	email := queued[0]
	assert.Equal(t, []any{first, second}, email["to"], "one email to every recipient")
	assert.Equal(t, subject, jsonField(email, "subject"))
	assert.True(t, strings.HasPrefix(jsonField(email, "attachment_filename"), "account-statement-"), jsonField(email, "attachment_filename"))
	assert.True(t, strings.HasSuffix(jsonField(email, "attachment_filename"), ".xlsx"))
	assert.NotEmpty(t, jsonField(email, "sent_by_id"))

	status, body, err = apiClient.Post(path, map[string]any{"recipient_emails": []string{first}}, key)
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, body)
	requireErrorResponse(t, body, "validation_failed", "idempotency_error")
	assert.Equal(t, 1, len(dashInvoicesQueuedEmails(t, token)))

	logged := dashInvoicesAwaitEmailLog(t, first, subject)
	assert.Equal(t, []any{first, second}, logged["recipients"])
	assert.True(t, strings.HasPrefix(jsonField(logged, "filename"), "account-statement-"))
	assert.Equal(t, "sent", jsonField(logged, "send_status"))
	assert.NotNil(t, jsonObject(logged, "sent_by"), "a statement someone sent is attributed to them")

	status, body, err = apiClient.Post(path, map[string]any{"recipient_emails": []string{first, second}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusAccepted, status, body)
	assert.Equal(t, 2, len(dashInvoicesQueuedEmails(t, token)), "a new request sends again")
}

// --- Email invoice ---

// Emailing an invoice sends it, with its PDF, to the contacts its order names for invoices and
// marks it sent; replaying the request answers the same and sends nothing more.
func TestDashInvoices_EmailInvoiceSendsToItsContactsOnce(t *testing.T) {
	t.Parallel()
	customerID := parityCustomer(t, "")
	contactID, contactEmail := dashInvoicesContact(t, customerID)
	description := uniqueName("e2e emailed invoice line")
	inv := parityInvoiceFor(t, customerID, map[string]any{"invoice_email_contacts": []map[string]any{{"account_user_id": contactID}}},
		dashInvoicesSoldLine(description))
	subject := "Invoice " + paddedNumber(inv.number)
	sent := func() []map[string]any {
		t.Helper()
		return dashInvoicesWithSubject(dashInvoicesQueuedEmails(t, description), subject)
	}

	before := getInvoice(t, inv.invoiceID)
	assert.Equal(t, "true", jsonField(before, "accepts_invoice_emails"), "the order names an invoice contact")
	assert.Equal(t, "false", jsonField(before, "has_been_sent"), "shipping without email_customer sends nothing")
	assert.Zero(t, len(sent()))

	key := newIdempotencyKey()
	request := map[string]any{"type": "invoice", "id": inv.invoiceID}
	status, body := dashInvoicesEmailRecord(t, apiClient, request, key)
	requireStatus(t, http.StatusAccepted, status, body)
	assert.Equal(t, "true", jsonField(getInvoice(t, inv.invoiceID), "has_been_sent"))

	status, replay := dashInvoicesEmailRecord(t, apiClient, request, key)
	requireStatus(t, http.StatusAccepted, status, replay)
	assert.JSONEq(t, string(body), string(replay))

	emails := sent()
	require.Equal(t, 1, len(emails), "the replay queued nothing")
	assert.Equal(t, []any{contactEmail}, emails[0]["to"])
	assert.Equal(t, "invoice-"+inv.number+".pdf", jsonField(emails[0], "attachment_filename"))
	assert.NotEmpty(t, jsonField(emails[0], "attachment_data"), "the invoice PDF is attached")

	status, body = dashInvoicesEmailRecord(t, apiClient, map[string]any{"type": "sales_order", "id": inv.orderID}, key)
	requireStatus(t, http.StatusBadRequest, status, body)
	requireErrorResponse(t, body, "validation_failed", "idempotency_error")

	logged := dashInvoicesAwaitEmailLog(t, contactEmail, subject)
	assert.Equal(t, []any{contactEmail}, logged["recipients"])
	assert.Equal(t, "invoice-"+inv.number+".pdf", jsonField(logged, "filename"))
	assert.Equal(t, "sent", jsonField(logged, "send_status"))

	status, body = dashInvoicesEmailRecord(t, apiClient, request, newIdempotencyKey())
	requireStatus(t, http.StatusAccepted, status, body)
	assert.Equal(t, 2, len(sent()), "a new request resends")
}

// A manual send is attributed to whoever pressed it; only the platform's own sends go unattributed.
func TestDashInvoices_EmailedInvoiceIsAttributedToItsSender(t *testing.T) {
	t.Parallel()
	customerID := parityCustomer(t, "")
	contactID, contactEmail := dashInvoicesContact(t, customerID)
	inv := parityInvoiceFor(t, customerID, map[string]any{"invoice_email_contacts": []map[string]any{{"account_user_id": contactID}}})

	status, body := dashInvoicesEmailRecord(t, apiClient, map[string]any{"type": "invoice", "id": inv.invoiceID}, newIdempotencyKey())
	requireStatus(t, http.StatusAccepted, status, body)

	logged := dashInvoicesAwaitEmailLog(t, contactEmail, "Invoice "+paddedNumber(inv.number))
	assert.NotNil(t, jsonObject(logged, "sent_by"), "the email log names the sender: %v", logged)
}

// An invoice whose order names no invoice contact is still marked sent, so the resend sweep leaves
// it alone, but nothing is queued.
func TestDashInvoices_EmailInvoiceWithoutContactsMarksItSentAndSendsNothing(t *testing.T) {
	t.Parallel()
	description := uniqueName("e2e unaddressed invoice line")
	inv := parityInvoiceFor(t, parityCustomer(t, ""), nil, dashInvoicesSoldLine(description))
	assert.Equal(t, "false", jsonField(getInvoice(t, inv.invoiceID), "accepts_invoice_emails"))

	status, body := dashInvoicesEmailRecord(t, apiClient, map[string]any{"type": "invoice", "id": inv.invoiceID}, newIdempotencyKey())
	requireStatus(t, http.StatusAccepted, status, body)
	assert.Equal(t, "true", jsonField(getInvoice(t, inv.invoiceID), "has_been_sent"))
	assert.Zero(t, len(dashInvoicesQueuedEmails(t, description)))
}

func TestDashInvoices_EmailRecordValidatesTheRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		body        map[string]any
		code, param string
	}{
		{map[string]any{"type": "invoice"}, "missing_field", "id"},
		{map[string]any{"type": "invoice", "id": ""}, "missing_field", "id"},
		{map[string]any{"id": SeedInvoiceID}, "missing_field", "type"},
		{map[string]any{"id": SeedInvoiceID, "type": "credit_memo"}, "parameter_invalid", "type"},
		{map[string]any{"id": SeedInvoiceID, "type": 1}, "invalid_format", "type"},
	} {
		status, body := dashInvoicesEmailRecord(t, apiClient, tc.body, newIdempotencyKey())
		dashInvoicesRequireBadRequest(t, status, body, tc.code, tc.param)
	}

	// The refusal is cached under its key like a success would be.
	key := newIdempotencyKey()
	unknown := map[string]any{"type": "invoice", "id": "iv_01nosuchinvoice0000"}
	for range 2 {
		status, body := dashInvoicesEmailRecord(t, apiClient, unknown, key)
		requireStatus(t, http.StatusNotFound, status, body)
		requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
	}
}

// Each record type names its own kind of record: an id that is not one of the account's records of
// that type is a 404, as it is for an invoice, rather than an accepted send to nobody.
func TestDashInvoices_EmailRecordOfAnotherTypeNeedsARecordOfThatType(t *testing.T) {
	t.Parallel()
	inv := parityInvoiceFor(t, parityCustomer(t, ""), nil)
	for _, tc := range []struct{ typ, id string }{
		{"sales_order", "or_01nosuchsalesorder00"},
		{"purchase_order", "or_01nosuchpurchase000"},
		{"sales_order", inv.invoiceID},
		{"purchase_order", inv.invoiceID},
		{"invoice", inv.orderID},
	} {
		status, body := dashInvoicesEmailRecord(t, apiClient, map[string]any{"type": tc.typ, "id": tc.id}, newIdempotencyKey())
		assert.Equal(t, http.StatusNotFound, status, "%s %s: %s", tc.typ, tc.id, string(body))
	}
	status, body := dashInvoicesEmailRecord(t, getTenantBClient(), map[string]any{"type": "sales_order", "id": inv.orderID}, newIdempotencyKey())
	assert.Equal(t, http.StatusNotFound, status, "another tenant's order: %s", string(body))
}

// The endpoint admits any of invoices, sales-order or purchase-order read; the record type sent
// decides which one the caller needs.
func TestDashInvoices_EmailRecordChecksThePermissionForTheTypeSent(t *testing.T) {
	t.Parallel()
	inv := parityInvoiceFor(t, parityCustomer(t, ""), nil)
	ordersOnly := customRoleClient(t, "sales_orders:read")
	invoicesOnly := customRoleClient(t, "invoices:read")

	status, body := dashInvoicesEmailRecord(t, ordersOnly, map[string]any{"type": "invoice", "id": inv.invoiceID}, newIdempotencyKey())
	assert.Equal(t, http.StatusForbidden, status, string(body))
	assert.Equal(t, "false", jsonField(getInvoice(t, inv.invoiceID), "has_been_sent"), "the refused send marked nothing")

	status, body = dashInvoicesEmailRecord(t, invoicesOnly, map[string]any{"type": "sales_order", "id": inv.orderID}, newIdempotencyKey())
	assert.Equal(t, http.StatusForbidden, status, string(body))

	status, body = dashInvoicesEmailRecord(t, invoicesOnly, map[string]any{"type": "invoice", "id": inv.invoiceID}, newIdempotencyKey())
	requireStatus(t, http.StatusAccepted, status, body)
	assert.Equal(t, "true", jsonField(getInvoice(t, inv.invoiceID), "has_been_sent"))
}
