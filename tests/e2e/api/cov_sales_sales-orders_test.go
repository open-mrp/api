//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file extends the public sales_sales-orders group
// (GET/POST /v1/sales/sales-orders, GET /v1/sales/sales-orders/statuses) with
// coverage gaps identified against the resource-struct field tables,
// request-body matrix, and query-param matrix in
// docs/patterns/e2e-test-patterns.md's companion task file for this group.
// Existing coverage in crud_sales_orders_create_test.go,
// crud_sales_orders_includes_test.go, crud_sales_orders_payment_status_test.go,
// list_sales_orders_test.go, included_fields_test.go, array_filter_union_test.go,
// customer_portal_access_test.go, and crud_partial_includes_test.go is not
// duplicated here. It reuses salesOrdersPath (included_fields_test.go),
// setupOrderCustomer / deleteOrder / productLineAccessPath
// (crud_sales_orders_create_test.go), minimalSalesOrderCreateBody / createE2EAddress
// (helpers_test.go), and getCustomerPortalClient (customer_portal_access_test.go).
//
// Two confirmed backend bugs are pinned here as tests asserting the CORRECT
// desired behavior (they currently fail red against the live stack — see the
// doc comments on TestCovSalesSalesOrders_UnvalidatedForeignKeysAcceptedBug and
// TestCovSalesSalesOrders_NonexistentEmailContactAccountUserAcceptedBug):
// service_level_id / payment_term_id / shipping_term_id / sales_rep_id / an
// email-contact account_user_id are never existence-checked at create time, so
// a garbage id is silently accepted (201) and stored as a dangling reference
// instead of being rejected with 400/404.

const covSalesSalesOrdersStatusesPath = "/v1/sales/sales-orders/statuses"

// covSalesSalesOrdersFullCreateBody returns a create body with every optional
// field populated (customer_purchase_order_number, note, carrier_billing_type,
// carrier_billing_account_number, sales_rep_id, order_discount_id, promised_at,
// line-level product_sku/product_description overrides, and both email-contact
// lists), for a buyer with product-line access to SeedProductID.
func covSalesSalesOrdersFullCreateBody(buyerAccountID, billAddrID, shipAddrID, poNumber string) map[string]any {
	return map[string]any{
		"buyer_account_id":               buyerAccountID,
		"customer_purchase_order_number": poNumber,
		"note":                           "Rush order",
		"carrier_id":                     SeedCarrierID,
		"service_level_id":               SeedServiceLevelID,
		"carrier_billing_type":           "sender",
		"carrier_billing_account_number": "ACCT-123",
		"priority_code":                  "normal",
		"sales_rep_id":                   SeedAccountUserID,
		"shipping_term_id":               SeedShippingTermID,
		"payment_term_id":                SeedPaymentTermID,
		"order_discount_id":              SeedOrderDiscountID,
		"promised_at":                    "2026-09-01T00:00:00Z",
		"bill_to_address_id":             billAddrID,
		"ship_to_address_id":             shipAddrID,
		"lines": []map[string]any{
			{
				"product_id":          SeedProductID,
				"quantity":            map[string]any{"value": "3", "unit_id": SeedUnitID},
				"product_sku":         "SKU-OVERRIDE",
				"product_description": "Custom desc",
			},
		},
		"acknowledgement_email_contacts": []map[string]any{{"account_user_id": SeedAccountUserID}},
		"invoice_email_contacts":         []map[string]any{{"account_user_id": SeedAccountUserID}},
	}
}

// ──────────────────────────────────────────────
// allFields / responseShape
// ──────────────────────────────────────────────

