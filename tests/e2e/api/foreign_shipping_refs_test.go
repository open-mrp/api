//go:build e2e

package api_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A record that stores a bare carrier, service level or term id must refuse another tenant's record rather than store it and later read it back.

// tenantBShippingRefs is a carrier, a service level on it, a shipping term and a payment term owned by tenant B.
type tenantBShippingRefs struct {
	carrierID      string
	serviceLevelID string
	shippingTermID string
	paymentTermID  string
}

func createTenantBShippingRefs(t *testing.T) tenantBShippingRefs {
	t.Helper()
	tenantB := getTenantBClient()
	create := func(path string, body map[string]any) string {
		t.Helper()
		status, resp, err := tenantB.Post(path, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, resp)
		id := jsonField(parseJSON(resp), "id")
		require.NotEmpty(t, id)
		return id
	}

	var refs tenantBShippingRefs
	refs.carrierID = create(carriersPath, map[string]any{"name": uniqueName("e2e-foreign-carrier"), "code": "will_call"})
	t.Cleanup(func() { _, _, _ = tenantB.Delete(carriersPath + "/" + refs.carrierID) })
	refs.serviceLevelID = create(serviceLevelsPath(refs.carrierID), map[string]any{
		"name": uniqueName("e2e-foreign-sl"), "code": uniqueName("e2e-foreign-sl-code"),
	})
	refs.shippingTermID = create(shippingTermsPath, map[string]any{"name": uniqueName("e2e-foreign-shipterm"), "type": "free_freight"})
	t.Cleanup(func() { _, _, _ = tenantB.Delete(shippingTermsPath + "/" + refs.shippingTermID) })
	refs.paymentTermID = create(paymentTermsPath, map[string]any{"name": uniqueName("e2e-foreign-payterm")})
	t.Cleanup(func() { _, _, _ = tenantB.Delete(paymentTermsPath + "/" + refs.paymentTermID) })
	return refs
}

func assertRefNotFound(t *testing.T, status int, body []byte, param string) {
	t.Helper()
	if assert.Equal(t, 404, status, "%s", string(body)) {
		assertErrorParam(t, requireErrorResponse(t, body, "resource_not_found", "invalid_request_error"), param)
	}
}

// --- Sales orders ---

func TestForeignShippingRefs_SalesOrderUpdateRefusedLeavesTheOrderAndItsShipments(t *testing.T) {
	t.Parallel()
	foreign := createTenantBShippingRefs(t)
	orderID, _, shipmentID := dashShipmentsPacked(t)

	carrier, serviceLevel, shippingTerm, paymentTerm := orderRefIDs(t, orderID)
	shipmentCarrier, shipmentServiceLevel := dashShipmentsFreightIDs(t, shipmentID)
	require.Equal(t, SeedCarrierID, shipmentCarrier, "the shipment starts on the order's carrier")

	for _, tc := range []struct{ field, id string }{
		{"carrier_id", foreign.carrierID},
		{"service_level_id", foreign.serviceLevelID},
		{"shipping_term_id", foreign.shippingTermID},
		{"payment_term_id", foreign.paymentTermID},
	} {
		t.Run(tc.field, func(t *testing.T) {
			status, body, err := apiClient.Patch(salesOrdersPath+"/"+orderID, map[string]any{tc.field: tc.id}, newIdempotencyKey())
			require.NoError(t, err)
			assertRefNotFound(t, status, body, tc.field)

			gotCarrier, gotServiceLevel, gotShippingTerm, gotPaymentTerm := orderRefIDs(t, orderID)
			assert.Equal(t, carrier, gotCarrier, "the order keeps its carrier")
			assert.Equal(t, serviceLevel, gotServiceLevel, "the order keeps its service level")
			assert.Equal(t, shippingTerm, gotShippingTerm, "the order keeps its shipping term")
			assert.Equal(t, paymentTerm, gotPaymentTerm, "the order keeps its payment term")

			gotShipmentCarrier, gotShipmentServiceLevel := dashShipmentsFreightIDs(t, shipmentID)
			assert.Equal(t, shipmentCarrier, gotShipmentCarrier, "the shipment keeps its carrier")
			assert.Equal(t, shipmentServiceLevel, gotShipmentServiceLevel, "the shipment keeps its service level")
		})
	}
}

