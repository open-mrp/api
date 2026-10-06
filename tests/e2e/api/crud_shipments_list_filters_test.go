//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Covers the filter set the shipping index page sends (shipment.api.ts fetchShipments): q, status,
// customer_ids, item_ids, product_line_ids, customer_group_ids, sales_rep_ids and the date window.
//
// A filter the server silently ignores still returns 200 with a full page, so every case pairs a
// positive match with a nonsense id that must narrow the list to nothing.
const (
	// SHP-002 is the only permanently shipped seed shipment; SHP-003 stays packed.
	seedShippedShipmentID = "sh_01k0a87w33fw0shhsahaa0yq6r"
	seedPackedShipmentID  = "sh_01k0a87w33emw8pmkz1mf86cg2"
)

// Collects the ids returned by the shipment list under the given filters.
func shipmentIDsFiltered(t *testing.T, params url.Values) []string {
	t.Helper()
	return listIDs(t, shipmentsPath, params)
}

func TestShipmentsList_SearchMatchesShipmentNumber(t *testing.T) {
	t.Parallel()

	assert.Contains(t, shipmentIDsFiltered(t, url.Values{"q": {"SHP-002"}}), seedShippedShipmentID,
		"searching a shipment number should surface that shipment")
	assert.Empty(t, shipmentIDsFiltered(t, url.Values{"q": {"zzz-no-such-shipment-zzz"}}),
		"a search matching nothing must return nothing")
}

func TestShipmentsList_StatusSplitsPackedFromShipped(t *testing.T) {
	t.Parallel()

	shipped := shipmentIDsFiltered(t, url.Values{"status": {"shipped"}})
	assert.Contains(t, shipped, seedShippedShipmentID)
	assert.NotContains(t, shipped, seedPackedShipmentID, "a packed shipment must not appear under shipped")

	packed := shipmentIDsFiltered(t, url.Values{"status": {"packed"}})
	assert.Contains(t, packed, seedPackedShipmentID)
	assert.NotContains(t, packed, seedShippedShipmentID, "a shipped shipment must not appear under packed")
}

func TestShipmentsList_FiltersByCustomer(t *testing.T) {
	t.Parallel()

	assert.Contains(t, shipmentIDsFiltered(t, url.Values{"customer_ids": {SeedCustomerAccountID}}), seedPackedShipmentID)
	assert.Empty(t, shipmentIDsFiltered(t, url.Values{"customer_ids": {"ac_01nosuchcustomer000"}}),
		"an unknown customer must narrow the list to nothing rather than be ignored")
}

// The item and product-line filters reach through the shipment's lines to the order line's product.
func TestShipmentsList_FiltersByItemAndProductLine(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, shipmentIDsFiltered(t, url.Values{"item_ids": {SeedItemID}}),
		"the seeded item ships on at least one shipment")
	assert.Empty(t, shipmentIDsFiltered(t, url.Values{"item_ids": {"it_01nosuchitem0000000"}}))

	assert.NotEmpty(t, shipmentIDsFiltered(t, url.Values{"product_line_ids": {SeedProductLineID}}),
		"the seeded product line ships on at least one shipment")
	assert.Empty(t, shipmentIDsFiltered(t, url.Values{"product_line_ids": {"pdln_01nosuchline00000"}}))
}

// The group is the customer's; the sales rep is the one the order credits (ORD-001's is the seed user).
func TestShipmentsList_FiltersByCustomerGroupAndSalesRep(t *testing.T) {
	t.Parallel()

	assert.Contains(t, shipmentIDsFiltered(t, url.Values{"customer_group_ids": {SeedCustomerGroupID}}), seedPackedShipmentID)
	assert.Empty(t, shipmentIDsFiltered(t, url.Values{"customer_group_ids": {"acgp_01nosuchgroup0000"}}))

	assert.Contains(t, shipmentIDsFiltered(t, url.Values{"sales_rep_ids": {SeedAccountUserID}}), seedPackedShipmentID)
	assert.Empty(t, shipmentIDsFiltered(t, url.Values{"sales_rep_ids": {"acus_nosuchsalesrep00"}}))
}

