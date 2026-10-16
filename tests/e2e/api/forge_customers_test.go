//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// forgeCustomer creates a customer from validCustomerBody plus extra and returns it as created.
func forgeCustomer(t *testing.T, prefix string, extra map[string]any) map[string]any {
	t.Helper()
	body := validCustomerBody(uniqueName(prefix))
	for k, v := range extra {
		if k == "defaults" {
			for dk, dv := range v.(map[string]any) {
				customerDefaults(body)[dk] = dv
			}
			continue
		}
		body[k] = v
	}
	return createAndCleanup(t, customersPath, body)
}

func forgeCustomerRead(t *testing.T, customerID string, includes ...string) map[string]any {
	t.Helper()
	params := url.Values{}
	for _, inc := range includes {
		params.Add("include[]", inc)
	}
	status, body, err := apiClient.GetListRaw(customersPath+"/"+customerID, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func forgeCustomerPatch(t *testing.T, customerID string, body map[string]any, includes ...string) (int, []byte) {
	t.Helper()
	path := customersPath + "/" + customerID
	if len(includes) > 0 {
		params := url.Values{}
		for _, inc := range includes {
			params.Add("include[]", inc)
		}
		path += "?" + params.Encode()
	}
	status, resp, err := apiClient.Patch(path, body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "a customer update must not 5xx: %s", resp)
	return status, resp
}

// --- CRUD ---

func TestForgeCustomers_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-fc-all")
	number := uniqueName("FC")
	body := validCustomerBody(name)
	body["number"] = number
	body["standing"] = "hold_shipment"
	body["instructions"] = "Send invoices to accounts payable."
	body["commission_policy"] = "applied"
	body["freight_policy"] = "free"
	body["credit_limit"] = map[string]any{"amount": "2500.00", "currency": "usd"}
	body["contact"] = map[string]any{"email": "ap@fc.e2e.openmrp.ai", "phone": "555-0100", "url": "https://fc.e2e.openmrp.ai"}
	body["carrier_billing"] = map[string]any{"type": "third_party", "account_number": "TP-1"}
	defaults := customerDefaults(body)
	defaults["service_level_id"] = SeedServiceLevelID
	defaults["sales_rep_id"] = SeedAccountUserID
	defaults["lead_time"] = dayLeadTime(7)
	defaults["supply"] = "make"

	created := createAndCleanup(t, customersPath, body)
	customerID := jsonField(created, "id")
	assertIDFormat(t, customerID, id.AccountIDPrefix)
	requireDefaultSite(t, customerID, name)
	assertObjectField(t, created, "customer")
	assert.Equal(t, "active", jsonField(created, "status"))
	assert.Equal(t, "hold_shipment", jsonField(created, "standing"))
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, number, jsonField(created, "number"))
	assert.Equal(t, "standalone", jsonField(created, "relationship_type"))
	assert.Equal(t, "Send invoices to accounts payable.", jsonField(created, "instructions"))
	assert.Equal(t, "applied", jsonField(created, "commission_policy"))
	assert.Equal(t, "free", jsonField(created, "freight_policy"))
	assert.Equal(t, map[string]any{"amount": "2500.00", "currency": "usd"}, jsonObject(created, "credit_limit"))
	assert.Equal(t, map[string]any{"email": "ap@fc.e2e.openmrp.ai", "phone": "555-0100", "url": "https://fc.e2e.openmrp.ai"}, jsonObject(created, "contact"))
	assert.Equal(t, map[string]any{"type": "third_party", "account_number": "TP-1"}, jsonObject(created, "carrier_billing"))
	gotDefaults := jsonObject(created, "defaults")
	require.NotNil(t, gotDefaults, "defaults is always present")
	assert.NotContains(t, gotDefaults, "object")
	for _, ref := range []string{"carrier", "service_level", "payment_term", "shipping_term", "sales_rep", "bill_to_address", "ship_to_address"} {
		assert.Contains(t, gotDefaults, ref)
		assert.Nil(t, gotDefaults[ref], "defaults.%s is null unless included", ref)
	}
	leadTime := jsonObject(gotDefaults, "lead_time")
	assert.Equal(t, "7", jsonField(leadTime, "value"))
	assert.Equal(t, "day", jsonField(jsonObject(leadTime, "unit"), "id"))
	assert.Equal(t, "make", jsonField(gotDefaults, "supply"))
	for _, ref := range []string{"group", "parent_customer", "child_customers", "notification_recipients"} {
		assertNilField(t, created, ref)
	}
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
	for _, gone := range []string{"note", "contact_info", "freight_preferences", "type", "parent_account", "child_accounts", "notification_preferences", "bill_to_address"} {
		assert.NotContains(t, created, gone)
	}

	// Section fields merge: only what is sent changes.
	status, resp := forgeCustomerPatch(t, customerID, map[string]any{
		"standing":        "normal",
		"status":          "archived",
		"contact":         map[string]any{"phone": "555-0199"},
		"carrier_billing": map[string]any{"type": "sender"},
		"defaults":        map[string]any{"lead_time": nil, "supply": "stock", "sales_rep_id": nil},
		"instructions":    nil,
	})
	requireStatus(t, 200, status, resp)
	updated := parseJSON(resp)
	assert.Equal(t, "archived", jsonField(updated, "status"))
	assert.Equal(t, "normal", jsonField(updated, "standing"))
	assertNilField(t, updated, "instructions")
	assert.Equal(t, map[string]any{"email": "ap@fc.e2e.openmrp.ai", "phone": "555-0199", "url": "https://fc.e2e.openmrp.ai"}, jsonObject(updated, "contact"))
	assert.Equal(t, map[string]any{"type": "sender", "account_number": "TP-1"}, jsonObject(updated, "carrier_billing"))
	updDefaults := jsonObject(updated, "defaults")
	assertNilField(t, updDefaults, "lead_time")
	assert.Equal(t, "stock", jsonField(updDefaults, "supply"))
	assert.Equal(t, "applied", jsonField(updated, "commission_policy"), "a field the update left out keeps its value")
	assert.Equal(t, map[string]any{"amount": "2500.00", "currency": "usd"}, jsonObject(updated, "credit_limit"))

	withRefs := forgeCustomerRead(t, customerID, "defaults.carrier", "defaults.service_level", "defaults.payment_term", "defaults.shipping_term", "defaults.sales_rep", "group")
	refDefaults := jsonObject(withRefs, "defaults")
	assert.Equal(t, SeedCarrierID, jsonField(jsonObject(refDefaults, "carrier"), "id"))
	assert.Equal(t, SeedServiceLevelID, jsonField(jsonObject(refDefaults, "service_level"), "id"))
	assert.Equal(t, SeedPaymentTermID, jsonField(jsonObject(refDefaults, "payment_term"), "id"))
	assert.Equal(t, SeedShippingTermID, jsonField(jsonObject(refDefaults, "shipping_term"), "id"))
	assertNilField(t, refDefaults, "sales_rep")
	assert.Equal(t, SeedCustomerGroupID, jsonField(jsonObject(withRefs, "group"), "id"))
	assert.Equal(t, "carrier", jsonField(jsonObject(refDefaults, "carrier"), "object"))
	assert.Equal(t, "account_group", jsonField(jsonObject(withRefs, "group"), "object"))
}

