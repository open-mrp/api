//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The dashboard calls this while a user types a number into a create form, to warn them before
// they submit one that would collide. It is a PUT: the check reads and never reserves.
const checkDuplicatesPath = "/v1/core/actions/check-duplicates"

// checkDuplicate PUTs a duplicate check as client and returns the raw answer, refusing any 5xx.
func checkDuplicate(t *testing.T, client *Client, body map[string]any) (int, []byte) {
	t.Helper()
	status, resp, err := client.Put(checkDuplicatesPath, body)
	require.NoError(t, err)
	require.Less(t, status, 500, "check-duplicates must not 5xx: %s", string(resp))
	return status, resp
}

// requireCheckResult runs a duplicate check that must succeed and returns its result.
func requireCheckResult(t *testing.T, client *Client, body map[string]any) map[string]any {
	t.Helper()
	status, resp := checkDuplicate(t, client, body)
	requireStatus(t, 200, status, resp)
	got := parseJSON(resp)
	require.NotNil(t, got, "check result must be JSON: %s", string(resp))
	assert.Equal(t, "check_duplicate_result", jsonField(got, "object"))
	return got
}

// assertNotDuplicate asserts a result reports the number as free, with no message to show.
func assertNotDuplicate(t *testing.T, got map[string]any) {
	t.Helper()
	assert.Equal(t, "false", jsonField(got, "is_duplicate"), "the number is free: %v", got)
	assertNilField(t, got, "message")
}

// orderWithCustomerPO creates a sales order for a fresh customer carrying a customer PO number no
// other test uses, and returns the customer, the order number the system assigned, and the PO.
func orderWithCustomerPO(t *testing.T) (customerID, orderNumber, po string) {
	t.Helper()
	customerID = setupOrderCustomer(t)
	po = uniqueName("e2e-dup-po")

	body := minimalSalesOrderCreateBody(t, customerID)
	body["customer_purchase_order_number"] = po
	status, resp, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	order := parseJSON(resp)
	deleteOrder(t, jsonField(order, "id"))

	orderNumber = jsonField(order, "number")
	require.NotEmpty(t, orderNumber, "an order is numbered on create: %s", string(resp))
	require.Equal(t, po, jsonField(order, "customer_purchase_order_number"))
	return customerID, orderNumber, po
}

// seedInvoiceNumber reads the seeded invoice's number, so the check is asserted against a record
// known to exist rather than a literal that could drift from the seed.
func seedInvoiceNumber(t *testing.T) string {
	t.Helper()
	number := jsonField(parseJSON(mustGet(t, invoicesPath+"/"+SeedInvoiceID)), "number")
	require.NotEmpty(t, number, "the seeded invoice carries a number")
	return number
}

// countInAccount counts rows matching a number in one account, for preconditions the API cannot
// state: that another tenant does not happen to hold the same number.
func countInAccount(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, authDB(t).QueryRow(query, args...).Scan(&n))
	return n
}

// roleScopedClient returns a client holding a fresh API key on the seed account under roleID, so a
// test can act with a role's permissions rather than the admin's.
func roleScopedClient(t *testing.T, roleID string) *Client {
	t.Helper()
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    uniqueName("e2e-role-scoped"),
		"role_id": roleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	created := parseJSON(body)
	keyID := jsonField(jsonObject(created, "api_key_info"), "id")
	require.NotEmpty(t, keyID)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(apiKeysPath + "/" + keyID) })
	secret := jsonField(created, "api_key_secret")
	require.NotEmpty(t, secret)
	return apiClient.WithBearerToken(secret, SeedAccountID)
}

// --- Duplicates ---

// An invoice number already issued has to be flagged, and the message is what the form shows under
// the field, so it must name the kind of number and the number itself.
func TestCheckDuplicates_InvoiceNumberInUseIsReported(t *testing.T) {
	t.Parallel()
	number := seedInvoiceNumber(t)

	got := requireCheckResult(t, apiClient, map[string]any{"type": "invoice_number", "record_number": number})
	assert.Equal(t, "true", jsonField(got, "is_duplicate"), "the seeded invoice's number is taken: %v", got)
	assert.Equal(t, "This invoice number "+number+" already exists", jsonField(got, "message"))
}