func TestCovSalesSalesOrders_CreateAllFields(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	bill := createE2EAddress(t, "E2E AllFields Bill")
	ship := createE2EAddress(t, "E2E AllFields Ship")
	po := uniqueName("PO")

	createResp, err := apiClient.PostFull(salesOrdersPath, covSalesSalesOrdersFullCreateBody(customerID, bill, ship, po), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	deleteOrder(t, id)

	// Response shape.
	assertIDFormat(t, id, "or")
	assertCreatedLocation(t, createResp.Header, id)
	assertObjectField(t, created, "sales_order")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
	assert.Equal(t, jsonField(created, "created_at"), jsonField(created, "updated_at"), "created_at should equal updated_at on create")

	// Scalar fields.
	assert.NotEmpty(t, jsonField(created, "number"))
	assert.Equal(t, po, jsonField(created, "customer_purchase_order_number"))
	assert.Equal(t, "Rush order", jsonField(created, "note"))
	assert.Equal(t, "estimate", jsonField(created, "status"), "new orders default to estimate")
	assert.Equal(t, "normal", jsonField(created, "priority"), "priority is a plain code string")
	assert.Equal(t, "unpaid", jsonField(created, "payment_status"))
	assert.Equal(t, "not_sent", jsonField(created, "acknowledgment_status"))
	assert.EqualValues(t, 3, created["line_count"], "1 input line + synthesized shipping line + synthesized discount line")
	assert.Equal(t, "2026-09-01T00:00:00Z", jsonField(created, "promised_at"))
	assertNilField(t, created, "issued_at")
	assertNilField(t, created, "completed_at")
	assertNilField(t, created, "first_ship_at")
	assertNilField(t, created, "expired_at")

	// Expandables are all null on the bare create response.
	for _, field := range []string{"customer", "sales_rep", "created_by", "bill_to_address", "ship_to_address", "freight", "payment_term", "shipping_term", "order_discount", "lines", "totals", "contacts"} {
		assertNilField(t, created, field)
	}
	assertNilField(t, created, "related")

	// Re-fetch with every include exercised on create and verify each
	// sub-resource resolves to what was set.
	gStatus, gBody, err := apiClient.GetListRaw(salesOrdersPath+"/"+id, url.Values{"include": {
		"customer", "sales_rep", "bill_to_address", "ship_to_address", "freight",
		"payment_term", "shipping_term", "order_discount", "totals", "contacts",
		"lines", "lines.product",
	}})
	require.NoError(t, err)
	requireStatus(t, 200, gStatus, gBody)
	full := parseJSON(gBody)

	cust := jsonObject(full, "customer")
	require.NotNil(t, cust)
	assert.Equal(t, customerID, jsonField(cust, "id"))

	rep := jsonObject(full, "sales_rep")
	require.NotNil(t, rep)
	assert.Equal(t, SeedAccountUserID, jsonField(rep, "id"))

	billAddr := jsonObject(full, "bill_to_address")
	require.NotNil(t, billAddr)
	assert.Equal(t, bill, jsonField(billAddr, "id"))

	shipAddr := jsonObject(full, "ship_to_address")
	require.NotNil(t, shipAddr)
	assert.Equal(t, ship, jsonField(shipAddr, "id"))

	freight := jsonObject(full, "freight")
	require.NotNil(t, freight)
	assert.Equal(t, "sender", jsonField(freight, "billing_type"))
	assert.Equal(t, "ACCT-123", jsonField(freight, "billing_account_number"))
	carrier := jsonObject(freight, "carrier")
	require.NotNil(t, carrier)
	assert.Equal(t, SeedCarrierID, jsonField(carrier, "id"))
	svcLevel := jsonObject(freight, "service_level")
	require.NotNil(t, svcLevel)
	assert.Equal(t, SeedServiceLevelID, jsonField(svcLevel, "id"))

	pt := jsonObject(full, "payment_term")
	require.NotNil(t, pt)
	assert.Equal(t, SeedPaymentTermID, jsonField(pt, "id"))

	st := jsonObject(full, "shipping_term")
	require.NotNil(t, st)
	assert.Equal(t, SeedShippingTermID, jsonField(st, "id"))

	disc := jsonObject(full, "order_discount")
	require.NotNil(t, disc)
	assert.Equal(t, SeedOrderDiscountID, jsonField(disc, "id"))

	contacts := jsonObject(full, "contacts")
	require.NotNil(t, contacts)
	assert.Equal(t, "order_contact", jsonField(contacts, "object"))
	assert.Contains(t, jsonStringSlice(contacts, "acknowledgement"), "dane@augno.com")
	assert.Contains(t, jsonStringSlice(contacts, "invoice"), "dane@augno.com")

	totals := jsonObject(full, "totals")
	require.NotNil(t, totals)
	assert.Equal(t, "sales_order_totals", jsonField(totals, "object"))
	assert.NotEmpty(t, jsonField(totals, "ordered"))
	assert.NotEmpty(t, jsonField(totals, "packed"))
	assert.NotEmpty(t, jsonField(totals, "invoiced"))

	lines := jsonListData(full, "lines")
	require.Len(t, lines, 3, "input line + synthesized shipping line + synthesized discount line")
	firstLine, ok := lines[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "SKU-OVERRIDE", jsonField(firstLine, "product_sku"), "line-level product_sku override should be honored")
	assert.Equal(t, "Custom desc", jsonField(firstLine, "product_description"), "line-level product_description override should be honored")
	product := jsonObject(firstLine, "product")
	require.NotNil(t, product)
	assert.Equal(t, SeedProductID, jsonField(product, "id"))
}

func TestCovSalesSalesOrders_CreateResponseShape(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)

	createResp, err := apiClient.PostFull(salesOrdersPath, minimalSalesOrderCreateBody(t, customerID), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	deleteOrder(t, id)

	assertIDFormat(t, id, "or")
	assertCreatedLocation(t, createResp.Header, id)
	assertObjectField(t, created, "sales_order")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
}

// TestCovSalesSalesOrders_PriorityAndAcknowledgmentStatusRealValues closes a
// dead-assertion gap: included_fields_test.go calls jsonObject(got,"priority")
// on this endpoint even though Priority is a plain string field, so the
// assertion always silently no-ops. This asserts the real string value.
// acknowledgment_status had zero prior assertions anywhere in the suite.
func TestCovSalesSalesOrders_PriorityAndAcknowledgmentStatusRealValues(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, "normal", jsonField(got, "priority"))
	assert.Equal(t, "not_sent", jsonField(got, "acknowledgment_status"))
}

