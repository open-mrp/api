//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// orderCounterpartyRefs reads the customer, addresses and discount an order holds ("" when unset).
func orderCounterpartyRefs(t *testing.T, orderID string) map[string]string {
	t.Helper()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID,
		url.Values{"include": {"customer,bill_to_address,ship_to_address,order_discount"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	refs := map[string]string{}
	for _, key := range []string{"customer", "bill_to_address", "ship_to_address", "order_discount"} {
		if obj := jsonObject(got, key); obj != nil {
			refs[key] = jsonField(obj, "id")
		} else {
			refs[key] = ""
		}
	}
	return refs
}

func customerBillToAddressID(t *testing.T, customerID string) string {
	t.Helper()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+customerID, url.Values{"include": {"bill_to_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	id := jsonField(jsonObject(parseJSON(body), "bill_to_address"), "id")
	require.NotEmpty(t, id)
	return id
}

// tenantBCustomer is a customer of tenant B, built on the platform carrier and terms open to every account.
func tenantBCustomer(t *testing.T) string {
	t.Helper()
	tenantB := getTenantBClient()
	name := uniqueName("e2e-foreign-cust")
	status, body, err := tenantB.Post(customersPath, map[string]any{
		"name":                     name,
		"status":                   "normal",
		"default_carrier_id":       SeedSystemCarrierID,
		"default_payment_term_id":  SeedDefaultPaymentTermID,
		"default_shipping_term_id": SeedShippingTermID,
		"customer_type_group_id":   dashCustomersTenantBGroup(t, "type_group"),
		"bill_to_address":          map[string]any{"name": name + " Billing", "country": "US"},
		"ship_to_address":          map[string]any{"name": name + " Shipping", "country": "US"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { _, _, _ = tenantB.Delete(customersPath + "/" + id) })
	return id
}

func TestSalesOrderCounterpartyRefs_UpdateRefusesOnesTheAccountCannotUse(t *testing.T) {
	t.Parallel()
	tenantB := getTenantBClient()
	post := func(path string, body map[string]any) string {
		t.Helper()
		status, resp, err := tenantB.Post(path, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, resp)
		id := jsonField(parseJSON(resp), "id")
		t.Cleanup(func() { _, _, _ = tenantB.Delete(path + "/" + id) })
		return id
	}
	foreignAddress := post(addressesPath, map[string]any{"name": uniqueName("e2e-foreign-addr"), "country": "US"})
	foreignDiscount := post(orderDiscountsPath, map[string]any{
		"name": uniqueName("e2e-foreign-ords"), "code": uniqueName("E2EFOREIGN"), "discount_type": "percentage", "percentage": "0.1",
	})
	foreignCustomer := tenantBCustomer(t)
	otherCustomersAddress := customerBillToAddressID(t, setupOrderCustomer(t))

	status, body, err := apiClient.Post(salesOrdersPath, minimalSalesOrderCreateBody(t, setupOrderCustomer(t)), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	orderID := jsonField(parseJSON(body), "id")
	deleteOrder(t, orderID)
	before := orderCounterpartyRefs(t, orderID)

	// An order moved to an account it has no relation with cannot be read back, so the supplier case runs first and stops the test on a stack that still accepts it.
	for _, tc := range []struct {
		name, field, id string
	}{
		{"a supplier as the customer", "customer_id", SeedSupplierAccountID},
		{"another tenant's customer", "customer_id", foreignCustomer},
		{"another tenant's bill-to", "billing_address_id", foreignAddress},
		{"another tenant's ship-to", "shipping_address_id", foreignAddress},
		{"another customer's ship-to", "shipping_address_id", otherCustomersAddress},
		{"another tenant's discount", "order_discount_id", foreignDiscount},
	} {
		status, body, err := apiClient.Patch(salesOrdersPath+"/"+orderID, map[string]any{tc.field: tc.id}, newIdempotencyKey())
		require.NoError(t, err)
		require.Equal(t, 404, status, "%s: %s", tc.name, string(body))
		assertErrorParam(t, requireErrorResponse(t, body, "resource_not_found", "invalid_request_error"), tc.field)
		assert.Equal(t, before, orderCounterpartyRefs(t, orderID), "%s: the order is unchanged", tc.name)
	}
}

func TestSalesOrderCounterpartyRefs_UpdateTakesTheBuyersAndTheAccountsOwn(t *testing.T) {
	t.Parallel()
	nextCustomer := setupOrderCustomer(t)
	nextCustomersAddress := customerBillToAddressID(t, nextCustomer)
	discountID := jsonField(createOrderDiscount(t, nil), "id")

	status, body, err := apiClient.Post(salesOrdersPath, minimalSalesOrderCreateBody(t, setupOrderCustomer(t)), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	orderID := jsonField(parseJSON(body), "id")
	deleteOrder(t, orderID)

	patch := func(fields map[string]any) {
		t.Helper()
		status, body, err := apiClient.Patch(salesOrdersPath+"/"+orderID, fields, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
	}

	ownAddress := createE2EAddress(t, "E2E Own Ship-To")
	patch(map[string]any{"shipping_address_id": ownAddress, "order_discount_id": discountID})
	got := orderCounterpartyRefs(t, orderID)
	assert.Equal(t, ownAddress, got["ship_to_address"], "an address of the account itself")
	assert.Equal(t, discountID, got["order_discount"], "the account's own discount")

	patch(map[string]any{"customer_id": nextCustomer, "billing_address_id": nextCustomersAddress})
	got = orderCounterpartyRefs(t, orderID)
	assert.Equal(t, nextCustomer, got["customer"], "another of the account's customers")
	assert.Equal(t, nextCustomersAddress, got["bill_to_address"], "the new customer's own address")

	patch(map[string]any{
		"customer_id":         got["customer"],
		"billing_address_id":  got["bill_to_address"],
		"shipping_address_id": got["ship_to_address"],
		"order_discount_id":   got["order_discount"],
	})
	assert.Equal(t, got, orderCounterpartyRefs(t, orderID), "re-sending what the order holds")
}
