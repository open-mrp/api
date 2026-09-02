//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The seed item is stocked in pairs (its unit_value rate is denominated in
// un_01seedpair000000000, ratio 2:1) while allocations are written in the unit of the receipt they
// draw from, which for stock allocated by the dashboard is the group's base unit — each. Subtracting
// those columns from one another without converting says an issue of 5 pair covered by 10 each is
// short by -5, when the two sides cancel exactly.
const (
	shortSignReceiptID    = "ivrc_01e2eshortsign00000"
	shortSignIssueID      = "ivis_01e2eshortsign00000"
	shortSignAllocationID = "ivac_01e2eshortsign00000"
	shortSignIssueQtyID   = "qu_01e2eshortsignissue"
	shortSignReceiptQtyID = "qu_01e2eshortsignrecpt"
	shortSignAllocQtyID   = "qu_01e2eshortsignalloc"
	shortSignAllocCostID  = "qu_01e2eshortsigncost0"
	shortSignReceiptRate  = "rt_01e2eshortsignrecpt"
	shortSignAllocRate    = "rt_01e2eshortsignalloc"
)

// seedCrossUnitAllocatedIssue writes an open issue recorded in pairs alongside the allocation that
// covers it recorded in each, the shape the dashboard's allocator produces.
func seedCrossUnitAllocatedIssue(t *testing.T) {
	t.Helper()
	db := authDB(t)

	statements := []struct {
		query string
		args  []any
	}{
		{"INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, ?, ?, NOW(3), NOW(3))",
			[]any{shortSignIssueQtyID, "5", SeedUnitID}},
		{"INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, ?, ?, NOW(3), NOW(3))",
			[]any{shortSignReceiptQtyID, "10", SeedSystemUnitID}},
		{"INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, ?, ?, NOW(3), NOW(3))",
			[]any{shortSignAllocQtyID, "10", SeedSystemUnitID}},
		{"INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, ?, ?, NOW(3), NOW(3))",
			[]any{shortSignAllocCostID, "0", SeedSystemUnitID}},
		{"INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES (?, ?, 'dollar', ?, NOW(3), NOW(3))",
			[]any{shortSignReceiptRate, "0", SeedSystemUnitID}},
		{"INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES (?, ?, 'dollar', ?, NOW(3), NOW(3))",
			[]any{shortSignAllocRate, "0", SeedSystemUnitID}},
		// Allocated, so the receipt itself is not free stock — the test is about the issue.
		{"INSERT INTO inventory_receipt (id, owner_account_id, holder_account_id, item_id, quantity_id, unit_cost_id, status_code, received_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, 'allocated', NOW(3), NOW(3), NOW(3))",
			[]any{shortSignReceiptID, SeedAccountID, SeedAccountID, SeedItemID, shortSignReceiptQtyID, shortSignReceiptRate}},
		{"INSERT INTO inventory_issue (id, account_id, item_id, quantity_id, status_code, issued_at, created_at, updated_at) VALUES (?, ?, ?, ?, 'open', NOW(3), NOW(3), NOW(3))",
			[]any{shortSignIssueID, SeedAccountID, SeedItemID, shortSignIssueQtyID}},
		{"INSERT INTO inventory_allocation (id, inventory_receipt_id, inventory_issue_id, quantity_id, unit_cost_id, total_cost_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, NOW(3), NOW(3))",
			[]any{shortSignAllocationID, shortSignReceiptID, shortSignIssueID, shortSignAllocQtyID, shortSignAllocRate, shortSignAllocCostID}},
	}

	for _, statement := range statements {
		_, err := db.Exec(statement.query, statement.args...)
		require.NoError(t, err, "seeding over-allocated issue: %s", statement.query)
	}

	t.Cleanup(func() {
		for _, statement := range []struct {
			query string
			args  []any
		}{
			{"DELETE FROM inventory_allocation WHERE id = ?", []any{shortSignAllocationID}},
			{"DELETE FROM inventory_issue WHERE id = ?", []any{shortSignIssueID}},
			{"DELETE FROM inventory_receipt WHERE id = ?", []any{shortSignReceiptID}},
			{"DELETE FROM rate WHERE id IN (?, ?)", []any{shortSignReceiptRate, shortSignAllocRate}},
			{"DELETE FROM quantity WHERE id IN (?, ?, ?, ?)", []any{
				shortSignIssueQtyID, shortSignReceiptQtyID, shortSignAllocQtyID, shortSignAllocCostID,
			}},
		} {
			_, _ = db.Exec(statement.query, statement.args...)
		}
	})
}

// inventoryMeasure reads one figure off the item inventory endpoint as a number.
func inventoryMeasure(t *testing.T, itemID, field string) float64 {
	t.Helper()
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+itemID+"/inventory", url.Values{
		"include": {"on_hand,reserved,available_to_promise,short"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	raw := jsonField(jsonObject(parseJSON(body), field), "value")
	require.NotEmpty(t, raw, "%s should carry a value", field)
	measure, parseErr := strconv.ParseFloat(raw, 64)
	require.NoError(t, parseErr, "%s value %q should be numeric", field, raw)
	return measure
}

// A shortage is demand no receipt has covered, and an allocation recorded in a different unit of the
// same group covers exactly what it says once both sides are normalized through their ratios.
// Subtracting the raw columns instead reports the covered issue as short by -5.
func TestItemInventory_ShortNetsAllocationsAcrossUnits(t *testing.T) {
	seedCrossUnitAllocatedIssue(t)

	// 10 each is exactly the 5 pair the issue asked for, so the demand is covered and nothing is
	// short. Reading -5 here is the raw subtraction of two different units.
	assert.Equal(t, 0.0, inventoryMeasure(t, SeedItemID, "short"),
		"an issue covered in another unit of the same group is not short")
}