// TestCovSalesSalesOrders_LifecycleTimestampsPopulatedOnFulfilledOrder covers
// issued_at/completed_at/first_ship_at, which previously had zero assertions.
func TestCovSalesSalesOrders_LifecycleTimestampsPopulatedOnFulfilledOrder(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedFulfilledPaidOrderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assertValidTimestamp(t, jsonField(got, "issued_at"), "issued_at")
	assertValidTimestamp(t, jsonField(got, "completed_at"), "completed_at")
	assertValidTimestamp(t, jsonField(got, "first_ship_at"), "first_ship_at")
	assertNilField(t, got, "expired_at")
}

// TestCovSalesSalesOrders_TotalsExpandable closes the zero-coverage gap on the
// `totals` expandable (prodBugSuspect #2): null without ?include=totals, and a
// populated non-zero value with it, using the seed order with a real
// settlement/invoice history.
func TestCovSalesSalesOrders_TotalsExpandable(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, parseJSON(body), "totals")

	iStatus, iBody, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, url.Values{"include": {"totals"}})
	require.NoError(t, err)
	requireStatus(t, 200, iStatus, iBody)
	totals := jsonObject(parseJSON(iBody), "totals")
	require.NotNil(t, totals, "totals should be populated with ?include=totals")
	assert.Equal(t, "sales_order_totals", jsonField(totals, "object"))
	assert.NotEqual(t, "0", jsonField(totals, "ordered"), "the seed order's lines sum to a nonzero ordered total")
	assert.NotEqual(t, "0", jsonField(totals, "invoiced"), "the seed order has a nonzero invoiced amount")
	_, hasPacked := totals["packed"]
	assert.True(t, hasPacked, "packed should always be present even when zero")
}

// ──────────────────────────────────────────────
// omittedFields
// ──────────────────────────────────────────────

func TestCovSalesSalesOrders_OmittedOptionalFieldsDefaultNull(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)

	status, body, err := apiClient.Post(salesOrdersPath, minimalSalesOrderCreateBody(t, customerID), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	created := parseJSON(body)
	deleteOrder(t, jsonField(created, "id"))

	assertNilField(t, created, "customer_purchase_order_number")
	assertNilField(t, created, "note")
	assertNilField(t, created, "promised_at")
	assertNilField(t, created, "issued_at")
	assertNilField(t, created, "completed_at")
	assertNilField(t, created, "first_ship_at")
	assertNilField(t, created, "expired_at")
}

