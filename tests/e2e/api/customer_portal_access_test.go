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
// OpenMRP-Account header to the vendor's account ID.
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

// TestCustomerPortalAccess_OrderRetrieveRelatedShipments verifies that a customer
// portal actor can retrieve an order they bought with related.shipments expanded
// (for tracking), while seller-internal related.pick / related.production_run
// remain omitted rather than 403ing the retrieve. Before customer-safe GetShipment,
// resolving related.shipments also failed authorization and the include was dropped.
// ORD-001 (or_01k0a8bs2yejxbsvqhrx4drkq1) is bought by the seed customer and has
// both a pick and SHP-003, so the loaders actually fire.
func TestCustomerPortalAccess_OrderRetrieveRelatedShipments(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	status, body, err := client.GetListRaw("/v1/sales/sales-orders/or_01k0a8bs2yejxbsvqhrx4drkq1", url.Values{
		"include": {"related.pick,related.shipments,related.production_run"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	parsed := parseJSON(body)
	assert.Equal(t, "sales_order", jsonField(parsed, "object"))
	assert.Equal(t, "or_01k0a8bs2yejxbsvqhrx4drkq1", jsonField(parsed, "id"))

	related := jsonObject(parsed, "related")
	require.NotNil(t, related, "related must be present so shipments can surface for tracking")
	assertNilField(t, related, "pick")
	assertNilField(t, related, "production_run")

	shipments := jsonObject(related, "shipments")
	require.NotNil(t, shipments, "related.shipments must populate for a customer on an order they bought")
	assert.Equal(t, "list", jsonField(shipments, "object"))
	data := jsonArray(shipments, "data")
	require.NotEmpty(t, data, "ORD-001 seed has SHP-003")

	var found bool
	for _, item := range data {
		rec, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if jsonField(rec, "id") != "sh_01k0a87w33emw8pmkz1mf86cg2" {
			continue
		}
		found = true
		assert.Equal(t, "SHP-003", jsonField(rec, "number"))
		assert.Equal(t, "packed", jsonField(rec, "status"))
		// SHP-003 is packed without a tracking number; carrier metadata still previews.
		meta := jsonObject(rec, "metadata")
		if meta != nil {
			assert.Equal(t, "packed", jsonField(meta, "status"))
		}
	}
	assert.True(t, found, "related.shipments must include seed SHP-003 for ORD-001")
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

// TestCustomerPortalAccess_SalesOrderLineUnitsHydrated verifies that a sales order's line
// quantity/rate UNIT sub-objects hydrate for a customer relation actor when requested via
// nested includes. Regression: the gateway built the line quantity/rate stubs without stashing
// their unit FKs, so ?include=lines.quantity_ordered.unit (and unit_price.numerator_unit)
// returned null — which broke unit display and the client-side order total on the portal.
func TestCustomerPortalAccess_SalesOrderLineUnitsHydrated(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	// Seed estimate EST-001 (or_01k0a8bs2yfhev5begay245wez): owned by the vendor, bought by the
	// seed customer, whose lines carry quantity units (pair / each) and unit prices.
	status, body, err := client.GetListRaw("/v1/sales/sales-orders/or_01k0a8bs2yfhev5begay245wez", url.Values{
		"include": {"lines,lines.quantity_ordered,lines.quantity_ordered.unit,lines.unit_price,lines.unit_price.numerator_unit,lines.unit_price.denominator_unit"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	parsed := parseJSON(body)
	linesObj := jsonObject(parsed, "lines")
	require.NotNil(t, linesObj, "lines must be present")
	data, ok := linesObj["data"].([]any)
	require.True(t, ok, "lines.data must be an array")
	require.NotEmpty(t, data, "seed estimate has lines")

	for i, raw := range data {
		line := raw.(map[string]any)

		qty := jsonObject(line, "quantity_ordered")
		require.NotNil(t, qty, "line %d: quantity_ordered", i)
		unit := jsonObject(qty, "unit")
		require.NotNil(t, unit, "line %d: quantity_ordered.unit must be hydrated (regression: was null)", i)
		assert.Equal(t, "unit", jsonField(unit, "object"), "line %d: quantity_ordered.unit object", i)
		assert.NotEmpty(t, jsonField(unit, "id"), "line %d: quantity_ordered.unit.id", i)
		assert.NotEmpty(t, jsonField(unit, "abbreviation"), "line %d: quantity_ordered.unit.abbreviation", i)

		if up := jsonObject(line, "unit_price"); up != nil {
			num := jsonObject(up, "numerator_unit")
			require.NotNil(t, num, "line %d: unit_price.numerator_unit must be hydrated (regression: was null)", i)
			assert.NotEmpty(t, jsonField(num, "id"), "line %d: unit_price.numerator_unit.id", i)
			den := jsonObject(up, "denominator_unit")
			require.NotNil(t, den, "line %d: unit_price.denominator_unit must be hydrated", i)
			assert.NotEmpty(t, jsonField(den, "id"), "line %d: unit_price.denominator_unit.id", i)
		}
	}
}

// TestCustomerPortalAccess_FrequentlyOrderedProductsHydrated verifies the frequently-ordered
// list returns FULLY-hydrated item and unit resources for a customer relation actor, rather
// than the partial stubs it used to emit. Regression guards:
//   - the gateway used to build a stub item{id, sku} where sku was actually the item
//     DESCRIPTION, breaking the portal "quick reorder" SKU lookup ("Could not find product ...").
//   - every other item/unit field was the Go zero value, serializing as ""/null/zero timestamps.
func TestCustomerPortalAccess_FrequentlyOrderedProductsHydrated(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	status, body, err := client.GetListRaw("/v1/sales/customers/"+SeedCustomerAccountID+"/frequently-ordered-products", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	parsed := parseJSON(body)
	assert.Equal(t, "list", jsonField(parsed, "object"))
	data, ok := parsed["data"].([]any)
	require.True(t, ok, "data must be an array")
	require.NotEmpty(t, data, "seed customer has order history with the vendor, so the list must not be empty")

	for i, raw := range data {
		row := raw.(map[string]any)
		assert.Equal(t, "frequently_ordered_product", jsonField(row, "object"), "row %d object", i)

		item := jsonObject(row, "item")
		require.NotNil(t, item, "row %d: item must be present", i)
		assert.Equal(t, "item", jsonField(item, "object"), "row %d: item must be a hydrated item resource", i)
		assert.NotEmpty(t, jsonField(item, "id"), "row %d: item.id must be set", i)

		sku := jsonField(item, "sku")
		assert.NotEmpty(t, sku, "row %d: item.sku must be populated (regression: was empty/stubbed)", i)
		if desc := jsonField(item, "description"); desc != "" {
			assert.NotEqualf(t, desc, sku, "row %d: item.sku must not equal item.description (regression: description mapped into sku)", i)
		}
		// A zero timestamp proves a stub was serialized instead of the real hydrated DB row.
		assert.NotEqual(t, "0001-01-01T00:00:00Z", jsonField(item, "created_at"), "row %d: item.created_at must be a real timestamp", i)
		assert.NotEqual(t, "0001-01-01T00:00:00Z", jsonField(item, "updated_at"), "row %d: item.updated_at must be a real timestamp", i)

		if unit := jsonObject(row, "unit"); unit != nil {
			assert.Equal(t, "unit", jsonField(unit, "object"), "row %d: unit must be a hydrated unit resource", i)
			assert.NotEmpty(t, jsonField(unit, "id"), "row %d: unit.id must be set", i)
			assert.NotEmpty(t, jsonField(unit, "abbreviation"), "row %d: unit.abbreviation must be set", i)
			assert.NotEqual(t, "0001-01-01T00:00:00Z", jsonField(unit, "created_at"), "row %d: unit.created_at must be a real timestamp", i)
		}
	}
}
