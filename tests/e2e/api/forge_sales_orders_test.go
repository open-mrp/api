//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/id"
)

// The sales order as 1.0.forge reads and writes it. Suites that only set orders up for other resources still go
// through preview.9 (see requestVersion); these pin forge with forgeClient.

// forgeClient is apiClient pinned to 1.0.forge on every route.
func forgeClient() *Client {
	return apiClient.WithAPIVersion(defaultAPIVersion)
}

// forgeOrderLines is one line of the seed product.
func forgeOrderLines() []map[string]any {
	return []map[string]any{{"product_id": SeedProductID, "quantity": map[string]any{"value": "2", "unit_id": SeedUnitID}}}
}

// forgeOrderBody is a forge create naming only the customer and the lines; the rest comes from the customer.
func forgeOrderBody(customerID string) map[string]any {
	return map[string]any{"customer_id": customerID, "lines": forgeOrderLines()}
}

// forgeCreateOrder creates an order as forge, failing the test unless it is created, and deletes it afterwards.
func forgeCreateOrder(t *testing.T, body map[string]any, includes ...string) map[string]any {
	t.Helper()
	path := salesOrdersPath
	if len(includes) > 0 {
		path += "?" + url.Values{"include[]": includes}.Encode()
	}
	status, resp, err := forgeClient().Post(path, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	order := parseJSON(resp)
	orderID := jsonField(order, "id")
	t.Cleanup(func() { _, _, _ = forgeClient().Delete(salesOrdersPath + "/" + orderID) })
	return order
}

func forgeReadOrder(t *testing.T, orderID string, includes ...string) map[string]any {
	t.Helper()
	params := url.Values{}
	if len(includes) > 0 {
		params["include[]"] = includes
	}
	status, resp, err := forgeClient().GetListRaw(salesOrdersPath+"/"+orderID, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	return parseJSON(resp)
}

func forgeOrderAction(t *testing.T, orderID, verb string, body map[string]any) (int, []byte) {
	t.Helper()
	if body == nil {
		body = map[string]any{}
	}
	status, resp, err := forgeClient().Post(salesOrdersPath+"/"+orderID+"/actions/"+verb, body, newIdempotencyKey())
	require.NoError(t, err)
	return status, resp
}

// --- CRUD ---

func TestForgeSalesOrders_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	po := uniqueName("PO")

	order := forgeCreateOrder(t, map[string]any{
		"customer_id":                    customerID,
		"customer_purchase_order_number": po,
		"currency":                       "usd",
		"instructions":                   "Pack in double-wall cartons.",
		"bill_to":                        map[string]any{"name": "Accounts Payable", "line_1": "1 Main St", "city": "Columbus", "state": "OH", "postal_code": "43204", "country": "US", "email": "ap@so.e2e.openmrp.ai"},
		"ship_to":                        map[string]any{"address_id": SeedAddressID},
		"freight": map[string]any{
			"carrier_id": SeedCarrierID, "service_level_id": SeedServiceLevelID,
			"carrier_billing": map[string]any{"type": "third_party", "account_number": "TP-77"},
		},
		"sales_rep_id":     SeedAccountUserID,
		"payment_term_id":  SeedPaymentTermID,
		"shipping_term_id": SeedShippingTermID,
		"commitment":       map[string]any{"lead_time_override": dayLeadTime(4), "firmness": "hard"},
		"email_recipients": []map[string]any{},
		"lines":            forgeOrderLines(),
		"metadata":         map[string]any{"erp_ref": "E-1"},
	})

	orderID := jsonField(order, "id")
	assertIDFormat(t, orderID, id.OrderIDPrefix)
	assertObjectField(t, order, "sales_order")
	assert.NotEmpty(t, jsonField(order, "number"))
	assert.Equal(t, "draft", jsonField(order, "status"))
	assert.Equal(t, "usd", jsonField(order, "currency"))
	assert.Equal(t, po, jsonField(order, "customer_purchase_order_number"))
	assert.Equal(t, "Pack in double-wall cartons.", jsonField(order, "instructions"))
	assert.Equal(t, "unpaid", jsonField(order, "payment_status"))
	assert.Equal(t, "not_sent", jsonField(order, "acknowledgment_status"))
	for _, expandable := range []string{"customer", "site", "sales_rep", "created_by", "payment_term", "shipping_term", "order_discount", "email_recipients", "lines", "totals"} {
		assertNilField(t, order, expandable)
	}
	assert.Equal(t, map[string]any{
		"name": "Accounts Payable", "line_1": "1 Main St", "line_2": nil, "city": "Columbus", "state": "OH", "postal_code": "43204",
		"country": "US", "phone": nil, "email": "ap@so.e2e.openmrp.ai", "address": nil,
	}, jsonObject(order, "bill_to"))
	shipTo := jsonObject(order, "ship_to")
	require.NotNil(t, shipTo, "a saved ship-to is copied onto the order")
	assert.NotEmpty(t, jsonField(shipTo, "name"))
	assertNilField(t, shipTo, "address")
	assert.Equal(t, map[string]any{
		"carrier": nil, "service_level": nil, "carrier_billing": map[string]any{"type": "third_party", "account_number": "TP-77"},
	}, jsonObject(order, "freight"))
	commitment := jsonObject(order, "commitment")
	assert.NotContains(t, commitment, "object")
	assert.Equal(t, "4", jsonField(jsonObject(commitment, "lead_time_override"), "value"))
	assert.Equal(t, "hard", jsonField(commitment, "firmness"))
	assertNilField(t, commitment, "promised_on")
	assertNilField(t, commitment, "ship_by_override_on")
	assertNilField(t, commitment, "ship_by_on")
	assertNilField(t, commitment, "promised_lead_time")
	assertNilField(t, commitment, "promised_lead_time_source")
	assert.EqualValues(t, 2, order["line_count"], "the line and its shipping line")
	assertNilField(t, order, "related")
	assertNilField(t, order, "issued_at")
	assertNilField(t, order, "fulfilled_at")
	assertNilField(t, order, "first_shipped_at")
	assertNilField(t, order, "expires_on")
	assert.Equal(t, map[string]any{"erp_ref": "E-1"}, jsonObject(order, "metadata"))
	assertValidTimestamp(t, jsonField(order, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(order, "updated_at"), "updated_at")
	for _, gone := range []string{"note", "priority", "payment_intent_ids", "completed_at", "first_ship_at", "expired_at"} {
		assert.NotContains(t, order, gone)
	}

	// The saved address the ship-to came from is its source; the inline bill-to has none.
	read := forgeReadOrder(t, orderID, "ship_to.address", "bill_to.address", "site", "sales_rep", "customer", "freight.carrier", "freight.service_level", "payment_term", "shipping_term", "email_recipients")
	assert.Equal(t, SeedAddressID, jsonField(jsonObject(jsonObject(read, "ship_to"), "address"), "id"))
	assertNilField(t, jsonObject(read, "bill_to"), "address")
	assert.Equal(t, SeedLocationParentID, jsonField(jsonObject(read, "site"), "id"), "orders ship from the default site")
	assert.Equal(t, "account_user", jsonField(jsonObject(read, "sales_rep"), "object"))
	assert.Equal(t, SeedAccountUserID, jsonField(jsonObject(read, "sales_rep"), "id"))
	assert.Equal(t, customerID, jsonField(jsonObject(read, "customer"), "id"))
	assert.Equal(t, SeedCarrierID, jsonField(jsonObject(jsonObject(read, "freight"), "carrier"), "id"))
	assert.Equal(t, SeedServiceLevelID, jsonField(jsonObject(jsonObject(read, "freight"), "service_level"), "id"))
	assert.Equal(t, SeedPaymentTermID, jsonField(jsonObject(read, "payment_term"), "id"))
	assert.Equal(t, SeedShippingTermID, jsonField(jsonObject(read, "shipping_term"), "id"))
	assert.Empty(t, jsonListData(read, "email_recipients"), "an empty list sent stays empty")

	// Update every writable field; what isn't sent stays.
	newPO := uniqueName("PO")
	status, resp, err := forgeClient().Patch(salesOrdersPath+"/"+orderID, map[string]any{
		"customer_purchase_order_number": newPO,
		"instructions":                   nil,
		"bill_to":                        map[string]any{"address_id": SeedAddressID},
		"freight":                        map[string]any{"carrier_billing": map[string]any{"type": "sender"}},
		"sales_rep_id":                   nil,
		"commitment":                     map[string]any{"lead_time_override": nil, "promised_on": "2026-12-01", "firmness": nil},
		"acknowledgment_status":          "sent",
		"metadata":                       map[string]any{"erp_ref": nil, "batch": "B7"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	updated := parseJSON(resp)
	assert.Equal(t, newPO, jsonField(updated, "customer_purchase_order_number"))
	assertNilField(t, updated, "instructions")
	assert.NotEqual(t, "Accounts Payable", jsonField(jsonObject(updated, "bill_to"), "name"), "the bill-to is now the saved address's copy")
	assert.Equal(t, map[string]any{"type": "sender", "account_number": nil}, jsonObject(jsonObject(updated, "freight"), "carrier_billing"))
	assert.Equal(t, "sent", jsonField(updated, "acknowledgment_status"))
	c := jsonObject(updated, "commitment")
	assert.Equal(t, "2026-12-01", jsonField(c, "promised_on"))
	assertNilField(t, c, "lead_time_override")
	assertNilField(t, c, "firmness")
	assert.Equal(t, map[string]any{"batch": "B7"}, jsonObject(updated, "metadata"))
	assert.Equal(t, jsonObject(order, "ship_to"), jsonObject(updated, "ship_to"), "an address not sent is kept")
	assert.Nil(t, forgeReadOrder(t, orderID, "sales_rep")["sales_rep"], "a cleared rep reads null")
}

func TestForgeSalesOrders_OmittedFieldsCopyTheCustomerDefaults(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	contact := createContact(t, apiClient.WithAccountID(customerID), uniqueName("so-recipient"), uniqueName("so-recipient")+"@e2e-test.openmrp.ai", nil)
	contactID := jsonField(contact, "id")
	status, resp := forgeCustomerPatch(t, customerID, map[string]any{
		"instructions":            "Call before delivery.",
		"carrier_billing":         map[string]any{"type": "third_party", "account_number": "CUST-9"},
		"defaults":                map[string]any{"service_level_id": SeedServiceLevelID, "sales_rep_id": SeedAccountUser2ID},
		"notification_recipients": []map[string]any{{"account_user_id": contactID, "documents": []string{"acknowledgment", "statement"}}},
	})
	requireStatus(t, 200, status, resp)
	customer := forgeCustomerRead(t, customerID, "defaults.bill_to_address", "defaults.ship_to_address")
	defaults := jsonObject(customer, "defaults")

	order := forgeCreateOrder(t, forgeOrderBody(customerID), "freight.carrier", "freight.service_level", "payment_term", "shipping_term", "sales_rep", "email_recipients", "bill_to.address", "ship_to.address")
	assert.Equal(t, "Call before delivery.", jsonField(order, "instructions"))
	assert.Equal(t, jsonField(jsonObject(defaults, "bill_to_address"), "id"), jsonField(jsonObject(jsonObject(order, "bill_to"), "address"), "id"))
	assert.Equal(t, jsonField(jsonObject(defaults, "ship_to_address"), "id"), jsonField(jsonObject(jsonObject(order, "ship_to"), "address"), "id"))
	freight := jsonObject(order, "freight")
	assert.Equal(t, SeedCarrierID, jsonField(jsonObject(freight, "carrier"), "id"))
	assert.Equal(t, SeedServiceLevelID, jsonField(jsonObject(freight, "service_level"), "id"))
	assert.Equal(t, map[string]any{"type": "third_party", "account_number": "CUST-9"}, jsonObject(freight, "carrier_billing"))
	assert.Equal(t, SeedPaymentTermID, jsonField(jsonObject(order, "payment_term"), "id"))
	assert.Equal(t, SeedShippingTermID, jsonField(jsonObject(order, "shipping_term"), "id"))
	recipients := jsonListData(order, "email_recipients")
	require.Len(t, recipients, 1, "the customer's recipients are copied")
	r := recipients[0].(map[string]any)
	assert.Equal(t, []any{"acknowledgment"}, r["documents"], "statements aren't an order's document")
	assertNilField(t, r, "account_user")

	withUser := forgeReadOrder(t, jsonField(order, "id"), "email_recipients.account_user")
	assert.Equal(t, contactID, jsonField(jsonListData(withUser, "email_recipients")[0].(map[string]any)["account_user"].(map[string]any), "id"))

	// Changing the customer afterwards leaves the order alone.
	status, resp = forgeCustomerPatch(t, customerID, map[string]any{"instructions": "Changed."})
	requireStatus(t, 200, status, resp)
	assert.Equal(t, "Call before delivery.", jsonField(forgeReadOrder(t, jsonField(order, "id")), "instructions"))
}

func TestForgeSalesOrders_ValidationAndReferences(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)

	post := func(body map[string]any) (int, []byte) {
		status, resp, err := forgeClient().Post(salesOrdersPath, body, newIdempotencyKey())
		require.NoError(t, err)
		return status, resp
	}

	body := forgeOrderBody(customerID)
	body["currency"] = "eur"
	status, resp := post(body)
	requireStatus(t, 422, status, resp)
	requireErrorResponse(t, resp, "currency_not_supported", "invalid_request_error")

	body = forgeOrderBody(customerID)
	body["ship_to"] = map[string]any{"name": "Dock"}
	status, resp = post(body)
	requireStatus(t, 422, status, resp)
	assert.Equal(t, "ship_to.country", errorParam(resp))

	body = forgeOrderBody(customerID)
	body["ship_to"] = map[string]any{"address_id": "ad_doesnotexist0"}
	status, resp = post(body)
	requireStatus(t, 404, status, resp)
	assert.Equal(t, "ship_to.address_id", errorParam(resp))

	body = forgeOrderBody(customerID)
	body["commitment"] = map[string]any{"lead_time_override": dayLeadTime(2), "promised_on": "2026-12-01"}
	status, resp = post(body)
	requireStatus(t, 422, status, resp)

	body = forgeOrderBody(customerID)
	body["commitment"] = map[string]any{"firmness": "rigid"}
	status, resp = post(body)
	requireStatus(t, 422, status, resp)

	body = forgeOrderBody(customerID)
	body["priority_code"] = "normal"
	status, resp = post(body)
	requireStatus(t, 400, status, resp)
	requireErrorResponse(t, resp, "parameter_unknown", "invalid_request_error")
}

func TestForgeSalesOrders_Idempotency(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	key := newIdempotencyKey()
	body := forgeOrderBody(customerID)

	status, first, err := forgeClient().Post(salesOrdersPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status, first)
	orderID := jsonField(parseJSON(first), "id")
	t.Cleanup(func() { _, _, _ = forgeClient().Delete(salesOrdersPath + "/" + orderID) })

	status, second, err := forgeClient().Post(salesOrdersPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status, second)
	assert.Equal(t, orderID, jsonField(parseJSON(second), "id"))
}

// --- Lifecycle ---

func TestForgeSalesOrders_IssueUnissueCancel(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	order := forgeCreateOrder(t, forgeOrderBody(customerID))
	orderID := jsonField(order, "id")

	status, resp := forgeOrderAction(t, orderID, "issue", map[string]any{"notify_customer": false})
	requireStatus(t, 200, status, resp)
	issued := parseJSON(resp)
	assert.Equal(t, "issued", jsonField(issued, "status"))
	assertValidTimestamp(t, jsonField(issued, "issued_at"), "issued_at")
	c := jsonObject(issued, "commitment")
	assert.NotEmpty(t, jsonField(c, "ship_by_on"), "issuing stamps the ship-by date")
	assert.Equal(t, "day", jsonField(jsonObject(jsonObject(c, "promised_lead_time"), "unit"), "id"))
	assert.Equal(t, "customer", jsonField(c, "promised_lead_time_source"))

	status, resp = forgeOrderAction(t, orderID, "issue", nil)
	requireStatus(t, 409, status, resp)
	requireErrorResponse(t, resp, "invalid_state_transition", "invalid_request_error")

	related := jsonObject(forgeReadOrder(t, orderID, "related.pick"), "related")
	require.NotNil(t, related, "the pick is related once issued")
	assert.Equal(t, "pick", jsonField(jsonObject(related, "pick"), "type"))

	status, resp = forgeOrderAction(t, orderID, "unissue", nil)
	requireStatus(t, 200, status, resp)
	unissued := parseJSON(resp)
	assert.Equal(t, "draft", jsonField(unissued, "status"))
	assertNilField(t, jsonObject(unissued, "commitment"), "ship_by_on")

	status, resp = forgeOrderAction(t, orderID, "cancel", nil)
	requireStatus(t, 409, status, resp)
	requireErrorResponse(t, resp, "invalid_state_transition", "invalid_request_error")

	status, resp = forgeOrderAction(t, orderID, "issue", nil)
	requireStatus(t, 200, status, resp)
	status, resp = forgeOrderAction(t, orderID, "cancel", nil)
	requireStatus(t, 200, status, resp)
	canceled := parseJSON(resp)
	assert.Equal(t, "canceled", jsonField(canceled, "status"), "nothing shipped")
	assertValidTimestamp(t, jsonField(canceled, "fulfilled_at"), "fulfilled_at")

	// A customer can't be changed once the order is issued.
	other := setupOrderCustomer(t)
	status, resp, err := forgeClient().Patch(salesOrdersPath+"/"+orderID, map[string]any{"customer_id": other}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, resp)
	assert.Equal(t, "customer_id", errorParam(resp))
}

func TestForgeSalesOrders_PreviewStatusRoutesAreGoneOnForge(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	orderID := jsonField(forgeCreateOrder(t, forgeOrderBody(customerID)), "id")

	status, resp, err := forgeClient().Put(salesOrdersPath+"/"+orderID+"/actions/issue", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 404, status, resp)

	status, resp, err = forgeClient().GetListRaw(salesOrdersPath+"/statuses", nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, resp)
}

// --- Delete ---

func TestForgeSalesOrders_DeleteReturnsStubAndRefusesShipped(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	orderID := jsonField(forgeCreateOrder(t, forgeOrderBody(customerID)), "id")

	status, resp, err := forgeClient().Delete(salesOrdersPath + "/" + orderID)
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	assert.Equal(t, map[string]any{"id": orderID, "object": "sales_order", "deleted": true}, parseJSON(resp))

	status, resp, err = forgeClient().GetListRaw(salesOrdersPath+"/"+orderID, nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, resp)

	// The seed order was invoiced.
	status, resp, err = forgeClient().Delete(salesOrdersPath + "/" + SeedFulfilledPaidOrderID)
	require.NoError(t, err)
	requireStatus(t, 409, status, resp)
	assert.Contains(t, jsonField(requireErrorResponse(t, resp, "resource_in_use", "invalid_request_error"), "message"), "invoices")
}

// --- List ---

func TestForgeSalesOrders_ListFilters(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	po := uniqueName("PO")
	body := forgeOrderBody(customerID)
	body["customer_purchase_order_number"] = po
	order := forgeCreateOrder(t, body)
	orderID := jsonField(order, "id")

	list := func(params url.Values) []any {
		t.Helper()
		status, resp, err := forgeClient().GetListRaw(salesOrdersPath, params)
		require.NoError(t, err)
		requireStatus(t, 200, status, resp)
		return parseJSON(resp)["data"].([]any)
	}
	ids := func(data []any) []string {
		out := make([]string, 0, len(data))
		for _, o := range data {
			out = append(out, jsonField(o.(map[string]any), "id"))
		}
		return out
	}

	assert.Equal(t, []string{orderID}, ids(list(url.Values{"q": {po}})), "q is an exact match on the customer PO")
	assert.Equal(t, []string{orderID}, ids(list(url.Values{"q": {jsonField(order, "number")}, "customer_ids[]": {customerID}})), "q is an exact match on the number")
	assert.Empty(t, list(url.Values{"q": {po[:len(po)-1]}}), "q is not a prefix match")
	assert.Equal(t, []string{orderID}, ids(list(url.Values{"customer_ids[]": {customerID}, "statuses[]": {"draft"}})))
	assert.Empty(t, list(url.Values{"customer_ids[]": {customerID}, "statuses[]": {"issued", "canceled"}}))
	assert.Equal(t, []string{orderID}, ids(list(url.Values{"customer_ids[]": {customerID}, "group_ids[]": {SeedCustomerGroupID}})))
	assert.Equal(t, []string{orderID}, ids(list(url.Values{"customer_ids[]": {customerID}, "payment_statuses[]": {"unpaid"}})))
	assert.Empty(t, list(url.Values{"customer_ids[]": {customerID}, "payment_statuses[]": {"paid"}}))
	assert.Empty(t, list(url.Values{"customer_ids[]": {customerID}, "created_before": {jsonField(order, "created_at")}}), "created_before is exclusive")
	assert.Equal(t, []string{orderID}, ids(list(url.Values{"customer_ids[]": {customerID}, "created_after": {jsonField(order, "created_at")}})))

	status, resp, err := forgeClient().GetListRaw(salesOrdersPath, url.Values{"statuses[]": {"estimate"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, resp)
	assert.Equal(t, "statuses", errorParam(resp))
}