func TestCovSalesSalesOrders_MissingRequiredFieldsRejected(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)

	fields := []string{"buyer_account_id", "bill_to_address_id", "ship_to_address_id", "priority_code", "lines"}
	for _, field := range fields {
		field := field
		t.Run(field, func(t *testing.T) {
			t.Parallel()
			body := minimalSalesOrderCreateBody(t, customerID)
			delete(body, field)

			status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
			require.NoError(t, err)
			require.GreaterOrEqual(t, status, 400, "missing %s should be rejected: %s", field, string(respBody))
			require.Less(t, status, 500, "missing %s must not 5xx: %s", field, string(respBody))
			errObj := requireErrorResponse(t, respBody, "missing_field", "invalid_request_error")
			assertErrorParam(t, errObj, field)
		})
	}
}

func TestCovSalesSalesOrders_EmptyLinesRejected(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	body := minimalSalesOrderCreateBody(t, customerID)
	body["lines"] = []map[string]any{}

	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "lines")
}

// ──────────────────────────────────────────────
// idempotency
// ──────────────────────────────────────────────

func TestCovSalesSalesOrders_CreateIdempotent(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	body := minimalSalesOrderCreateBody(t, customerID)
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(salesOrdersPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")
	deleteOrder(t, id1)

	status2, body2, err := apiClient.Post(salesOrdersPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"), "replaying the same idempotency key must return the same order")
}

// ──────────────────────────────────────────────
// validation — optional scalar fields
// ──────────────────────────────────────────────

func TestCovSalesSalesOrders_OptionalScalarFieldValidation(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)

	cases := []struct {
		name  string
		field string
		value any
	}{
		{"CustomerPurchaseOrderNumberBlank", "customer_purchase_order_number", ""},
		{"CustomerPurchaseOrderNumberNull", "customer_purchase_order_number", nil},
		{"CustomerPurchaseOrderNumberTooLong", "customer_purchase_order_number", strings.Repeat("X", 256)},
		{"NoteBlank", "note", ""},
		{"NoteNull", "note", nil},
		{"CarrierBillingAccountNumberTooLong", "carrier_billing_account_number", strings.Repeat("X", 256)},
		{"PromisedAtMalformed", "promised_at", "not-a-date"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := minimalSalesOrderCreateBody(t, customerID)
			body[tc.field] = tc.value

			status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
			require.NoError(t, err)
			require.GreaterOrEqual(t, status, 400, "%s should be rejected: %s", tc.name, string(respBody))
			require.Less(t, status, 500, "%s must not 5xx: %s", tc.name, string(respBody))
		})
	}
}

func TestCovSalesSalesOrders_CarrierBillingTypeInvalidEnumRejected(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	body := minimalSalesOrderCreateBody(t, customerID)
	body["carrier_billing_type"] = "bogus_billing_type"

	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "carrier_billing_type")
}

func TestCovSalesSalesOrders_UnknownTopLevelFieldRejected(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	body := minimalSalesOrderCreateBody(t, customerID)
	body[bogusE2EJSONField] = "x"

	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	assertJSONUnknownFieldRejected(t, "POST", salesOrdersPath, status, respBody)
}

// ──────────────────────────────────────────────
// validation — line-level fields
// ──────────────────────────────────────────────

