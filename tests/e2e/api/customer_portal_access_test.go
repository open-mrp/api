//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Customer portal access tests verify that a customer API key (owned by the
// customer account) can read product lines and unit groups on the vendor's
// account. The customer authenticates with their own API key and sets the
// Augno-Account header to the vendor's account ID.
//
// Seed data used:
//   - Customer account:  ac_01k09wm2fgevdsc344gpbcj30f (Global Manufacturing Solutions)
//   - Vendor account:    ac_01k0a5smf9ekb8rqg12555zjqa (Acme Inc.)
//   - Account relation:  acre_01seedcustomer00000 (customer role)
//   - Customer API key:  SeedCustomerAPIKey (owned by customer account)
//   - Product line:      pdln_01k0a735ype5e8nrhv1n5dhq1q (Socks)
//   - Unit group:        ungp_01k0a5ecy9edg9za40dnccw53n

// customerPortalClient is a Client authenticated as the customer account
// but targeting the vendor account (cross-account access).
var customerPortalClient *Client

func getCustomerPortalClient() *Client {
	if customerPortalClient == nil {
		baseURL := envOr("E2E_BASE_URL", defaultBaseURL)
		customerPortalClient = NewClient(baseURL, SeedCustomerAPIKey, SeedAccountID)
	}
	return customerPortalClient
}

func TestCustomerPortalAccess_ListProductLines(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	status, body, err := client.GetListRaw("/v1/catalog/product-lines", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	parsed := parseJSON(body)
	assert.Equal(t, "list", jsonField(parsed, "object"))
}

func TestCustomerPortalAccess_GetProductLine(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	status, body, err := client.GetListRaw("/v1/catalog/product-lines/"+SeedProductLineID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	parsed := parseJSON(body)
	assert.Equal(t, "product_line", jsonField(parsed, "object"))
	assert.Equal(t, SeedProductLineID, jsonField(parsed, "id"))
}

func TestCustomerPortalAccess_ListProducts(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	status, body, err := client.GetListRaw("/v1/catalog/products", url.Values{
		"include": {"product_line,item,item.category,item.unit_value,item.attributes"},
		"q":       {"SCK-001"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	parsed := parseJSON(body)
	assert.Equal(t, "list", jsonField(parsed, "object"))
	data := parsed["data"].([]any)
	require.NotEmpty(t, data, "customer should see portal-ready products from accessible product lines")

	first := data[0].(map[string]any)
	assert.Equal(t, "product", jsonField(first, "object"))
	assert.NotNil(t, jsonField(first, "product_line"))
	assert.NotNil(t, jsonField(first, "item"))
}

func TestCustomerPortalAccess_GetProduct(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	status, body, err := client.GetListRaw("/v1/catalog/products/"+SeedProductID, url.Values{
		"include": {"product_line,item,item.category,item.unit_value,item.attributes"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	parsed := parseJSON(body)
	assert.Equal(t, "product", jsonField(parsed, "object"))
	assert.Equal(t, SeedProductID, jsonField(parsed, "id"))
	assert.NotNil(t, jsonField(parsed, "product_line"))
	assert.NotNil(t, jsonField(parsed, "item"))
}

func TestCustomerPortalAccess_ListUnitGroups(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	status, body, err := client.GetListRaw("/v1/catalog/unit-groups", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	parsed := parseJSON(body)
	assert.Equal(t, "list", jsonField(parsed, "object"))
}

func TestCustomerPortalAccess_GetUnitGroup(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	status, body, err := client.GetListRaw("/v1/catalog/unit-groups/"+SeedUnitGroupID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	parsed := parseJSON(body)
	assert.Equal(t, "unit_group", jsonField(parsed, "object"))
	assert.Equal(t, SeedUnitGroupID, jsonField(parsed, "id"))
}

func TestCustomerPortalAccess_GetUnitGroupWithIncludes(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	status, body, err := client.GetListRaw("/v1/catalog/unit-groups/currency_group", url.Values{
		"include": {"owner,base_unit,associated_units"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	parsed := parseJSON(body)
	assert.Equal(t, "unit_group", jsonField(parsed, "object"))
	assert.Equal(t, "currency_group", jsonField(parsed, "id"))
	assert.NotNil(t, jsonField(parsed, "owner"))
	assert.NotNil(t, jsonField(parsed, "base_unit"))
	assert.NotNil(t, jsonField(parsed, "associated_units"))
}

func TestCustomerPortalAccess_ListUnitGroupUnits(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	status, body, err := client.GetListRaw("/v1/catalog/unit-groups/"+SeedUnitGroupID+"/units", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	parsed := parseJSON(body)
	assert.Equal(t, "list", jsonField(parsed, "object"))
}

func TestCustomerPortalAccess_GetUnitGroupUnit(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	status, body, err := client.GetListRaw("/v1/catalog/unit-groups/"+SeedUnitGroupID+"/units/"+SeedUnitGroupUnitID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	parsed := parseJSON(body)
	assert.Equal(t, "unit_group_unit", jsonField(parsed, "object"))
	assert.Equal(t, SeedUnitGroupUnitID, jsonField(parsed, "id"))
}

// TestCustomerPortalAccess_ListSalesOrders verifies a customer portal actor can
// list sales orders on the vendor's account WITHOUT sales_orders:read, scoped to
// orders where they are the buyer. Regression guard for the portal 403: the
// portal must target the vendor account (the order owner), not the customer's
// own account — targeting their own account classifies them as an internal actor
// and trips the sales_orders:read permission check.
func TestCustomerPortalAccess_ListSalesOrders(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	status, body, err := client.GetListRaw(salesOrdersPath, url.Values{
		"include": {"customer"},
		"limit":   {"100"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	parsed := parseJSON(body)
	assert.Equal(t, "list", jsonField(parsed, "object"))

	var sawBoughtOrder, sawVendorInternalOrder bool
	for _, item := range jsonArray(parsed, "data") {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		// Every visible order must belong to the customer (buyer == customer account).
		cust := jsonObject(row, "customer")
		require.NotNil(t, cust, "customer should be expanded with ?include=customer")
		assert.Equal(t, SeedCustomerAccountID, jsonField(cust, "id"),
			"a customer must only see orders where they are the buyer")
		switch jsonField(row, "id") {
		case SeedPOSalesOrderID:
			sawBoughtOrder = true
		case SeedInternalSalesOrderID:
			sawVendorInternalOrder = true
		}
	}
	assert.True(t, sawBoughtOrder, "customer should see an order they bought (%s)", SeedPOSalesOrderID)
	assert.False(t, sawVendorInternalOrder,
		"customer must not see the vendor's internal order (%s)", SeedInternalSalesOrderID)
}

// TestCustomerPortalAccess_CannotCreateProductLine verifies that customer
// portal users cannot create product lines (write access restricted to internal actors).
func TestCustomerPortalAccess_CannotCreateProductLine(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	status, body, err := client.Post("/v1/catalog/product-lines", map[string]any{
		"name":              uniqueName("e2e-cust-pl"),
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, status, "customer should not be able to create product lines: %s", string(body))
}

// TestCustomerPortalAccess_CannotCreateUnitGroup verifies that customer
// portal users cannot create unit groups.
func TestCustomerPortalAccess_CannotCreateUnitGroup(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	status, body, err := client.Post("/v1/catalog/unit-groups", map[string]any{
		"name":         uniqueName("e2e-cust-ug"),
		"type":         "quantity",
		"base_unit_id": SeedUnitID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, status, "customer should not be able to create unit groups: %s", string(body))
}
