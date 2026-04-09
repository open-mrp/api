//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Helpers ────────────────────────────────────

// expectAuditEvent polls the audit events endpoint until an event matching the
// given resource_id, resource_type, and action appears (or the timeout expires).
func expectAuditEvent(t *testing.T, resourceID, resourceType, action string) {
	t.Helper()
	eventually(t, 30*time.Second, 1*time.Second, func() error {
		list, _, err := apiClient.GetList(auditEventsPath, url.Values{
			"resource_id":   {resourceID},
			"resource_type": {resourceType},
			"action":        {action},
		})
		if err != nil {
			return err
		}
		if len(list.Data) == 0 {
			return fmt.Errorf("no %s audit event yet for %s %s", action, resourceType, resourceID)
		}
		return nil
	})
}

// expectAuditEventWithChanges polls audit events until one matching the resource
// and action appears, then returns the full event payload with include=changes.
func expectAuditEventWithChanges(t *testing.T, resourceID, resourceType, action string) map[string]any {
	t.Helper()

	var matched map[string]any
	eventually(t, 30*time.Second, 1*time.Second, func() error {
		status, body, err := apiClient.GetListRaw(auditEventsPath, url.Values{
			"resource_id":   {resourceID},
			"resource_type": {resourceType},
			"action":        {action},
			"include":       {"changes"},
			"limit":         {"25"},
		})
		if err != nil {
			return err
		}
		if status != 200 {
			return fmt.Errorf("unexpected status %d while querying audit events", status)
		}

		resp := parseJSON(body)
		for _, item := range jsonArray(resp, "data") {
			event, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if jsonField(event, "resource_id") != resourceID ||
				jsonField(event, "resource_type") != resourceType ||
				jsonField(event, "action") != action {
				continue
			}

			if _, ok := event["changes"]; !ok {
				return fmt.Errorf("event found but changes not included yet")
			}

			matched = event
			return nil
		}

		return fmt.Errorf("no %s audit event yet for %s %s", action, resourceType, resourceID)
	})

	return matched
}

func changeForField(changes []any, field string) (map[string]any, bool) {
	for _, item := range changes {
		change, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if jsonField(change, "field") == field {
			return change, true
		}
	}
	return nil, false
}

// expectRequestLog polls the request logs endpoint until a log matching the
// given method, status code, and path appears (or the timeout expires).
//
// Request logs go through a longer pipeline than audit events (goroutine →
// outbox → enqueuer poll → RabbitMQ → consumer), so we use a longer timeout.
func expectRequestLog(t *testing.T, method, statusCode, path string) {
	t.Helper()
	eventually(t, 30*time.Second, 1*time.Second, func() error {
		list, _, err := apiClient.GetList(requestLogsPath, url.Values{
			"method":      {method},
			"status_code": {statusCode},
			"limit":       {"100"},
		})
		if err != nil {
			return err
		}
		for _, item := range list.Data {
			m := parseJSON(item)
			if jsonField(m, "path") == path {
				return nil
			}
		}
		return fmt.Errorf("no %s %s request log yet for %s", method, statusCode, path)
	})
}

// ── Account Groups ─────────────────────────────

func TestAccountGroups_AuditEvents(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": uniqueName("e2e-acgrp-audit"),
		"type": "type_group",
	})
	id := jsonField(created, "id")

	expectAuditEvent(t, id, "account_group", "create")

	_, patchBody, err := apiClient.Patch(accountGroupsPath+"/"+id, map[string]any{
		"name": uniqueName("e2e-acgrp-audit-u"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotNil(t, patchBody)

	expectAuditEvent(t, id, "account_group", "update")
}

func TestAccountGroups_RequestLogs(t *testing.T) {
	t.Parallel()
	createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": uniqueName("e2e-acgrp-rlog"),
		"type": "type_group",
	})

	expectRequestLog(t, "POST", "201", accountGroupsPath)
}

// ── Customers ──────────────────────────────────