func TestCovSalesSalesOrders_LineValidation(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)

	t.Run("MissingProductID", func(t *testing.T) {
		t.Parallel()
		body := minimalSalesOrderCreateBody(t, customerID)
		body["lines"] = []map[string]any{{"quantity": map[string]any{"value": "1", "unit_id": SeedUnitID}}}

		status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, respBody)
		requireErrorResponse(t, respBody, "missing_field", "invalid_request_error")
	})

	t.Run("NonexistentProductID", func(t *testing.T) {
		t.Parallel()
		body := minimalSalesOrderCreateBody(t, customerID)
		body["lines"] = []map[string]any{{
			"product_id": "pd_00000000000000000000",
			"quantity":   map[string]any{"value": "1", "unit_id": SeedUnitID},
		}}

		status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
		require.NoError(t, err)
		require.GreaterOrEqual(t, status, 400, "nonexistent line product_id should be rejected: %s", string(respBody))
		require.Less(t, status, 500, "nonexistent line product_id must not 5xx: %s", string(respBody))
		errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
		assertErrorParam(t, errObj, "product_id")
	})

	// A blank product_sku is treated the same as an omitted one (falls back to
	// the product's own SKU), not rejected -- unlike the top-level optional
	// string fields, whose blank/null rejection (RejectExplicitJSONNulls) only
	// walks the top-level request struct, not nested slice elements.
	t.Run("BlankProductSKUDefaultsFromProduct", func(t *testing.T) {
		t.Parallel()
		body := minimalSalesOrderCreateBody(t, customerID)
		body["lines"] = []map[string]any{{
			"product_id":  SeedProductID,
			"quantity":    map[string]any{"value": "1", "unit_id": SeedUnitID},
			"product_sku": "",
		}}

		status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, respBody)
		id := jsonField(parseJSON(respBody), "id")
		deleteOrder(t, id)

		gStatus, gBody, err := apiClient.GetListRaw(salesOrdersPath+"/"+id, url.Values{"include": {"lines"}})
		require.NoError(t, err)
		requireStatus(t, 200, gStatus, gBody)
		lines := jsonListData(parseJSON(gBody), "lines")
		require.NotEmpty(t, lines)
		first, ok := lines[0].(map[string]any)
		require.True(t, ok)
		assert.NotEmpty(t, jsonField(first, "product_sku"), "a blank product_sku falls back to the product's own SKU rather than staying blank")
	})
}

// ──────────────────────────────────────────────
// validation — foreign keys that ARE checked at create
// ──────────────────────────────────────────────

func TestCovSalesSalesOrders_NonexistentValidatedForeignKeysRejected(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)

	cases := []struct {
		name  string
		field string
		value string
	}{
		{"CarrierID", "carrier_id", "crr_00000000000000000000"},
		{"OrderDiscountID", "order_discount_id", "ords_00000000000000000000"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := minimalSalesOrderCreateBody(t, customerID)
			body[tc.field] = tc.value

			status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
			require.NoError(t, err)
			require.GreaterOrEqual(t, status, 400, "nonexistent %s should be rejected: %s", tc.field, string(respBody))
			require.Less(t, status, 500, "nonexistent %s must not 5xx: %s", tc.field, string(respBody))
		})
	}
}

// TestCovSalesSalesOrders_UnknownPriorityCodeRejectedNotGhosted empirically
// establishes the prodBugSuspect #1 behavior: PriorityCode is a bare
// `string` on the gateway request (no enum validation), and
// core-service reads sales orders back via an INNER JOIN against the
// priority lookup table, which could in theory create an order that 201s
// but is then unretrievable ("ghost record"). Empirically the live stack
// instead rejects the create outright, so this pins the safe observed
// behavior while still guarding against the ghost-record / 5xx failure
// modes if that ever regresses.
func TestCovSalesSalesOrders_UnknownPriorityCodeRejectedNotGhosted(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	body := minimalSalesOrderCreateBody(t, customerID)
	body["priority_code"] = "bogus_priority_code"

	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		id := jsonField(parseJSON(respBody), "id")
		deleteOrder(t, id)
		gStatus, gBody, err := apiClient.GetListRaw(salesOrdersPath+"/"+id, nil)
		require.NoError(t, err)
		require.Equal(t, 200, gStatus, "an order created with an unknown priority_code must remain retrievable by its own id, not a ghost record: %s", string(gBody))
		return
	}
	require.GreaterOrEqual(t, status, 400, "unknown priority_code should be rejected: %s", string(respBody))
	require.Less(t, status, 500, "unknown priority_code must not 5xx: %s", string(respBody))
}

