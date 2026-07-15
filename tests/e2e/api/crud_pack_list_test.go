//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Behavioral coverage for POST /v1/core/records/actions/generate-pack-list, which
// assembles a printable pack-list document for a shipment by composing the shipment
// (lines + shipping cases) with its parent sales order (header, parties, terms, and
// back-ordered lines) and the selling account.
//
// Seed facts (verified against the e2e DB — see 0011_orders.sql + e2e/0014_e2e_extras.sql):
//   - SeedShipmentID sh_...86cg1 = "SHP-001", status packed, belongs to order ORD-002
//     (or_...4dfmwe), seller account "Acme Inc." (SeedAccountID).
//   - ORD-002 lines: SCK-003 "Small black sock" (sale, ordered 20 pr, packed 20),
//     SCK-005 "Small beige sock" (sale, ordered 15 pr, packed 15), Freight (shipping,
//     ordered 1, packed 0).
//   - SHP-001 shipment lines: SCK-003 x20 pr, SCK-005 x15 pr (the two sale lines).
//   - SHP-001 shipping cases: SHP-001-1, SHP-001-2, freight weight 0 lbs, no tracking
//     (packed, never shipped).
//   - ORD-002 has no customer PO; bill-to is the customer "Global Manufacturing Solutions".

const genPackListPath = "/v1/core/records/actions/generate-pack-list"

// packListSubList returns the `data` array of a pack-list sub-list (line_items,
// back_orders, shipping_cases), asserting the wrapper is a List resource.
func packListSubList(t *testing.T, packList map[string]any, key string) []any {
	t.Helper()
	list := jsonObject(packList, key)
	require.NotNil(t, list, "%s should be present", key)
	assert.Equal(t, "list", jsonField(list, "object"), "%s should be a list resource", key)
	return jsonArray(list, "data")
}

// packListFloat parses a decimal-string field (quantities are returned as decimal
// strings) to a float for numeric comparison.
func packListFloat(t *testing.T, m map[string]any, key string) float64 {
	t.Helper()
	f, err := strconv.ParseFloat(jsonField(m, key), 64)
	require.NoError(t, err, "%s should parse as a number: %v", key, m[key])
	return f
}

func TestGenPackList_AssemblesShipmentDocument(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(genPackListPath, map[string]any{"shipment_id": SeedShipmentID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	pl := parseJSON(body)
	assert.Equal(t, "pack_list", jsonField(pl, "object"))
	assert.Equal(t, "Acme Inc.", jsonField(pl, "account_name"))
	assert.Equal(t, "ORD-002", jsonField(pl, "sales_order_number"))
	assert.Equal(t, "SHP-001", jsonField(pl, "shipment_number"))
	assert.Nil(t, pl["customer_po"], "ORD-002 has no customer PO")

	// Parties.
	billTo := jsonObject(pl, "bill_to")
	require.NotNil(t, billTo, "bill_to should be present")
	assert.Equal(t, "pack_list_party", jsonField(billTo, "object"))
	assert.Equal(t, "Global Manufacturing Solutions", jsonField(billTo, "name"))
	shipTo := jsonObject(pl, "ship_to")
	require.NotNil(t, shipTo, "ship_to should be present")
	assert.Equal(t, "pack_list_party", jsonField(shipTo, "object"))

	// Line items: the two packed sale socks; the Freight charge line is never a
	// packed line item.
	items := packListSubList(t, pl, "line_items")
	require.Len(t, items, 2)
	bySKU := map[string]map[string]any{}
	for _, it := range items {
		m, ok := it.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "pack_list_line_item", jsonField(m, "object"))
		bySKU[jsonField(m, "sku")] = m
	}
	require.Contains(t, bySKU, "SCK-003")
	require.Contains(t, bySKU, "SCK-005")
	assert.NotContains(t, bySKU, "Freight", "shipping-charge lines are not packed line items")
	assert.Equal(t, "Small black sock", jsonField(bySKU["SCK-003"], "description"))
	assert.Equal(t, "Pair", jsonField(bySKU["SCK-003"], "unit"))
	assert.InDelta(t, 20.0, packListFloat(t, bySKU["SCK-003"], "quantity"), 0.001)
	assert.InDelta(t, 15.0, packListFloat(t, bySKU["SCK-005"], "quantity"), 0.001)

	// Shipping cases: sorted by number, pound weight unit, no tracking (packed, not shipped).
	cases := packListSubList(t, pl, "shipping_cases")
	require.Len(t, cases, 2)
	c0, _ := cases[0].(map[string]any)
	c1, _ := cases[1].(map[string]any)
	assert.Equal(t, "pack_list_case", jsonField(c0, "object"))
	assert.Equal(t, "SHP-001-1", jsonField(c0, "number"))
	assert.Equal(t, "SHP-001-2", jsonField(c1, "number"))
	assert.Equal(t, "lbs", jsonField(c0, "weight_unit"))
	assert.InDelta(t, 0.0, packListFloat(t, c0, "weight"), 0.001)
	assert.Nil(t, c0["tracking_number"], "a packed (unshipped) case has no tracking number")

	// contact_information is always an array (order email recipients + billing phone).
	assert.NotNil(t, pl["contact_information"], "contact_information should be present")
	_, ok := pl["contact_information"].([]any)
	assert.True(t, ok, "contact_information should be an array")
}

func TestGenPackList_ExcludesNonSaleLinesFromBackOrders(t *testing.T) {
	t.Parallel()

	// ORD-002's two sale lines are fully packed (20/20, 15/15), so neither is
	// back-ordered. Its Freight line is ordered 1 / packed 0 — it WOULD surface as a
	// back-order of 1 if the back-order table did not filter to sale-type products.
	// An empty back_orders list therefore pins both the fully-packed case and the
	// product-type exclusion (regression guard for the product_type_code filter).
	status, body, err := apiClient.Post(genPackListPath, map[string]any{"shipment_id": SeedShipmentID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	backOrders := packListSubList(t, parseJSON(body), "back_orders")
	assert.Empty(t, backOrders, "no back-orders: sale lines fully packed and the Freight (shipping) line is excluded")
}

func TestGenPackList_LineItemsMatchShipmentLines(t *testing.T) {
	t.Parallel()

	// Cross-check against the shipment resource: the pack list's line items are exactly
	// the shipment's lines, so it must not drop or duplicate any. Robust to seed drift.
	shStatus, shBody, err := apiClient.GetListRaw(shipmentsPath+"/"+SeedShipmentID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, shStatus, shBody)
	shipmentLines := jsonArray(jsonObject(parseJSON(shBody), "lines"), "data")

	plStatus, plBody, err := apiClient.Post(genPackListPath, map[string]any{"shipment_id": SeedShipmentID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, plStatus, plBody)
	packListItems := packListSubList(t, parseJSON(plBody), "line_items")

	assert.Equal(t, len(shipmentLines), len(packListItems), "pack list line-item count should match the shipment's line count")
}

func TestGenPackList_ShipmentNotFound(t *testing.T) {
	t.Parallel()

	// Well-formed but nonexistent shipment id → 404, not a 5xx.
	status, body, err := apiClient.Post(genPackListPath, map[string]any{"shipment_id": "sh_01k0a87w33emw8pmkz1mf80000"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

func TestGenPackList_RequiresShipmentID(t *testing.T) {
	t.Parallel()

	// shipment_id is required; omitting it is a request-validation error, not a 5xx.
	status, body, err := apiClient.Post(genPackListPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}
