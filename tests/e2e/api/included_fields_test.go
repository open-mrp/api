//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Shared Assertion Helpers
// ──────────────────────────────────────────────

// assertCarrierFieldsPopulated checks that a Carrier sub-resource has no empty enum fields.
func assertCarrierFieldsPopulated(t *testing.T, carrier map[string]any, prefix string) {
	t.Helper()
	assert.NotEmpty(t, jsonField(carrier, "id"), prefix+".id must not be empty")
	assert.NotEmpty(t, jsonField(carrier, "customer_portal_visibility"), prefix+".customer_portal_visibility must not be empty")
}

// assertServiceLevelFieldsPopulated checks that a ServiceLevel sub-resource has no empty enum fields.
func assertServiceLevelFieldsPopulated(t *testing.T, sl map[string]any, prefix string) {
	t.Helper()
	assert.NotEmpty(t, jsonField(sl, "id"), prefix+".id must not be empty")
	assert.NotEmpty(t, jsonField(sl, "customer_portal_visibility"), prefix+".customer_portal_visibility must not be empty")
}

// assertPaymentTermFieldsPopulated checks that a PaymentTerm sub-resource has no empty enum fields.
func assertPaymentTermFieldsPopulated(t *testing.T, pt map[string]any, prefix string) {
	t.Helper()
	assert.NotEmpty(t, jsonField(pt, "id"), prefix+".id must not be empty")
	assert.NotEmpty(t, jsonField(pt, "status"), prefix+".status must not be empty")
}

// assertShippingTermFieldsPopulated checks that a ShippingTerm sub-resource has no empty enum fields.
func assertShippingTermFieldsPopulated(t *testing.T, st map[string]any, prefix string) {
	t.Helper()
	assert.NotEmpty(t, jsonField(st, "id"), prefix+".id must not be empty")
	assert.NotEmpty(t, jsonField(st, "type"), prefix+".type must not be empty")
}

// assertPriorityFieldsPopulated checks that a Priority sub-resource has no empty fields.
func assertPriorityFieldsPopulated(t *testing.T, pri map[string]any, prefix string) {
	t.Helper()
	assert.NotEmpty(t, jsonField(pri, "id"), prefix+".id must not be empty")
	assert.NotEmpty(t, jsonField(pri, "code"), prefix+".code must not be empty")
}

// ──────────────────────────────────────────────
// Customer — Included Sub-Resource Completeness
// ──────────────────────────────────────────────

const allCustomerIncludes = "type,defaults.payment_term,defaults.shipping_term,defaults.priority,freight_preferences.carrier"

func assertCustomerIncludedFieldsPopulated(t *testing.T, got map[string]any) {
	t.Helper()

	if typeGroup := jsonObject(got, "type"); typeGroup != nil {
		assert.NotEmpty(t, jsonField(typeGroup, "type"), "type.type must not be empty")
		assert.NotEmpty(t, jsonField(typeGroup, "commission_policy"), "type.commission_policy must not be empty")
		assert.NotEmpty(t, jsonField(typeGroup, "freight_policy"), "type.freight_policy must not be empty")
	}

	if defaults := jsonObject(got, "defaults"); defaults != nil {
		if pt := jsonObject(defaults, "payment_term"); pt != nil {
			assertPaymentTermFieldsPopulated(t, pt, "defaults.payment_term")
		}
		if st := jsonObject(defaults, "shipping_term"); st != nil {
			assertShippingTermFieldsPopulated(t, st, "defaults.shipping_term")
		}
		if pri := jsonObject(defaults, "priority"); pri != nil {
			assertPriorityFieldsPopulated(t, pri, "defaults.priority")
		}
	}

	if fp := jsonObject(got, "freight_preferences"); fp != nil {
		if carrier := jsonObject(fp, "carrier"); carrier != nil {
			assertCarrierFieldsPopulated(t, carrier, "freight_preferences.carrier")
		}
	}
}