// ──────────────────────────────────────────────
// validation — foreign keys that are NOT checked at create (confirmed bugs)
// ──────────────────────────────────────────────

// TestCovSalesSalesOrders_UnvalidatedForeignKeysAcceptedBug documents a
// CONFIRMED backend bug (prodBugSuspects #3 in TASK-sales_sales-orders.md):
// unlike carrier_id/order_discount_id (see
// TestCovSalesSalesOrders_NonexistentValidatedForeignKeysRejected, which
// correctly 400/404), service_level_id, payment_term_id, shipping_term_id,
// and sales_rep_id are never existence-checked at create time. A garbage id
// for any of these is silently accepted and stored as-is, producing a 201
// whose expanded sub-resource is a dangling reference (empty name, zero-value
// timestamps) instead of a validation error. Verified live:
//
//	POST /v1/sales/sales-orders {..., "service_level_id": "crop_00000000000000000000"} -> 201
//	GET  .../{id}?include=freight -> freight.service_level = {"id":"crop_...","name":"","created_at":"0001-01-01T00:00:00Z",...}
//
// This test asserts the CORRECT desired behavior (400/404) and will fail red
// against the current build until the backend adds existence checks for
// these fields.
func TestCovSalesSalesOrders_UnvalidatedForeignKeysAcceptedBug(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)

	cases := []struct {
		name  string
		field string
		value string
	}{
		{"ServiceLevelID", "service_level_id", "crop_00000000000000000000"},
		{"PaymentTermID", "payment_term_id", "pytm_00000000000000000000"},
		{"ShippingTermID", "shipping_term_id", "bogus_shipping_term_e2e"},
		{"SalesRepID", "sales_rep_id", "acus_00000000000000000000"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := minimalSalesOrderCreateBody(t, customerID)
			body[tc.field] = tc.value

			status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
			require.NoError(t, err)
			if status == 201 {
				id := jsonField(parseJSON(respBody), "id")
				deleteOrder(t, id)
			}
			assert.True(t, status == 400 || status == 404,
				"nonexistent %s should be rejected with 400/404, got %d: %s (confirmed backend bug: unvalidated foreign key accepted on create)",
				tc.field, status, string(respBody))
		})
	}
}

// TestCovSalesSalesOrders_NonexistentEmailContactAccountUserAcceptedBug
// documents the same class of confirmed bug for email-contact account_user_id
// values: a nonexistent id is silently dropped (the order is still created,
// and the resulting contacts.acknowledgement/invoice arrays are simply empty)
// rather than the create being rejected.
func TestCovSalesSalesOrders_NonexistentEmailContactAccountUserAcceptedBug(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	body := minimalSalesOrderCreateBody(t, customerID)
	body["acknowledgement_email_contacts"] = []map[string]any{{"account_user_id": "acus_00000000000000000000"}}

	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		id := jsonField(parseJSON(respBody), "id")
		deleteOrder(t, id)
	}
	assert.True(t, status == 400 || status == 404,
		"a nonexistent acknowledgement_email_contacts[].account_user_id should be rejected with 400/404, got %d: %s (confirmed backend bug: unvalidated account_user_id silently dropped on create)",
		status, string(respBody))
}

// ──────────────────────────────────────────────
// Email contacts (valid) and line unit_price override authorization
// ──────────────────────────────────────────────

func TestCovSalesSalesOrders_EmailContactsPopulateContacts(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	body := minimalSalesOrderCreateBody(t, customerID)
	body["acknowledgement_email_contacts"] = []map[string]any{{"account_user_id": SeedAccountUserID}}
	body["invoice_email_contacts"] = []map[string]any{{"account_user_id": SeedAccountUserID}}

	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	id := jsonField(parseJSON(respBody), "id")
	deleteOrder(t, id)

	gStatus, gBody, err := apiClient.GetListRaw(salesOrdersPath+"/"+id, url.Values{"include": {"contacts"}})
	require.NoError(t, err)
	requireStatus(t, 200, gStatus, gBody)
	contacts := jsonObject(parseJSON(gBody), "contacts")
	require.NotNil(t, contacts)
	assert.Equal(t, "order_contact", jsonField(contacts, "object"))
	assert.Contains(t, jsonStringSlice(contacts, "acknowledgement"), "dane@augno.com")
	assert.Contains(t, jsonStringSlice(contacts, "invoice"), "dane@augno.com")
}