func TestForgeCustomers_OmittedFieldsTakeDefaults(t *testing.T) {
	t.Parallel()

	created := forgeCustomer(t, "e2e-fc-omit", nil)
	assert.Equal(t, "active", jsonField(created, "status"))
	assert.Equal(t, "normal", jsonField(created, "standing"))
	assert.Equal(t, "exempt", jsonField(created, "commission_policy"))
	assert.Equal(t, "billed", jsonField(created, "freight_policy"))
	assertNilField(t, created, "instructions")
	assertNilField(t, created, "credit_limit")
	assert.Equal(t, map[string]any{"email": nil, "phone": nil, "url": nil}, jsonObject(created, "contact"))
	assert.Equal(t, map[string]any{"type": "sender", "account_number": nil}, jsonObject(created, "carrier_billing"))
	defaults := jsonObject(created, "defaults")
	assertNilField(t, defaults, "lead_time")
	assertNilField(t, defaults, "supply")
	assert.NotEmpty(t, jsonField(created, "number"), "a number is assigned when omitted")
}

func TestForgeCustomers_CreateValidation(t *testing.T) {
	t.Parallel()

	missingDefaults := validCustomerBody(uniqueName("e2e-fc-val"))
	delete(missingDefaults, "defaults")
	status, body, err := apiClient.Post(customersPath, missingDefaults, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)

	badUnit := validCustomerBody(uniqueName("e2e-fc-val"))
	customerDefaults(badUnit)["lead_time"] = map[string]any{"value": "3", "unit_id": SeedUnitID}
	status, body, err = apiClient.Post(customersPath, badUnit, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	assert.Equal(t, "defaults.lead_time.unit_id", errorParam(body))

	partial := validCustomerBody(uniqueName("e2e-fc-val"))
	customerDefaults(partial)["lead_time"] = dayLeadTime(1)
	customerDefaults(partial)["lead_time"].(map[string]any)["value"] = "1.5"
	status, body, err = apiClient.Post(customersPath, partial, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	assert.Equal(t, "defaults.lead_time.value", errorParam(body))

	preferred := validCustomerBody(uniqueName("e2e-fc-val"))
	preferred["standing"] = "preferred"
	status, body, err = apiClient.Post(customersPath, preferred, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)

	badCarrier := validCustomerBody(uniqueName("e2e-fc-val"))
	customerDefaults(badCarrier)["carrier_id"] = "car_doesnotexist0000"
	status, body, err = apiClient.Post(customersPath, badCarrier, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	assert.Equal(t, "defaults.carrier_id", errorParam(body))
}

func TestForgeCustomers_CreateIdempotent(t *testing.T) {
	t.Parallel()

	key := newIdempotencyKey()
	body := validCustomerBody(uniqueName("e2e-fc-idem"))
	status, first, err := apiClient.Post(customersPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status, first)
	firstID := jsonField(parseJSON(first), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(customersPath + "/" + firstID) })

	status, second, err := apiClient.Post(customersPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status, second)
	assert.Equal(t, firstID, jsonField(parseJSON(second), "id"))
}

func TestForgeCustomers_DefaultAddressesAndIncludes(t *testing.T) {
	t.Parallel()

	created := forgeCustomer(t, "e2e-fc-addr", nil)
	id := jsonField(created, "id")
	got := forgeCustomerRead(t, id, "defaults.bill_to_address", "defaults.ship_to_address")
	defaults := jsonObject(got, "defaults")
	bill := jsonObject(defaults, "bill_to_address")
	require.NotNil(t, bill)
	assert.Equal(t, "address", jsonField(bill, "object"))
	assert.Equal(t, "US", jsonField(bill, "country"))
	ship := jsonObject(defaults, "ship_to_address")
	require.NotNil(t, ship)

	// A new default must be one of the customer's own addresses.
	addrStatus, addrBody, err := apiClient.WithAccountID(id).Post(addressesPath, map[string]any{"name": uniqueName("e2e-fc-addr-new"), "country": "US"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, addrStatus, addrBody)
	addrID := jsonField(parseJSON(addrBody), "id")
	status, resp := forgeCustomerPatch(t, id, map[string]any{"defaults": map[string]any{"ship_to_address_id": addrID}}, "defaults.ship_to_address")
	requireStatus(t, 200, status, resp)
	assert.Equal(t, addrID, jsonField(jsonObject(jsonObject(parseJSON(resp), "defaults"), "ship_to_address"), "id"))

	status, resp = forgeCustomerPatch(t, id, map[string]any{"defaults": map[string]any{"bill_to_address_id": SeedAddressID}})
	assert.Equal(t, 404, status, "another account's address is refused: %s", resp)
	assert.Equal(t, "defaults.bill_to_address_id", errorParam(resp))
}

// --- Hierarchy ---

func TestForgeCustomers_ParentCustomer(t *testing.T) {
	t.Parallel()

	parent := forgeCustomer(t, "e2e-fc-parent", nil)
	parentID := jsonField(parent, "id")
	child := forgeCustomer(t, "e2e-fc-child", map[string]any{"parent_customer_id": parentID})
	childID := jsonField(child, "id")
	assert.Equal(t, "child", jsonField(child, "relationship_type"))

	gotParent := forgeCustomerRead(t, parentID, "child_customers")
	assert.Equal(t, "parent", jsonField(gotParent, "relationship_type"))
	children := jsonListData(gotParent, "child_customers")
	require.Len(t, children, 1)
	assert.Equal(t, childID, jsonField(children[0].(map[string]any), "id"))

	gotChild := forgeCustomerRead(t, childID, "parent_customer")
	assert.Equal(t, parentID, jsonField(jsonObject(gotChild, "parent_customer"), "id"))

	// A customer can't be placed under itself or its own child.
	status, resp := forgeCustomerPatch(t, parentID, map[string]any{"parent_customer_id": childID})
	assert.Equal(t, 422, status, "%s", resp)
	assert.Equal(t, "parent_customer_id", errorParam(resp))
	status, resp = forgeCustomerPatch(t, parentID, map[string]any{"parent_customer_id": parentID})
	assert.Equal(t, 422, status, "%s", resp)

	// The parent can't be deleted while a child is under it.
	delStatus, delBody, err := apiClient.Delete(customersPath + "/" + parentID)
	require.NoError(t, err)
	requireStatus(t, 409, delStatus, delBody)
	assert.Contains(t, jsonField(requireErrorResponse(t, delBody, "resource_in_use", "invalid_request_error"), "message"), "child customers")

	status, resp = forgeCustomerPatch(t, childID, map[string]any{"parent_customer_id": nil})
	requireStatus(t, 200, status, resp)
	assert.Equal(t, "standalone", jsonField(parseJSON(resp), "relationship_type"))
}

func TestForgeCustomers_ArchivedParentIsRefused(t *testing.T) {
	t.Parallel()

	parent := forgeCustomer(t, "e2e-fc-arch-parent", map[string]any{"status": "archived"})
	body := validCustomerBody(uniqueName("e2e-fc-arch-child"))
	body["parent_customer_id"] = jsonField(parent, "id")
	status, resp, err := apiClient.Post(customersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, resp)
	assertErrorParam(t, requireErrorResponse(t, resp, "resource_archived", "invalid_request_error"), "parent_customer_id")
}

// --- Lifecycle ---

func TestForgeCustomers_ArchivedCustomerTakesNoNewOrders(t *testing.T) {
	t.Parallel()

	customerID := setupOrderCustomer(t)
	status, resp := forgeCustomerPatch(t, customerID, map[string]any{"status": "archived"})
	requireStatus(t, 200, status, resp)

	orderStatus, orderBody, err := forgeClient().Post(salesOrdersPath, forgeOrderBody(customerID), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, orderStatus, orderBody)
	assertErrorParam(t, requireErrorResponse(t, orderBody, "resource_archived", "invalid_request_error"), "customer_id")

	status, resp = forgeCustomerPatch(t, customerID, map[string]any{"status": "active"})
	requireStatus(t, 200, status, resp)
}

func TestForgeCustomers_DeleteReturnsStubAndRefusesInUse(t *testing.T) {
	t.Parallel()

	free := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-fc-del")))
	freeID := jsonField(free, "id")
	status, body, err := apiClient.Delete(customersPath + "/" + freeID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	stub := parseJSON(body)
	assert.Equal(t, freeID, jsonField(stub, "id"))
	assert.Equal(t, "customer", jsonField(stub, "object"))
	assert.Equal(t, "true", jsonField(stub, "deleted"))
	getStatus, getBody, err := apiClient.GetListRaw(customersPath+"/"+freeID, nil)
	require.NoError(t, err)
	assert.Contains(t, []int{404, 410}, getStatus, "%s", getBody)

	ordering := setupOrderCustomer(t)
	orderStatus, orderBody, err := apiClient.Post(salesOrdersPath, minimalSalesOrderCreateBody(t, ordering), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, orderStatus, orderBody)
	orderID := jsonField(parseJSON(orderBody), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(salesOrdersPath + "/" + orderID) })

	status, body, err = apiClient.Delete(customersPath + "/" + ordering)
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	assert.Contains(t, jsonField(requireErrorResponse(t, body, "resource_in_use", "invalid_request_error"), "message"), "sales orders")
}

func TestForgeCustomers_MergeRunsAsAJob(t *testing.T) {
	t.Parallel()

	target := forgeCustomer(t, "e2e-fc-merge-tgt", nil)
	targetID := jsonField(target, "id")
	source := forgeCustomer(t, "e2e-fc-merge-src", nil)
	sourceID := jsonField(source, "id")

	status, body, err := apiClient.Post(customersPath+"/"+targetID+"/actions/merge", map[string]any{"source_customer_ids": []string{sourceID}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, body)
	accepted := parseJSON(body)
	assert.Equal(t, "async_job", jsonField(accepted, "object"))

	job := pollJobUntilTerminal(t, jsonField(accepted, "id"))
	require.Equal(t, "completed", jsonField(job, "status"), "%v", job)
	results := jsonListData(job, "results")
	require.Len(t, results, 1)
	assert.Equal(t, targetID, jsonField(jsonObject(results[0].(map[string]any), "resource"), "id"))

	getStatus, getBody, err := apiClient.GetListRaw(customersPath+"/"+sourceID, nil)
	require.NoError(t, err)
	assert.Contains(t, []int{404, 410}, getStatus, "the source is deleted: %s", getBody)

	bad, badBody, err := apiClient.Post(customersPath+"/"+targetID+"/actions/merge", map[string]any{"source_customer_ids": []string{targetID}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, bad, badBody)
}

// --- Notification recipients ---

func TestForgeCustomers_NotificationRecipients(t *testing.T) {
	t.Parallel()

	created := forgeCustomer(t, "e2e-fc-recip", nil)
	id := jsonField(created, "id")
	name, email := contactEmail("e2e-fc-recip-user")
	contact := createContact(t, apiClient.WithAccountID(id), name, email, nil)
	contactID := jsonField(contact, "id")

	status, resp := forgeCustomerPatch(t, id, map[string]any{
		"notification_recipients": []map[string]any{{"account_user_id": contactID, "documents": []string{"acknowledgment", "invoice", "statement"}}},
	}, "notification_recipients.account_user")
	requireStatus(t, 200, status, resp)
	recipients := jsonListData(parseJSON(resp), "notification_recipients")
	require.Len(t, recipients, 1)
	recipient := recipients[0].(map[string]any)
	assert.ElementsMatch(t, []any{"acknowledgment", "invoice", "statement"}, jsonArray(recipient, "documents"))
	assert.Equal(t, contactID, jsonField(jsonObject(recipient, "account_user"), "id"))

	withoutUser := forgeCustomerRead(t, id, "notification_recipients")
	recipients = jsonListData(withoutUser, "notification_recipients")
	require.Len(t, recipients, 1)
	assertNilField(t, recipients[0].(map[string]any), "account_user")
	assertNilField(t, forgeCustomerRead(t, id), "notification_recipients")

	status, resp = forgeCustomerPatch(t, id, map[string]any{
		"notification_recipients": []map[string]any{{"account_user_id": SeedAccountUserID, "documents": []string{"invoice"}}},
	})
	assert.Equal(t, 404, status, "a user of another account is refused: %s", resp)
	assert.Equal(t, "notification_recipients[0].account_user_id", errorParam(resp))

	status, resp = forgeCustomerPatch(t, id, map[string]any{"notification_recipients": []map[string]any{}}, "notification_recipients")
	requireStatus(t, 200, status, resp)
	assert.Empty(t, jsonListData(parseJSON(resp), "notification_recipients"))
}

// --- Lead time ---

func TestForgeCustomers_LeadTimeResolution(t *testing.T) {
	t.Parallel()

	created := forgeCustomer(t, "e2e-fc-lead", map[string]any{"defaults": map[string]any{"lead_time": dayLeadTime(9)}})
	id := jsonField(created, "id")
	status, body, err := apiClient.GetListRaw(customersPath+"/"+id+"/lead-time", url.Values{"include[]": {"customer"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, "customer_lead_time", jsonField(got, "object"))
	assert.Equal(t, "customer", jsonField(got, "source"))
	assert.Equal(t, "9", jsonField(jsonObject(got, "lead_time"), "value"))
	assert.Equal(t, "day", jsonField(jsonObject(jsonObject(got, "lead_time"), "unit"), "id"))
	assert.Equal(t, id, jsonField(jsonObject(got, "customer"), "id"))
	assertNilField(t, got, "account_group")
	assertNilField(t, got, "parent_customer")
}

// --- List ---

func TestForgeCustomers_ListFilters(t *testing.T) {
	t.Parallel()

	marker := uniqueName("e2e-fc-list")
	parent := forgeCustomer(t, "e2e-fc-list-p", map[string]any{"name": marker + " parent", "standing": "hold_all"})
	parentID := jsonField(parent, "id")
	child := forgeCustomer(t, "e2e-fc-list-c", map[string]any{"name": marker + " child", "parent_customer_id": parentID, "status": "archived"})
	childID := jsonField(child, "id")

	ids := func(params url.Values) []string {
		params.Set("q", marker)
		params.Set("limit", "100")
		list, _, err := apiClient.GetList(customersPath, params)
		require.NoError(t, err)
		var out []string
		for _, item := range list.Data {
			out = append(out, DataItemField(item, "id"))
		}
		return out
	}
	assert.ElementsMatch(t, []string{parentID, childID}, ids(url.Values{}))
	assert.Equal(t, []string{childID}, ids(url.Values{"statuses[]": {"archived"}}))
	assert.Equal(t, []string{parentID}, ids(url.Values{"statuses[]": {"active"}}))
	assert.Equal(t, []string{parentID}, ids(url.Values{"standings[]": {"hold_all"}}))
	assert.Equal(t, []string{childID}, ids(url.Values{"relationship_types[]": {"child"}}))
	assert.Equal(t, []string{parentID}, ids(url.Values{"relationship_types[]": {"parent"}}))
	assert.Empty(t, ids(url.Values{"relationship_types[]": {"standalone"}}))
	assert.ElementsMatch(t, []string{parentID, childID}, ids(url.Values{"group_ids[]": {SeedCustomerGroupID}}))
	assert.Empty(t, ids(url.Values{"created_before": {"2001-01-01T00:00:00Z"}}))

	byNumber, _, err := apiClient.GetList(customersPath, url.Values{"q": {jsonField(child, "number")}})
	require.NoError(t, err)
	found := false
	for _, item := range byNumber.Data {
		found = found || DataItemField(item, "id") == childID
	}
	assert.True(t, found, "q matches the number")

	empty, _, err := apiClient.GetList(customersPath, url.Values{"q": {uniqueName("e2e-fc-none")}})
	require.NoError(t, err)
	assertEmptyListData(t, empty.Data, "a search matching nothing")
}

// --- Account groups ---

func TestForgeAccountGroups_DefaultsAndInUse(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(accountGroupsPath, map[string]any{
		"name":     uniqueName("e2e-fag"),
		"defaults": map[string]any{"lead_time": dayLeadTime(4)},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	group := parseJSON(body)
	groupID := jsonField(group, "id")
	assertIDFormat(t, groupID, id.AccountGroupIDPrefix)
	assert.Equal(t, "4", jsonField(jsonObject(jsonObject(group, "defaults"), "lead_time"), "value"))
	assert.NotContains(t, group, "default_lead_time_days")

	status, body, err = apiClient.Patch(accountGroupsPath+"/"+groupID, map[string]any{"defaults": map[string]any{"lead_time": nil}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, jsonObject(parseJSON(body), "defaults"), "lead_time")

	customer := forgeCustomer(t, "e2e-fag-cust", map[string]any{"group_id": groupID})
	status, body, err = apiClient.Delete(accountGroupsPath + "/" + groupID)
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	requireErrorResponse(t, body, "resource_in_use", "invalid_request_error")

	status, body = forgeCustomerPatch(t, jsonField(customer, "id"), map[string]any{"group_id": SeedCustomerGroupID})
	requireStatus(t, 200, status, body)
	status, body, err = apiClient.Delete(accountGroupsPath + "/" + groupID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "true", jsonField(parseJSON(body), "deleted"))
}

// --- preview.9 ---

func TestForgeCustomers_Preview9KeepsItsShape(t *testing.T) {
	t.Parallel()

	body := map[string]any{
		"name": uniqueName("e2e-fc-p9"), "status": "hold_all", "note": "p9 note",
		"default_carrier_id": SeedCarrierID, "default_payment_term_id": SeedPaymentTermID, "default_shipping_term_id": SeedShippingTermID,
		"customer_type_group_id": SeedCustomerGroupID, "lead_time_days": 3, "fulfillment_policy": "make_to_order",
		"email": "p9@fc.e2e.openmrp.ai", "carrier_billing_type": "sender", "default_priority": "normal",
		"bill_to_address": map[string]any{"name": "Bill", "country": "US"},
		"ship_to_address": map[string]any{"name": "Ship", "country": "US"},
	}
	created := createAndCleanupP9(t, customersPath, body)
	id := jsonField(created, "id")
	assert.Equal(t, "hold_all", jsonField(created, "status"))
	assert.Equal(t, "p9 note", jsonField(created, "note"))

	status, resp, err := p9Client().GetListRaw(customersPath+"/"+id, url.Values{"include": {"type,freight_preferences.carrier,bill_to_address,contact_info,defaults"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	got := parseJSON(resp)
	assert.Equal(t, SeedCustomerGroupID, jsonField(jsonObject(got, "type"), "id"))
	assert.Equal(t, "p9@fc.e2e.openmrp.ai", jsonField(jsonObject(got, "contact_info"), "email"))
	freight := jsonObject(got, "freight_preferences")
	assert.Equal(t, "billed_freight", jsonField(freight, "status"))
	assert.Equal(t, SeedCarrierID, jsonField(jsonObject(freight, "carrier"), "id"))
	defaults := jsonObject(got, "defaults")
	assert.Equal(t, "3", jsonField(defaults, "lead_time_days"))
	assert.Equal(t, "make_to_order", jsonField(defaults, "fulfillment_policy"))
	assertNilField(t, defaults, "priority")
	assert.NotNil(t, jsonObject(got, "bill_to_address"))

	forge := forgeCustomerRead(t, id)
	assert.Equal(t, "hold_all", jsonField(forge, "standing"))
	assert.Equal(t, "make", jsonField(jsonObject(forge, "defaults"), "supply"))
}
