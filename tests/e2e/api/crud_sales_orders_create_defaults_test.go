//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Coverage for the customer-default fallback on POST /v1/sales/sales-orders. Carrier,
// service level, shipping term, and payment term are filled from the buyer's customer
// relation whenever the request omits them (mirroring the Dashboard create form, which
// pre-fills them from the selected customer). This is the regression guard for the
// "Carrier is required" 500 that GET /v1/sales-orders/{id} threw after an order was
// created via the API without those references: the read adapter requires carrier,
// shipping term, and payment term, so the create must never persist an order missing
// them.
//
// Note on reachable permutations: default_carrier_id, default_payment_term_id, and
// default_shipping_term_id are required on customer create and non-clearable on update,
// so a customer created through the API always carries them. The variation that
// actually produced the incident — and the one exercised here — is the create *request*
// including or excluding each reference. The service-level default additionally only
// applies when the carrier is also defaulted, so the two never reference different
// carriers; that pairing is pinned below too.

// setupOrderCustomerWithServiceLevel creates a buyer whose customer relation carries all
// four order defaults (carrier, service level, shipping term, payment term) and grants it
// access to the seed product line. Returns the customer account id.
func setupOrderCustomerWithServiceLevel(t *testing.T) string {
	t.Helper()
	body := validCustomerBody(uniqueName("e2e-so-defaults-cust"))
	body["default_service_level_id"] = SeedServiceLevelID

	status, respBody, err := apiClient.Post(customersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	customerID := jsonField(parseJSON(respBody), "id")

	plStatus, plBody, err := apiClient.Post(productLineAccessPath, map[string]any{
		"customer_id":      customerID,
		"product_line_ids": []string{SeedProductLineID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, plStatus, plBody)

	t.Cleanup(func() {
		_, _, _ = apiClient.Delete(productLineAccessPath + "/" + customerID)
		_, _, _ = apiClient.Delete(customersPath + "/" + customerID)
	})
	return customerID
}

// orderRefIDs GETs the order with the freight (carrier + service level), shipping term,
// and payment term expansions and returns each referenced id ("" when the field is null).
// It also asserts the GET itself succeeds — the whole point of the fixture is that the
// order is readable, which is where the original bug 500'd. Carrier and service level are
// nested under the freight sub-resource, so they come from ?include=freight.
func orderRefIDs(t *testing.T, orderID string) (carrier, serviceLevel, shippingTerm, paymentTerm string) {
	t.Helper()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID,
		url.Values{"include": {"freight,shipping_term,payment_term"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	idOf := func(obj map[string]any) string {
		if obj != nil {
			return jsonField(obj, "id")
		}
		return ""
	}
	freight := jsonObject(got, "freight")
	if freight != nil {
		carrier = idOf(jsonObject(freight, "carrier"))
		serviceLevel = idOf(jsonObject(freight, "service_level"))
	}
	return carrier, serviceLevel, idOf(jsonObject(got, "shipping_term")), idOf(jsonObject(got, "payment_term"))
}

// TestCreateSalesOrder_RequiredRefsFillFromCustomerDefaults_AllRequestPermutations drives
// every subset of the three read-required references the request can omit — including
// omitting all of them — and asserts each resulting order is created (201) and reads back
// (200) with all three references resolved to the customer defaults. Service level is
// supplied throughout so this test isolates the three required fields.
func TestCreateSalesOrder_RequiredRefsFillFromCustomerDefaults_AllRequestPermutations(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomerWithServiceLevel(t)

	requiredFields := []string{"carrier_id", "shipping_term_id", "payment_term_id"}

	for mask := 0; mask < (1 << len(requiredFields)); mask++ {
		// Bit i set → field i is supplied; bit i clear → field i is omitted.
		body := minimalSalesOrderCreateBody(t, customerID)
		var omitted []string
		for i, f := range requiredFields {
			if mask&(1<<i) == 0 {
				delete(body, f)
				omitted = append(omitted, f)
			}
		}
		name := "omit=none"
		if len(omitted) > 0 {
			name = "omit=" + strings.Join(omitted, "+")
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 201, status, respBody)
			orderID := jsonField(parseJSON(respBody), "id")
			deleteOrder(t, orderID)

			carrier, _, shippingTerm, paymentTerm := orderRefIDs(t, orderID)
			assert.Equal(t, SeedCarrierID, carrier, "carrier must resolve from request or customer default")
			assert.Equal(t, SeedShippingTermID, shippingTerm, "shipping term must resolve from request or customer default")
			assert.Equal(t, SeedPaymentTermID, paymentTerm, "payment term must resolve from request or customer default")
		})
	}
}

// TestCreateSalesOrder_ServiceLevelDefaultsOnlyWithCarrier pins the carrier/service-level
// pairing: the customer's default service level is adopted only when the carrier is also
// defaulted (request omits both), never when the request supplies its own carrier.
func TestCreateSalesOrder_ServiceLevelDefaultsOnlyWithCarrier(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomerWithServiceLevel(t)

	t.Run("omit carrier and service level -> both from customer default", func(t *testing.T) {
		t.Parallel()
		body := minimalSalesOrderCreateBody(t, customerID)
		delete(body, "carrier_id")
		delete(body, "service_level_id")

		status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, respBody)
		orderID := jsonField(parseJSON(respBody), "id")
		deleteOrder(t, orderID)

		carrier, serviceLevel, _, _ := orderRefIDs(t, orderID)
		assert.Equal(t, SeedCarrierID, carrier, "carrier fills from the customer default")
		assert.Equal(t, SeedServiceLevelID, serviceLevel, "service level fills from the customer default when the carrier is also defaulted")
	})

	t.Run("supply carrier, omit service level -> service level stays unset", func(t *testing.T) {
		t.Parallel()
		body := minimalSalesOrderCreateBody(t, customerID)
		delete(body, "service_level_id") // carrier_id kept

		status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, respBody)
		orderID := jsonField(parseJSON(respBody), "id")
		deleteOrder(t, orderID)

		carrier, serviceLevel, _, _ := orderRefIDs(t, orderID)
		assert.Equal(t, SeedCarrierID, carrier, "the request carrier is used")
		assert.Empty(t, serviceLevel, "the customer's default service level is NOT attached to a caller-supplied carrier")
	})
}

// TestCreateSalesOrder_RequestValuesOverrideCustomerDefaults proves the default is only a
// fallback: a caller-supplied carrier wins over the customer default, while the omitted
// shipping and payment terms still fill from the customer default in the same request.
func TestCreateSalesOrder_RequestValuesOverrideCustomerDefaults(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomerWithServiceLevel(t)

	body := minimalSalesOrderCreateBody(t, customerID)
	// Supply a carrier + service level that differ from the customer defaults (system
	// carrier, usable by any account), and omit shipping/payment so they must default.
	body["carrier_id"] = SeedSystemCarrierID
	body["service_level_id"] = SeedSystemServiceLevelID
	delete(body, "shipping_term_id")
	delete(body, "payment_term_id")

	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	orderID := jsonField(parseJSON(respBody), "id")
	deleteOrder(t, orderID)

	carrier, serviceLevel, shippingTerm, paymentTerm := orderRefIDs(t, orderID)
	assert.Equal(t, SeedSystemCarrierID, carrier, "request carrier overrides the customer default")
	assert.Equal(t, SeedSystemServiceLevelID, serviceLevel, "request service level overrides the customer default")
	assert.Equal(t, SeedShippingTermID, shippingTerm, "omitted shipping term still fills from the customer default")
	assert.Equal(t, SeedPaymentTermID, paymentTerm, "omitted payment term still fills from the customer default")
}