// A system carrier and the account's own carrier are both accepted, and the order's shipments follow the order onto each.
func TestForeignShippingRefs_SalesOrderUpdateTakesASystemOrOwnCarrier(t *testing.T) {
	t.Parallel()
	orderID, _, shipmentID := dashShipmentsPacked(t)

	for _, tc := range []struct{ name, carrierID, serviceLevelID string }{
		{"system carrier", SeedSystemCarrierID, SeedSystemServiceLevelID},
		{"own carrier", SeedCarrierID, SeedServiceLevelID},
	} {
		status, body, err := apiClient.Patch(salesOrdersPath+"/"+orderID, map[string]any{
			"carrier_id": tc.carrierID, "service_level_id": tc.serviceLevelID,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		carrier, serviceLevel, _, _ := orderRefIDs(t, orderID)
		assert.Equal(t, tc.carrierID, carrier, tc.name)
		assert.Equal(t, tc.serviceLevelID, serviceLevel, tc.name)

		eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
			gotCarrier, gotServiceLevel := dashShipmentsFreightIDs(t, shipmentID)
			if gotCarrier != tc.carrierID || gotServiceLevel != tc.serviceLevelID {
				return fmt.Errorf("%s: shipment is on %s/%s", tc.name, gotCarrier, gotServiceLevel)
			}
			return nil
		})
	}
}

// A freight-exempt order never prices freight, so its carrier has to be checked on its own.
func TestForeignShippingRefs_SalesOrderCreateRefusesAForeignCarrier(t *testing.T) {
	t.Parallel()
	foreign := createTenantBShippingRefs(t)
	billed := setupOrderCustomer(t)
	exempt := setupFreightExemptOrderCustomer(t)

	for _, tc := range []struct{ name, customerID string }{
		{"billed freight", billed},
		{"freight exempt", exempt},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := minimalSalesOrderCreateBody(t, tc.customerID)
			body["carrier_id"] = foreign.carrierID
			delete(body, "service_level_id")

			status, resp, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
			require.NoError(t, err)
			if status == 201 {
				deleteOrder(t, jsonField(parseJSON(resp), "id"))
			}
			assertRefNotFound(t, status, resp, "carrier_id")
		})
	}
}

// Create refuses another tenant's service level with the 400 its other reference checks answer.
func TestForeignShippingRefs_SalesOrderCreateRefusesAForeignServiceLevel(t *testing.T) {
	t.Parallel()
	foreign := createTenantBShippingRefs(t)
	body := minimalSalesOrderCreateBody(t, setupFreightExemptOrderCustomer(t))
	body["service_level_id"] = foreign.serviceLevelID

	status, resp, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		deleteOrder(t, jsonField(parseJSON(resp), "id"))
	}
	requireStatus(t, 400, status, resp)
	assertErrorParam(t, requireErrorResponse(t, resp, "", "invalid_request_error"), "service_level_id")
}

func TestForeignShippingRefs_SalesOrderCreateTakesASystemOrOwnCarrier(t *testing.T) {
	t.Parallel()
	exempt := setupFreightExemptOrderCustomer(t)

	for _, tc := range []struct{ name, carrierID, serviceLevelID string }{
		{"system carrier", SeedSystemCarrierID, SeedSystemServiceLevelID},
		{"own carrier", SeedCarrierID, SeedServiceLevelID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := minimalSalesOrderCreateBody(t, exempt)
			body["carrier_id"] = tc.carrierID
			body["service_level_id"] = tc.serviceLevelID

			status, resp, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 201, status, resp)
			orderID := jsonField(parseJSON(resp), "id")
			deleteOrder(t, orderID)

			carrier, serviceLevel, _, _ := orderRefIDs(t, orderID)
			assert.Equal(t, tc.carrierID, carrier)
			assert.Equal(t, tc.serviceLevelID, serviceLevel)
		})
	}
}

// setupFreightExemptOrderCustomer is a customer on free freight, with access to the seed product's line.
func setupFreightExemptOrderCustomer(t *testing.T) string {
	t.Helper()
	body := validCustomerBody(uniqueName("e2e-so-exempt-cust"))
	body["freight_policy"] = "free_freight"
	status, resp, err := apiClient.Post(customersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	customerID := jsonField(parseJSON(resp), "id")

	status, resp, err = apiClient.Post(productLineAccessPath, map[string]any{
		"customer_id":      customerID,
		"product_line_ids": []string{SeedProductLineID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)

	t.Cleanup(func() {
		_, _, _ = apiClient.Delete(productLineAccessPath + "/" + customerID)
		_, _, _ = apiClient.Delete(customersPath + "/" + customerID)
	})
	return customerID
}