func TestCustomers_GetIncludedFieldsNotEmpty(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(
		customersPath+"/"+SeedCustomerAccountID,
		url.Values{"include": {allCustomerIncludes}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assertCustomerIncludedFieldsPopulated(t, got)
}

func TestCustomers_CreateIncludedFieldsNotEmpty(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cust-noempty-c")
	payload := validCustomerBody(name)
	payload["default_priority_code"] = SeedPriorityCode
	status, body, err := apiClient.Post(
		customersPath+"?include="+allCustomerIncludes,
		payload,
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	assertCustomerIncludedFieldsPopulated(t, got)

	apiClient.Delete(customersPath + "/" + jsonField(got, "id"))
}

func TestCustomers_UpdateIncludedFieldsNotEmpty(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cust-noempty-u")
	updPayload := validCustomerBody(name)
	updPayload["default_priority_code"] = SeedPriorityCode
	createStatus, createBody, err := apiClient.Post(customersPath, updPayload, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	patchStatus, patchBody, err := apiClient.Patch(
		customersPath+"/"+id+"?include="+allCustomerIncludes,
		map[string]any{"note": "verify enum fields"},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	got := parseJSON(patchBody)
	assertCustomerIncludedFieldsPopulated(t, got)

	apiClient.Delete(customersPath + "/" + id)
}

// ──────────────────────────────────────────────
// Sales Order — Included Sub-Resource Completeness
// ──────────────────────────────────────────────

const salesOrdersPath = "/v1/sales/sales-orders"
const allSalesOrderIncludes = "carrier,service_level,payment_term,shipping_term"

func TestSalesOrders_GetIncludedFieldsNotEmpty(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(
		salesOrdersPath+"/"+SeedSalesOrderID,
		url.Values{"include": {allSalesOrderIncludes}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)

	// Priority is always present (not expandable), check its ID
	if pri := jsonObject(got, "priority"); pri != nil {
		assertPriorityFieldsPopulated(t, pri, "priority")
	}

	// Expandable sub-resources
	if carrier := jsonObject(got, "carrier"); carrier != nil {
		assertCarrierFieldsPopulated(t, carrier, "carrier")
	}

	if sl := jsonObject(got, "service_level"); sl != nil {
		assertServiceLevelFieldsPopulated(t, sl, "service_level")
	}

	if pt := jsonObject(got, "payment_term"); pt != nil {
		assertPaymentTermFieldsPopulated(t, pt, "payment_term")
	}

	if st := jsonObject(got, "shipping_term"); st != nil {
		assertShippingTermFieldsPopulated(t, st, "shipping_term")
	}
}

func TestSalesOrders_ListPriorityIDNotEmpty(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(salesOrdersPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	data, ok := got["data"].([]any)
	require.True(t, ok, "response should have data array")
	require.NotEmpty(t, data, "should have at least one sales order")

	first, ok := data[0].(map[string]any)
	require.True(t, ok, "first element should be an object")

	if pri := jsonObject(first, "priority"); pri != nil {
		assertPriorityFieldsPopulated(t, pri, "list[0].priority")
	}
}

// ──────────────────────────────────────────────
// Shipment — Included Sub-Resource Completeness
// ──────────────────────────────────────────────

const shipmentsPath = "/v1/operations/shipments"
const allShipmentIncludes = "carrier,service_level"

func TestShipments_GetIncludedFieldsNotEmpty(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(
		shipmentsPath+"/"+SeedShipmentID,
		url.Values{"include": {allShipmentIncludes}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)

	if carrier := jsonObject(got, "carrier"); carrier != nil {
		assertCarrierFieldsPopulated(t, carrier, "carrier")
	}

	if sl := jsonObject(got, "service_level"); sl != nil {
		assertServiceLevelFieldsPopulated(t, sl, "service_level")
	}
}

func TestShipments_ListIncludedFieldsNotEmpty(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shipmentsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	data, ok := got["data"].([]any)
	require.True(t, ok, "response should have data array")
	require.NotEmpty(t, data, "should have at least one shipment")

	// Carrier and service_level are always present on shipment summaries (not expandable)
	first, ok := data[0].(map[string]any)
	require.True(t, ok, "first element should be an object")

	if carrier := jsonObject(first, "carrier"); carrier != nil {
		assertCarrierFieldsPopulated(t, carrier, "list[0].carrier")
	}

	if sl := jsonObject(first, "service_level"); sl != nil {
		assertServiceLevelFieldsPopulated(t, sl, "list[0].service_level")
	}
}

// ──────────────────────────────────────────────
// Invoice — Included Sub-Resource Completeness
// ──────────────────────────────────────────────

const invoicesPath = "/v1/finance/invoices"

func TestInvoices_ListPaymentTermFieldsNotEmpty(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(invoicesPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	data, ok := got["data"].([]any)
	require.True(t, ok, "response should have data array")
	require.NotEmpty(t, data, "should have at least one invoice")

	first, ok := data[0].(map[string]any)
	require.True(t, ok, "first element should be an object")

	if pt := jsonObject(first, "payment_term"); pt != nil {
		assertPaymentTermFieldsPopulated(t, pt, "list[0].payment_term")
	}
}