func TestCustomers_AuditEvents(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cust-audit")
	created := createAndCleanup(t, customersPath, validCustomerBody(name))
	id := jsonField(created, "id")

	expectAuditEvent(t, id, "customer", "create")

	_, patchBody, err := apiClient.Patch(customersPath+"/"+id, map[string]any{
		"note": "audit test update",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotNil(t, patchBody)

	expectAuditEvent(t, id, "customer", "update")
}

func TestCustomers_AuditEvents_Changes(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-cust-audit-chg")
	createStatus, createBody, err := apiClient.Post(customersPath, validCustomerBody(name), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)

	createdEvent := expectAuditEventWithChanges(t, id, "customer", "create")
	createChanges := jsonArray(createdEvent, "changes")
	require.NotEmpty(t, createChanges, "create audit event should include changes")

	nameCreateChange, ok := changeForField(createChanges, "name")
	require.True(t, ok, "create audit event should include a name change")
	assert.Nil(t, nameCreateChange["old_value"])
	assert.Equal(t, name, jsonField(nameCreateChange, "new_value"))

	note := "audit change test note"
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+id, map[string]any{
		"note": note,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updatedEvent := expectAuditEventWithChanges(t, id, "customer", "update")
	updateChanges := jsonArray(updatedEvent, "changes")
	require.NotEmpty(t, updateChanges, "update audit event should include changes")

	noteUpdateChange, ok := changeForField(updateChanges, "note")
	require.True(t, ok, "update audit event should include a note change")
	assert.Nil(t, noteUpdateChange["old_value"])
	assert.Equal(t, note, jsonField(noteUpdateChange, "new_value"))

	delStatus, delBody, err := apiClient.Delete(customersPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	deletedEvent := expectAuditEventWithChanges(t, id, "customer", "delete")
	deleteChanges := jsonArray(deletedEvent, "changes")
	require.NotEmpty(t, deleteChanges, "delete audit event should include changes")

	nameDeleteChange, ok := changeForField(deleteChanges, "name")
	require.True(t, ok, "delete audit event should include a name change")
	assert.Equal(t, name, jsonField(nameDeleteChange, "old_value"))
	assert.Nil(t, nameDeleteChange["new_value"])
}

func TestCustomers_AuditEvents_UpdateAllFields(t *testing.T) {
	t.Parallel()

	// Create a customer with all fields populated so we can test changing each one.
	name := uniqueName("e2e-cust-audit-all")
	body := validCustomerBody(name)
	body["carrier_billing_type"] = "sender"
	body["carrier_billing_account"] = "ORIG-ACCT"
	body["default_priority_code"] = SeedPriorityCode
	body["default_sales_rep_user_id"] = SeedUserID

	createStatus, createBody, err := apiClient.Post(customersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	created := parseJSON(createBody)
	id := jsonField(created, "id")
	t.Cleanup(func() { apiClient.Delete(customersPath + "/" + id) })

	expectAuditEvent(t, id, "customer", "create")

	// Get initial address IDs.
	getStatus, getBody, err := apiClient.GetListRaw(
		customersPath+"/"+id,
		url.Values{"include": {"bill_to_address,ship_to_address"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	origBillToID := jsonField(jsonObject(got, "bill_to_address"), "id")
	origShipToID := jsonField(jsonObject(got, "ship_to_address"), "id")
	require.NotEmpty(t, origBillToID)
	require.NotEmpty(t, origShipToID)

	// Create new addresses for the update.
	customerClient := apiClient.WithAccountID(id)

	billAddrStatus, billAddrBody, err := customerClient.Post(addressesPath, map[string]any{
		"name": uniqueName("e2e-addr-bill"), "country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, billAddrStatus, billAddrBody)
	newBillToID := jsonField(parseJSON(billAddrBody), "id")
	t.Cleanup(func() { customerClient.Delete(addressesPath + "/" + newBillToID) })

	shipAddrStatus, shipAddrBody, err := customerClient.Post(addressesPath, map[string]any{
		"name": uniqueName("e2e-addr-ship"), "country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, shipAddrStatus, shipAddrBody)
	newShipToID := jsonField(parseJSON(shipAddrBody), "id")
	t.Cleanup(func() { customerClient.Delete(addressesPath + "/" + newShipToID) })

	// Update ALL changeable fields at once.
	updatedName := uniqueName("e2e-cust-audit-upd")
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+id, map[string]any{
		"name":                    updatedName,
		"note":                    "audit update note",
		"email":                   "audit-update@e2e.augno.com",
		"phone":                   "555-AUDIT",
		"url":                     "https://audit.e2e.augno.com",
		"is_edi_enabled":          true,
		"commission_policy":       "commission_exempt",
		"freight_policy":          "free_freight",
		"carrier_billing_type":    "third_party",
		"carrier_billing_account": "NEW-ACCT",
		"default_payment_term_id": SeedDefaultPaymentTermID,
		"bill_to_address_id":      newBillToID,
		"ship_to_address_id":      newShipToID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	event := expectAuditEventWithChanges(t, id, "customer", "update")
	changes := jsonArray(event, "changes")
	require.NotEmpty(t, changes, "update audit event should include changes")

	// Verify string field changes.
	nameChange, ok := changeForField(changes, "name")
	require.True(t, ok, "should include name change")
	assert.Equal(t, name, jsonField(nameChange, "old_value"))
	assert.Equal(t, updatedName, jsonField(nameChange, "new_value"))

	// Verify nullable string fields set from nil.
	for _, tc := range []struct {
		field string
		value string
	}{
		{"note", "audit update note"},
		{"email", "audit-update@e2e.augno.com"},
		{"phone", "555-AUDIT"},
		{"url", "https://audit.e2e.augno.com"},
	} {
		change, ok := changeForField(changes, tc.field)
		require.True(t, ok, "should include %s change", tc.field)
		assert.Nil(t, change["old_value"], "%s old_value should be null", tc.field)
		assert.Equal(t, tc.value, jsonField(change, "new_value"), "%s new_value mismatch", tc.field)
	}

	// Verify boolean field change.
	ediChange, ok := changeForField(changes, "is_edi_enabled")
	require.True(t, ok, "should include is_edi_enabled change")
	assert.Equal(t, false, ediChange["old_value"], "is_edi_enabled old_value should be false")
	assert.Equal(t, true, ediChange["new_value"], "is_edi_enabled new_value should be true")

	// Verify enum/policy field changes.
	commChange, ok := changeForField(changes, "commission_policy")
	require.True(t, ok, "should include commission_policy change")
	assert.Equal(t, "commission_applied", jsonField(commChange, "old_value"))
	assert.Equal(t, "commission_exempt", jsonField(commChange, "new_value"))

	freightChange, ok := changeForField(changes, "freight_policy")
	require.True(t, ok, "should include freight_policy change")
	assert.Equal(t, "billed_freight", jsonField(freightChange, "old_value"))
	assert.Equal(t, "free_freight", jsonField(freightChange, "new_value"))

	cbtChange, ok := changeForField(changes, "carrier_billing_type")
	require.True(t, ok, "should include carrier_billing_type change")
	assert.Equal(t, "sender", jsonField(cbtChange, "old_value"))
	assert.Equal(t, "third_party", jsonField(cbtChange, "new_value"))

	// Verify carrier billing account change.
	cbaChange, ok := changeForField(changes, "carrier_billing_account")
	require.True(t, ok, "should include carrier_billing_account change")
	assert.Equal(t, "ORIG-ACCT", jsonField(cbaChange, "old_value"))
	assert.Equal(t, "NEW-ACCT", jsonField(cbaChange, "new_value"))

	// Verify ID-reference field change.
	ptChange, ok := changeForField(changes, "default_payment_term_id")
	require.True(t, ok, "should include default_payment_term_id change")
	assert.Equal(t, SeedPaymentTermID, jsonField(ptChange, "old_value"))
	assert.Equal(t, SeedDefaultPaymentTermID, jsonField(ptChange, "new_value"))

	// Verify address ID changes.
	billChange, ok := changeForField(changes, "bill_to_address_id")
	require.True(t, ok, "should include bill_to_address_id change")
	assert.Equal(t, origBillToID, jsonField(billChange, "old_value"))
	assert.Equal(t, newBillToID, jsonField(billChange, "new_value"))

	shipChange, ok := changeForField(changes, "ship_to_address_id")
	require.True(t, ok, "should include ship_to_address_id change")
	assert.Equal(t, origShipToID, jsonField(shipChange, "old_value"))
	assert.Equal(t, newShipToID, jsonField(shipChange, "new_value"))
}

func TestCustomers_AuditEvents_ClearAllNullableFields(t *testing.T) {
	t.Parallel()

	// Create a customer with ALL nullable fields populated.
	name := uniqueName("e2e-cust-audit-clr")
	body := validCustomerBody(name)
	body["note"] = "clear-test note"
	body["email"] = "clear-test@e2e.augno.com"
	body["phone"] = "555-CLEAR"
	body["url"] = "https://clear-test.e2e.augno.com"
	body["carrier_billing_type"] = "third_party"
	body["carrier_billing_account"] = "CLR-ACCT"
	body["default_priority_code"] = SeedPriorityCode
	body["default_sales_rep_user_id"] = SeedUserID

	createStatus, createBody, err := apiClient.Post(customersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	created := parseJSON(createBody)
	id := jsonField(created, "id")
	t.Cleanup(func() { apiClient.Delete(customersPath + "/" + id) })

	expectAuditEvent(t, id, "customer", "create")

	// Get the customer to capture initial IDs for address and sales rep.
	getStatus, getBody, err := apiClient.GetListRaw(
		customersPath+"/"+id,
		url.Values{"include": {"bill_to_address,ship_to_address,defaults.sales_rep"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)

	origBillToID := jsonField(jsonObject(got, "bill_to_address"), "id")
	origShipToID := jsonField(jsonObject(got, "ship_to_address"), "id")
	origSalesRepID := jsonField(jsonObject(jsonObject(got, "defaults"), "sales_rep"), "id")
	require.NotEmpty(t, origBillToID)
	require.NotEmpty(t, origShipToID)
	require.NotEmpty(t, origSalesRepID)

	// Clear ALL nullable fields at once.
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+id, map[string]any{
		"note":                      nil,
		"email":                     nil,
		"phone":                     nil,
		"url":                       nil,
		"carrier_billing_account":   nil,
		"default_carrier_id":        nil,
		"default_payment_term_id":   nil,
		"default_shipping_term_id":  nil,
		"default_sales_rep_user_id": nil,
		"bill_to_address_id":        nil,
		"ship_to_address_id":        nil,
		"customer_type_group_id":    nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	event := expectAuditEventWithChanges(t, id, "customer", "update")
	changes := jsonArray(event, "changes")
	require.NotEmpty(t, changes, "update audit event should include changes")

	// Verify nullable string fields cleared to null.
	for _, tc := range []struct {
		field    string
		oldValue string
	}{
		{"note", "clear-test note"},
		{"email", "clear-test@e2e.augno.com"},
		{"phone", "555-CLEAR"},
		{"url", "https://clear-test.e2e.augno.com"},
		{"carrier_billing_account", "CLR-ACCT"},
	} {
		change, ok := changeForField(changes, tc.field)
		require.True(t, ok, "should include %s change", tc.field)
		assert.Equal(t, tc.oldValue, jsonField(change, "old_value"), "%s old_value mismatch", tc.field)
		assert.Nil(t, change["new_value"], "%s new_value should be null", tc.field)
	}

	// Verify ID-reference fields cleared to null.
	for _, tc := range []struct {
		field    string
		oldValue string
	}{
		{"default_carrier_id", SeedCarrierID},
		{"default_payment_term_id", SeedPaymentTermID},
		{"default_shipping_term_id", SeedShippingTermID},
		{"type_group_id", SeedCustomerGroupID},
	} {
		change, ok := changeForField(changes, tc.field)
		require.True(t, ok, "should include %s change", tc.field)
		assert.Equal(t, tc.oldValue, jsonField(change, "old_value"), "%s old_value mismatch", tc.field)
		assert.Nil(t, change["new_value"], "%s new_value should be null", tc.field)
	}

	// Verify sales rep cleared (API field: default_sales_rep_user_id → audit field: default_sales_rep_id).
	salesRepChange, ok := changeForField(changes, "default_sales_rep_id")
	require.True(t, ok, "should include default_sales_rep_id change")
	assert.Equal(t, origSalesRepID, jsonField(salesRepChange, "old_value"))
	assert.Nil(t, salesRepChange["new_value"])

	// Verify address IDs cleared to null.
	billChange, ok := changeForField(changes, "bill_to_address_id")
	require.True(t, ok, "should include bill_to_address_id change")
	assert.Equal(t, origBillToID, jsonField(billChange, "old_value"))
	assert.Nil(t, billChange["new_value"])

	shipChange, ok := changeForField(changes, "ship_to_address_id")
	require.True(t, ok, "should include ship_to_address_id change")
	assert.Equal(t, origShipToID, jsonField(shipChange, "old_value"))
	assert.Nil(t, shipChange["new_value"])
}

func TestCustomers_RequestLogs(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cust-rlog")
	createAndCleanup(t, customersPath, validCustomerBody(name))

	expectRequestLog(t, "POST", "201", customersPath)
}

// ── Account Users ──────────────────────────────

func TestAccountUsers_AuditEvents(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-acuser-audit")
	email := name + "@e2e-test.augno.com"

	status, body, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { apiClient.Delete(accountUsersPath + "/" + id) })

	expectAuditEvent(t, id, "account_user", "create")

	_, patchBody, err := apiClient.Patch(accountUsersPath+"/"+id, map[string]any{
		"name": uniqueName("e2e-acuser-audit-u"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotNil(t, patchBody)

	expectAuditEvent(t, id, "account_user", "update")
}

func TestAccountUsers_RequestLogs(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-acuser-rlog")
	email := name + "@e2e-test.augno.com"

	status, body, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { apiClient.Delete(accountUsersPath + "/" + id) })

	expectRequestLog(t, "POST", "201", accountUsersPath)
}

// ── Addresses ──────────────────────────────────

func TestAddresses_AuditEvents(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, addressesPath, map[string]any{
		"name":    uniqueName("e2e-addr-audit"),
		"country": "US",
	})
	id := jsonField(created, "id")

	expectAuditEvent(t, id, "address", "create")

	_, patchBody, err := apiClient.Patch(addressesPath+"/"+id, map[string]any{
		"name": uniqueName("e2e-addr-audit-u"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotNil(t, patchBody)

	expectAuditEvent(t, id, "address", "update")
}

func TestAddresses_RequestLogs(t *testing.T) {
	t.Parallel()
	createAndCleanup(t, addressesPath, map[string]any{
		"name":    uniqueName("e2e-addr-rlog"),
		"country": "US",
	})

	expectRequestLog(t, "POST", "201", addressesPath)
}

// ── API Keys ───────────────────────────────────

func TestAPIKeys_AuditEvents(t *testing.T) {
	t.Parallel()
	created := createAPIKeyAndCleanup(t, uniqueName("e2e-key-audit"))
	info := jsonObject(created, "api_key_info")
	id := jsonField(info, "id")

	expectAuditEvent(t, id, "api_key", "create")
}

func TestAPIKeys_RequestLogs(t *testing.T) {
	t.Parallel()
	created := createAPIKeyAndCleanup(t, uniqueName("e2e-key-rlog"))
	info := jsonObject(created, "api_key_info")
	id := jsonField(info, "id")
	require.NotEmpty(t, id)

	expectRequestLog(t, "POST", "201", apiKeysPath)
}

// ── Attributes ─────────────────────────────────

func TestAttributes_AuditEvents(t *testing.T) {
	t.Parallel()
	path := attributesPath(SeedPropertyID)

	status, body, err := apiClient.Post(path, map[string]any{
		"value":      uniqueName("e2e-attr-audit"),
		"color_code": "blue",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { apiClient.Delete(path + "/" + id) })

	expectAuditEvent(t, id, "attribute", "create")

	_, patchBody, err := apiClient.Patch(path+"/"+id, map[string]any{
		"value": uniqueName("e2e-attr-audit-u"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotNil(t, patchBody)

	expectAuditEvent(t, id, "attribute", "update")
}

func TestAttributes_RequestLogs(t *testing.T) {
	t.Parallel()
	path := attributesPath(SeedPropertyID)

	status, body, err := apiClient.Post(path, map[string]any{
		"value":      uniqueName("e2e-attr-rlog"),
		"color_code": "blue",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { apiClient.Delete(path + "/" + id) })

	expectRequestLog(t, "POST", "201", path)
}

// ── Carriers ───────────────────────────────────

func TestCarriers_AuditEvents(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, carriersPath, map[string]any{
		"name": uniqueName("e2e-carrier-audit"),
		"code": "will_call",
	})
	id := jsonField(created, "id")

	expectAuditEvent(t, id, "carrier", "create")

	_, patchBody, err := apiClient.Patch(carriersPath+"/"+id, map[string]any{
		"name": uniqueName("e2e-carrier-audit-u"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotNil(t, patchBody)

	expectAuditEvent(t, id, "carrier", "update")
}

func TestCarriers_RequestLogs(t *testing.T) {
	t.Parallel()
	createAndCleanup(t, carriersPath, map[string]any{
		"name": uniqueName("e2e-carrier-rlog"),
		"code": "will_call",
	})

	expectRequestLog(t, "POST", "201", carriersPath)
}

// ── Item Categories ────────────────────────────

func TestItemCategories_AuditEvents(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, itemCategoriesPath, map[string]any{
		"name":          uniqueName("e2e-itcg-audit"),
		"type":          "material_category",
		"unit_group_id": SeedUnitGroupID,
	})
	id := jsonField(created, "id")

	expectAuditEvent(t, id, "item_category", "create")

	_, patchBody, err := apiClient.Patch(itemCategoriesPath+"/"+id, map[string]any{
		"name": uniqueName("e2e-itcg-audit-u"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotNil(t, patchBody)

	expectAuditEvent(t, id, "item_category", "update")
}

func TestItemCategories_RequestLogs(t *testing.T) {
	t.Parallel()
	createAndCleanup(t, itemCategoriesPath, map[string]any{
		"name":          uniqueName("e2e-itcg-rlog"),
		"type":          "material_category",
		"unit_group_id": SeedUnitGroupID,
	})

	expectRequestLog(t, "POST", "201", itemCategoriesPath)
}

// ── Locations ──────────────────────────────────

func TestLocations_AuditEvents(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, locationsPath, map[string]any{
		"name":      uniqueName("e2e-loc-audit"),
		"type_code": "building",
	})
	id := jsonField(created, "id")

	expectAuditEvent(t, id, "location", "create")

	_, patchBody, err := apiClient.Patch(locationsPath+"/"+id, map[string]any{
		"name": uniqueName("e2e-loc-audit-u"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotNil(t, patchBody)

	expectAuditEvent(t, id, "location", "update")
}

func TestLocations_RequestLogs(t *testing.T) {
	t.Parallel()
	createAndCleanup(t, locationsPath, map[string]any{
		"name":      uniqueName("e2e-loc-rlog"),
		"type_code": "building",
	})

	expectRequestLog(t, "POST", "201", locationsPath)
}

// ── Machines ───────────────────────────────────

func TestMachines_AuditEvents(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, machinesPath, map[string]any{
		"name":          uniqueName("e2e-machine-audit"),
		"serial_number": uniqueName("SN"),
		"department_id": SeedDepartmentID,
	})
	id := jsonField(created, "id")

	expectAuditEvent(t, id, "machine", "create")

	_, patchBody, err := apiClient.Patch(machinesPath+"/"+id, map[string]any{
		"name": uniqueName("e2e-machine-audit-u"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotNil(t, patchBody)

	expectAuditEvent(t, id, "machine", "update")
}

func TestMachines_RequestLogs(t *testing.T) {
	t.Parallel()
	createAndCleanup(t, machinesPath, map[string]any{
		"name":          uniqueName("e2e-machine-rlog"),
		"serial_number": uniqueName("SN"),
		"department_id": SeedDepartmentID,
	})

	expectRequestLog(t, "POST", "201", machinesPath)
}

// ── Payment Terms ──────────────────────────────

func TestPaymentTerms_AuditEvents(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, paymentTermsPath, map[string]any{
		"name": uniqueName("e2e-payterm-audit"),
	})
	id := jsonField(created, "id")

	expectAuditEvent(t, id, "payment_term", "create")

	_, patchBody, err := apiClient.Patch(paymentTermsPath+"/"+id, map[string]any{
		"name": uniqueName("e2e-payterm-audit-u"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotNil(t, patchBody)

	expectAuditEvent(t, id, "payment_term", "update")
}

func TestPaymentTerms_RequestLogs(t *testing.T) {
	t.Parallel()
	createAndCleanup(t, paymentTermsPath, map[string]any{
		"name": uniqueName("e2e-payterm-rlog"),
	})

	expectRequestLog(t, "POST", "201", paymentTermsPath)
}

// ── Product Lines ──────────────────────────────

func TestProductLines_AuditEvents(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, productLinesPath, map[string]any{
		"name":              uniqueName("e2e-pdln-audit"),
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
	})
	id := jsonField(created, "id")

	expectAuditEvent(t, id, "product_line", "create")

	_, patchBody, err := apiClient.Patch(productLinesPath+"/"+id, map[string]any{
		"name": uniqueName("e2e-pdln-audit-u"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotNil(t, patchBody)

	expectAuditEvent(t, id, "product_line", "update")
}

func TestProductLines_RequestLogs(t *testing.T) {
	t.Parallel()
	createAndCleanup(t, productLinesPath, map[string]any{
		"name":              uniqueName("e2e-pdln-rlog"),
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
	})

	expectRequestLog(t, "POST", "201", productLinesPath)
}

// ── Properties ─────────────────────────────────

func TestProperties_AuditEvents(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, propertiesPath, map[string]any{
		"name": uniqueName("e2e-prop-audit"),
	})
	id := jsonField(created, "id")

	expectAuditEvent(t, id, "property", "create")

	_, patchBody, err := apiClient.Patch(propertiesPath+"/"+id, map[string]any{
		"name": uniqueName("e2e-prop-audit-u"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotNil(t, patchBody)

	expectAuditEvent(t, id, "property", "update")
}

func TestProperties_RequestLogs(t *testing.T) {
	t.Parallel()
	createAndCleanup(t, propertiesPath, map[string]any{
		"name": uniqueName("e2e-prop-rlog"),
	})

	expectRequestLog(t, "POST", "201", propertiesPath)
}

// ── Roles ──────────────────────────────────────

func TestRoles_AuditEvents(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, rolesPath, map[string]any{
		"name":        uniqueName("e2e-role-audit"),
		"permissions": []string{"customers:read"},
	})
	id := jsonField(created, "id")

	expectAuditEvent(t, id, "role", "create")

	_, patchBody, err := apiClient.Patch(rolesPath+"/"+id, map[string]any{
		"name": uniqueName("e2e-role-audit-u"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotNil(t, patchBody)

	expectAuditEvent(t, id, "role", "update")
}

func TestRoles_RequestLogs(t *testing.T) {
	t.Parallel()
	createAndCleanup(t, rolesPath, map[string]any{
		"name":        uniqueName("e2e-role-rlog"),
		"permissions": []string{"customers:read"},
	})

	expectRequestLog(t, "POST", "201", rolesPath)
}

// ── Sandboxes ──────────────────────────────────

func TestSandboxes_AuditEvents(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(sandboxesPath, map[string]any{
		"name": uniqueName("e2e-sandbox-audit"),
		"mode": "blank",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { apiClient.Delete(sandboxesPath + "/" + id) })

	expectAuditEvent(t, id, "sandbox", "create")
}

func TestSandboxes_RequestLogs(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(sandboxesPath, map[string]any{
		"name": uniqueName("e2e-sandbox-rlog"),
		"mode": "blank",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { apiClient.Delete(sandboxesPath + "/" + id) })

	expectRequestLog(t, "POST", "201", sandboxesPath)
}

// ── Scanning Stations ──────────────────────────

func TestScanningStations_AuditEvents(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, scanningStationsPath, map[string]any{
		"name":          uniqueName("e2e-station-audit"),
		"type":          "init_batch",
		"department_id": SeedDepartmentID,
	})
	id := jsonField(created, "id")

	expectAuditEvent(t, id, "scanning_station", "create")

	_, patchBody, err := apiClient.Patch(scanningStationsPath+"/"+id, map[string]any{
		"name": uniqueName("e2e-station-audit-u"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotNil(t, patchBody)

	expectAuditEvent(t, id, "scanning_station", "update")
}

func TestScanningStations_RequestLogs(t *testing.T) {
	t.Parallel()
	createAndCleanup(t, scanningStationsPath, map[string]any{
		"name":          uniqueName("e2e-station-rlog"),
		"type":          "init_batch",
		"department_id": SeedDepartmentID,
	})

	expectRequestLog(t, "POST", "201", scanningStationsPath)
}

// ── Shipping Terms ─────────────────────────────

func TestShippingTerms_AuditEvents(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, shippingTermsPath, map[string]any{
		"name": uniqueName("e2e-shipterm-audit"),
		"type": "free_freight",
	})
	id := jsonField(created, "id")

	expectAuditEvent(t, id, "shipping_term", "create")

	_, patchBody, err := apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{
		"name": uniqueName("e2e-shipterm-audit-u"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotNil(t, patchBody)

	expectAuditEvent(t, id, "shipping_term", "update")
}

func TestShippingTerms_RequestLogs(t *testing.T) {
	t.Parallel()
	createAndCleanup(t, shippingTermsPath, map[string]any{
		"name": uniqueName("e2e-shipterm-rlog"),
		"type": "free_freight",
	})

	expectRequestLog(t, "POST", "201", shippingTermsPath)
}

// ── Unit Groups ────────────────────────────────

func TestUnitGroups_AuditEvents(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         uniqueName("e2e-unitgrp-audit"),
		"type":         "quantity",
		"base_unit_id": "each",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { apiClient.Delete(unitGroupsPath + "/" + id) })

	expectAuditEvent(t, id, "unit_group", "create")

	_, patchBody, err := apiClient.Patch(unitGroupsPath+"/"+id, map[string]any{
		"name": uniqueName("e2e-unitgrp-audit-u"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotNil(t, patchBody)

	expectAuditEvent(t, id, "unit_group", "update")
}

func TestUnitGroups_RequestLogs(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         uniqueName("e2e-unitgrp-rlog"),
		"type":         "quantity",
		"base_unit_id": "each",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { apiClient.Delete(unitGroupsPath + "/" + id) })

	expectRequestLog(t, "POST", "201", unitGroupsPath)
}
