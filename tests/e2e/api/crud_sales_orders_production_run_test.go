//go:build e2e

package api_test

import (
	"database/sql"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Behavioral coverage for POST /v1/sales/sales-orders/{id}/actions/create-production-run,
// pinning the legacy material-demand semantics: the BOM is exploded to LEAF raw materials
// only (intermediate parts and the finished good are never reserved), each consumption's
// waste is folded into demand, and the order-line unit is normalized before scaling.
//
// The reservations are read straight from the e2e DB (via authDB) keyed by order id — the
// item-inventory API does not surface production-run material reservations, and a per-order
// read is exact and isolated (so no global before/after bookkeeping is needed).

// Seed BOM: "Large Beige sock" is produced by a single Pack step whose flow chains
// Pack → Board → Dye (consumes 50× the sewn sock) → Sew → Knit → raw yarn. All part
// quantities are in `pair`; the sellable product may be ordered in pair or dozen.
const (
	largeBeigeProductID = "pd_01k0a65nx5e67rd1rahv4tdnrp"
	largeBeigeItemID    = "it_01k0a7100ae85v16mmxx5gx2w3"
	yarn1ItemID         = "it_01seedyrn1item00000" // leaf, 0.06 lb/pair, zero waste
	beigeDyeItemID      = "it_01seeddye1item00000" // leaf, 1.5 g + 0.15 g waste
	lknItemID           = "it_01seedlknitem000000" // intermediate (large knitted sock)
	noFlowProductID     = "pd_01k0a65nx5e3haz2fgfm34hmcz"
	noFlowItemID        = "it_01k0a7100aedgv8416p4p2v9ks"
	// The sellable-sock unit group is pair/dozen. dozen = 12 each = 6 pair, so ordering
	// 1 dozen must scale demand ×6.
	dozenUnitID = "un_01seeddozen00000000"
)

// reservedForOrderItem sums the reserved-material quantity a production run created for a
// given (order, item), read directly from the e2e DB.
func reservedForOrderItem(t *testing.T, orderID, itemID string) float64 {
	t.Helper()
	var total sql.NullFloat64
	err := authDB(t).QueryRow(
		`SELECT COALESCE(SUM(q.value), 0)
		   FROM inventory_issue ii
		   JOIN quantity q ON q.id = ii.quantity_id
		  WHERE ii.order_id = ? AND ii.item_id = ? AND ii.status_code = 'reserved'`,
		orderID, itemID,
	).Scan(&total)
	require.NoError(t, err, "querying reserved material for order %s item %s", orderID, itemID)
	if !total.Valid {
		return 0
	}
	return total.Float64
}

// reservedRowCountForOrderItem counts the distinct reserved-material rows a production run
// created for a given (order, item) — used to prove per-material aggregation.
func reservedRowCountForOrderItem(t *testing.T, orderID, itemID string) int {
	t.Helper()
	var n int
	err := authDB(t).QueryRow(
		`SELECT COUNT(*) FROM inventory_issue WHERE order_id = ? AND item_id = ? AND status_code = 'reserved'`,
		orderID, itemID,
	).Scan(&n)
	require.NoError(t, err, "counting reserved rows for order %s item %s", orderID, itemID)
	return n
}

// productionRunUser logs in as the seeded admin user. The production-run endpoint records
// a responsible account user, so the actor must resolve to one — the default API-key
// client has no user actor.
func productionRunUser(t *testing.T) *Client {
	t.Helper()
	return loginAsUser(t, seedUserEmail, seedUserPassword, SeedAccountID)
}

// createProductionOrder creates an estimate order for one product line at the given
// quantity/unit and registers cleanup. Returns the order id.
func createProductionOrder(t *testing.T, customerID, productID, qtyValue, qtyUnitID string) string {
	t.Helper()
	body := minimalSalesOrderCreateBody(t, customerID)
	body["lines"] = []map[string]any{
		{
			"product_id": productID,
			"quantity":   map[string]any{"value": qtyValue, "unit_id": qtyUnitID},
		},
	}
	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	orderID := jsonField(parseJSON(respBody), "id")
	deleteOrder(t, orderID)
	return orderID
}

func runProductionForOrder(t *testing.T, client *Client, orderID string) (int, []byte) {
	t.Helper()
	status, body, err := client.Post(salesOrdersPath+"/"+orderID+"/actions/create-production-run", nil, newIdempotencyKey())
	require.NoError(t, err)
	return status, body
}

func TestProductionRun_ExplodesBOMToLeafMaterialsWithWaste(t *testing.T) {
	user := productionRunUser(t)
	customerID := setupOrderCustomer(t)
	orderID := createProductionOrder(t, customerID, largeBeigeProductID, "1", SeedUnitID) // 1 pair

	status, body := runProductionForOrder(t, user, orderID)
	requireStatus(t, 201, status, body)

	// The order links to the run (exposed as the related.production_run record).
	related := jsonObject(getSalesOrder(t, orderID, url.Values{"include": {"related.production_run"}}), "related")
	require.NotNil(t, related, "related is present")
	run := jsonObject(related, "production_run")
	require.NotNil(t, run, "the order links to the created production run")
	assert.NotEmpty(t, jsonField(run, "id"))

	// A second run on the same order conflicts.
	status2, body2 := runProductionForOrder(t, user, orderID)
	requireClientError(t, status2, body2)

	// Leaf raw materials are reserved; the 50× consumption in the Dye step amplifies the
	// whole sew/knit subtree.
	assert.InDelta(t, 3.0, reservedForOrderItem(t, orderID, yarn1ItemID), 1e-6,
		"Yarn1 = 0.06 lb/pair × 50 (dye amplifier) = 3 lb")
	// Waste is folded into demand: 1.5 g consumption + 0.15 g waste.
	assert.InDelta(t, 1.65, reservedForOrderItem(t, orderID, beigeDyeItemID), 1e-6,
		"Beige Dye = 1.5 g + 0.15 g waste = 1.65 g")
	// Intermediate parts and the finished good itself are never reserved.
	assert.InDelta(t, 0.0, reservedForOrderItem(t, orderID, lknItemID), 1e-6,
		"an intermediate part (knitted sock) is not a raw-material reservation")
	assert.InDelta(t, 0.0, reservedForOrderItem(t, orderID, largeBeigeItemID), 1e-6,
		"the finished good is not reserved as its own material")
}

func TestProductionRun_NormalizesNonBaseOrderUnit(t *testing.T) {
	user := productionRunUser(t)
	customerID := setupOrderCustomer(t)
	// 1 dozen = 6 pair. Demand must scale ×6, not ×1 (raw math treats dozen as pair).
	orderID := createProductionOrder(t, customerID, largeBeigeProductID, "1", dozenUnitID)

	status, body := runProductionForOrder(t, user, orderID)
	requireStatus(t, 201, status, body)

	assert.InDelta(t, 18.0, reservedForOrderItem(t, orderID, yarn1ItemID), 1e-6,
		"order unit normalized: 1 dozen = 6 pair → Yarn1 = 0.06 × 50 × 6 = 18 lb (raw math would give 3)")
}

func TestProductionRun_AggregatesReservationsPerMaterial(t *testing.T) {
	user := productionRunUser(t)
	customerID := setupOrderCustomer(t)

	// Two lines of the same product each demand Yarn1; the run must aggregate them into a
	// single reservation per material (not one row per line).
	body := minimalSalesOrderCreateBody(t, customerID)
	body["lines"] = []map[string]any{
		{"product_id": largeBeigeProductID, "quantity": map[string]any{"value": "1", "unit_id": SeedUnitID}},
		{"product_id": largeBeigeProductID, "quantity": map[string]any{"value": "1", "unit_id": SeedUnitID}},
	}
	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	orderID := jsonField(parseJSON(respBody), "id")
	deleteOrder(t, orderID)

	rStatus, rBody := runProductionForOrder(t, user, orderID)
	requireStatus(t, 201, rStatus, rBody)

	assert.InDelta(t, 6.0, reservedForOrderItem(t, orderID, yarn1ItemID), 1e-6,
		"two 1-pair lines → Yarn1 = 2 × 3 lb = 6 lb")
	assert.Equal(t, 1, reservedRowCountForOrderItem(t, orderID, yarn1ItemID),
		"the shared material aggregates into a single reservation row, not one per line")
}

func TestProductionRun_NoFlowFinishedGoodReservesNothing(t *testing.T) {
	user := productionRunUser(t)
	customerID := setupOrderCustomer(t)
	orderID := createProductionOrder(t, customerID, noFlowProductID, "5", SeedUnitID)

	status, body := runProductionForOrder(t, user, orderID)
	requireStatus(t, 201, status, body)

	assert.InDelta(t, 0.0, reservedForOrderItem(t, orderID, noFlowItemID), 1e-6,
		"a finished good with no production flow reserves no materials, and not itself")
}