// Order numbers are matched across the whole account, so an order's number is taken whoever the
// customer is.
func TestCheckDuplicates_SalesOrderNumberInUseIsReported(t *testing.T) {
	t.Parallel()
	_, orderNumber, _ := orderWithCustomerPO(t)

	got := requireCheckResult(t, apiClient, map[string]any{"type": "order_number", "record_number": orderNumber})
	assert.Equal(t, "true", jsonField(got, "is_duplicate"), "the new order's number is taken: %v", got)
	assert.Equal(t, "This sales order number "+orderNumber+" already exists", jsonField(got, "message"))
}

// This is the check the dashboard actually makes, from the sales order form's customer PO field. A
// customer reusing their own PO number is the mistake it exists to catch.
func TestCheckDuplicates_CustomerPONumberInUseIsReported(t *testing.T) {
	t.Parallel()
	customerID, _, po := orderWithCustomerPO(t)

	got := requireCheckResult(t, apiClient, map[string]any{
		"type": "customer_po_number", "record_number": po, "customer_id": customerID,
	})
	assert.Equal(t, "true", jsonField(got, "is_duplicate"), "the customer already used this PO: %v", got)
	assert.Equal(t, "This customer PO number "+po+" already exists", jsonField(got, "message"))
}

// A number nothing carries must come back free with no message; a false warning would block a
// user from entering a perfectly good number.
func TestCheckDuplicates_UnusedNumberIsNotADuplicate(t *testing.T) {
	t.Parallel()
	fresh := uniqueName("e2e-dup-free")

	for _, body := range []map[string]any{
		{"type": "invoice_number", "record_number": fresh},
		{"type": "order_number", "record_number": fresh},
		{"type": "customer_po_number", "record_number": fresh, "customer_id": SeedCustomerAccountID},
	} {
		t.Run(body["type"].(string), func(t *testing.T) {
			assertNotDuplicate(t, requireCheckResult(t, apiClient, body))
		})
	}
}

// Two customers can send the same PO number: each numbers their own purchase orders. Flagging
// another customer's PO would warn on every common number like "PO-1001".
func TestCheckDuplicates_CustomerPONumberIsScopedToTheCustomer(t *testing.T) {
	t.Parallel()
	_, _, po := orderWithCustomerPO(t)

	got := requireCheckResult(t, apiClient, map[string]any{
		"type": "customer_po_number", "record_number": po, "customer_id": SeedCustomerAccountID,
	})
	assertNotDuplicate(t, got)
}

// Without a customer there is nothing to scope a PO number to, so the check is refused rather
// than answered for the whole account. The dashboard only calls once a customer is chosen.
func TestCheckDuplicates_CustomerPONumberRequiresACustomer(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body map[string]any
		code string
	}{
		{"omitted", map[string]any{"type": "customer_po_number", "record_number": "PO-1"}, "validation_failed"},
		{"empty", map[string]any{"type": "customer_po_number", "record_number": "PO-1", "customer_id": ""}, "invalid_format"},
		{"blank", map[string]any{"type": "customer_po_number", "record_number": "PO-1", "customer_id": "   "}, "invalid_format"},
		{"null", map[string]any{"type": "customer_po_number", "record_number": "PO-1", "customer_id": nil}, "invalid_format"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := checkDuplicate(t, apiClient, tc.body)
			requireStatus(t, 400, status, body)
			assertErrorParam(t, requireErrorResponse(t, body, tc.code, "invalid_request_error"), "customer_id")
		})
	}
}