// The sales rep is the order's, as on the legacy dashboard list, not the customer's default rep: a
// customer defaulting to one rep can place an order another rep is credited with.
func TestShipmentsList_SalesRepIsTheOrdersRepNotTheCustomers(t *testing.T) {
	t.Parallel()

	_, orderRep := paritySalesRep(t)
	_, defaultRep := paritySalesRep(t)
	groupID := leadTimeAccountGroup(t, "e2e-shp-rep-grp", nil)
	customerID := parityCustomer(t, groupID)
	status, body, err := apiClient.Patch(customersPath+"/"+customerID, map[string]any{"default_sales_rep_id": defaultRep}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	order := parityOrderBody(t, customerID)
	order["sales_rep_id"] = orderRep
	orderID, _, shipmentID := packedOrder(t, order)

	status, body, err = apiClient.GetListRaw(salesOrdersPath+"/"+orderID, url.Values{"include": {"sales_rep"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	require.Equal(t, orderRep, jsonField(jsonObject(parseJSON(body), "sales_rep"), "id"), "the order credits its own rep")
	status, body, err = apiClient.GetListRaw(customersPath+"/"+customerID, url.Values{"include": {"defaults.sales_rep"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	require.Equal(t, defaultRep, jsonField(jsonObject(jsonObject(parseJSON(body), "defaults"), "sales_rep"), "id"))

	only := []string{shipmentID}
	assert.Equal(t, only, shipmentIDsFiltered(t, url.Values{"sales_rep_ids": {orderRep}}))
	assert.Empty(t, shipmentIDsFiltered(t, url.Values{"sales_rep_ids": {defaultRep}}), "the customer's default rep is not the order's")
	assert.Equal(t, only, shipmentIDsFiltered(t, url.Values{"sales_rep_ids": {defaultRep, orderRep}}), "reps match any of")

	// The group still narrows through the customer relation, alongside the order's rep.
	assert.Equal(t, only, shipmentIDsFiltered(t, url.Values{"customer_group_ids": {groupID}, "sales_rep_ids": {orderRep}}))
	assert.Empty(t, shipmentIDsFiltered(t, url.Values{"customer_group_ids": {groupID}, "sales_rep_ids": {defaultRep}}))
	assert.Empty(t, shipmentIDsFiltered(t, url.Values{"customer_group_ids": {SeedCustomerGroupID}, "sales_rep_ids": {orderRep}}),
		"the rep's shipment is for a customer outside that group")

	// Two customers can drive the read instead, leaving the rep to be checked per shipment.
	pair := []string{customerID, "ac_01nosuchcustomer000"}
	assert.Equal(t, only, shipmentIDsFiltered(t, url.Values{"customer_ids": pair, "sales_rep_ids": {orderRep}}))
	assert.Empty(t, shipmentIDsFiltered(t, url.Values{"customer_ids": pair, "sales_rep_ids": {defaultRep}}))

	status, body, err = getTenantBClient().GetListRaw(shipmentsPath, url.Values{"sales_rep_ids": {orderRep}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Empty(t, jsonArray(parseJSON(body), "data"), "another account filtering on the rep sees none of this account's shipments")
}

// The window filters on creation, not on when the shipment shipped.
func TestShipmentsList_FiltersByCreatedDateWindow(t *testing.T) {
	t.Parallel()

	today := time.Now().UTC().Format("2006-01-02")
	assert.Contains(t, shipmentIDsFiltered(t, url.Values{"starts_at": {"2000-01-01"}, "ends_at": {today}}),
		seedPackedShipmentID, "a window covering today must include shipments seeded now")

	assert.Empty(t, shipmentIDsFiltered(t, url.Values{"starts_at": {"2000-01-01"}, "ends_at": {"2000-01-02"}}),
		"a window that closed decades ago must exclude every shipment")
}

// The page asks for its filters and its includes in one request, so prove they compose.
func TestShipmentsList_FiltersComposeWithIncludes(t *testing.T) {
	t.Parallel()

	params := url.Values{
		"status":       {"packed"},
		"customer_ids": {SeedCustomerAccountID},
	}
	for _, inc := range []string{"customer", "freight", "shipping_address", "related.sales_order"} {
		params.Add("include", inc)
	}

	status, body, err := apiClient.GetListRaw(shipmentsPath, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	rows := jsonArray(parseJSON(body), "data")
	require.NotEmpty(t, rows, "the filtered page must still return rows")

	row, ok := rows[0].(map[string]any)
	require.True(t, ok)
	assert.NotNil(t, jsonObject(row, "customer"), "customer must expand alongside the filters")
	assert.Equal(t, "packed", jsonField(row, "status"))
}
