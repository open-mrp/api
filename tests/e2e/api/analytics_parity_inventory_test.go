//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Inventory receipts and material analytics on yarn, stocked in pounds and grains (7000 gr to the pound), with
// the item's base unit the pound. Every figure is chosen so a conversion that never happens reads as a wrong
// answer: 7000 gr counted as 7000 lb, or priced at $2 each.

// parityYarn creates a material costing $2/lb, optionally with an order point, and returns its id and item id.
func parityYarn(t *testing.T, orderPoint map[string]any) (materialID, itemID string) {
	t.Helper()
	sku := uniqueName("e2e-parity-yarn")
	body := validMaterialBody(sku)
	body["unit_cost"] = map[string]any{"value": "2.00", "numerator_unit_id": dollarUnitID, "denominator_unit_id": poundUnitID}
	body["lead_time"] = map[string]any{"value": "3", "unit_id": "day"}
	if orderPoint != nil {
		body["order_point"] = orderPoint
	}
	status, raw, err := apiClient.Post(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	materialID = jsonField(parseJSON(raw), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(materialsPath + "/" + materialID) })

	list, _, err := apiClient.GetList(itemsPath, url.Values{"q": {sku}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	return materialID, DataItemField(list.Data[0], "id")
}

func receiveInto(t *testing.T, itemID, value, unitID, locationID, lot string) {
	t.Helper()
	body := map[string]any{"quantity": map[string]any{"value": value, "unit_id": unitID}, "operation": "adjust"}
	if locationID != "" {
		body["location_id"] = locationID
	}
	if lot != "" {
		body["lot_number"] = lot
	}
	status, raw, err := apiClient.Patch(itemsPath+"/"+itemID+"/inventory", body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
}

func receiptSummaries(t *testing.T, client *Client, body map[string]any) []map[string]any {
	t.Helper()
	return listRows(t, mustPutAnalytics(t, client, inventoryReceiptsPath, nil, body))
}

// awaitRemaining polls until the item's one summary row reports remaining: allocation runs behind the
// request that draws stock.
func awaitRemaining(t *testing.T, itemID, remaining string) map[string]any {
	t.Helper()
	var row map[string]any
	eventually(t, 30*time.Second, 300*time.Millisecond, func() error {
		rows := receiptSummaries(t, apiClient, map[string]any{"item_ids": []string{itemID}})
		if len(rows) != 1 {
			return fmt.Errorf("want one row, got %d", len(rows))
		}
		row = rows[0]
		if got := jsonField(jsonObject(row, "remaining_quantity"), "value"); got != remaining {
			return fmt.Errorf("remaining %s, want %s", got, remaining)
		}
		return nil
	})
	return row
}

func assertMeasure(t *testing.T, obj map[string]any, field, value, display, unitAbbr, unitType string) {
	t.Helper()
	q := jsonObject(obj, field)
	require.NotNil(t, q, "%s: %v", field, obj)
	assert.Equal(t, value, jsonField(q, "value"), field)
	assert.Equal(t, display, jsonField(q, "display_value"), field)
	unit := jsonObject(q, "unit")
	require.NotNil(t, unit, field)
	assert.Equal(t, unitAbbr, jsonField(unit, "abbreviation"), field)
	assert.Equal(t, unitType, jsonField(unit, "type"), field)
}

// allocateAgainst draws value of unitID against the item's only available receipt, the way a fulfilled issue
// would. There is no API to allocate a receipt to nothing.
func allocateAgainst(t *testing.T, itemID, value, unitID string) {
	t.Helper()
	db := authDB(t)
	var receiptID string
	require.NoError(t, db.QueryRow("SELECT id FROM inventory_receipt WHERE item_id = ? AND status_code = 'available'", itemID).Scan(&receiptID))
	s := uuid.New().String()
	ids := map[string]string{"issueQty": "qu_" + s + "a", "allocQty": "qu_" + s + "b", "unitCost": "rt_" + s + "c", "totalCost": "rt_" + s + "d", "issue": "ivis_" + s, "alloc": "ivac_" + s}
	exec := func(q string, args ...any) {
		_, err := db.Exec(q, args...)
		require.NoError(t, err)
	}
	exec("INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, ?, ?, NOW(3), NOW(3))", ids["issueQty"], value, unitID)
	exec("INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, ?, ?, NOW(3), NOW(3))", ids["allocQty"], value, unitID)
	exec("INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES (?, 0, ?, ?, NOW(3), NOW(3))", ids["unitCost"], dollarUnitID, unitID)
	exec("INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES (?, 0, ?, ?, NOW(3), NOW(3))", ids["totalCost"], dollarUnitID, unitID)
	exec("INSERT INTO inventory_issue (id, account_id, item_id, quantity_id, status_code, issued_at, created_at, updated_at) VALUES (?, ?, ?, ?, 'fulfilled', NOW(3), NOW(3), NOW(3))",
		ids["issue"], SeedAccountID, itemID, ids["issueQty"])
	exec("INSERT INTO inventory_allocation (id, inventory_receipt_id, inventory_issue_id, quantity_id, unit_cost_id, total_cost_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, NOW(3), NOW(3))",
		ids["alloc"], receiptID, ids["issue"], ids["allocQty"], ids["unitCost"], ids["totalCost"])
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM inventory_allocation WHERE id = ?", ids["alloc"])
		_, _ = db.Exec("DELETE FROM inventory_issue WHERE id = ?", ids["issue"])
		_, _ = db.Exec("DELETE FROM rate WHERE id IN (?, ?)", ids["unitCost"], ids["totalCost"])
		_, _ = db.Exec("DELETE FROM quantity WHERE id IN (?, ?)", ids["issueQty"], ids["allocQty"])
	})
}

// --- Inventory receipts ---

func TestAnalyticsParityInventoryReceipts_GroupsAcrossUnitsInTheBaseUnit(t *testing.T) {
	t.Parallel()
	_, item := parityYarn(t, nil)
	receiveInto(t, item, "2", poundUnitID, "", "")
	receiveInto(t, item, "7000", grainUnitID, "", "") // one more pound
	receiveInto(t, item, "-0.5", poundUnitID, "", "") // drawn against them

	// 2 lb + 1 lb - 0.5 lb, one row whatever unit each receipt was recorded in.
	row := awaitRemaining(t, item, "2.5")
	assert.Equal(t, item, jsonField(jsonObject(row, "item"), "id"))
	assert.Equal(t, "item", jsonField(jsonObject(row, "item"), "object"))
	assertNilField(t, row, "location")
	assertNilField(t, row, "lot")
	assert.Equal(t, SeedAccountID, jsonField(jsonObject(row, "owner_account"), "id"))
	assert.Equal(t, SeedAccountID, jsonField(jsonObject(row, "holder_account"), "id"))
	assertMeasure(t, row, "remaining_quantity", "2.5", "2.5 lbs", "lbs", "mass")

	// $2 a pound, the grains included: priced per pound, not per grain.
	cost := jsonObject(row, "weighted_average_unit_cost")
	require.NotNil(t, cost)
	assertMeasure(t, cost, "numerator", "2.00", "$2.00", "$", "currency")
	assertMeasure(t, cost, "denominator", "1", "1 lbs", "lbs", "mass")
	assertMeasure(t, row, "inventory_value", "5.00", "$5.00", "$", "currency")
	assertValidTimestamp(t, jsonField(row, "oldest_receipt_at"), "oldest_receipt_at")
	assertValidTimestamp(t, jsonField(row, "newest_receipt_at"), "newest_receipt_at")
	assert.Less(t, jsonField(row, "oldest_receipt_at"), jsonField(row, "newest_receipt_at"))
}

func TestAnalyticsParityInventoryReceipts_NothingLeftHasNoValue(t *testing.T) {
	t.Parallel()
	_, item := parityYarn(t, nil)
	receiveInto(t, item, "1", poundUnitID, "", "")
	allocateAgainst(t, item, "7000", grainUnitID) // the whole pound, in grains

	rows := receiptSummaries(t, apiClient, map[string]any{"item_ids": []string{item}})
	require.Len(t, rows, 1, "an available receipt with nothing left is still listed")
	assertMeasure(t, rows[0], "remaining_quantity", "0", "0 lbs", "lbs", "mass")
	assertNilField(t, rows[0], "inventory_value")
	assertMeasure(t, jsonObject(rows[0], "weighted_average_unit_cost"), "numerator", "0.00", "$0.00", "$", "currency")
}

func TestAnalyticsParityInventoryReceipts_OldestFirstAndFilteredInSQL(t *testing.T) {
	t.Parallel()
	_, older := parityYarn(t, nil)
	_, newer := parityYarn(t, nil)
	lot := uniqueName("LOT-PARITY")
	receiveInto(t, older, "1", poundUnitID, SeedLocationID, lot)
	time.Sleep(10 * time.Millisecond)
	receiveInto(t, newer, "3", poundUnitID, "", "")

	rows := receiptSummaries(t, apiClient, map[string]any{"item_ids": []string{newer, older}})
	require.Len(t, rows, 2)
	assert.Equal(t, older, jsonField(jsonObject(rows[0], "item"), "id"), "groups are listed oldest receipt first")
	assert.Equal(t, newer, jsonField(jsonObject(rows[1], "item"), "id"))
	location := jsonObject(rows[0], "location")
	require.NotNil(t, location)
	assert.Equal(t, SeedLocationID, jsonField(location, "id"))
	lotObj := jsonObject(rows[0], "lot")
	require.NotNil(t, lotObj)
	assert.Equal(t, "lot", jsonField(lotObj, "object"))
	assert.Equal(t, lot, jsonField(lotObj, "number"))
	lotID := jsonField(lotObj, "id")

	both := []string{newer, older}
	for name, body := range map[string]map[string]any{
		"location": {"item_ids": both, "location_ids": []string{SeedLocationID}},
		"lot":      {"item_ids": both, "lot_ids": []string{lotID}},
	} {
		rows := receiptSummaries(t, apiClient, body)
		require.Len(t, rows, 1, name)
		assert.Equal(t, older, jsonField(jsonObject(rows[0], "item"), "id"), name)
	}
	assert.Empty(t, receiptSummaries(t, apiClient, map[string]any{"item_ids": []string{newer}, "lot_ids": []string{lotID}}), "filters combine with AND")
}

// --- Materials ---

func materialRow(t *testing.T, materialID string) map[string]any {
	t.Helper()
	for _, row := range listRows(t, mustPutAnalytics(t, apiClient, analyticsMaterialsPath, nil, nil)) {
		if jsonField(row, "id") == materialID {
			return row
		}
	}
	t.Fatalf("material %s is not in the analytics", materialID)
	return nil
}

func TestAnalyticsParityMaterials_StockAndDemandInTheOrderPointUnit(t *testing.T) {
	t.Parallel()
	stocked, stockedItem := parityYarn(t, map[string]any{"value": "14000", "unit_id": grainUnitID})
	receiveInto(t, stockedItem, "2", poundUnitID, "", "")
	receiveInto(t, stockedItem, "7000", grainUnitID, "", "")
	receiveInto(t, stockedItem, "-0.5", poundUnitID, "", "")
	awaitRemaining(t, stockedItem, "2.5")

	row := materialRow(t, stocked)
	assert.Equal(t, stockedItem, jsonField(row, "item_id"))
	// 2 lb + 7000 gr - 0.5 lb = 2.5 lb, stated in the order point's grains.
	assertMeasure(t, row, "quantity_in_inventory", "17500", "17,500 gr", "gr", "mass")
	assertMeasure(t, row, "quantity_in_demand", "0", "0 gr", "gr", "mass")
	assertMeasure(t, row, "order_point", "14000", "14,000 gr", "gr", "mass")
	assertMeasure(t, row, "lead_time", "3", "3 day", "day", "time")

	// Demand with no stock to cover it: nothing can allocate, so it stays open, and stock goes negative rather than clamping at zero.
	short, shortItem := parityYarn(t, map[string]any{"value": "14000", "unit_id": grainUnitID})
	seedOpenIssue(t, shortItem, "0.25", poundUnitID)
	row = materialRow(t, short)
	assertMeasure(t, row, "quantity_in_inventory", "-1750", "-1,750 gr", "gr", "mass")
	assertMeasure(t, row, "quantity_in_demand", "1750", "1,750 gr", "gr", "mass")
}

// A material whose order-point row is gone is still listed, its stock in the item's base unit.
func TestAnalyticsParityMaterials_KeepsAMaterialWithoutAnOrderPoint(t *testing.T) {
	t.Parallel()
	material, item := parityYarn(t, map[string]any{"value": "1", "unit_id": poundUnitID})
	receiveInto(t, item, "7000", grainUnitID, "", "")

	db := authDB(t)
	var orderPointID string
	require.NoError(t, db.QueryRow("SELECT order_point_id FROM material WHERE id = ?", material).Scan(&orderPointID))
	_, err := db.Exec("DELETE FROM quantity WHERE id = ?", orderPointID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec("INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, 1, ?, NOW(3), NOW(3))", orderPointID, poundUnitID)
	})

	row := materialRow(t, material)
	assertNilField(t, row, "order_point")
	assertMeasure(t, row, "quantity_in_inventory", "1", "1 lbs", "lbs", "mass")
	assertMeasure(t, row, "quantity_in_demand", "0", "0 lbs", "lbs", "mass")
}