// customer_id only scopes PO numbers; invoice and order numbers stay account-wide even when one
// is sent, or a stray customer_id would hide a real collision.
func TestCheckDuplicates_CustomerIDDoesNotNarrowOtherTypes(t *testing.T) {
	t.Parallel()
	number := seedInvoiceNumber(t)
	_, orderNumber, _ := orderWithCustomerPO(t)

	inv := requireCheckResult(t, apiClient, map[string]any{
		"type": "invoice_number", "record_number": number, "customer_id": SeedSupplierAccountID,
	})
	assert.Equal(t, "true", jsonField(inv, "is_duplicate"), "invoice numbers are account-wide: %v", inv)

	ord := requireCheckResult(t, apiClient, map[string]any{
		"type": "order_number", "record_number": orderNumber, "customer_id": SeedCustomerAccountID,
	})
	assert.Equal(t, "true", jsonField(ord, "is_duplicate"), "order numbers are account-wide: %v", ord)
}

// Pasted numbers often carry stray spaces. They are trimmed before matching, and the message names
// the number as it is stored, not with the padding.
func TestCheckDuplicates_SurroundingWhitespaceIsIgnored(t *testing.T) {
	t.Parallel()
	number := seedInvoiceNumber(t)

	got := requireCheckResult(t, apiClient, map[string]any{"type": "invoice_number", "record_number": "  " + number + "\t"})
	assert.Equal(t, "true", jsonField(got, "is_duplicate"), "padding must not hide a taken number: %v", got)
	assert.Equal(t, "This invoice number "+number+" already exists", jsonField(got, "message"))
}

// A number of only whitespace is empty once trimmed, and an empty number is refused (see
// TestCheckDuplicates_RequestIsValidated): answering "not a duplicate" would be an answer about a
// number no record could carry. Blank customer_id is refused the same way.
func TestCheckDuplicates_WhitespaceOnlyNumberIsRefused(t *testing.T) {
	t.Parallel()

	status, body := checkDuplicate(t, apiClient, map[string]any{"type": "order_number", "record_number": "   "})
	requireStatus(t, 400, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), "record_number")
}

// --- Tenancy ---

// Numbering is per account: another merchant using the same invoice, order or PO number is not a
// collision, and saying it was would leak that the number exists elsewhere.
func TestCheckDuplicates_AnotherTenantsNumbersAreNotDuplicates(t *testing.T) {
	t.Parallel()
	tenantB := getTenantBClient()
	invoiceNumber := seedInvoiceNumber(t)
	customerID, orderNumber, po := orderWithCustomerPO(t)

	require.Zero(t, countInAccount(t, "SELECT COUNT(*) FROM invoice WHERE account_id = ? AND number = ?",
		SeedTenantBAccountID, invoiceNumber), "precondition: tenant B has no invoice %s", invoiceNumber)
	require.Zero(t, countInAccount(t, "SELECT COUNT(*) FROM sales_order WHERE owner_account_id = ? AND number = ?",
		SeedTenantBAccountID, orderNumber), "precondition: tenant B has no order %s", orderNumber)

	assertNotDuplicate(t, requireCheckResult(t, tenantB, map[string]any{"type": "invoice_number", "record_number": invoiceNumber}))
	assertNotDuplicate(t, requireCheckResult(t, tenantB, map[string]any{"type": "order_number", "record_number": orderNumber}))
	assertNotDuplicate(t, requireCheckResult(t, tenantB, map[string]any{
		"type": "customer_po_number", "record_number": po, "customer_id": customerID,
	}))
}

// An internal user acting on one of their customers' accounts checks that account's numbers. The
// merchant's own invoice number is not taken there.
func TestCheckDuplicates_ActingOnACustomerAccountChecksThatAccount(t *testing.T) {
	t.Parallel()
	number := seedInvoiceNumber(t)
	require.Zero(t, countInAccount(t, "SELECT COUNT(*) FROM invoice WHERE account_id = ? AND number = ?",
		SeedCustomerAccountID, number), "precondition: the customer account has no invoice %s", number)

	got := requireCheckResult(t, apiClient.WithAccountID(SeedCustomerAccountID), map[string]any{
		"type": "invoice_number", "record_number": number,
	})
	assertNotDuplicate(t, got)
}

// --- Validation ---

