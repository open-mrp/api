//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// customersPath is defined in list_customers_test.go

// ──────────────────────────────────────────────
// CRUD Lifecycle
// ──────────────────────────────────────────────

func TestCustomers_CRUD(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cust")

	// CREATE
	createStatus, createBody, err := apiClient.Post(customersPath, validCustomerBody(name), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assert.Equal(t, "customer", jsonField(created, "object"))
	assert.Equal(t, name, jsonField(created, "name"))
	assert.NotEmpty(t, jsonField(created, "number"))
	assert.NotEmpty(t, jsonField(created, "created_at"))
	assert.NotEmpty(t, jsonField(created, "updated_at"))

	// GET
	getStatus, getBody, err := apiClient.GetListRaw(customersPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, id, jsonField(got, "id"))
	assert.Equal(t, name, jsonField(got, "name"))

	// UPDATE
	newNote := "updated note"
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+id, map[string]any{
		"note": newNote,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	updated := parseJSON(patchBody)
	assert.Equal(t, newNote, jsonField(updated, "note"))
	assert.Equal(t, name, jsonField(updated, "name"))

	// DELETE
	delStatus, delBody, err := apiClient.Delete(customersPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify deletion
	getStatus2, _, err := apiClient.GetListRaw(customersPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus2)
}

// ──────────────────────────────────────────────
// Create Tests
// ──────────────────────────────────────────────

func TestCustomers_CreateResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cust-shape")
	status, body, err := apiClient.Post(customersPath, validCustomerBody(name), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	assert.NotEmpty(t, jsonField(got, "id"))
	assert.Equal(t, "customer", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.NotEmpty(t, jsonField(got, "number"), "number should be auto-generated")
	assert.Equal(t, "normal", jsonField(got, "status"))
	assert.Equal(t, "false", jsonField(got, "is_edi_enabled"))
	assert.Equal(t, "false", jsonField(got, "is_parent_account"))
	assert.Equal(t, "commission_applied", jsonField(got, "commission_policy"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))

	// All expandable fields should be null without ?include
	assert.Nil(t, got["contact_info"], "contact_info should be null without ?include=contact_info")
	assert.Nil(t, got["freight_preferences"], "freight_preferences should be null without ?include=freight_preferences")
	assert.Nil(t, got["defaults"], "defaults should be null without ?include=defaults")
	assert.Nil(t, got["notification_preferences"], "notification_preferences should be null without ?include=notification_preferences")
	assert.Nil(t, got["bill_to_address"], "bill_to_address should be null without ?include=bill_to_address")
	assert.Nil(t, got["ship_to_address"], "ship_to_address should be null without ?include=ship_to_address")
	assert.Nil(t, got["type"], "type should be null without ?include=type")
	assert.Nil(t, got["price_groups"], "price_groups should be null without ?include=price_groups")
	assert.Nil(t, got["parent_account"], "parent_account should be null without ?include=parent_account")
	assert.Nil(t, got["child_accounts"], "child_accounts should be null without ?include=child_accounts")
	assert.Nil(t, got["credit_limit"], "credit_limit should be null without ?include=credit_limit")

	apiClient.Delete(customersPath + "/" + jsonField(got, "id"))
}

func TestCustomers_CreateWithAllFields(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cust-all")
	status, body, err := apiClient.Post(customersPath, map[string]any{
		"name":                      name,
		"note":                      "Test customer note",
		"email":                     name + "@e2e-test.augno.com",
		"phone":                     "555-000-1234",
		"url":                       "https://e2e.augno.com",
		"status_code":               "normal",
		"is_edi_enabled":            true,
		"commission_policy":         "commission_exempt",
		"freight_policy":            "free_freight",
		"default_carrier_id":        SeedCarrierID,
		"default_payment_term_id":   SeedPaymentTermID,
		"default_shipping_term_id":  SeedShippingTermID,
		"default_priority_code":     SeedPriorityCode,
		"default_sales_rep_user_id": SeedUserID,
		"customer_type_group_id":    SeedCustomerGroupID,
		"carrier_billing_type":      "third_party",
		"carrier_billing_account":   "ACCT-12345",
		"credit_limit": map[string]any{
			"value":   "7500.00",
			"unit_id": SeedUnitID,
		},
		"bill_to_address": map[string]any{
			"name":    name + " Billing",
			"country": "US",
		},
		"ship_to_address": map[string]any{
			"name":    name + " Shipping",
			"country": "US",
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	id := jsonField(got, "id")
	assert.NotEmpty(t, id)
	assert.Equal(t, "customer", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.NotEmpty(t, jsonField(got, "number"), "number should be auto-generated")
	assert.Equal(t, "normal", jsonField(got, "status"))
	assert.Equal(t, "true", jsonField(got, "is_edi_enabled"))
	assert.Equal(t, "false", jsonField(got, "is_parent_account"))
	assert.Equal(t, "commission_exempt", jsonField(got, "commission_policy"))
	assert.Equal(t, "Test customer note", jsonField(got, "note"))
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	apiClient.Delete(customersPath + "/" + id)
}

func TestCustomers_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	includeStr := "contact_info,defaults.payment_term,defaults.shipping_term,defaults.priority,defaults.sales_rep,freight_preferences.carrier,type,bill_to_address,ship_to_address,notification_preferences,freight_preferences,credit_limit"

	// ── CREATE with every settable field ──

	name := uniqueName("e2e-cust-allf")
	createStatus, createBody, err := apiClient.Post(
		customersPath+"?include="+includeStr,
		map[string]any{
			"name":                      name,
			"note":                      "Create note",
			"email":                     name + "@e2e-test.augno.com",
			"phone":                     "555-000-1234",
			"url":                       "https://create.e2e.augno.com",
			"status_code":               "normal",
			"is_edi_enabled":            true,
			"commission_policy":         "commission_exempt",
			"freight_policy":            "free_freight",
			"default_carrier_id":        SeedCarrierID,
			"default_payment_term_id":   SeedPaymentTermID,
			"default_shipping_term_id":  SeedShippingTermID,
			"default_priority_code":     SeedPriorityCode,
			"default_sales_rep_user_id": SeedUserID,
			"customer_type_group_id":    SeedCustomerGroupID,
			"carrier_billing_type":      "third_party",
			"carrier_billing_account":   "ACCT-12345",
			"credit_limit": map[string]any{
				"value":   "10000.00",
				"unit_id": SeedUnitID,
			},
			"bill_to_address": map[string]any{
				"name":    name + " Billing",
				"country": "US",
			},
			"ship_to_address": map[string]any{
				"name":    name + " Shipping",
				"country": "US",
			},
		},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	got := parseJSON(createBody)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(customersPath + "/" + id)

	// Top-level scalar fields
	assert.Equal(t, "customer", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "Create note", jsonField(got, "note"))
	assert.NotEmpty(t, jsonField(got, "number"))
	assert.Equal(t, "normal", jsonField(got, "status"))
	assert.Equal(t, "true", jsonField(got, "is_edi_enabled"))
	assert.Equal(t, "false", jsonField(got, "is_parent_account"))
	assert.Equal(t, "commission_exempt", jsonField(got, "commission_policy"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))

	// contact_info
	contactInfo := jsonObject(got, "contact_info")
	require.NotNil(t, contactInfo, "contact_info should be expanded")
	assert.Equal(t, "customer_contact_info", jsonField(contactInfo, "object"))
	assert.Equal(t, name+"@e2e-test.augno.com", jsonField(contactInfo, "email"))
	assert.Equal(t, "555-000-1234", jsonField(contactInfo, "phone"))
	assert.Equal(t, "https://create.e2e.augno.com", jsonField(contactInfo, "url"))

	// defaults — payment_term (this catches the known bug)
	defaults := jsonObject(got, "defaults")
	require.NotNil(t, defaults, "defaults should be expanded")
	assert.Equal(t, "customer_defaults", jsonField(defaults, "object"))

	pt := jsonObject(defaults, "payment_term")
	require.NotNil(t, pt, "defaults.payment_term must be set after create")
	assert.Equal(t, SeedPaymentTermID, jsonField(pt, "id"), "payment_term ID must match what was sent on create")

	st := jsonObject(defaults, "shipping_term")
	require.NotNil(t, st, "defaults.shipping_term must be set after create")
	assert.Equal(t, SeedShippingTermID, jsonField(st, "id"))

	pri := jsonObject(defaults, "priority")
	require.NotNil(t, pri, "defaults.priority must be set after create")
	assert.Equal(t, SeedPriorityCode, jsonField(pri, "code"))

	salesRep := jsonObject(defaults, "sales_rep")
	require.NotNil(t, salesRep, "defaults.sales_rep must be set after create")
	assert.Equal(t, SeedAccountUserID, jsonField(salesRep, "id"))

	// freight_preferences
	fp := jsonObject(got, "freight_preferences")
	require.NotNil(t, fp, "freight_preferences should be expanded")
	assert.Equal(t, "customer_freight_preferences", jsonField(fp, "object"))
	assert.Equal(t, "free_freight", jsonField(fp, "status"))
	assert.Equal(t, "third_party", jsonField(fp, "billing_type"))
	assert.Equal(t, "ACCT-12345", jsonField(fp, "billing_account"))

	carrier := jsonObject(fp, "carrier")
	require.NotNil(t, carrier, "freight_preferences.carrier must be set after create")
	assert.Equal(t, SeedCarrierID, jsonField(carrier, "id"))

	// type
	typeGroup := jsonObject(got, "type")
	require.NotNil(t, typeGroup, "type should be expanded after create")
	assert.Equal(t, SeedCustomerGroupID, jsonField(typeGroup, "id"))

	// addresses (set on create via inline address objects)
	billAddr := jsonObject(got, "bill_to_address")
	require.NotNil(t, billAddr, "bill_to_address should be expanded after create")
	assert.Equal(t, name+" Billing", jsonField(billAddr, "name"))

	shipAddr := jsonObject(got, "ship_to_address")
	require.NotNil(t, shipAddr, "ship_to_address should be expanded after create")
	assert.Equal(t, name+" Shipping", jsonField(shipAddr, "name"))

	// notification_preferences
	np := jsonObject(got, "notification_preferences")
	require.NotNil(t, np, "notification_preferences should be expanded after create")
	assert.Equal(t, "customer_notification_preferences", jsonField(np, "object"))

	// credit_limit
	cl := jsonObject(got, "credit_limit")
	require.NotNil(t, cl, "credit_limit should be expanded after create")
	assert.Equal(t, "quantity", jsonField(cl, "object"))
	assert.Equal(t, "10000", jsonField(cl, "value"))
	assert.NotEmpty(t, jsonField(cl, "id"))
	clUnit := jsonObject(cl, "unit")
	require.NotNil(t, clUnit, "credit_limit.unit should be set")
	assert.Equal(t, SeedUnitID, jsonField(clUnit, "id"))

	// ── UPDATE with different values for changeable fields ──

	updatedName := uniqueName("e2e-cust-allf-upd")
	patchStatus, patchBody, err := apiClient.Patch(
		customersPath+"/"+id,
		map[string]any{
			"name":                    updatedName,
			"note":                    "Updated note",
			"email":                   updatedName + "@e2e-test.augno.com",
			"phone":                   "555-999-8888",
			"url":                     "https://update.e2e.augno.com",
			"is_edi_enabled":          false,
			"commission_policy":       "commission_applied",
			"freight_policy":          "billed_freight",
			"default_payment_term_id": SeedDefaultPaymentTermID,
			"carrier_billing_type":    "sender",
			"carrier_billing_account": "ACCT-99999",
			"credit_limit": map[string]any{
				"value":   "25000.50",
				"unit_id": SeedUnitID,
			},
		},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	// Verify the update via GET with include
	getStatus, getBody, err := apiClient.GetListRaw(customersPath+"/"+id, url.Values{"include": {includeStr}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	updated := parseJSON(getBody)

	// Updated top-level fields
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "Updated note", jsonField(updated, "note"))
	assert.Equal(t, "false", jsonField(updated, "is_edi_enabled"))
	assert.Equal(t, "commission_applied", jsonField(updated, "commission_policy"))

	// Preserved top-level fields
	assert.Equal(t, "normal", jsonField(updated, "status"), "status should be preserved")

	// Updated contact_info
	updContactInfo := jsonObject(updated, "contact_info")
	require.NotNil(t, updContactInfo)
	assert.Equal(t, updatedName+"@e2e-test.augno.com", jsonField(updContactInfo, "email"))
	assert.Equal(t, "555-999-8888", jsonField(updContactInfo, "phone"))
	assert.Equal(t, "https://update.e2e.augno.com", jsonField(updContactInfo, "url"))

	// Updated defaults — payment_term changed
	updDefaults := jsonObject(updated, "defaults")
	require.NotNil(t, updDefaults)

	updPt := jsonObject(updDefaults, "payment_term")
	require.NotNil(t, updPt, "defaults.payment_term must be set after update")
	assert.Equal(t, SeedDefaultPaymentTermID, jsonField(updPt, "id"), "payment_term should be updated")

	// Preserved defaults
	updSt := jsonObject(updDefaults, "shipping_term")
	require.NotNil(t, updSt, "defaults.shipping_term should be preserved")
	assert.Equal(t, SeedShippingTermID, jsonField(updSt, "id"))

	updPri := jsonObject(updDefaults, "priority")
	require.NotNil(t, updPri, "defaults.priority should be preserved")
	assert.Equal(t, SeedPriorityCode, jsonField(updPri, "code"))

	updSalesRep := jsonObject(updDefaults, "sales_rep")
	require.NotNil(t, updSalesRep, "defaults.sales_rep should be preserved")
	assert.Equal(t, SeedAccountUserID, jsonField(updSalesRep, "id"))

	// Updated freight_preferences
	updFp := jsonObject(updated, "freight_preferences")
	require.NotNil(t, updFp)
	assert.Equal(t, "billed_freight", jsonField(updFp, "status"))
	assert.Equal(t, "sender", jsonField(updFp, "billing_type"))
	assert.Equal(t, "ACCT-99999", jsonField(updFp, "billing_account"))

	// Preserved carrier
	updCarrier := jsonObject(updFp, "carrier")
	require.NotNil(t, updCarrier, "freight_preferences.carrier should be preserved")
	assert.Equal(t, SeedCarrierID, jsonField(updCarrier, "id"))

	// Preserved type
	updType := jsonObject(updated, "type")
	require.NotNil(t, updType, "type should be preserved")
	assert.Equal(t, SeedCustomerGroupID, jsonField(updType, "id"))

	// Preserved top-level fields
	assert.Equal(t, "false", jsonField(updated, "is_parent_account"), "is_parent_account should be preserved")
	assert.NotEmpty(t, jsonField(updated, "number"), "number should be preserved")
	assertValidTimestamp(t, jsonField(updated, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")

	// notification_preferences should be preserved
	updNp := jsonObject(updated, "notification_preferences")
	require.NotNil(t, updNp, "notification_preferences should be preserved")
	assert.Equal(t, "customer_notification_preferences", jsonField(updNp, "object"))

	// freight_preferences.service_level was not set, should be nil
	assert.Nil(t, updFp["service_level"], "freight_preferences.service_level should be nil when not set")

	// Updated credit_limit
	updCL := jsonObject(updated, "credit_limit")
	require.NotNil(t, updCL, "credit_limit should be expanded after update")
	assert.Equal(t, "25000.5", jsonField(updCL, "value"))

	// ── CLEAR all nullable fields by sending null ──

	clearStatus, clearBody, err := apiClient.Patch(
		customersPath+"/"+id+"?include="+includeStr,
		map[string]any{
			"note":                      nil,
			"email":                     nil,
			"phone":                     nil,
			"url":                       nil,
			"default_carrier_id":        nil,
			"default_payment_term_id":   nil,
			"default_shipping_term_id":  nil,
			"default_sales_rep_user_id": nil,
			"customer_type_group_id":    nil,
			"credit_limit":              nil,
		},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, clearStatus, clearBody)

	cleared := parseJSON(clearBody)

	// Nullable fields should be cleared.
	assertNilField(t, cleared, "note")

	clearedCI := jsonObject(cleared, "contact_info")
	require.NotNil(t, clearedCI, "contact_info should still be expanded")
	assertNilField(t, clearedCI, "email")
	assertNilField(t, clearedCI, "phone")
	assertNilField(t, clearedCI, "url")

	clearedDefaults := jsonObject(cleared, "defaults")
	require.NotNil(t, clearedDefaults, "defaults should still be expanded")
	assertNilField(t, clearedDefaults, "payment_term")
	assertNilField(t, clearedDefaults, "shipping_term")
	assertNilField(t, clearedDefaults, "sales_rep")

	clearedFP := jsonObject(cleared, "freight_preferences")
	require.NotNil(t, clearedFP, "freight_preferences should still be expanded")
	assertNilField(t, clearedFP, "carrier")

	assertNilField(t, cleared, "type")
	assertNilField(t, cleared, "credit_limit")

	// Non-nullable fields should be preserved.
	assert.Equal(t, updatedName, jsonField(cleared, "name"), "name should be preserved")
	assert.NotEmpty(t, jsonField(cleared, "number"), "number should be preserved")
	assert.Equal(t, "normal", jsonField(cleared, "status"), "status should be preserved")
	assert.Equal(t, "false", jsonField(cleared, "is_edi_enabled"), "is_edi_enabled should be preserved")
	assert.Equal(t, "commission_applied", jsonField(cleared, "commission_policy"), "commission_policy should be preserved")

	// ── Verify cleared state survives an unrelated update ──

	preserveStatus, preserveBody, err := apiClient.Patch(
		customersPath+"/"+id+"?include="+includeStr,
		map[string]any{
			"note": "re-added note",
		},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, preserveStatus, preserveBody)

	preserved := parseJSON(preserveBody)
	assert.Equal(t, "re-added note", jsonField(preserved, "note"), "note should be re-set")

	// Previously cleared fields should still be null.
	preservedCI := jsonObject(preserved, "contact_info")
	require.NotNil(t, preservedCI)
	assertNilField(t, preservedCI, "email")
	assertNilField(t, preservedCI, "phone")
	assertNilField(t, preservedCI, "url")

	preservedDefaults := jsonObject(preserved, "defaults")
	require.NotNil(t, preservedDefaults)
	assertNilField(t, preservedDefaults, "payment_term")
	assertNilField(t, preservedDefaults, "shipping_term")
	assertNilField(t, preservedDefaults, "sales_rep")

	preservedFP := jsonObject(preserved, "freight_preferences")
	require.NotNil(t, preservedFP)
	assertNilField(t, preservedFP, "carrier")
	assertNilField(t, preserved, "type")
	assertNilField(t, preserved, "credit_limit")
}

func TestCustomers_CreateValidation_MissingName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(customersPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing name should return 400 or 422, got %d: %s", status, string(body))
}

func TestCustomers_CreateValidation_EmptyName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(customersPath, map[string]any{
		"name": "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Empty name should return 400 or 422, got %d: %s", status, string(body))
}

func TestCustomers_CreateValidation_MissingName_ErrorShape(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(customersPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

func TestCustomers_GetNotFound_ErrorShape(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/ac_000000000000000000000000", nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)

	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestCustomers_ResponseFieldFormats(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cust-fmt")
	status, body, err := apiClient.Post(customersPath, validCustomerBody(name), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	id := jsonField(got, "id")

	assertIDFormat(t, id, "ac")
	assertObjectField(t, got, "customer")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	apiClient.Delete(customersPath + "/" + id)
}

// ──────────────────────────────────────────────
// Idempotency
// ──────────────────────────────────────────────

func TestCustomers_CreateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-cust")
	idemKey := newIdempotencyKey()
	payload := validCustomerBody(name)

	status1, body1, err := apiClient.Post(customersPath, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(customersPath, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(customersPath + "/" + id1)
}

func TestCustomers_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-cust-upd")
	createStatus, createBody, err := apiClient.Post(customersPath, validCustomerBody(name), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	idemKey := newIdempotencyKey()
	payload := map[string]any{"note": "idempotent note"}

	status1, body1, err := apiClient.Patch(customersPath+"/"+id, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(customersPath+"/"+id, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	assert.Equal(t, jsonField(parseJSON(body1), "note"), jsonField(parseJSON(body2), "note"))

	apiClient.Delete(customersPath + "/" + id)
}

// ──────────────────────────────────────────────
// Get Tests
// ──────────────────────────────────────────────

func TestCustomers_GetSeeded(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedCustomerAccountID, jsonField(got, "id"))
	assert.Equal(t, "customer", jsonField(got, "object"))
	assert.Equal(t, SeedCustomerName, jsonField(got, "name"))
	assert.NotEmpty(t, jsonField(got, "number"))
	assert.Equal(t, "normal", jsonField(got, "status"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))
}

func TestCustomers_GetNotFound(t *testing.T) {
	t.Parallel()
	getStatus, _, err := apiClient.GetListRaw(customersPath+"/ac_000000000000000000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus)
}

// ──────────────────────────────────────────────
// Update Tests
// ──────────────────────────────────────────────

func TestCustomers_UpdateOnlyNote(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cust-upd-note")
	createStatus, createBody, err := apiClient.Post(customersPath, validCustomerBody(name), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+id, map[string]any{
		"note": "a note",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, "a note", jsonField(patched, "note"))
	assert.Equal(t, name, jsonField(patched, "name"), "name should be preserved")

	apiClient.Delete(customersPath + "/" + id)
}

func TestCustomers_UpdateMultipleFields(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cust-upd-multi")
	createStatus, createBody, err := apiClient.Post(customersPath, validCustomerBody(name), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+id, map[string]any{
		"note":              "multi-field update",
		"phone":             "555-999-8888",
		"is_edi_enabled":    true,
		"commission_policy": "commission_exempt",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, name, jsonField(patched, "name"), "name should be preserved")
	assert.Equal(t, "multi-field update", jsonField(patched, "note"))
	assert.Equal(t, "true", jsonField(patched, "is_edi_enabled"))
	assert.Equal(t, "commission_exempt", jsonField(patched, "commission_policy"))

	apiClient.Delete(customersPath + "/" + id)
}

// ──────────────────────────────────────────────
// Delete Tests
// ──────────────────────────────────────────────

func TestCustomers_DeleteThenGetReturns404(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cust-del")
	createStatus, createBody, err := apiClient.Post(customersPath, validCustomerBody(name), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	delStatus, delBody, err := apiClient.Delete(customersPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// GET after delete → 404
	getStatus, _, err := apiClient.GetListRaw(customersPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus)
}

// ──────────────────────────────────────────────
// Expandable Fields — Null Without Include
// ──────────────────────────────────────────────

func TestCustomers_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["contact_info"], "contact_info should be null without ?include=contact_info")
	assert.Nil(t, got["freight_preferences"], "freight_preferences should be null without ?include=freight_preferences")
	assert.Nil(t, got["defaults"], "defaults should be null without ?include=defaults")
	assert.Nil(t, got["notification_preferences"], "notification_preferences should be null without ?include=notification_preferences")
	assert.Nil(t, got["bill_to_address"], "bill_to_address should be null without ?include=bill_to_address")
	assert.Nil(t, got["ship_to_address"], "ship_to_address should be null without ?include=ship_to_address")
	assert.Nil(t, got["type"], "type should be null without ?include=type")
	assert.Nil(t, got["price_groups"], "price_groups should be null without ?include=price_groups")
	assert.Nil(t, got["parent_account"], "parent_account should be null without ?include=parent_account")
	assert.Nil(t, got["child_accounts"], "child_accounts should be null without ?include=child_accounts")
	assert.Nil(t, got["credit_limit"], "credit_limit should be null without ?include=credit_limit")
}

// ──────────────────────────────────────────────
// Single Include Tests
// ──────────────────────────────────────────────

func TestCustomers_IncludeContactInfo(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"contact_info"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	contactInfo := jsonObject(got, "contact_info")
	require.NotNil(t, contactInfo, "contact_info should be present with ?include=contact_info")
	assert.Equal(t, "customer_contact_info", jsonField(contactInfo, "object"))
}

func TestCustomers_IncludeFreightPreferences(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"freight_preferences"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	fp := jsonObject(got, "freight_preferences")
	require.NotNil(t, fp, "freight_preferences should be present with ?include=freight_preferences")
	assert.Equal(t, "customer_freight_preferences", jsonField(fp, "object"))
	assert.NotEmpty(t, jsonField(fp, "status"))

	// Nested expandable fields should be null when not explicitly included
	assert.Nil(t, fp["carrier"], "carrier should be null without ?include=freight_preferences.carrier")
	assert.Nil(t, fp["service_level"], "service_level should be null without ?include=freight_preferences.service_level")
}

func TestCustomers_IncludeDefaults(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"defaults"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	defaults := jsonObject(got, "defaults")
	require.NotNil(t, defaults, "defaults should be present with ?include=defaults")
	assert.Equal(t, "customer_defaults", jsonField(defaults, "object"))

	// Nested expandable fields should be null when not explicitly included
	assert.Nil(t, defaults["payment_term"], "payment_term should be null without ?include=defaults.payment_term")
	assert.Nil(t, defaults["shipping_term"], "shipping_term should be null without ?include=defaults.shipping_term")
	assert.Nil(t, defaults["sales_rep"], "sales_rep should be null without ?include=defaults.sales_rep")
}

func TestCustomers_IncludeNotificationPreferences(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"notification_preferences"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	np := jsonObject(got, "notification_preferences")
	require.NotNil(t, np, "notification_preferences should be present with ?include=notification_preferences")
	assert.Equal(t, "customer_notification_preferences", jsonField(np, "object"))
}

func TestCustomers_IncludeBillToAddress(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"bill_to_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	addr := jsonObject(got, "bill_to_address")
	require.NotNil(t, addr, "bill_to_address should be present with ?include=bill_to_address")
	assert.Equal(t, "address", jsonField(addr, "object"))
	assert.Equal(t, SeedAddressID, jsonField(addr, "id"))
}

func TestCustomers_IncludeShipToAddress(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"ship_to_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	addr := jsonObject(got, "ship_to_address")
	require.NotNil(t, addr, "ship_to_address should be present with ?include=ship_to_address")
	assert.Equal(t, "address", jsonField(addr, "object"))
	assert.NotEmpty(t, jsonField(addr, "id"))
}

func TestCustomers_IncludeType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"type"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	typeGroup := jsonObject(got, "type")
	require.NotNil(t, typeGroup, "type should be present with ?include=type")
	assert.Equal(t, "account_group", jsonField(typeGroup, "object"))
	assert.Equal(t, SeedCustomerGroupID, jsonField(typeGroup, "id"))
}

func TestCustomers_IncludePriceGroups(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"price_groups"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	pg, ok := got["price_groups"]
	require.True(t, ok, "price_groups key should be present with ?include=price_groups")
	require.NotNil(t, pg, "price_groups should not be null with ?include=price_groups")

	pgMap, ok := pg.(map[string]any)
	require.True(t, ok, "price_groups should be a list object")
	assert.Equal(t, "list", pgMap["object"])
}

func TestCustomers_IncludeCreditLimit(t *testing.T) {
	t.Parallel()

	// Create a customer with a credit limit.
	payload := validCustomerBody(uniqueName("e2e-cust-cl-inc"))
	payload["credit_limit"] = map[string]any{
		"value":   "5000.00",
		"unit_id": SeedUnitID,
	}
	cust := createAndCleanup(t, customersPath+"?include=credit_limit", payload)

	cl := jsonObject(cust, "credit_limit")
	require.NotNil(t, cl, "credit_limit should be present with ?include=credit_limit")
	assert.Equal(t, "quantity", jsonField(cl, "object"))
	assert.Equal(t, "5000", jsonField(cl, "value"))
	assert.NotEmpty(t, jsonField(cl, "display_value"))
	assert.NotEmpty(t, jsonField(cl, "id"))

	clUnit := jsonObject(cl, "unit")
	require.NotNil(t, clUnit, "credit_limit.unit should be set")
	assert.Equal(t, SeedUnitID, jsonField(clUnit, "id"))
	assert.Equal(t, "unit", jsonField(clUnit, "object"))
}

func TestCustomers_IncludeCreditLimitNullWhenNotSet(t *testing.T) {
	t.Parallel()

	// Create a customer without a credit limit.
	cust := createAndCleanup(t, customersPath+"?include=credit_limit", validCustomerBody(uniqueName("e2e-cust-cl-nil")))
	assertNilField(t, cust, "credit_limit")
}

// ──────────────────────────────────────────────
// Nested Include Tests
// ──────────────────────────────────────────────

func TestCustomers_IncludeFreightPreferencesCarrier(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"freight_preferences.carrier"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	fp := jsonObject(got, "freight_preferences")
	require.NotNil(t, fp, "freight_preferences should be present with ?include=freight_preferences.carrier")

	carrier := jsonObject(fp, "carrier")
	require.NotNil(t, carrier, "carrier should be present with ?include=freight_preferences.carrier")
	assert.Equal(t, "carrier", jsonField(carrier, "object"))
	assert.Equal(t, SeedCarrierID, jsonField(carrier, "id"))
}

func TestCustomers_IncludeDefaultsPaymentTerm(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"defaults.payment_term"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	defaults := jsonObject(got, "defaults")
	require.NotNil(t, defaults, "defaults should be present with ?include=defaults.payment_term")

	pt := jsonObject(defaults, "payment_term")
	require.NotNil(t, pt, "payment_term should be present with ?include=defaults.payment_term")
	assert.Equal(t, "payment_term", jsonField(pt, "object"))
	assert.Equal(t, SeedPaymentTermID, jsonField(pt, "id"))

	// Other nested fields should remain null
	assert.Nil(t, defaults["shipping_term"], "shipping_term should be null when only defaults.payment_term is included")
	assert.Nil(t, defaults["sales_rep"], "sales_rep should be null when only defaults.payment_term is included")
}

func TestCustomers_IncludeDefaultsSalesRep(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"defaults.sales_rep"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	defaults := jsonObject(got, "defaults")
	require.NotNil(t, defaults, "defaults should be present with ?include=defaults.sales_rep")
	assert.Equal(t, "customer_defaults", jsonField(defaults, "object"))
	// sales_rep may be null if the seeded customer has no sales rep assigned
}

// ──────────────────────────────────────────────
// Multiple Includes
// ──────────────────────────────────────────────

func TestCustomers_IncludeMultiple(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"contact_info,defaults"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	contactInfo := jsonObject(got, "contact_info")
	require.NotNil(t, contactInfo, "contact_info should be present with ?include=contact_info,defaults")
	assert.Equal(t, "customer_contact_info", jsonField(contactInfo, "object"))

	defaults := jsonObject(got, "defaults")
	require.NotNil(t, defaults, "defaults should be present with ?include=contact_info,defaults")
	assert.Equal(t, "customer_defaults", jsonField(defaults, "object"))

	// Other expandable fields should remain null
	assert.Nil(t, got["freight_preferences"], "freight_preferences should be null when not included")
	assert.Nil(t, got["bill_to_address"], "bill_to_address should be null when not included")
}

func TestCustomers_IncludeMultipleWithNested(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{
		"include": {"contact_info,defaults.payment_term,freight_preferences.carrier"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)

	// contact_info expanded
	contactInfo := jsonObject(got, "contact_info")
	require.NotNil(t, contactInfo, "contact_info should be present")
	assert.Equal(t, "customer_contact_info", jsonField(contactInfo, "object"))

	// defaults expanded with payment_term nested
	defaults := jsonObject(got, "defaults")
	require.NotNil(t, defaults, "defaults should be present")
	pt := jsonObject(defaults, "payment_term")
	require.NotNil(t, pt, "defaults.payment_term should be present")
	assert.Equal(t, "payment_term", jsonField(pt, "object"))

	// freight_preferences expanded with carrier nested
	fp := jsonObject(got, "freight_preferences")
	require.NotNil(t, fp, "freight_preferences should be present")
	carrier := jsonObject(fp, "carrier")
	require.NotNil(t, carrier, "freight_preferences.carrier should be present")
	assert.Equal(t, "carrier", jsonField(carrier, "object"))
}

// ──────────────────────────────────────────────
// Include on Create / Update
// ──────────────────────────────────────────────

func TestCustomers_CreateWithInclude(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cust-inc-create")
	email := name + "@e2e-test.augno.com"
	payload := validCustomerBody(name)
	payload["email"] = email
	payload["phone"] = "555-111-2222"
	status, body, err := apiClient.Post(customersPath+"?include=contact_info", payload, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	id := jsonField(got, "id")
	contactInfo := jsonObject(got, "contact_info")
	require.NotNil(t, contactInfo, "contact_info should be present with ?include=contact_info on create")
	assert.Equal(t, "customer_contact_info", jsonField(contactInfo, "object"))
	assert.Equal(t, email, jsonField(contactInfo, "email"))
	assert.Equal(t, "555-111-2222", jsonField(contactInfo, "phone"))

	// Other expandable fields should still be null
	assert.Nil(t, got["defaults"], "defaults should be null when not included")

	apiClient.Delete(customersPath + "/" + id)
}

func TestCustomers_UpdateWithInclude(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cust-inc-upd")
	createPayload := validCustomerBody(name)
	createPayload["email"] = name + "@e2e-test.augno.com"
	createStatus, createBody, err := apiClient.Post(customersPath, createPayload, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	newEmail := uniqueName("e2e-cust-upd") + "@e2e-test.augno.com"
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+id+"?include=contact_info", map[string]any{
		"email": newEmail,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	got := parseJSON(patchBody)
	contactInfo := jsonObject(got, "contact_info")
	require.NotNil(t, contactInfo, "contact_info should be present with ?include=contact_info on update")
	assert.Equal(t, "customer_contact_info", jsonField(contactInfo, "object"))
	assert.Equal(t, newEmail, jsonField(contactInfo, "email"))

	apiClient.Delete(customersPath + "/" + id)
}

// Included sub-resource field completeness tests are in included_fields_test.go.

// ──────────────────────────────────────────────
// Omitted Fields
// ──────────────────────────────────────────────

func TestCustomers_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		name := uniqueName("e2e-cust-omit")
		includeStr := "contact_info,defaults,freight_preferences,notification_preferences"

		status, body, err := apiClient.Post(customersPath+"?include="+includeStr, validCustomerBody(name), newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(customersPath + "/" + id)

		// Top-level defaults
		assertIDFormat(t, id, "ac")
		assertObjectField(t, got, "customer")
		assert.Equal(t, name, jsonField(got, "name"))
		assert.NotEmpty(t, jsonField(got, "number"), "number should be auto-generated")
		assert.Equal(t, "normal", jsonField(got, "status"))
		assert.Equal(t, "false", jsonField(got, "is_edi_enabled"))
		assert.Equal(t, "false", jsonField(got, "is_parent_account"))
		assert.Equal(t, "commission_applied", jsonField(got, "commission_policy"))
		assertNilField(t, got, "note")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

		// Expandable fields not included should be nil
		assertNilField(t, got, "bill_to_address")
		assertNilField(t, got, "ship_to_address")
		assertNilField(t, got, "type")
		assertNilField(t, got, "price_groups")
		assertNilField(t, got, "parent_account")
		assertNilField(t, got, "child_accounts")

		// contact_info should be expanded but with nil sub-fields
		contactInfo := jsonObject(got, "contact_info")
		require.NotNil(t, contactInfo, "contact_info should be expanded via ?include")
		assert.Equal(t, "customer_contact_info", jsonField(contactInfo, "object"))
		assertNilField(t, contactInfo, "email")
		assertNilField(t, contactInfo, "phone")
		assertNilField(t, contactInfo, "url")

		// defaults should be expanded but with nil sub-resources
		defaults := jsonObject(got, "defaults")
		require.NotNil(t, defaults, "defaults should be expanded via ?include")
		assert.Equal(t, "customer_defaults", jsonField(defaults, "object"))
		assertNilField(t, defaults, "payment_term")
		assertNilField(t, defaults, "shipping_term")
		assertNilField(t, defaults, "priority")
		assertNilField(t, defaults, "sales_rep")

		// freight_preferences should be expanded but with nil carrier/service_level
		fp := jsonObject(got, "freight_preferences")
		require.NotNil(t, fp, "freight_preferences should be expanded via ?include")
		assert.Equal(t, "customer_freight_preferences", jsonField(fp, "object"))
		assertNilField(t, fp, "carrier")
		assertNilField(t, fp, "service_level")
		assertNilField(t, fp, "billing_type")
		assertNilField(t, fp, "billing_account")

		// notification_preferences should be expanded with default
		np := jsonObject(got, "notification_preferences")
		require.NotNil(t, np, "notification_preferences should be expanded via ?include")
		assert.Equal(t, "customer_notification_preferences", jsonField(np, "object"))
		assert.Equal(t, "false", jsonField(np, "accepts_invoice_emails"))
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		includeStr := "contact_info,defaults.payment_term,defaults.shipping_term,defaults.priority,defaults.sales_rep,freight_preferences.carrier,type,notification_preferences,freight_preferences"

		// Create with all fields
		name := uniqueName("e2e-cust-pres")
		createPayload := validCustomerBody(name)
		createPayload["note"] = "Original note"
		createPayload["email"] = name + "@e2e-test.augno.com"
		createPayload["phone"] = "555-000-1234"
		createPayload["url"] = "https://original.e2e.augno.com"
		createPayload["is_edi_enabled"] = true
		createPayload["commission_policy"] = "commission_exempt"
		createPayload["freight_policy"] = "free_freight"
		createPayload["default_priority_code"] = SeedPriorityCode
		createPayload["default_sales_rep_user_id"] = SeedUserID
		createPayload["carrier_billing_type"] = "third_party"
		createPayload["carrier_billing_account"] = "ACCT-ORIG"
		createStatus, createBody, err := apiClient.Post(customersPath+"?include="+includeStr, createPayload, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)

		created := parseJSON(createBody)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(customersPath + "/" + id)
		origNumber := jsonField(created, "number")
		origCreatedAt := jsonField(created, "created_at")

		// Update ONLY the note
		patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+id, map[string]any{
			"note": "Changed note",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		// GET with full include to verify preservation
		getStatus, getBody, err := apiClient.GetListRaw(customersPath+"/"+id, url.Values{"include": {includeStr}})
		require.NoError(t, err)
		requireStatus(t, 200, getStatus, getBody)

		got := parseJSON(getBody)

		// Updated field
		assert.Equal(t, "Changed note", jsonField(got, "note"))

		// Preserved top-level fields
		assert.Equal(t, id, jsonField(got, "id"))
		assert.Equal(t, "customer", jsonField(got, "object"))
		assert.Equal(t, name, jsonField(got, "name"), "name should be preserved")
		assert.Equal(t, origNumber, jsonField(got, "number"), "number should be preserved")
		assert.Equal(t, "normal", jsonField(got, "status"), "status should be preserved")
		assert.Equal(t, "true", jsonField(got, "is_edi_enabled"), "is_edi_enabled should be preserved")
		assert.Equal(t, "false", jsonField(got, "is_parent_account"), "is_parent_account should be preserved")
		assert.Equal(t, "commission_exempt", jsonField(got, "commission_policy"), "commission_policy should be preserved")
		assert.Equal(t, origCreatedAt, jsonField(got, "created_at"), "created_at should not change")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

		// Preserved contact_info
		ci := jsonObject(got, "contact_info")
		require.NotNil(t, ci)
		assert.Equal(t, name+"@e2e-test.augno.com", jsonField(ci, "email"), "email should be preserved")
		assert.Equal(t, "555-000-1234", jsonField(ci, "phone"), "phone should be preserved")
		assert.Equal(t, "https://original.e2e.augno.com", jsonField(ci, "url"), "url should be preserved")

		// Preserved defaults
		defaults := jsonObject(got, "defaults")
		require.NotNil(t, defaults)
		pt := jsonObject(defaults, "payment_term")
		require.NotNil(t, pt, "payment_term should be preserved")
		assert.Equal(t, SeedPaymentTermID, jsonField(pt, "id"))
		st := jsonObject(defaults, "shipping_term")
		require.NotNil(t, st, "shipping_term should be preserved")
		assert.Equal(t, SeedShippingTermID, jsonField(st, "id"))
		pri := jsonObject(defaults, "priority")
		require.NotNil(t, pri, "priority should be preserved")
		assert.Equal(t, SeedPriorityCode, jsonField(pri, "code"))
		sr := jsonObject(defaults, "sales_rep")
		require.NotNil(t, sr, "sales_rep should be preserved")
		assert.Equal(t, SeedAccountUserID, jsonField(sr, "id"))

		// Preserved freight_preferences
		fp := jsonObject(got, "freight_preferences")
		require.NotNil(t, fp)
		assert.Equal(t, "free_freight", jsonField(fp, "status"), "freight status should be preserved")
		assert.Equal(t, "third_party", jsonField(fp, "billing_type"), "billing_type should be preserved")
		assert.Equal(t, "ACCT-ORIG", jsonField(fp, "billing_account"), "billing_account should be preserved")
		carrier := jsonObject(fp, "carrier")
		require.NotNil(t, carrier, "carrier should be preserved")
		assert.Equal(t, SeedCarrierID, jsonField(carrier, "id"))

		// Preserved type
		typeGroup := jsonObject(got, "type")
		require.NotNil(t, typeGroup, "type should be preserved")
		assert.Equal(t, SeedCustomerGroupID, jsonField(typeGroup, "id"))

		// Preserved notification_preferences
		np := jsonObject(got, "notification_preferences")
		require.NotNil(t, np, "notification_preferences should be preserved")
		assert.Equal(t, "customer_notification_preferences", jsonField(np, "object"))
	})
}

// ──────────────────────────────────────────────
// Account Address Linking
// ──────────────────────────────────────────────

// listContainsID returns true if any item in the list has the given ID.
func listContainsID(list *ListResponse, targetID string) bool {
	for _, item := range list.Data {
		if DataItemField(item, "id") == targetID {
			return true
		}
	}
	return false
}

func TestCustomers_CreateWithBillingAddressLinksAccountAddress(t *testing.T) {
	t.Parallel()

	// Create a customer first.
	cust := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-cust-addr-link")))
	custID := jsonField(cust, "id")
	customerClient := apiClient.WithAccountID(custID)

	// Create an address on the customer's account.
	addrStatus, addrBody, err := customerClient.Post(addressesPath, map[string]any{
		"name":          uniqueName("e2e-addr-link"),
		"street_line_1": "100 Link St",
		"country":       "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, addrStatus, addrBody)
	addr := parseJSON(addrBody)
	addrID := jsonField(addr, "id")
	t.Cleanup(func() { customerClient.Delete(addressesPath + "/" + addrID) })

	// Update customer's billing address to the new address.
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+custID, map[string]any{
		"bill_to_address_id": addrID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	// Verify the address appears in the customer's address list.
	list, _, err := customerClient.GetList(addressesPath, nil)
	require.NoError(t, err)
	assert.True(t, listContainsID(list, addrID),
		"billing address should appear in customer's address list")

	// Verify billing address is set on the customer via include.
	getStatus, getBody, err := apiClient.GetListRaw(
		customersPath+"/"+custID,
		url.Values{"include": {"bill_to_address"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	billTo := jsonObject(got, "bill_to_address")
	require.NotNil(t, billTo, "bill_to_address should be set")
	assert.Equal(t, addrID, jsonField(billTo, "id"))
}

func TestCustomers_UpdateBillingAddressLinksAccountAddress(t *testing.T) {
	t.Parallel()

	// Create a customer.
	cust := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-cust-upd-addr")))
	custID := jsonField(cust, "id")
	customerClient := apiClient.WithAccountID(custID)

	// Create two addresses on the customer's account.
	addrAStatus, addrABody, err := customerClient.Post(addressesPath, map[string]any{
		"name":          uniqueName("e2e-addr-a"),
		"street_line_1": "100 A St",
		"country":       "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, addrAStatus, addrABody)
	addrAID := jsonField(parseJSON(addrABody), "id")
	t.Cleanup(func() { customerClient.Delete(addressesPath + "/" + addrAID) })

	addrBStatus, addrBBody, err := customerClient.Post(addressesPath, map[string]any{
		"name":          uniqueName("e2e-addr-b"),
		"street_line_1": "200 B St",
		"country":       "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, addrBStatus, addrBBody)
	addrBID := jsonField(parseJSON(addrBBody), "id")
	t.Cleanup(func() { customerClient.Delete(addressesPath + "/" + addrBID) })

	// Set billing address to A.
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+custID, map[string]any{
		"bill_to_address_id": addrAID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	// Verify address A is in the customer's address list.
	list, _, err := customerClient.GetList(addressesPath, nil)
	require.NoError(t, err)
	assert.True(t, listContainsID(list, addrAID),
		"address A should be in customer's address list after create")

	// Update customer's billing address to B.
	patchStatus2, patchBody2, err := apiClient.Patch(customersPath+"/"+custID, map[string]any{
		"bill_to_address_id": addrBID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus2, patchBody2)

	// Verify BOTH addresses are in the customer's address list.
	list2, _, err := customerClient.GetList(addressesPath, nil)
	require.NoError(t, err)
	assert.True(t, listContainsID(list2, addrAID),
		"address A should still be in customer's address list after update")
	assert.True(t, listContainsID(list2, addrBID),
		"address B should be in customer's address list after update")

	// Verify the customer's billing address is now B.
	getStatus, getBody, err := apiClient.GetListRaw(
		customersPath+"/"+custID,
		url.Values{"include": {"bill_to_address"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	billTo := jsonObject(got, "bill_to_address")
	require.NotNil(t, billTo, "bill_to_address should be set after update")
	assert.Equal(t, addrBID, jsonField(billTo, "id"))
}

func TestCustomers_UpdateShippingAddressLinksAccountAddress(t *testing.T) {
	t.Parallel()

	// Create a customer.
	cust := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-cust-ship-addr")))
	custID := jsonField(cust, "id")
	customerClient := apiClient.WithAccountID(custID)

	// Create two addresses on the customer's account.
	addrAStatus, addrABody, err := customerClient.Post(addressesPath, map[string]any{
		"name":          uniqueName("e2e-ship-a"),
		"street_line_1": "100 Ship A St",
		"country":       "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, addrAStatus, addrABody)
	addrAID := jsonField(parseJSON(addrABody), "id")
	t.Cleanup(func() { customerClient.Delete(addressesPath + "/" + addrAID) })

	addrBStatus, addrBBody, err := customerClient.Post(addressesPath, map[string]any{
		"name":          uniqueName("e2e-ship-b"),
		"street_line_1": "200 Ship B St",
		"country":       "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, addrBStatus, addrBBody)
	addrBID := jsonField(parseJSON(addrBBody), "id")
	t.Cleanup(func() { customerClient.Delete(addressesPath + "/" + addrBID) })

	// Set shipping address to A.
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+custID, map[string]any{
		"ship_to_address_id": addrAID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	// Verify address A is in the customer's address list.
	list, _, err := customerClient.GetList(addressesPath, nil)
	require.NoError(t, err)
	assert.True(t, listContainsID(list, addrAID),
		"address A should be in customer's address list after create")

	// Update customer's shipping address to B.
	patchStatus2, patchBody2, err := apiClient.Patch(customersPath+"/"+custID, map[string]any{
		"ship_to_address_id": addrBID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus2, patchBody2)

	// Verify BOTH addresses are in the customer's address list.
	list2, _, err := customerClient.GetList(addressesPath, nil)
	require.NoError(t, err)
	assert.True(t, listContainsID(list2, addrAID),
		"address A should still be in customer's address list after update")
	assert.True(t, listContainsID(list2, addrBID),
		"address B should be in customer's address list after update")
}

func TestCustomers_SameAddressForBillingAndShipping(t *testing.T) {
	t.Parallel()

	// Create a customer.
	cust := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-cust-same-addr")))
	custID := jsonField(cust, "id")
	customerClient := apiClient.WithAccountID(custID)

	// Create a single address on the customer's account.
	addrStatus, addrBody, err := customerClient.Post(addressesPath, map[string]any{
		"name":          uniqueName("e2e-addr-both"),
		"street_line_1": "100 Both St",
		"country":       "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, addrStatus, addrBody)
	addrID := jsonField(parseJSON(addrBody), "id")
	t.Cleanup(func() { customerClient.Delete(addressesPath + "/" + addrID) })

	// Set the same address for both billing and shipping (no duplicate key error).
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+custID, map[string]any{
		"bill_to_address_id": addrID,
		"ship_to_address_id": addrID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	// Verify address is in the customer's address list.
	list, _, err := customerClient.GetList(addressesPath, nil)
	require.NoError(t, err)
	assert.True(t, listContainsID(list, addrID),
		"address should appear in customer's address list")

	// Verify both billing and shipping point to the same address.
	getStatus, getBody, err := apiClient.GetListRaw(
		customersPath+"/"+custID,
		url.Values{"include": {"bill_to_address,ship_to_address"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)

	billTo := jsonObject(got, "bill_to_address")
	require.NotNil(t, billTo, "bill_to_address should be set")
	assert.Equal(t, addrID, jsonField(billTo, "id"))

	shipTo := jsonObject(got, "ship_to_address")
	require.NotNil(t, shipTo, "ship_to_address should be set")
	assert.Equal(t, addrID, jsonField(shipTo, "id"))
}

// ──────────────────────────────────────────────
// Price Group Update Tests
// ──────────────────────────────────────────────

func TestCustomers_UpdatePriceGroups(t *testing.T) {
	t.Parallel()

	// Create two pricing groups.
	pg1 := createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": uniqueName("e2e-pg1"),
		"type": "pricing_group",
	})
	pg1ID := jsonField(pg1, "id")

	pg2 := createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": uniqueName("e2e-pg2"),
		"type": "pricing_group",
	})
	pg2ID := jsonField(pg2, "id")

	// Create a customer.
	cust := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-cust-pg")))
	custID := jsonField(cust, "id")

	// Verify no price groups initially.
	getStatus, getBody, err := apiClient.GetListRaw(
		customersPath+"/"+custID,
		url.Values{"include": {"price_groups"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	pgList := jsonObject(got, "price_groups")
	require.NotNil(t, pgList, "price_groups should be present with ?include")
	pgData, _ := pgList["data"].([]any)
	assert.Empty(t, pgData, "price_groups should be empty initially")

	// Set price groups to [pg1, pg2].
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+custID, map[string]any{
		"customer_price_group_ids": []string{pg1ID, pg2ID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	// Verify both price groups are set.
	getStatus, getBody, err = apiClient.GetListRaw(
		customersPath+"/"+custID,
		url.Values{"include": {"price_groups"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got = parseJSON(getBody)
	pgList = jsonObject(got, "price_groups")
	require.NotNil(t, pgList)
	pgData, _ = pgList["data"].([]any)
	assert.Len(t, pgData, 2, "should have 2 price groups")

	pgIDs := make(map[string]bool)
	for _, item := range pgData {
		if m, ok := item.(map[string]any); ok {
			pgIDs[jsonField(m, "id")] = true
		}
	}
	assert.True(t, pgIDs[pg1ID], "pg1 should be present")
	assert.True(t, pgIDs[pg2ID], "pg2 should be present")

	// Replace price groups with only [pg1].
	patchStatus, patchBody, err = apiClient.Patch(customersPath+"/"+custID, map[string]any{
		"customer_price_group_ids": []string{pg1ID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	// Verify only pg1 remains.
	getStatus, getBody, err = apiClient.GetListRaw(
		customersPath+"/"+custID,
		url.Values{"include": {"price_groups"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got = parseJSON(getBody)
	pgList = jsonObject(got, "price_groups")
	require.NotNil(t, pgList)
	pgData, _ = pgList["data"].([]any)
	assert.Len(t, pgData, 1, "should have 1 price group after replace")
	if len(pgData) > 0 {
		if m, ok := pgData[0].(map[string]any); ok {
			assert.Equal(t, pg1ID, jsonField(m, "id"))
		}
	}

	// Clear all price groups by sending empty array.
	patchStatus, patchBody, err = apiClient.Patch(customersPath+"/"+custID, map[string]any{
		"customer_price_group_ids": []string{},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	// Verify price groups are cleared.
	getStatus, getBody, err = apiClient.GetListRaw(
		customersPath+"/"+custID,
		url.Values{"include": {"price_groups"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got = parseJSON(getBody)
	pgList = jsonObject(got, "price_groups")
	require.NotNil(t, pgList)
	pgData, _ = pgList["data"].([]any)
	assert.Empty(t, pgData, "price_groups should be empty after clearing")
}

func TestCustomers_UpdateContactInfo(t *testing.T) {
	t.Parallel()

	// Create a customer with contact info.
	name := uniqueName("e2e-cust-ci")
	payload := validCustomerBody(name)
	payload["email"] = name + "@test.com"
	payload["phone"] = "555-111-2222"
	payload["url"] = "https://original.example.com"
	cust := createAndCleanup(t, customersPath+"?include=contact_info", payload)
	custID := jsonField(cust, "id")

	// Verify contact info was set.
	ci := jsonObject(cust, "contact_info")
	require.NotNil(t, ci)
	assert.Equal(t, name+"@test.com", jsonField(ci, "email"))
	assert.Equal(t, "555-111-2222", jsonField(ci, "phone"))
	assert.Equal(t, "https://original.example.com", jsonField(ci, "url"))

	// Update email only — phone and url should be preserved.
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+custID+"?include=contact_info", map[string]any{
		"email": name + "-updated@test.com",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updatedCI := jsonObject(parseJSON(patchBody), "contact_info")
	require.NotNil(t, updatedCI)
	assert.Equal(t, name+"-updated@test.com", jsonField(updatedCI, "email"), "email should be updated")
	assert.Equal(t, "555-111-2222", jsonField(updatedCI, "phone"), "phone should be preserved")
	assert.Equal(t, "https://original.example.com", jsonField(updatedCI, "url"), "url should be preserved")

	// Clear email by sending null — phone and url should be preserved.
	patchStatus, patchBody, err = apiClient.Patch(customersPath+"/"+custID+"?include=contact_info", map[string]any{
		"email": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	clearedCI := jsonObject(parseJSON(patchBody), "contact_info")
	require.NotNil(t, clearedCI)
	assertNilField(t, clearedCI, "email")
	assert.Equal(t, "555-111-2222", jsonField(clearedCI, "phone"), "phone should be preserved after clearing email")
	assert.Equal(t, "https://original.example.com", jsonField(clearedCI, "url"), "url should be preserved after clearing email")

	// Update without sending any contact fields — all should be preserved.
	patchStatus, patchBody, err = apiClient.Patch(customersPath+"/"+custID+"?include=contact_info", map[string]any{
		"note": "unrelated update",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	preservedCI := jsonObject(parseJSON(patchBody), "contact_info")
	require.NotNil(t, preservedCI)
	assertNilField(t, preservedCI, "email")
	assert.Equal(t, "555-111-2222", jsonField(preservedCI, "phone"), "phone should still be preserved")
	assert.Equal(t, "https://original.example.com", jsonField(preservedCI, "url"), "url should still be preserved")
}

func TestCustomers_UpdateAndClearNote(t *testing.T) {
	t.Parallel()

	notePayload := validCustomerBody(uniqueName("e2e-cust-note"))
	notePayload["note"] = "initial note"
	cust := createAndCleanup(t, customersPath, notePayload)
	custID := jsonField(cust, "id")
	assert.Equal(t, "initial note", jsonField(cust, "note"))

	// Update note to a new value.
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+custID, map[string]any{
		"note": "updated note",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, "updated note", jsonField(parseJSON(patchBody), "note"))

	// Clear note by sending null.
	patchStatus, patchBody, err = apiClient.Patch(customersPath+"/"+custID, map[string]any{
		"note": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assertNilField(t, parseJSON(patchBody), "note")

	// Update unrelated field — note should remain cleared.
	patchStatus, patchBody, err = apiClient.Patch(customersPath+"/"+custID, map[string]any{
		"number": "NUM-999",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assertNilField(t, parseJSON(patchBody), "note")
}

func TestCustomers_ClearNullableDefaults(t *testing.T) {
	t.Parallel()

	includeAll := url.Values{
		"include[]": {
			"defaults",
			"defaults.payment_term",
			"defaults.shipping_term",
			"defaults.sales_rep",
			"freight_preferences",
			"freight_preferences.carrier",
			"freight_preferences.service_level",
		},
	}
	includeQS := "?" + includeAll.Encode()

	// Create customer with all default fields set.
	clrPayload := validCustomerBody(uniqueName("e2e-cust-clr"))
	clrPayload["default_sales_rep_user_id"] = SeedUserID
	cust := createAndCleanup(t, customersPath+includeQS, clrPayload)
	custID := jsonField(cust, "id")

	// Verify defaults were set.
	defaults := jsonObject(cust, "defaults")
	require.NotNil(t, defaults)
	pt := jsonObject(defaults, "payment_term")
	require.NotNil(t, pt, "payment_term should be set")
	assert.Equal(t, SeedPaymentTermID, jsonField(pt, "id"))

	fp := jsonObject(cust, "freight_preferences")
	require.NotNil(t, fp)
	carrier := jsonObject(fp, "carrier")
	require.NotNil(t, carrier, "carrier should be set")

	// Clear payment term by sending null — other defaults should be preserved.
	patchStatus, patchBody, err := apiClient.Patch(
		customersPath+"/"+custID+includeQS,
		map[string]any{
			"default_payment_term_id": nil,
		},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	patchedDefaults := jsonObject(patched, "defaults")
	require.NotNil(t, patchedDefaults)
	assertNilField(t, patchedDefaults, "payment_term")

	// Shipping term, sales rep, carrier should be preserved.
	patchedST := jsonObject(patchedDefaults, "shipping_term")
	require.NotNil(t, patchedST, "shipping_term should be preserved")
	assert.Equal(t, SeedShippingTermID, jsonField(patchedST, "id"))

	patchedSR := jsonObject(patchedDefaults, "sales_rep")
	require.NotNil(t, patchedSR, "sales_rep should be preserved")

	patchedFP := jsonObject(patched, "freight_preferences")
	require.NotNil(t, patchedFP)
	patchedCarrier := jsonObject(patchedFP, "carrier")
	require.NotNil(t, patchedCarrier, "carrier should be preserved")

	// Update unrelated field — all remaining defaults should be preserved.
	patchStatus2, patchBody2, err := apiClient.Patch(
		customersPath+"/"+custID+includeQS,
		map[string]any{
			"note": "unrelated change",
		},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus2, patchBody2)

	preserved := parseJSON(patchBody2)
	preservedDefaults := jsonObject(preserved, "defaults")
	require.NotNil(t, preservedDefaults)
	assertNilField(t, preservedDefaults, "payment_term")
	preservedST := jsonObject(preservedDefaults, "shipping_term")
	require.NotNil(t, preservedST, "shipping_term should still be preserved")
	preservedSR := jsonObject(preservedDefaults, "sales_rep")
	require.NotNil(t, preservedSR, "sales_rep should still be preserved")
}

// ──────────────────────────────────────────────
// Nullable Field Validation
// ──────────────────────────────────────────────

func TestCustomers_UpdateRejectsNullForNonNullableFields(t *testing.T) {
	t.Parallel()

	cust := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-cust-nonnull")))
	custID := jsonField(cust, "id")

	tests := []struct {
		name  string
		field string
		param string
	}{
		{"CommissionPolicy", "commission_policy", "commission_policy"},
		{"FreightPolicy", "freight_policy", "freight_policy"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := apiClient.Patch(customersPath+"/"+custID, map[string]any{
				tc.field: nil,
			}, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
			assertErrorParam(t, errObj, tc.param)
		})
	}
}

func TestCustomers_CreateRejectsNullForNonNullableFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		field string
		param string
	}{
		{"CommissionPolicy", "commission_policy", "commission_policy"},
		{"FreightPolicy", "freight_policy", "freight_policy"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := validCustomerBody(uniqueName("e2e-cust-nonnull"))
			payload[tc.field] = nil
			status, body, err := apiClient.Post(customersPath, payload, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
			assertErrorParam(t, errObj, tc.param)
		})
	}
}

func TestCustomers_UpdatePreservesPriceGroupsWhenNotSent(t *testing.T) {
	t.Parallel()

	// Create a pricing group.
	pg := createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": uniqueName("e2e-pg-pres"),
		"type": "pricing_group",
	})
	pgID := jsonField(pg, "id")

	// Create a customer with the pricing group.
	pgPresPayload := validCustomerBody(uniqueName("e2e-cust-pg-pres"))
	pgPresPayload["customer_price_group_ids"] = []string{pgID}
	cust := createAndCleanup(t, customersPath, pgPresPayload)
	custID := jsonField(cust, "id")

	// Update an unrelated field (note) — should NOT clear price groups.
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+custID, map[string]any{
		"note": "Updated note, price groups should remain",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	// Verify price group is still set.
	getStatus, getBody, err := apiClient.GetListRaw(
		customersPath+"/"+custID,
		url.Values{"include": {"price_groups"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	pgList := jsonObject(got, "price_groups")
	require.NotNil(t, pgList)
	pgData, _ := pgList["data"].([]any)
	require.Len(t, pgData, 1, "price group should be preserved when not sent in update")
	if m, ok := pgData[0].(map[string]any); ok {
		assert.Equal(t, pgID, jsonField(m, "id"))
	}

	// Send explicit null for price group IDs — should also preserve (null = absent for slices).
	patchStatus, patchBody, err = apiClient.Patch(customersPath+"/"+custID, map[string]any{
		"customer_price_group_ids": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	getStatus, getBody, err = apiClient.GetListRaw(
		customersPath+"/"+custID,
		url.Values{"include": {"price_groups"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got = parseJSON(getBody)
	pgList = jsonObject(got, "price_groups")
	require.NotNil(t, pgList)
	pgData, _ = pgList["data"].([]any)
	require.Len(t, pgData, 1, "price group should be preserved when null is sent")
}