// TestCovSalesSalesOrders_LineUnitPriceOverride_InternalHonoredCustomerIgnored
// exercises the authorization boundary documented on
// CreateSalesOrderLineInput.UnitPrice: honored only for internal actors, and
// silently ignored (server-calculated price used instead) for customer
// portal actors. This was previously untested from either actor type.
func TestCovSalesSalesOrders_LineUnitPriceOverride_InternalHonoredCustomerIgnored(t *testing.T) {
	t.Parallel()

	overrideLine := func() []map[string]any {
		return []map[string]any{{
			"product_id": SeedProductID,
			"quantity":   map[string]any{"value": "1", "unit_id": SeedUnitID},
			"unit_price": map[string]any{"value": "999.00", "numerator_unit_id": "dollar", "denominator_unit_id": SeedUnitID},
		}}
	}

	t.Run("InternalActorOverrideHonored", func(t *testing.T) {
		t.Parallel()
		customerID := setupOrderCustomer(t)
		body := minimalSalesOrderCreateBody(t, customerID)
		body["lines"] = overrideLine()

		status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, respBody)
		id := jsonField(parseJSON(respBody), "id")
		deleteOrder(t, id)

		gStatus, gBody, err := apiClient.GetListRaw(salesOrdersPath+"/"+id, url.Values{"include": {"lines", "lines.unit_price"}})
		require.NoError(t, err)
		requireStatus(t, 200, gStatus, gBody)
		lines := jsonListData(parseJSON(gBody), "lines")
		require.NotEmpty(t, lines)
		first, ok := lines[0].(map[string]any)
		require.True(t, ok)
		price := jsonObject(first, "unit_price")
		require.NotNil(t, price)
		assert.Equal(t, "999.000000000000000000000000000000", jsonField(price, "value"), "an internal actor's unit_price override must be honored")
	})

	t.Run("CustomerActorOverrideIgnored", func(t *testing.T) {
		t.Parallel()
		portal := getCustomerPortalClient()

		plStatus, _, err := apiClient.Post(productLineAccessPath, map[string]any{
			"customer_id":      SeedCustomerAccountID,
			"product_line_ids": []string{SeedProductLineID},
		}, newIdempotencyKey())
		require.NoError(t, err)
		require.True(t, plStatus == 201 || plStatus == 409, "unexpected access-grant status: %d", plStatus)

		body := minimalSalesOrderCreateBody(t, SeedCustomerAccountID)
		body["lines"] = overrideLine()

		status, respBody, err := portal.Post(salesOrdersPath, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, respBody)
		id := jsonField(parseJSON(respBody), "id")
		deleteOrder(t, id)

		gStatus, gBody, err := apiClient.GetListRaw(salesOrdersPath+"/"+id, url.Values{"include": {"lines", "lines.unit_price"}})
		require.NoError(t, err)
		requireStatus(t, 200, gStatus, gBody)
		lines := jsonListData(parseJSON(gBody), "lines")
		require.NotEmpty(t, lines)
		first, ok := lines[0].(map[string]any)
		require.True(t, ok)
		price := jsonObject(first, "unit_price")
		require.NotNil(t, price)
		assert.NotEqual(t, "999.000000000000000000000000000000", jsonField(price, "value"), "a customer actor's unit_price override must be ignored, not honored")
	})
}

// ──────────────────────────────────────────────
// list — query-param gaps not covered elsewhere
// ──────────────────────────────────────────────