// A malformed check must say which field is wrong, so a client can surface it next to that field.
func TestCheckDuplicates_RequestIsValidated(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		body  map[string]any
		code  string
		param string
	}{
		{"missing type", map[string]any{"record_number": "X-1"}, "missing_field", "type"},
		// The dashboard's own enum also lists customer_number and user_name; the API checks neither.
		{"unsupported type", map[string]any{"type": "customer_number", "record_number": "X-1"}, "parameter_invalid", "type"},
		{"missing record_number", map[string]any{"type": "invoice_number"}, "missing_field", "record_number"},
		{"empty record_number", map[string]any{"type": "invoice_number", "record_number": ""}, "missing_field", "record_number"},
		{"record_number not a string", map[string]any{"type": "invoice_number", "record_number": 1001}, "invalid_format", "record_number"},
		{"unknown field", map[string]any{"type": "invoice_number", "record_number": "X-1", bogusE2EJSONField: true}, "parameter_unknown", bogusE2EJSONField},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := checkDuplicate(t, apiClient, tc.body)
			requireStatus(t, 400, status, body)
			assertErrorParam(t, requireErrorResponse(t, body, tc.code, "invalid_request_error"), tc.param)
		})
	}
}

// --- Authorization ---

func TestCheckDuplicates_RequiresAuthentication(t *testing.T) {
	t.Parallel()

	status, body := checkDuplicate(t, apiClient.WithBearerToken("", SeedAccountID), map[string]any{
		"type": "invoice_number", "record_number": "INV-001",
	})
	requireStatus(t, 401, status, body)
	requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")
}

// The dashboard calls this as the signed-in user, not with an API key, so a session must get the
// same answer.
func TestCheckDuplicates_SignedInUserGetsTheSameAnswer(t *testing.T) {
	t.Parallel()
	number := seedInvoiceNumber(t)

	got := requireCheckResult(t, loginAsSeedUser(t), map[string]any{"type": "invoice_number", "record_number": number})
	assert.Equal(t, "true", jsonField(got, "is_duplicate"), "a session sees the same invoices: %v", got)
	assert.Equal(t, "This invoice number "+number+" already exists", jsonField(got, "message"))
}

// Each kind of number reveals a different kind of record, so each needs that record's read
// permission. A sales rep may read sales orders but not invoices.
func TestCheckDuplicates_EachTypeNeedsItsOwnReadPermission(t *testing.T) {
	t.Parallel()
	salesRep := roleScopedClient(t, SeedSalesRepRoleID)

	assertNotDuplicate(t, requireCheckResult(t, salesRep, map[string]any{
		"type": "order_number", "record_number": uniqueName("e2e-dup-rep"),
	}))
	assertNotDuplicate(t, requireCheckResult(t, salesRep, map[string]any{
		"type": "customer_po_number", "record_number": uniqueName("e2e-dup-rep"), "customer_id": SeedCustomerAccountID,
	}))

	status, body := checkDuplicate(t, salesRep, map[string]any{"type": "invoice_number", "record_number": "INV-001"})
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")

	// A role holding none of the record permissions is stopped before any number is looked at.
	scanner := roleScopedClient(t, SeedScannerRoleID)
	status, body = checkDuplicate(t, scanner, map[string]any{"type": "order_number", "record_number": "ORD-001"})
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

// A customer's portal key reaching the merchant's account must not be able to probe the merchant's
// invoice and order numbers, or other customers' PO numbers: nothing scopes the check to the
// customer, so the only safe answer is a refusal.
func TestCheckDuplicates_CustomerPortalKeyIsRefused(t *testing.T) {
	t.Parallel()
	portal := getCustomerPortalClient()

	for _, body := range []map[string]any{
		{"type": "invoice_number", "record_number": "INV-001"},
		{"type": "order_number", "record_number": "ORD-001"},
		{"type": "customer_po_number", "record_number": SeedSalesOrderPONumber, "customer_id": SeedCustomerAccountID},
	} {
		t.Run(body["type"].(string), func(t *testing.T) {
			status, resp := checkDuplicate(t, portal, body)
			requireStatus(t, 403, status, resp)
			requireErrorResponse(t, resp, "insufficient_permissions", "invalid_request_error")
		})
	}
}
