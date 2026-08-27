//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Behavioral coverage for POST /v1/sales/sales-orders, pinning the create-path
// semantics ported from the legacy Dashboard API: estimate-status default, automatic
// order numbering, synthesized shipping line, promised_at, duplicate-customer-PO
// conflict, required-field validation, and customer-actor self-create authorization.

const productLineAccessPath = "/v1/sales/product-line-access/customers"

// setupOrderCustomer creates a fresh customer with access to the seed product's line
// so it can be used as a buyer on a sales order, and registers cleanup.
func setupOrderCustomer(t *testing.T) string {
	t.Helper()

	status, body, err := apiClient.Post(customersPath, validCustomerBody(uniqueName("e2e-so-create-cust")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	customerID := jsonField(parseJSON(body), "id")

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

// deleteOrder removes an order created during a test (internal actor owns it).
func deleteOrder(t *testing.T, orderID string) {
	t.Helper()
	t.Cleanup(func() { _, _, _ = apiClient.Delete(salesOrdersPath + "/" + orderID) })
}

func TestCreateSalesOrder_HappyPath_DefaultsEstimateAndSynthesizesShippingLine(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)

	// Note: the body intentionally omits sales_order_type_code — it is a storage
	// discriminator set in the repository layer, not API surface.
	status, body, err := apiClient.Post(salesOrdersPath, minimalSalesOrderCreateBody(t, customerID), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	parsed := parseJSON(body)
	orderID := jsonField(parsed, "id")
	deleteOrder(t, orderID)

	assert.Equal(t, "sales_order", jsonField(parsed, "object"))
	assert.Equal(t, "estimate", jsonField(parsed, "status"), "new orders default to estimate")
	assert.NotEmpty(t, jsonField(parsed, "number"), "an order number is assigned automatically")
	// One input line + one synthesized shipping line == line_count 2.
	assert.EqualValues(t, 2, parsed["line_count"], "a shipping line is always synthesized")
}

func TestCreateSalesOrder_PromisedAtRoundTrips(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)

	bodyReq := minimalSalesOrderCreateBody(t, customerID)
	bodyReq["promised_at"] = "2026-09-01T00:00:00Z"

	status, body, err := apiClient.Post(salesOrdersPath, bodyReq, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	orderID := jsonField(parseJSON(body), "id")
	deleteOrder(t, orderID)

	gstatus, gbody, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, gstatus, gbody)
	promisedAt := jsonField(commitmentOf(parseJSON(gbody)), "promised_at")
	assert.Contains(t, promisedAt, "2026-09-01", "promised_at is stored at create time")
}

func TestCreateSalesOrder_DuplicateCustomerPONumberConflicts(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)

	po := uniqueName("PO")
	bodyReq := minimalSalesOrderCreateBody(t, customerID)
	bodyReq["customer_purchase_order_number"] = po

	status, body, err := apiClient.Post(salesOrdersPath, bodyReq, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	deleteOrder(t, jsonField(parseJSON(body), "id"))

	// A second order with the same customer PO (new idempotency key) conflicts.
	dupReq := minimalSalesOrderCreateBody(t, customerID)
	dupReq["customer_purchase_order_number"] = po
	dupStatus, dupBody, err := apiClient.Post(salesOrdersPath, dupReq, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, dupStatus, dupBody)
	assertErrorParam(t, jsonObject(parseJSON(dupBody), "error"), "customer_po_number")
}

func TestCreateSalesOrder_UnitNotInProductGroupRejected(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)

	bodyReq := minimalSalesOrderCreateBody(t, customerID)
	// "dollar" (the currency unit) is not in the seed product's unit group.
	bodyReq["lines"] = []map[string]any{
		{
			"product_id": SeedProductID,
			"quantity":   map[string]any{"value": "1", "unit_id": e2eCurrencyUnitID},
		},
	}

	status, body, err := apiClient.Post(salesOrdersPath, bodyReq, newIdempotencyKey())
	require.NoError(t, err)
	require.GreaterOrEqual(t, status, 400, "invalid unit must be rejected")
	require.Less(t, status, 500, "validation failure must not 5xx: %s", string(body))
	assertErrorParam(t, jsonObject(parseJSON(body), "error"), "quantity_unit_id")
}

func TestCreateSalesOrder_MissingBuyerAccountRejected(t *testing.T) {
	t.Parallel()

	bodyReq := minimalSalesOrderCreateBody(t, SeedCustomerAccountID)
	delete(bodyReq, "buyer_account_id")

	status, body, err := apiClient.Post(salesOrdersPath, bodyReq, newIdempotencyKey())
	require.NoError(t, err)
	require.GreaterOrEqual(t, status, 400, "missing buyer_account_id is rejected")
	require.Less(t, status, 500, "validation failure must not 5xx: %s", string(body))
}

func TestCreateSalesOrder_CustomerActor_SelfCreateAllowed_OtherBuyerForbidden(t *testing.T) {
	t.Parallel()
	portal := getCustomerPortalClient()

	// Ensure the seed customer has access to the product's line (idempotent: a 409
	// means it was already granted by the seed).
	plStatus, _, err := apiClient.Post(productLineAccessPath, map[string]any{
		"customer_id":      SeedCustomerAccountID,
		"product_line_ids": []string{SeedProductLineID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.True(t, plStatus == 201 || plStatus == 409, "unexpected access-grant status: %d", plStatus)

	// A customer may create an order for its own account.
	status, body, err := portal.Post(salesOrdersPath, minimalSalesOrderCreateBody(t, SeedCustomerAccountID), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	deleteOrder(t, jsonField(parseJSON(body), "id"))

	// A customer may NOT create an order for a different buyer (the vendor account).
	forbidReq := minimalSalesOrderCreateBody(t, SeedAccountID)
	fStatus, fBody, err := portal.Post(salesOrdersPath, forbidReq, newIdempotencyKey())
	require.NoError(t, err)
	require.Equal(t, 403, fStatus, "customer cannot create for another account: %s", string(fBody))
}