func TestCovSalesSalesOrders_ListStatusCodesUnknownReturnsEmpty(t *testing.T) {
	t.Parallel()
	list, status, err := apiClient.GetList(salesOrdersPath, url.Values{"status_codes": {"bogus_status_code_e2e"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assertEmptyListData(t, list.Data, "an unknown status code should match no orders, not error")
}

// TestCovSalesSalesOrders_ListMalformedDateFiltersIgnoredNot5xx pins the
// actual observed behavior for start_date/end_date, which are plain *string
// query params with no gateway-side date-format validation: a malformed
// value is currently silently ignored (dropped from the filter) rather than
// rejected with 400.
func TestCovSalesSalesOrders_ListMalformedDateFiltersIgnoredNot5xx(t *testing.T) {
	t.Parallel()
	for _, param := range []string{"start_date", "end_date"} {
		param := param
		t.Run(param, func(t *testing.T) {
			t.Parallel()
			list, status, err := apiClient.GetList(salesOrdersPath, url.Values{param: {"not-a-date"}})
			require.NoError(t, err)
			require.Equal(t, 200, status)
			assert.NotEmpty(t, list.Data, "a malformed %s must not error and must not silently exclude every order", param)
		})
	}
}

func TestCovSalesSalesOrders_ListUnknownIncludeRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(salesOrdersPath, url.Values{"include": {"bogus_include_e2e"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "include[]")
}

// ──────────────────────────────────────────────
// GET /v1/sales/sales-orders/statuses — list/pagination/search/fields
// (previously only had a single include=owner presence test)
// ──────────────────────────────────────────────

func TestCovSalesSalesOrders_StatusesListBasicShape(t *testing.T) {
	t.Parallel()
	list, status, err := apiClient.GetList(covSalesSalesOrdersStatusesPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.NotEmpty(t, list.Data)

	seenCodes := map[string]bool{}
	for _, item := range list.Data {
		row := parseJSON(item)
		assert.Equal(t, "sales_order_status", jsonField(row, "object"))
		assertIDFormat(t, jsonField(row, "id"), "orss")
		assert.NotEmpty(t, jsonField(row, "code"))
		assert.NotEmpty(t, jsonField(row, "name"))
		assertValidTimestamp(t, jsonField(row, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(row, "updated_at"), "updated_at")
		assertNilField(t, row, "owner")
		seenCodes[jsonField(row, "code")] = true
	}
	assert.True(t, seenCodes["estimate"], "estimate status should be in the list")
	assert.True(t, seenCodes["issued"], "issued status should be in the list")
	assert.True(t, seenCodes["fulfilled"], "fulfilled status should be in the list")
}

func TestCovSalesSalesOrders_StatusesListPaginationAdvances(t *testing.T) {
	t.Parallel()
	assertCursorPaginationAdvances(t, covSalesSalesOrdersStatusesPath, nil)
}

func TestCovSalesSalesOrders_StatusesListSearch(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covSalesSalesOrdersStatusesPath, url.Values{"q": {"estimate"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1, "q=estimate should match exactly the estimate status")
	assert.Equal(t, "estimate", DataItemField(list.Data[0], "code"))

	noMatch, status, err := apiClient.GetList(covSalesSalesOrdersStatusesPath, url.Values{"q": {"nonexistent_status_code_e2e"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assertEmptyListData(t, noMatch.Data, "an unmatched query should return no rows")
}

func TestCovSalesSalesOrders_StatusesListInvalidLimitRejected(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		limit string
	}{
		{"Zero", "0"},
		{"TooLarge", "1001"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.GetListRaw(covSalesSalesOrdersStatusesPath, url.Values{"limit": {tc.limit}})
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
		})
	}
}

func TestCovSalesSalesOrders_StatusesIncludeOwnerShape(t *testing.T) {
	t.Parallel()
	list, status, err := apiClient.GetList(covSalesSalesOrdersStatusesPath, url.Values{"include": {"owner"}, "q": {"estimate"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1)

	row := parseJSON(list.Data[0])
	owner := jsonObject(row, "owner")
	require.NotNil(t, owner, "owner should be populated with ?include=owner")
	assert.Equal(t, "owner", jsonField(owner, "object"))
	assert.Equal(t, "system", jsonField(owner, "type"), "sales order statuses are platform-provided, so owner is always the system owner")
	assertNilField(t, owner, "account")
}

func TestCovSalesSalesOrders_StatusesUnknownIncludeRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covSalesSalesOrdersStatusesPath, url.Values{"include": {"bogus_include_e2e"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "include[]")
}
