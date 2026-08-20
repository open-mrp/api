//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The Yarn unit group holds pounds (453.59237 to the gram) and grains (0.06479891), so 7000 grains
// make a pound exactly and neither unit is the dimension base. Every figure below is chosen so a
// conversion that rounds, or one that never happens, shows up as a wrong answer rather than a near
// one.
const (
	poundUnitID = "un_01seedpound00000000"
	grainUnitID = "un_01seedgrain00000000"
	grainsPerLb = 7000
)

// freshInventoryItem creates a material and returns the item behind it, stocked in pounds. Its own
// item, because these tests move stock and assert on totals.
func freshInventoryItem(t *testing.T) string {
	t.Helper()

	sku := uniqueName("e2e-inv")
	status, body, err := apiClient.Post(materialsPath, validMaterialBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	materialID := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { apiClient.Delete(materialsPath + "/" + materialID) })

	list, _, err := apiClient.GetList(itemsPath, url.Values{"q": {sku}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1, "exactly one item should match the unique SKU %q", sku)
	return DataItemField(list.Data[0], "id")
}

// inventoryPosition reads all four figures from one response.
//
// One request, because they are only consistent with each other within a single read: allocation
// runs behind the request that triggered it, and a reader that fetches on hand and short separately
// can catch the first before the consumer commits and the second after — which reads as stock that
// appeared from nowhere.
type inventoryPosition struct {
	onHand, reserved, short, availableToPromise decimal.Decimal
}

// level is what a reconcile measures its target against: on hand net of demand nothing has covered.
func (p inventoryPosition) level() decimal.Decimal {
	return p.onHand.Sub(p.short)
}

func readInventory(t *testing.T, itemID string) inventoryPosition {
	t.Helper()
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+itemID+"/inventory", url.Values{
		"include": {"on_hand,reserved,available_to_promise,short"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	figure := func(field string) decimal.Decimal {
		raw := jsonField(jsonObject(got, field), "value")
		require.NotEmpty(t, raw, "%s should carry a value", field)
		value, parseErr := decimal.NewFromString(raw)
		require.NoError(t, parseErr, "%s value %q should be a decimal", field, raw)
		return value
	}

	return inventoryPosition{
		onHand:             figure("on_hand"),
		reserved:           figure("reserved"),
		short:              figure("short"),
		availableToPromise: figure("available_to_promise"),
	}
}

func updateInventory(t *testing.T, itemID, value, unitID, operation string) (int, []byte) {
	t.Helper()
	status, body, err := apiClient.Patch(itemsPath+"/"+itemID+"/inventory", map[string]any{
		"quantity":  map[string]any{"value": value, "unit_id": unitID},
		"operation": operation,
	}, newIdempotencyKey())
	require.NoError(t, err)
	return status, body
}

func mustUpdateInventory(t *testing.T, itemID, value, unitID, operation string) {
	t.Helper()
	status, body := updateInventory(t, itemID, value, unitID, operation)
	requireStatus(t, 200, status, body)
}

// seedOpenIssue writes demand nothing has covered, the way an order that outran the shelf would.
func seedOpenIssue(t *testing.T, itemID, value, unitID string) {
	t.Helper()
	db := authDB(t)

	suffix := uuid.New().String()
	quantityID := "qu_" + suffix
	issueID := "ivis_" + suffix

	_, err := db.Exec(
		"INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, ?, ?, NOW(3), NOW(3))",
		quantityID, value, unitID)
	require.NoError(t, err)
	_, err = db.Exec(
		"INSERT INTO inventory_issue (id, account_id, item_id, quantity_id, status_code, issued_at, created_at, updated_at) VALUES (?, ?, ?, ?, 'open', NOW(3), NOW(3), NOW(3))",
		issueID, SeedAccountID, itemID, quantityID)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Exec("DELETE FROM inventory_allocation WHERE inventory_issue_id = ?", issueID)
		db.Exec("DELETE FROM inventory_issue WHERE id = ?", issueID)
		db.Exec("DELETE FROM quantity WHERE id = ?", quantityID)
	})
}

// ledgerRows reads back what the write path actually recorded, in the unit it recorded it in.
func ledgerRows(t *testing.T, itemID, table string) []struct{ Value, UnitID string } {
	t.Helper()
	rows, err := authDB(t).Query(fmt.Sprintf(
		"SELECT q.value, q.unit_id FROM %s l JOIN quantity q ON q.id = l.quantity_id WHERE l.item_id = ? ORDER BY l.created_at", table), itemID)
	require.NoError(t, err)
	defer rows.Close()

	var out []struct{ Value, UnitID string }
	for rows.Next() {
		var r struct{ Value, UnitID string }
		require.NoError(t, rows.Scan(&r.Value, &r.UnitID))
		out = append(out, r)
	}
	require.NoError(t, rows.Err())
	return out
}

// ledgerDump renders what the ledger holds, for a failure message that would otherwise say only that
// two numbers differ.
func ledgerDump(t *testing.T, itemID string) string {
	t.Helper()
	position := readInventory(t, itemID)
	out := fmt.Sprintf("\n  on_hand=%s short=%s reserved=%s", position.onHand, position.short, position.reserved)

	rows, err := authDB(t).Query(`
		SELECT 'receipt', ir.id, ir.status_code, q.value, q.unit_id,
		       COALESCE((SELECT SUM(aq.value) FROM inventory_allocation ia JOIN quantity aq ON aq.id = ia.quantity_id WHERE ia.inventory_receipt_id = ir.id), 0)
		FROM inventory_receipt ir JOIN quantity q ON q.id = ir.quantity_id WHERE ir.item_id = ?
		UNION ALL
		SELECT 'issue', ii.id, ii.status_code, q.value, q.unit_id,
		       COALESCE((SELECT SUM(aq.value) FROM inventory_allocation ia JOIN quantity aq ON aq.id = ia.quantity_id WHERE ia.inventory_issue_id = ii.id), 0)
		FROM inventory_issue ii JOIN quantity q ON q.id = ii.quantity_id WHERE ii.item_id = ?`, itemID, itemID)
	if err != nil {
		return out + "\n  (ledger unreadable: " + err.Error() + ")"
	}
	defer rows.Close()
	for rows.Next() {
		var side, rowID, status, value, unitID, allocated string
		if err := rows.Scan(&side, &rowID, &status, &value, &unitID, &allocated); err != nil {
			return out + "\n  (scan failed: " + err.Error() + ")"
		}
		out += fmt.Sprintf("\n  %s %s status=%s value=%s unit=%s allocated=%s", side, rowID, status, value, unitID, allocated)
	}
	return out
}

// settles waits on work the request handed to a consumer.
func settles(t *testing.T, what string, check func() bool) {
	t.Helper()
	eventually(t, 20*time.Second, 250*time.Millisecond, func() error {
		if check() {
			return nil
		}
		return fmt.Errorf("still waiting for %s", what)
	})
}

// ──────────────────────────────────────────────
// What a quantity means
// ──────────────────────────────────────────────

func TestUpdateItemInventory_AdjustMovesTheLevelByTheAmountSent(t *testing.T) {
	t.Parallel()
	itemID := freshInventoryItem(t)

	mustUpdateInventory(t, itemID, "7.25", poundUnitID, "adjust")

	assert.Equal(t, "7.25", readInventory(t, itemID).level().String(),
		"the level should move by exactly the quantity sent")
}

func TestUpdateItemInventory_ReconcileSetsTheLevel(t *testing.T) {
	t.Parallel()
	itemID := freshInventoryItem(t)

	mustUpdateInventory(t, itemID, "40", poundUnitID, "adjust")
	mustUpdateInventory(t, itemID, "13.5", poundUnitID, "reconcile")

	assert.Equal(t, "13.5", readInventory(t, itemID).level().String())
}

// Reconciling to the figure already reported is a no-op. It is the property that makes the number on
// the screen the number to type back, and every correction that appeared not to take was this
// failing: the target was measured against a figure nobody was shown.
func TestUpdateItemInventory_ReconcileToTheReportedFigureMovesNothing(t *testing.T) {
	t.Parallel()
	itemID := freshInventoryItem(t)
	mustUpdateInventory(t, itemID, "31.75", poundUnitID, "adjust")

	beforePosition := readInventory(t, itemID)
	before, onHandBefore := beforePosition.level(), beforePosition.onHand
	receiptsBefore := len(ledgerRows(t, itemID, "inventory_receipt"))
	issuesBefore := len(ledgerRows(t, itemID, "inventory_issue"))

	mustUpdateInventory(t, itemID, before.String(), poundUnitID, "reconcile")

	assert.Equal(t, before.String(), readInventory(t, itemID).level().String(), "the level should not move")
	assert.Equal(t, onHandBefore.String(), readInventory(t, itemID).onHand.String(),
		"and neither should what is on the shelf")
	assert.Len(t, ledgerRows(t, itemID, "inventory_receipt"), receiptsBefore, "no receipt should be written")
	assert.Len(t, ledgerRows(t, itemID, "inventory_issue"), issuesBefore, "no issue should be written")
}

// The target is the level, not the raw on-hand total. With a shortage standing, the two differ, and
// reconciling to on-hand would quietly move stock by the size of the shortage.
func TestUpdateItemInventory_ReconcileMeasuresAgainstTheLevelNotOnHand(t *testing.T) {
	t.Parallel()
	itemID := freshInventoryItem(t)

	// Demand first, so the arriving stock is what triggers the allocation that draws it down. 130
	// owed against 100 delivered settles at an empty shelf and thirty still short, which is the state
	// this test needs: on hand and the level are not the same number.
	seedOpenIssue(t, itemID, "130", poundUnitID)
	mustUpdateInventory(t, itemID, "100", poundUnitID, "adjust")

	settles(t, "the shelf to be drawn down against the demand", func() bool {
		return readInventory(t, itemID).onHand.IsZero()
	})

	settled := readInventory(t, itemID)
	onHand, short := settled.onHand, settled.short
	require.Equal(t, "0", onHand.String())
	require.Equal(t, "30", short.String(), "the demand the shelf could not cover")
	require.NotEqual(t, onHand.String(), onHand.Sub(short).String(),
		"the test is only meaningful while the two figures differ")

	mustUpdateInventory(t, itemID, onHand.Sub(short).String(), poundUnitID, "reconcile")

	assert.Equal(t, onHand.String(), readInventory(t, itemID).onHand.String(),
		"reconciling to the level should not have moved the shelf")
	assert.Equal(t, short.String(), readInventory(t, itemID).short.String(),
		"nor changed what is short")
}

// ──────────────────────────────────────────────
// Decimals
// ──────────────────────────────────────────────

// A quantity that has been through a binary float is not the quantity that was sent. This is the
// pair that exposed it: a tenth and a fifth are neither of them representable.
func TestUpdateItemInventory_KeepsDecimalsNoFloatCanHold(t *testing.T) {
	t.Parallel()
	itemID := freshInventoryItem(t)

	mustUpdateInventory(t, itemID, "0.1", poundUnitID, "adjust")
	mustUpdateInventory(t, itemID, "0.2", poundUnitID, "adjust")

	assert.Equal(t, "0.3", readInventory(t, itemID).level().String(),
		"a tenth and a fifth make three tenths, not 0.30000000000000004")
}

// The whole sequence that started this: add 20, take 40, add 21 back, expect 1.
func TestUpdateItemInventory_WholeUnitsSurviveARunOfCorrections(t *testing.T) {
	t.Parallel()
	itemID := freshInventoryItem(t)

	mustUpdateInventory(t, itemID, "20", poundUnitID, "reconcile")
	mustUpdateInventory(t, itemID, "-40", poundUnitID, "adjust")
	mustUpdateInventory(t, itemID, "21", poundUnitID, "adjust")

	assert.Equal(t, "1", readInventory(t, itemID).level().String(), "20 - 40 + 21 is one pound, to the digit%s", ledgerDump(t, itemID))
}

// A movement is recorded in the unit it was expressed in. Converting on the way in is what put
// 9071.847400000001 in the ledger, and it is also what left a reconcile measuring against a figure
// in one unit and a target in another.
func TestUpdateItemInventory_RecordsTheQuantityInTheUnitItWasSent(t *testing.T) {
	t.Parallel()
	itemID := freshInventoryItem(t)

	mustUpdateInventory(t, itemID, "20", poundUnitID, "adjust")

	receipts := ledgerRows(t, itemID, "inventory_receipt")
	require.Len(t, receipts, 1)
	assert.Equal(t, poundUnitID, receipts[0].UnitID, "recorded in pounds, the unit it was sent in")
	value, err := decimal.NewFromString(receipts[0].Value)
	require.NoError(t, err)
	assert.Equal(t, "20", value.String(), "and as twenty, not twenty and a tail")
}

// ──────────────────────────────────────────────
// Units that are not the same unit
// ──────────────────────────────────────────────

// Stock arriving in one unit has to cover demand raised in another. Comparing the raw column values
// instead drew 40 grains against an issue for 40 pounds and called the demand met.
func TestUpdateItemInventory_StockInOneUnitCoversDemandInAnother(t *testing.T) {
	t.Parallel()
	itemID := freshInventoryItem(t)
	seedOpenIssue(t, itemID, "2", poundUnitID)

	require.Equal(t, "2", readInventory(t, itemID).short.String())

	// Exactly the two pounds owed, counted out in grains.
	mustUpdateInventory(t, itemID, fmt.Sprint(2*grainsPerLb), grainUnitID, "adjust")

	settles(t, "the consumer to allocate the arriving stock", func() bool {
		return readInventory(t, itemID).short.IsZero()
	})

	assert.Equal(t, "0", readInventory(t, itemID).short.String(),
		"14,000 grains is two pounds and covers the demand exactly")
	assert.Equal(t, "0", readInventory(t, itemID).onHand.String(),
		"and leaves nothing spare on the shelf")
}

// Under-covering must under-cover by the right amount, in the item's own unit.
func TestUpdateItemInventory_PartialCoverAcrossUnitsLeavesTheRightShortfall(t *testing.T) {
	t.Parallel()
	itemID := freshInventoryItem(t)
	seedOpenIssue(t, itemID, "3", poundUnitID)

	mustUpdateInventory(t, itemID, fmt.Sprint(1*grainsPerLb), grainUnitID, "adjust")

	settles(t, "the consumer to allocate the arriving stock", func() bool {
		return readInventory(t, itemID).short.Equal(decimal.RequireFromString("2"))
	})

	assert.Equal(t, "2", readInventory(t, itemID).short.String(),
		"one pound of grain against three pounds owed leaves two short")
}

// A reconcile expressed in one unit is measured against a level held in another.
func TestUpdateItemInventory_ReconcileInAnotherUnitMeasuresAgainstTheSameLevel(t *testing.T) {
	t.Parallel()
	itemID := freshInventoryItem(t)
	mustUpdateInventory(t, itemID, "5", poundUnitID, "adjust")

	// Five pounds is 35,000 grains; reconciling to that figure asks for no change at all.
	mustUpdateInventory(t, itemID, fmt.Sprint(5*grainsPerLb), grainUnitID, "reconcile")

	assert.Equal(t, "5", readInventory(t, itemID).level().String(), "the level should still read five pounds")
	assert.Len(t, ledgerRows(t, itemID, "inventory_issue"), 0, "and nothing should have been taken off it")
}

// ──────────────────────────────────────────────
// Nothing is clamped
// ──────────────────────────────────────────────

// Inventory is the sum of the movements recorded against it. Demand beyond the shelf reads as a
// shortage and drives available below zero; flooring either at zero hides a real shortfall.
func TestUpdateItemInventory_DemandBeyondTheShelfIsNotClampedAway(t *testing.T) {
	t.Parallel()
	itemID := freshInventoryItem(t)

	// Demand first: allocation runs off the arriving stock, so an issue seeded after it would have
	// nothing to trigger the pass that draws it down.
	seedOpenIssue(t, itemID, "25", poundUnitID)
	mustUpdateInventory(t, itemID, "10", poundUnitID, "adjust")

	settles(t, "the shelf to be drawn down against the demand", func() bool {
		return readInventory(t, itemID).onHand.IsZero()
	})

	assert.Equal(t, "15", readInventory(t, itemID).short.String(),
		"ten against twenty-five leaves fifteen short")
	assert.Equal(t, "-15", readInventory(t, itemID).availableToPromise.String(),
		"and fifteen short is fifteen below what can be promised")
}

func TestUpdateItemInventory_AdjustingBelowZeroIsRecorded(t *testing.T) {
	t.Parallel()
	itemID := freshInventoryItem(t)

	mustUpdateInventory(t, itemID, "-4", poundUnitID, "adjust")

	assert.Equal(t, "-4", readInventory(t, itemID).level().String(),
		"taking stock that is not there is a shortage, not a floor at zero")
}

// ──────────────────────────────────────────────
// The response
// ──────────────────────────────────────────────

// The four figures are netted out of the ledger at read time, so they are computed quantities: no
// id, because there is no row behind them to have one.
func TestItemInventory_FiguresAreComputedNotStoredRows(t *testing.T) {
	t.Parallel()
	itemID := freshInventoryItem(t)
	mustUpdateInventory(t, itemID, "3", poundUnitID, "adjust")

	status, body, err := apiClient.GetListRaw(itemsPath+"/"+itemID+"/inventory", url.Values{
		"include": {"on_hand,reserved,available_to_promise,short"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	for _, field := range []string{"on_hand", "reserved", "available_to_promise", "short"} {
		figure := jsonObject(got, field)
		require.NotNil(t, figure, "%s should be present", field)
		assert.Equal(t, "computed_quantity", jsonField(figure, "object"), "%s should be a computed quantity", field)
		assert.NotContains(t, figure, "id", "%s has no row behind it to carry an id", field)
		assert.NotEmpty(t, jsonField(figure, "display_value"), "%s should convey its unit", field)
	}
}

// ──────────────────────────────────────────────
// Refusals
// ──────────────────────────────────────────────

func TestUpdateItemInventory_RejectsAQuantityThatIsNotADecimal(t *testing.T) {
	t.Parallel()
	status, _ := updateInventory(t, SeedItemID, "not-a-number", SeedUnitID, "adjust")
	assert.Equal(t, 400, status)
}

func TestUpdateItemInventory_RejectsAMissingQuantity(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.Patch(itemsPath+"/"+SeedItemID+"/inventory", map[string]any{
		"operation": "adjust",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status)
}

func TestUpdateItemInventory_NotFound(t *testing.T) {
	t.Parallel()
	status, _ := updateInventory(t, "it_01zzzzzzzzzzzzzzzzzzzzzzz", "1", poundUnitID, "adjust")
	assert.Equal(t, 404, status)
}

func TestUpdateItemInventory_UnknownLocationIsNotFound(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.Patch(itemsPath+"/"+SeedItemID+"/inventory", map[string]any{
		"quantity":    map[string]any{"value": "1", "unit_id": SeedUnitID},
		"operation":   "adjust",
		"location_id": "stlo_01zzzzzzzzzzzzzzzzzzzzzz",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}
