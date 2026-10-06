//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A record that saves an address with its own write (a purchase order's supplier addresses, a sales
// order's customer addresses, an account's default addresses) needs only the record's permission:
// the address is part of the write the caller is already allowed to make.

func inlineAddressBody(name, street string) map[string]any {
	return map[string]any{
		"name":          name,
		"street_line_1": street,
		"locality":      "Columbus",
		"state":         "OH",
		"postal_code":   "43204",
		"country":       "US",
	}
}

// trackInlineAddresses deletes, when the test ends, the addresses saved into accountID. Register it before the record that uses them, so the record is gone first.
func trackInlineAddresses(t *testing.T, accountID string) *[]string {
	t.Helper()
	ids := &[]string{}
	t.Cleanup(func() {
		for _, id := range *ids {
			_, _, _ = apiClient.WithAccountID(accountID).Delete(addressesPath + "/" + id)
		}
	})
	return ids
}

func addressIn(t *testing.T, accountID, addressID string) map[string]any {
	t.Helper()
	status, body, err := apiClient.WithAccountID(accountID).GetListRaw(addressesPath+"/"+addressID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func savedCounterpartyAddress(t *testing.T, accountID, name string) string {
	t.Helper()
	status, body, err := apiClient.WithAccountID(accountID).Post(addressesPath, map[string]any{
		"name":          name,
		"street_line_1": "9 Spindle Way",
		"locality":      "Los Angeles",
		"state":         "CA",
		"postal_code":   "90001",
		"country":       "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	return jsonField(parseJSON(body), "id")
}

func tenantBAddress(t *testing.T) string {
	t.Helper()
	status, body, err := getTenantBClient().Post(addressesPath, inlineAddressBody(uniqueName("e2e-inline-tenant-b"), "1 Elsewhere St"), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { _, _, _ = getTenantBClient().Delete(addressesPath + "/" + id) })
	return id
}

func inlineSupplier(t *testing.T) string {
	t.Helper()
	return jsonField(createAndCleanup(t, suppliersPath, map[string]any{"name": uniqueName("e2e-inline-sup"), "number": uniqueName("SUP")}), "id")
}

func inlinePurchaseOrderBody(supplierID string) map[string]any {
	return map[string]any{
		"supplier_account_id": supplierID,
		"priority_code":       SeedPriorityCode,
		"lines":               []map[string]any{purchaseOrderLineBody("E2E-PO-INLINE")},
	}
}

func inlineSalesOrderBody(customerID string) map[string]any {
	return map[string]any{
		"buyer_account_id": customerID,
		"carrier_id":       SeedCarrierID,
		"service_level_id": SeedServiceLevelID,
		"priority_code":    "normal",
		"payment_term_id":  SeedPaymentTermID,
		"shipping_term_id": SeedShippingTermID,
		"lines": []map[string]any{
			{"product_id": SeedProductID, "quantity": map[string]any{"value": "1", "unit_id": SeedUnitID}},
		},
	}
}

// getWithAddresses reads a record with both of its address includes.
func getWithAddresses(t *testing.T, path, billKey, shipKey string) (bill, ship map[string]any) {
	t.Helper()
	status, body, err := apiClient.GetListRaw(path, url.Values{"include": {billKey + "," + shipKey}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	bill, ship = jsonObject(got, billKey), jsonObject(got, shipKey)
	require.NotNil(t, bill, "%s must be returned: %s", billKey, body)
	require.NotNil(t, ship, "%s must be returned: %s", shipKey, body)
	return bill, ship
}

// --- Purchase orders ---

func TestInlineAddresses_PurchaseOrderCreateNeedsOnlyCreatePermission(t *testing.T) {
	t.Parallel()
	supplierID := inlineSupplier(t)
	saved := trackInlineAddresses(t, supplierID)
	existingID := savedCounterpartyAddress(t, supplierID, uniqueName("e2e-inline-po-existing"))
	*saved = append(*saved, existingID)
	client := customRoleClient(t, "purchase_orders:create")

	billName, shipName := uniqueName("e2e-inline-po-bill"), uniqueName("e2e-inline-po-ship")
	body := inlinePurchaseOrderBody(supplierID)
	body["bill_to_address"] = inlineAddressBody(billName, "1 Inline Way")
	body["ship_to_address"] = map[string]any{"id": existingID, "name": shipName, "street_line_1": "2 Updated Rd", "phone": "555-0101"}

	status, resp, err := client.Post(purchaseOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	orderID := jsonField(parseJSON(resp), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(purchaseOrdersPath + "/" + orderID) })

	bill, ship := getWithAddresses(t, purchaseOrdersPath+"/"+orderID, "bill_to_address", "ship_to_address")
	billID := jsonField(bill, "id")
	*saved = append(*saved, billID)
	assert.NotEqual(t, existingID, billID, "an inline address without an id is a new address")
	assert.Equal(t, billName, jsonField(bill, "name"))
	assert.Equal(t, existingID, jsonField(ship, "id"), "an inline address with an id keeps that address")
	assert.Equal(t, shipName, jsonField(ship, "name"))

	created := addressIn(t, supplierID, billID)
	assert.Equal(t, billName, jsonField(created, "name"), "the new address is saved to the supplier")
	assert.Equal(t, "1 Inline Way", jsonField(jsonObject(created, "geolocation"), "street_line_1"))
	assert.Equal(t, "US", jsonField(jsonObject(created, "geolocation"), "country"))

	updated := addressIn(t, supplierID, existingID)
	assert.Equal(t, shipName, jsonField(updated, "name"))
	assert.Equal(t, "555-0101", jsonField(updated, "phone"))
	assert.Equal(t, "2 Updated Rd", jsonField(jsonObject(updated, "geolocation"), "street_line_1"))
	assert.Equal(t, "Los Angeles", jsonField(jsonObject(updated, "geolocation"), "locality"), "fields the inline update leaves out are kept")

	expectAuditEvent(t, billID, "address", "create")
	expectAuditEvent(t, existingID, "address", "update")
}

func TestInlineAddresses_PurchaseOrderIdenticalAddressesAreSavedOnce(t *testing.T) {
	t.Parallel()
	supplierID := inlineSupplier(t)
	saved := trackInlineAddresses(t, supplierID)
	client := customRoleClient(t, "purchase_orders:create")

	address := inlineAddressBody(uniqueName("e2e-inline-po-same"), "3 Same St")
	body := inlinePurchaseOrderBody(supplierID)
	body["bill_to_address"] = address
	body["ship_to_address"] = address

	status, resp, err := client.Post(purchaseOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	orderID := jsonField(parseJSON(resp), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(purchaseOrdersPath + "/" + orderID) })

	bill, ship := getWithAddresses(t, purchaseOrdersPath+"/"+orderID, "bill_to_address", "ship_to_address")
	*saved = append(*saved, jsonField(bill, "id"))
	assert.Equal(t, jsonField(bill, "id"), jsonField(ship, "id"), "one address entered for both sides is saved once")
}

func TestInlineAddresses_PurchaseOrderUpdateNeedsOnlyUpdatePermission(t *testing.T) {
	t.Parallel()
	supplierID := inlineSupplier(t)
	saved := trackInlineAddresses(t, supplierID)
	existingID := savedCounterpartyAddress(t, supplierID, uniqueName("e2e-inline-po-upd-existing"))
	*saved = append(*saved, existingID)

	body := inlinePurchaseOrderBody(supplierID)
	body["bill_to_address_id"] = existingID
	body["ship_to_address_id"] = existingID
	status, resp, err := apiClient.Post(purchaseOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	orderID := jsonField(parseJSON(resp), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(purchaseOrdersPath + "/" + orderID) })

	client := customRoleClient(t, "purchase_orders:update")
	billName, shipName := uniqueName("e2e-inline-po-upd-bill"), uniqueName("e2e-inline-po-upd-ship")
	status, resp, err = client.Patch(purchaseOrdersPath+"/"+orderID, map[string]any{
		"billing_address":  map[string]any{"id": existingID, "name": billName, "phone": nil},
		"shipping_address": inlineAddressBody(shipName, "4 Shipping Ln"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)

	bill, ship := getWithAddresses(t, purchaseOrdersPath+"/"+orderID, "bill_to_address", "ship_to_address")
	shipID := jsonField(ship, "id")
	*saved = append(*saved, shipID)
	assert.Equal(t, existingID, jsonField(bill, "id"))
	assert.Equal(t, billName, jsonField(bill, "name"))
	assert.NotEqual(t, existingID, shipID, "the new shipping address replaces the shared one on the ship-to side only")
	assert.Equal(t, shipName, jsonField(addressIn(t, supplierID, shipID), "name"), "the new address is saved to the supplier")
	assert.Equal(t, billName, jsonField(addressIn(t, supplierID, existingID), "name"))
}

func TestInlineAddresses_PurchaseOrderRejections(t *testing.T) {
	t.Parallel()
	supplierID := inlineSupplier(t)
	saved := trackInlineAddresses(t, supplierID)
	supplierAddressID := savedCounterpartyAddress(t, supplierID, uniqueName("e2e-inline-po-rej"))
	*saved = append(*saved, supplierAddressID)
	ownAddressID := createE2EAddress(t, uniqueName("e2e-inline-po-own"))
	otherTenantAddressID := tenantBAddress(t)
	client := customRoleClient(t, "purchase_orders:create")

	cases := []struct {
		name   string
		mutate func(map[string]any)
		status int
		code   string
		param  string
	}{
		{"another supplier's address", func(b map[string]any) {
			b["bill_to_address"] = map[string]any{"id": SeedSupplierAddressID, "name": "x"}
		}, 404, "", "bill_to_address.id"},
		{"the account's own address", func(b map[string]any) {
			b["ship_to_address"] = map[string]any{"id": ownAddressID}
		}, 404, "", "ship_to_address.id"},
		{"another tenant's address", func(b map[string]any) {
			b["bill_to_address"] = map[string]any{"id": otherTenantAddressID}
		}, 404, "", "bill_to_address.id"},
		{"an id and an inline address", func(b map[string]any) {
			b["bill_to_address_id"] = supplierAddressID
			b["bill_to_address"] = map[string]any{"id": supplierAddressID}
		}, 400, "validation_failed", "bill_to_address"},
		{"flat fields and an inline address", func(b map[string]any) {
			b["ship_to_name"] = "Flat"
			b["ship_to_address"] = inlineAddressBody("Inline", "5 Both St")
		}, 400, "validation_failed", "ship_to_address"},
		{"a new address without a country", func(b map[string]any) {
			b["bill_to_address"] = map[string]any{"name": "No country"}
		}, 400, "missing_field", "bill_to_address.country"},
		{"a new address without a name", func(b map[string]any) {
			b["bill_to_address"] = map[string]any{"country": "US"}
		}, 400, "missing_field", "bill_to_address.name"},
		{"a blank name", func(b map[string]any) {
			b["bill_to_address"] = map[string]any{"name": "   ", "country": "US"}
		}, 400, "validation_failed", "bill_to_address.name"},
		{"a malformed email", func(b map[string]any) {
			b["ship_to_address"] = map[string]any{"name": "Bad email", "country": "US", "email": "not-an-email"}
		}, 400, "invalid_format", "ship_to_address.email"},
		{"one address edited two ways", func(b map[string]any) {
			b["bill_to_address"] = map[string]any{"id": supplierAddressID, "name": "One"}
			b["ship_to_address"] = map[string]any{"id": supplierAddressID, "name": "Two"}
		}, 400, "validation_failed", "ship_to_address.id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := inlinePurchaseOrderBody(supplierID)
			tc.mutate(body)
			status, resp, err := client.Post(purchaseOrdersPath, body, newIdempotencyKey())
			require.NoError(t, err)
			if status == 201 {
				_, _, _ = apiClient.Delete(purchaseOrdersPath + "/" + jsonField(parseJSON(resp), "id"))
			}
			requireStatus(t, tc.status, status, resp)
			assertErrorParam(t, requireErrorResponse(t, resp, tc.code, "invalid_request_error"), tc.param)
		})
	}
	assert.Equal(t, "9 Spindle Way", jsonField(jsonObject(addressIn(t, supplierID, supplierAddressID), "geolocation"), "street_line_1"), "a refused order saves no address")
}

// --- Sales orders ---

func TestInlineAddresses_SalesOrderCreateNeedsOnlyCreatePermission(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	saved := trackInlineAddresses(t, customerID)
	existingID := savedCounterpartyAddress(t, customerID, uniqueName("e2e-inline-so-existing"))
	*saved = append(*saved, existingID)
	client := customRoleClient(t, "sales_orders:create")

	billName, shipName := uniqueName("e2e-inline-so-bill"), uniqueName("e2e-inline-so-ship")
	body := inlineSalesOrderBody(customerID)
	body["bill_to_address"] = inlineAddressBody(billName, "6 Billing Blvd")
	body["ship_to_address"] = map[string]any{"id": existingID, "name": shipName, "street_line_2": "Dock 4"}

	status, resp, err := client.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	orderID := jsonField(parseJSON(resp), "id")
	deleteOrder(t, orderID)

	bill, ship := getWithAddresses(t, salesOrdersPath+"/"+orderID, "bill_to_address", "ship_to_address")
	billID := jsonField(bill, "id")
	*saved = append(*saved, billID)
	assert.Equal(t, billName, jsonField(bill, "name"))
	assert.Equal(t, existingID, jsonField(ship, "id"))
	assert.Equal(t, shipName, jsonField(ship, "name"))

	assert.Equal(t, "6 Billing Blvd", jsonField(jsonObject(addressIn(t, customerID, billID), "geolocation"), "street_line_1"), "the new address is saved to the customer")
	updated := addressIn(t, customerID, existingID)
	assert.Equal(t, shipName, jsonField(updated, "name"))
	assert.Equal(t, "Dock 4", jsonField(jsonObject(updated, "geolocation"), "street_line_2"))
	assert.Equal(t, "9 Spindle Way", jsonField(jsonObject(updated, "geolocation"), "street_line_1"), "fields the inline update leaves out are kept")
}

func TestInlineAddresses_SalesOrderUpdateNeedsOnlyUpdatePermission(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	saved := trackInlineAddresses(t, customerID)
	existingID := savedCounterpartyAddress(t, customerID, uniqueName("e2e-inline-so-upd-existing"))
	*saved = append(*saved, existingID)

	body := inlineSalesOrderBody(customerID)
	body["bill_to_address_id"] = existingID
	body["ship_to_address_id"] = existingID
	status, resp, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	orderID := jsonField(parseJSON(resp), "id")
	deleteOrder(t, orderID)

	client := customRoleClient(t, "sales_orders:update")
	billName, shipName := uniqueName("e2e-inline-so-upd-bill"), uniqueName("e2e-inline-so-upd-ship")
	status, resp, err = client.Patch(salesOrdersPath+"/"+orderID, map[string]any{
		"billing_address":  map[string]any{"id": existingID, "name": billName, "email": "billing@e2e-test.openmrp.ai"},
		"shipping_address": inlineAddressBody(shipName, "7 Shipping St"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)

	bill, ship := getWithAddresses(t, salesOrdersPath+"/"+orderID, "bill_to_address", "ship_to_address")
	shipID := jsonField(ship, "id")
	*saved = append(*saved, shipID)
	assert.Equal(t, existingID, jsonField(bill, "id"))
	assert.Equal(t, billName, jsonField(bill, "name"))
	assert.NotEqual(t, existingID, shipID)
	assert.Equal(t, shipName, jsonField(addressIn(t, customerID, shipID), "name"), "the new address is saved to the customer")
	assert.Equal(t, "billing@e2e-test.openmrp.ai", jsonField(addressIn(t, customerID, existingID), "email"))
}

func TestInlineAddresses_SalesOrderRejections(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	saved := trackInlineAddresses(t, customerID)
	customerAddressID := savedCounterpartyAddress(t, customerID, uniqueName("e2e-inline-so-rej"))
	*saved = append(*saved, customerAddressID)
	otherCustomerID := setupOrderCustomer(t)
	otherSaved := trackInlineAddresses(t, otherCustomerID)
	otherCustomerAddressID := savedCounterpartyAddress(t, otherCustomerID, uniqueName("e2e-inline-so-other"))
	*otherSaved = append(*otherSaved, otherCustomerAddressID)
	otherTenantAddressID := tenantBAddress(t)

	body := inlineSalesOrderBody(customerID)
	body["bill_to_address_id"] = customerAddressID
	body["ship_to_address_id"] = customerAddressID
	status, resp, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	orderID := jsonField(parseJSON(resp), "id")
	deleteOrder(t, orderID)

	createClient := customRoleClient(t, "sales_orders:create")
	updateClient := customRoleClient(t, "sales_orders:update")

	cases := []struct {
		name   string
		update bool
		mutate func(map[string]any)
		status int
		code   string
		param  string
	}{
		{"create: another customer's address", false, func(b map[string]any) {
			b["bill_to_address"] = map[string]any{"id": otherCustomerAddressID}
			b["ship_to_address_id"] = customerAddressID
		}, 404, "", "bill_to_address.id"},
		{"create: another tenant's address", false, func(b map[string]any) {
			b["bill_to_address_id"] = customerAddressID
			b["ship_to_address"] = map[string]any{"id": otherTenantAddressID}
		}, 404, "", "ship_to_address.id"},
		{"create: an id and an inline address", false, func(b map[string]any) {
			b["bill_to_address_id"] = customerAddressID
			b["bill_to_address"] = map[string]any{"id": customerAddressID}
			b["ship_to_address_id"] = customerAddressID
		}, 400, "validation_failed", "bill_to_address"},
		{"create: neither an id nor an inline address", false, func(b map[string]any) {
			b["bill_to_address_id"] = customerAddressID
		}, 400, "missing_field", "ship_to_address_id"},
		{"create: a new address without a country", false, func(b map[string]any) {
			b["bill_to_address"] = map[string]any{"name": "No country"}
			b["ship_to_address_id"] = customerAddressID
		}, 400, "missing_field", "bill_to_address.country"},
		{"update: another customer's address", true, func(b map[string]any) {
			b["billing_address"] = map[string]any{"id": otherCustomerAddressID, "name": "x"}
		}, 404, "", "billing_address.id"},
		{"update: another tenant's address", true, func(b map[string]any) {
			b["shipping_address"] = map[string]any{"id": otherTenantAddressID}
		}, 404, "", "shipping_address.id"},
		{"update: an id and an inline address", true, func(b map[string]any) {
			b["shipping_address_id"] = customerAddressID
			b["shipping_address"] = map[string]any{"id": customerAddressID}
		}, 400, "validation_failed", "shipping_address"},
		{"update: a too-long state", true, func(b map[string]any) {
			b["billing_address"] = map[string]any{"name": "Long state", "country": "US", "state": strings.Repeat("x", 256)}
		}, 400, "", "billing_address.state"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var status int
			var resp []byte
			var err error
			if tc.update {
				b := map[string]any{}
				tc.mutate(b)
				status, resp, err = updateClient.Patch(salesOrdersPath+"/"+orderID, b, newIdempotencyKey())
			} else {
				b := inlineSalesOrderBody(customerID)
				tc.mutate(b)
				status, resp, err = createClient.Post(salesOrdersPath, b, newIdempotencyKey())
				if status == 201 {
					deleteOrder(t, jsonField(parseJSON(resp), "id"))
				}
			}
			require.NoError(t, err)
			requireStatus(t, tc.status, status, resp)
			assertErrorParam(t, requireErrorResponse(t, resp, tc.code, "invalid_request_error"), tc.param)
		})
	}
	assert.Equal(t, "9 Spindle Way", jsonField(jsonObject(addressIn(t, otherCustomerID, otherCustomerAddressID), "geolocation"), "street_line_1"), "another customer's address is never touched")
}

// --- Account ---

func accountDefaultAddressIDs(t *testing.T) (billing, shipping string) {
	t.Helper()
	status, body, err := apiClient.GetListRaw(accountsPath+"/"+SeedAccountID, url.Values{"include": {"default_billing_address,default_shipping_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	return jsonField(jsonObject(got, "default_billing_address"), "id"), jsonField(jsonObject(got, "default_shipping_address"), "id")
}

// Not parallel: it changes the seeded account's default addresses, which it puts back.
func TestInlineAddresses_AccountUpdateNeedsOnlySelfUpdate(t *testing.T) {
	originalBilling, originalShipping := accountDefaultAddressIDs(t)
	require.NotEmpty(t, originalBilling)
	require.NotEmpty(t, originalShipping)
	ownAddressID := createE2EAddress(t, uniqueName("e2e-inline-acct-own"))
	saved := trackInlineAddresses(t, SeedAccountID)
	t.Cleanup(func() {
		apiClient.Patch(accountsPath+"/"+SeedAccountID, map[string]any{
			"default_billing_address_id":  originalBilling,
			"default_shipping_address_id": originalShipping,
		}, newIdempotencyKey())
	})
	client := customRoleClient(t, "self:update")

	billName, shipName := uniqueName("e2e-inline-acct-bill"), uniqueName("e2e-inline-acct-ship")
	status, resp, err := client.Patch(accountsPath+"/"+SeedAccountID, map[string]any{
		"default_billing_address":  map[string]any{"id": ownAddressID, "name": billName, "postal_code": "90002"},
		"default_shipping_address": inlineAddressBody(shipName, "8 Default Dr"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)

	billing, shipping := accountDefaultAddressIDs(t)
	*saved = append(*saved, shipping)
	assert.Equal(t, ownAddressID, billing)
	assert.NotEqual(t, ownAddressID, shipping)
	edited := addressIn(t, SeedAccountID, ownAddressID)
	assert.Equal(t, billName, jsonField(edited, "name"))
	assert.Equal(t, "90002", jsonField(jsonObject(edited, "geolocation"), "postal_code"))
	assert.Equal(t, shipName, jsonField(addressIn(t, SeedAccountID, shipping), "name"))
}

// Not parallel for the same reason as above, though every request here is refused.
func TestInlineAddresses_AccountUpdateRejections(t *testing.T) {
	originalBilling, originalShipping := accountDefaultAddressIDs(t)
	customerID := setupOrderCustomer(t)
	saved := trackInlineAddresses(t, customerID)
	customerAddressID := savedCounterpartyAddress(t, customerID, uniqueName("e2e-inline-acct-cust"))
	*saved = append(*saved, customerAddressID)
	otherTenantAddressID := tenantBAddress(t)
	client := customRoleClient(t, "self:update")

	cases := []struct {
		name   string
		body   map[string]any
		status int
		code   string
		param  string
	}{
		{"a customer's address", map[string]any{"default_billing_address": map[string]any{"id": customerAddressID}}, 404, "", "default_billing_address.id"},
		{"another tenant's address", map[string]any{"default_shipping_address": map[string]any{"id": otherTenantAddressID}}, 404, "", "default_shipping_address.id"},
		{"an id and an inline address", map[string]any{
			"default_billing_address_id": originalBilling,
			"default_billing_address":    map[string]any{"id": originalBilling},
		}, 400, "validation_failed", "default_billing_address"},
		{"a new address without a name", map[string]any{"default_shipping_address": map[string]any{"country": "US"}}, 400, "missing_field", "default_shipping_address.name"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, resp, err := client.Patch(accountsPath+"/"+SeedAccountID, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, tc.status, status, resp)
			assertErrorParam(t, requireErrorResponse(t, resp, tc.code, "invalid_request_error"), tc.param)
		})
	}
	billing, shipping := accountDefaultAddressIDs(t)
	assert.Equal(t, originalBilling, billing, "a refused update leaves the defaults alone")
	assert.Equal(t, originalShipping, shipping)
}

// --- The address endpoints keep their own permissions ---

func TestInlineAddresses_AddressEndpointsStillNeedAddressPermissions(t *testing.T) {
	t.Parallel()
	supplierID := inlineSupplier(t)
	saved := trackInlineAddresses(t, supplierID)
	supplierAddressID := savedCounterpartyAddress(t, supplierID, uniqueName("e2e-inline-perm"))
	*saved = append(*saved, supplierAddressID)
	ownAddressID := createE2EAddress(t, uniqueName("e2e-inline-perm-own"))
	client := customRoleClient(t, "purchase_orders:create", "sales_orders:create", "self:update")

	status, resp, err := client.WithAccountID(supplierID).Post(addressesPath, inlineAddressBody("Not allowed", "1 No Way"), newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		*saved = append(*saved, jsonField(parseJSON(resp), "id"))
	}
	requireStatus(t, 403, status, resp)

	status, resp, err = client.WithAccountID(supplierID).Patch(addressesPath+"/"+supplierAddressID, map[string]any{"name": "Not allowed"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, resp)

	status, resp, err = client.Patch(addressesPath+"/"+ownAddressID, map[string]any{"name": "Not allowed"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, resp)
}

// An update that points the order at a saved address takes one of the supplier's or one of the ordering account's own, which is what the order page's bill-to and ship-to pickers list. Before, any address id was taken, including another tenant's.
func TestInlineAddresses_PurchaseOrderUpdateTakesOnlyTheSuppliersOrOwnAddresses(t *testing.T) {
	t.Parallel()
	supplierID := inlineSupplier(t)
	saved := trackInlineAddresses(t, supplierID)
	supplierAddressID := savedCounterpartyAddress(t, supplierID, uniqueName("e2e-inline-po-upd-own"))
	*saved = append(*saved, supplierAddressID)
	ownAddressID := createE2EAddress(t, uniqueName("e2e-inline-po-upd-acct"))
	otherTenantAddressID := tenantBAddress(t)
	order := createAndCleanup(t, purchaseOrdersPath, inlinePurchaseOrderBody(supplierID))
	path := purchaseOrdersPath + "/" + jsonField(order, "id")

	for _, tc := range []struct{ name, field, addressID string }{
		{"another supplier's address", "billing_address_id", SeedSupplierAddressID},
		{"another tenant's address", "shipping_address_id", otherTenantAddressID},
	} {
		status, body, err := apiClient.Patch(path, map[string]any{tc.field: tc.addressID}, newIdempotencyKey())
		require.NoError(t, err)
		require.Equal(t, 400, status, "%s: %s", tc.name, string(body))
		assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), tc.field)
	}

	status, body, err := apiClient.Patch(path, map[string]any{"billing_address_id": ownAddressID, "shipping_address_id": supplierAddressID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	bill, ship := getWithAddresses(t, path, "bill_to_address", "ship_to_address")
	assert.Equal(t, ownAddressID, jsonField(bill, "id"), "the account's own address, as the order page offers")
	assert.Equal(t, supplierAddressID, jsonField(ship, "id"), "the supplier's address")
}
