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
	createResp, err := apiClient.PostFull(customersPath, forgeCustomerWrite(validCustomerBody(name)), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
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
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+id, forgeCustomerWrite(map[string]any{
		"note": newNote,
	}), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	updated := parseJSON(patchBody)
	assert.Equal(t, newNote, jsonField(updated, "instructions"))
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

func TestCustomers_CreateValidation_MissingName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(customersPath, forgeCustomerWrite(map[string]any{}), newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing name should return 400 or 422, got %d: %s", status, string(body))
}

func TestCustomers_CreateValidation_EmptyName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(customersPath, forgeCustomerWrite(map[string]any{
		"name": "",
	}), newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Empty name should return 400 or 422, got %d: %s", status, string(body))
}

func TestCustomers_CreateValidation_MissingName_ErrorShape(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(customersPath, forgeCustomerWrite(map[string]any{}), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)

	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

func TestCustomers_UpdateValidation_EmptyName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cust-blank")
	createStatus, createBody, err := apiClient.Post(customersPath, forgeCustomerWrite(validCustomerBody(name)), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	defer apiClient.Delete(customersPath + "/" + id)

	status, body, err := apiClient.Patch(customersPath+"/"+id, forgeCustomerWrite(map[string]any{"name": ""}), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
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
	status, body, err := apiClient.Post(customersPath, forgeCustomerWrite(validCustomerBody(name)), newIdempotencyKey())
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

	status1, body1, err := apiClient.Post(customersPath, forgeCustomerWrite(payload), idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(customersPath, forgeCustomerWrite(payload), idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(customersPath + "/" + id1)
}

func TestCustomers_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-cust-upd")
	createStatus, createBody, err := apiClient.Post(customersPath, forgeCustomerWrite(validCustomerBody(name)), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	idemKey := newIdempotencyKey()
	payload := map[string]any{"note": "idempotent note"}

	status1, body1, err := apiClient.Patch(customersPath+"/"+id, forgeCustomerWrite(payload), idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(customersPath+"/"+id, forgeCustomerWrite(payload), idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	assert.Equal(t, jsonField(parseJSON(body1), "instructions"), jsonField(parseJSON(body2), "instructions"))

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
	assert.Equal(t, "normal", jsonField(got, "standing"))
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
	createStatus, createBody, err := apiClient.Post(customersPath, forgeCustomerWrite(validCustomerBody(name)), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+id, forgeCustomerWrite(map[string]any{
		"note": "a note",
	}), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, "a note", jsonField(patched, "instructions"))
	assert.Equal(t, name, jsonField(patched, "name"), "name should be preserved")

	apiClient.Delete(customersPath + "/" + id)
}

func TestCustomers_UpdateMultipleFields(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cust-upd-multi")
	createStatus, createBody, err := apiClient.Post(customersPath, forgeCustomerWrite(validCustomerBody(name)), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+id, forgeCustomerWrite(map[string]any{
		"note":              "multi-field update",
		"phone":             "555-999-8888",
		"commission_policy": "exempt",
	}), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, name, jsonField(patched, "name"), "name should be preserved")
	assert.Equal(t, "multi-field update", jsonField(patched, "instructions"))
	assert.Equal(t, "exempt", jsonField(patched, "commission_policy"))

	apiClient.Delete(customersPath + "/" + id)
}

// ──────────────────────────────────────────────
// Delete Tests
// ──────────────────────────────────────────────

func TestCustomers_DeleteThenGetReturns404(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cust-del")
	createStatus, createBody, err := apiClient.Post(customersPath, forgeCustomerWrite(validCustomerBody(name)), newIdempotencyKey())
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

// ──────────────────────────────────────────────
// Single Include Tests
// ──────────────────────────────────────────────

func TestCustomers_IncludeBillToAddress(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"defaults.bill_to_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	addr := jsonObject(jsonObject(got, "defaults"), "bill_to_address")
	require.NotNil(t, addr, "bill_to_address should be present with ?include=defaults.bill_to_address")
	assert.Equal(t, "address", jsonField(addr, "object"))
	assert.Equal(t, SeedAddressID, jsonField(addr, "id"))
}

func TestCustomers_IncludeShipToAddress(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"defaults.ship_to_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	addr := jsonObject(jsonObject(got, "defaults"), "ship_to_address")
	require.NotNil(t, addr, "ship_to_address should be present with ?include=defaults.ship_to_address")
	assert.Equal(t, "address", jsonField(addr, "object"))
	assert.NotEmpty(t, jsonField(addr, "id"))
}

func TestCustomers_IncludeType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"group"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	typeGroup := jsonObject(got, "group")
	require.NotNil(t, typeGroup, "type should be present with ?include=group")
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
		"amount":   "5000.00",
		"currency": "usd",
	}
	cust := createAndCleanup(t, customersPath+"?include=", payload)

	cl := jsonObject(cust, "credit_limit")
	require.NotNil(t, cl, "credit_limit should be present with ?include=")
	assert.Equal(t, map[string]any{"amount": "5000.00", "currency": "usd"}, cl, "money is its amount and currency, nothing more")
}

func TestCustomers_IncludeCreditLimitNullWhenNotSet(t *testing.T) {
	t.Parallel()

	// Create a customer without a credit limit.
	cust := createAndCleanup(t, customersPath+"?include=", validCustomerBody(uniqueName("e2e-cust-cl-nil")))
	assertNilField(t, cust, "credit_limit")
}

// Money is dollars only: any other currency is refused on create and on update, naming the field.
func TestCustomers_CreditLimitRejectsAnUnsupportedCurrency(t *testing.T) {
	t.Parallel()
	euros := map[string]any{"amount": "5000.00", "currency": "eur"}

	payload := validCustomerBody(uniqueName("e2e-cust-cl-eur"))
	payload["credit_limit"] = euros
	status, body, err := apiClient.Post(customersPath, forgeCustomerWrite(payload), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "currency_not_supported", "invalid_request_error"), "credit_limit.currency")

	cust := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-cust-cl-eur-upd")))
	status, body, err = apiClient.Patch(customersPath+"/"+jsonField(cust, "id"), forgeCustomerWrite(map[string]any{"credit_limit": euros}), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "currency_not_supported", "invalid_request_error"), "credit_limit.currency")
}

// ──────────────────────────────────────────────
// Nested Include Tests
// ──────────────────────────────────────────────

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
	require.NotNil(t, defaults, "defaults is always present")
	// sales_rep may be null if the seeded customer has no sales rep assigned
}

// ──────────────────────────────────────────────
// Multiple Includes
// ──────────────────────────────────────────────

// ──────────────────────────────────────────────
// Include on Create / Update
// ──────────────────────────────────────────────

// Included sub-resource field completeness tests are in included_fields_test.go.

// ──────────────────────────────────────────────
// Omitted Fields
// ──────────────────────────────────────────────

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
	addrResp, err := customerClient.PostFull(addressesPath, map[string]any{
		"name":    uniqueName("e2e-addr-link"),
		"line_1":  "100 Link St",
		"country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, addrResp.StatusCode, addrResp.Body)
	addr := parseJSON(addrResp.Body)
	addrID := jsonField(addr, "id")
	assertCreatedLocation(t, addrResp.Header, addrID)
	t.Cleanup(func() { customerClient.Delete(addressesPath + "/" + addrID) })

	// Update customer's billing address to the new address.
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+custID, forgeCustomerWrite(map[string]any{
		"bill_to_address_id": addrID,
	}), newIdempotencyKey())
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
		url.Values{"include": {"defaults.bill_to_address"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	billTo := jsonObject(jsonObject(got, "defaults"), "bill_to_address")
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
		"name":    uniqueName("e2e-addr-a"),
		"line_1":  "100 A St",
		"country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, addrAStatus, addrABody)
	addrAID := jsonField(parseJSON(addrABody), "id")
	t.Cleanup(func() { customerClient.Delete(addressesPath + "/" + addrAID) })

	addrBStatus, addrBBody, err := customerClient.Post(addressesPath, map[string]any{
		"name":    uniqueName("e2e-addr-b"),
		"line_1":  "200 B St",
		"country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, addrBStatus, addrBBody)
	addrBID := jsonField(parseJSON(addrBBody), "id")
	t.Cleanup(func() { customerClient.Delete(addressesPath + "/" + addrBID) })

	// Set billing address to A.
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+custID, forgeCustomerWrite(map[string]any{
		"bill_to_address_id": addrAID,
	}), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	// Verify address A is in the customer's address list.
	list, _, err := customerClient.GetList(addressesPath, nil)
	require.NoError(t, err)
	assert.True(t, listContainsID(list, addrAID),
		"address A should be in customer's address list after create")

	// Update customer's billing address to B.
	patchStatus2, patchBody2, err := apiClient.Patch(customersPath+"/"+custID, forgeCustomerWrite(map[string]any{
		"bill_to_address_id": addrBID,
	}), newIdempotencyKey())
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
		url.Values{"include": {"defaults.bill_to_address"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	billTo := jsonObject(jsonObject(got, "defaults"), "bill_to_address")
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
		"name":    uniqueName("e2e-ship-a"),
		"line_1":  "100 Ship A St",
		"country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, addrAStatus, addrABody)
	addrAID := jsonField(parseJSON(addrABody), "id")
	t.Cleanup(func() { customerClient.Delete(addressesPath + "/" + addrAID) })

	addrBStatus, addrBBody, err := customerClient.Post(addressesPath, map[string]any{
		"name":    uniqueName("e2e-ship-b"),
		"line_1":  "200 Ship B St",
		"country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, addrBStatus, addrBBody)
	addrBID := jsonField(parseJSON(addrBBody), "id")
	t.Cleanup(func() { customerClient.Delete(addressesPath + "/" + addrBID) })

	// Set shipping address to A.
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+custID, forgeCustomerWrite(map[string]any{
		"ship_to_address_id": addrAID,
	}), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	// Verify address A is in the customer's address list.
	list, _, err := customerClient.GetList(addressesPath, nil)
	require.NoError(t, err)
	assert.True(t, listContainsID(list, addrAID),
		"address A should be in customer's address list after create")

	// Update customer's shipping address to B.
	patchStatus2, patchBody2, err := apiClient.Patch(customersPath+"/"+custID, forgeCustomerWrite(map[string]any{
		"ship_to_address_id": addrBID,
	}), newIdempotencyKey())
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
	addrResp, err := customerClient.PostFull(addressesPath, map[string]any{
		"name":    uniqueName("e2e-addr-both"),
		"line_1":  "100 Both St",
		"country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, addrResp.StatusCode, addrResp.Body)
	addrID := jsonField(parseJSON(addrResp.Body), "id")
	assertCreatedLocation(t, addrResp.Header, addrID)
	t.Cleanup(func() { customerClient.Delete(addressesPath + "/" + addrID) })

	// Set the same address for both billing and shipping (no duplicate key error).
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+custID, forgeCustomerWrite(map[string]any{
		"bill_to_address_id": addrID,
		"ship_to_address_id": addrID,
	}), newIdempotencyKey())
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
		url.Values{"include": {"defaults.bill_to_address,defaults.ship_to_address"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)

	billTo := jsonObject(jsonObject(got, "defaults"), "bill_to_address")
	require.NotNil(t, billTo, "bill_to_address should be set")
	assert.Equal(t, addrID, jsonField(billTo, "id"))

	shipTo := jsonObject(jsonObject(got, "defaults"), "ship_to_address")
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
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+custID, forgeCustomerWrite(map[string]any{
		"customer_price_group_ids": []string{pg1ID, pg2ID},
	}), newIdempotencyKey())
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
	patchStatus, patchBody, err = apiClient.Patch(customersPath+"/"+custID, forgeCustomerWrite(map[string]any{
		"customer_price_group_ids": []string{pg1ID},
	}), newIdempotencyKey())
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
	patchStatus, patchBody, err = apiClient.Patch(customersPath+"/"+custID, forgeCustomerWrite(map[string]any{
		"customer_price_group_ids": []string{},
	}), newIdempotencyKey())
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

func TestCustomers_UpdateAndClearNote(t *testing.T) {
	t.Parallel()

	notePayload := validCustomerBody(uniqueName("e2e-cust-note"))
	notePayload["note"] = "initial note"
	cust := createAndCleanup(t, customersPath, notePayload)
	custID := jsonField(cust, "id")
	assert.Equal(t, "initial note", jsonField(cust, "instructions"))

	// Update note to a new value.
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+custID, forgeCustomerWrite(map[string]any{
		"note": "updated note",
	}), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, "updated note", jsonField(parseJSON(patchBody), "instructions"))

	// Clear note by sending null.
	patchStatus, patchBody, err = apiClient.Patch(customersPath+"/"+custID, forgeCustomerWrite(map[string]any{
		"note": nil,
	}), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assertNilField(t, parseJSON(patchBody), "instructions")

	// Update unrelated field — note should remain cleared.
	patchStatus, patchBody, err = apiClient.Patch(customersPath+"/"+custID, forgeCustomerWrite(map[string]any{
		"number": "NUM-999",
	}), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assertNilField(t, parseJSON(patchBody), "instructions")
}

func TestCustomers_ClearNullableDefaults(t *testing.T) {
	t.Parallel()

	includeAll := url.Values{
		"include[]": {
			"defaults.payment_term",
			"defaults.shipping_term",
			"defaults.sales_rep",
			"defaults.carrier",
			"defaults.service_level",
		},
	}
	includeQS := "?" + includeAll.Encode()

	// Create customer with all default fields set.
	clrPayload := validCustomerBody(uniqueName("e2e-cust-clr"))
	clrPayload["default_sales_rep_id"] = SeedAccountUserID
	cust := createAndCleanup(t, customersPath+includeQS, clrPayload)
	custID := jsonField(cust, "id")

	// Verify defaults were set.
	defaults := jsonObject(cust, "defaults")
	require.NotNil(t, defaults)
	sr := jsonObject(defaults, "sales_rep")
	require.NotNil(t, sr, "sales_rep should be set")

	carrier := jsonObject(defaults, "carrier")
	require.NotNil(t, carrier, "carrier should be set")

	// Clear sales rep by sending null — other defaults should be preserved.
	patchStatus, patchBody, err := apiClient.Patch(
		customersPath+"/"+custID+includeQS,
		forgeCustomerWrite(map[string]any{
			"default_sales_rep_id": nil,
		}),
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	patchedDefaults := jsonObject(patched, "defaults")
	require.NotNil(t, patchedDefaults)
	assertNilField(t, patchedDefaults, "sales_rep")

	// Payment term, shipping term, carrier should be preserved.
	patchedPT := jsonObject(patchedDefaults, "payment_term")
	require.NotNil(t, patchedPT, "payment_term should be preserved")
	assert.Equal(t, SeedPaymentTermID, jsonField(patchedPT, "id"))

	patchedST := jsonObject(patchedDefaults, "shipping_term")
	require.NotNil(t, patchedST, "shipping_term should be preserved")
	assert.Equal(t, SeedShippingTermID, jsonField(patchedST, "id"))

	patchedCarrier := jsonObject(patchedDefaults, "carrier")
	require.NotNil(t, patchedCarrier, "carrier should be preserved")

	// Update unrelated field — all remaining defaults should be preserved.
	patchStatus2, patchBody2, err := apiClient.Patch(
		customersPath+"/"+custID+includeQS,
		forgeCustomerWrite(map[string]any{
			"note": "unrelated change",
		}),
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus2, patchBody2)

	preserved := parseJSON(patchBody2)
	preservedDefaults := jsonObject(preserved, "defaults")
	require.NotNil(t, preservedDefaults)
	assertNilField(t, preservedDefaults, "sales_rep")
	preservedPT := jsonObject(preservedDefaults, "payment_term")
	require.NotNil(t, preservedPT, "payment_term should still be preserved")
	preservedST := jsonObject(preservedDefaults, "shipping_term")
	require.NotNil(t, preservedST, "shipping_term should still be preserved")
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
			status, body, err := apiClient.Patch(customersPath+"/"+custID, forgeCustomerWrite(map[string]any{
				tc.field: nil,
			}), newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 422, status, body)
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
			status, body, err := apiClient.Post(customersPath, forgeCustomerWrite(payload), newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 422, status, body)
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
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+custID, forgeCustomerWrite(map[string]any{
		"note": "Updated note, price groups should remain",
	}), newIdempotencyKey())
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

	// Sending an empty array should clear the price groups.
	patchStatus, patchBody, err = apiClient.Patch(customersPath+"/"+custID, forgeCustomerWrite(map[string]any{
		"customer_price_group_ids": []string{},
	}), newIdempotencyKey())
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
	require.Len(t, pgData, 0, "price groups should be cleared when empty array is sent")
}
