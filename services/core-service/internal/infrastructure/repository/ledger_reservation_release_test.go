//go:build ledger

package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/core-service/internal/ledgerlock"
)

// Releasing an order's reservations hands the stock back, all the way back to allocatable.
//
// Unissuing used to delete the allocations covering an order's reserved issues and then the issues,
// and stop there. A receipt an allocation had closed out stayed at `allocated`, and
// FindReceiptsForAllocation considers nothing else — so stock physically on the shelf, and now
// unspoken for, was invisible to every later allocation: a shortage with no bad row to find, on a
// path a user can take twice a day. The shipment-void path had always done this correctly through
// FreeReleasedReceipts; the four order paths never learned to.
//
// A reserved issue carrying allocations is not a hypothetical state: dashboard/apps/api's
// allocateReservationsByInvoice writes exactly that pair, and the Go allocator's own
// FindReservedIssuesWithAllocationSums exists to subtract what it wrote.
func TestReleaseOrderReservations_FreesTheReceiptsItReleases(t *testing.T) {
	f := newFixture(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	orderID := f.nextID("or")

	// A receipt fully drawn by this order's reservation, exactly as the dashboard leaves it: the
	// allocation written straight in, the receipt flipped to `allocated` because nothing is left.
	receipt := f.insertReceipt(t, "available", "10", f.each, base)
	reserved := f.insertReservedIssueForOrder(t, orderID, "10", f.each, base)
	f.writeRawAllocation(t, reserved, receipt, "10", f.each)
	f.setReceiptStatus(t, receipt, "allocated")

	// Open demand for the same item, from another order, which the freed stock should be able to cover.
	openIssue := f.insertIssue(t, "open", "10", f.each, base.Add(time.Minute))

	releaser := f.actor(t, "unissue")
	released := releaseOrderReservations(t, releaser, f, orderID)
	releaser.commit(t)

	require.Equal(t, []string{f.itemID}, released,
		"the release must name the items it touched: they are what the caller enqueues allocation for")
	require.Empty(t, f.allocationIDsForIssue(t, reserved),
		"the reservation's allocations should be gone: that is the part the release already did")
	require.Zero(t, f.reservedIssueCount(t, orderID),
		"the reserved issues should be gone: that is the other part it already did")

	require.Equal(t, "available", f.receiptStatus(t, receipt),
		"receipt %s was drawn to zero by an allocation that has just been deleted, so it holds its full "+
			"quantity again and must be allocatable. Left at `allocated` it is real stock on the shelf "+
			"that no allocation can ever see again — a shortage with no bad row to find.", receipt)

	candidates, err := f.allocationCandidates(t)
	require.NoError(t, err)
	require.Len(t, candidates, 1,
		"the released receipt must be a candidate for the item's other open demand; "+
			"FindReceiptsForAllocation matches status_code = 'available' and sees nothing otherwise")

	// And the allocator can actually cover the waiting demand from it, which is the outcome the whole
	// release exists to produce.
	next := f.actor(t, "allocate")
	require.Nil(t, next.repo.AllocateOneOpenIssue(context.Background(), next.scope(t, f),
		f.accountID, f.itemID, openIssue), "covering the open issue from the released receipt")
	next.commit(t)

	covered := f.baseAllocatedForIssue(t, openIssue)
	require.True(t, covered.Equal(decimal.NewFromInt(10)),
		"the open issue should now be covered by the released receipt's 10, got %s", covered.String())
	assertReceiptNotOverDrawn(t, f, receipt)
}

// The satellites go with the allocation, or they are unreachable forever.
//
// An allocation owns its quantity, its unit-cost rate and its total-cost quantity, and they are
// reachable only through it. The statement this replaces deleted the allocation row alone, so every
// release leaked three rows per allocation into tables nothing would ever join them from again.
func TestReleaseOrderReservations_LeavesNoOrphanedSatellites(t *testing.T) {
	f := newFixture(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	orderID := f.nextID("or")

	receipt := f.insertReceipt(t, "available", "10", f.each, base)
	reserved := f.insertReservedIssueForOrder(t, orderID, "10", f.each, base)
	allocation := f.writeRawAllocation(t, reserved, receipt, "10", f.each)
	satellites := f.allocationSatellites(t, allocation)
	issueQuantityID := f.issueQuantityID(t, reserved)

	releaser := f.actor(t, "unissue")
	releaseOrderReservations(t, releaser, f, orderID)
	releaser.commit(t)

	for _, quantityID := range satellites.quantityIDs {
		require.False(t, f.quantityExists(t, quantityID),
			"quantity %s belonged to a deleted allocation and is now unreachable", quantityID)
	}
	require.False(t, f.rateExists(t, satellites.rateID),
		"rate %s was the deleted allocation's unit cost and is now unreachable", satellites.rateID)
	require.False(t, f.quantityExists(t, issueQuantityID),
		"quantity %s belonged to a deleted inventory issue and is now unreachable", issueQuantityID)
}

// Deleting the order was the same defect with a worse ending, and the reason the release runs before
// the cascade rather than inside it.
//
// DeleteCascade deleted the order's reserved issues and never touched their allocations at all, so
// the allocation rows survived pointing at an issue id that no longer existed. Every reader that
// weighs a receipt sums by inventory_receipt_id and does not care whether the issue is still there,
// so those rows held the stock down permanently: no issue left to release them from, and no path that
// would ever delete them.
func TestReleaseOrderReservations_OnDeleteLeavesNoOrphanAllocations(t *testing.T) {
	f := newFixture(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	orderID := f.nextID("or")

	receipt := f.insertReceipt(t, "available", "10", f.each, base)
	reserved := f.insertReservedIssueForOrder(t, orderID, "10", f.each, base)
	f.writeRawAllocation(t, reserved, receipt, "10", f.each)
	f.setReceiptStatus(t, receipt, "allocated")

	// The order delete releases the reservations before its cascade runs, for exactly this reason.
	deleter := f.actor(t, "delete-order")
	releaseOrderReservations(t, deleter, f, orderID)
	deleter.commit(t)

	require.Empty(t, f.allocationIDsForIssue(t, reserved),
		"allocation rows survived the deletion of the issue they belong to: nothing can release them now, "+
			"and they hold receipt %s down for good", receipt)
	require.Equal(t, "available", f.receiptStatus(t, receipt),
		"receipt %s must be allocatable again once the order holding it is gone", receipt)
}

// releaseOrderReservations drives the release the way every caller of it does: the item set resolved
// before the transaction, its ordering root taken as the transaction's first statement.
func releaseOrderReservations(t *testing.T, a *actor, f *fixture, orderID string) []string {
	t.Helper()
	ctx := context.Background()

	itemIDs, apiErr := a.repo.ListReservedItemIDsForOrders(ctx, f.accountID, []string{orderID})
	require.Nil(t, apiErr)

	scope, apiErr := ledgerlock.Acquire(ctx, a.repo, itemIDs)
	require.Nil(t, apiErr)

	released, apiErr := a.repo.ReleaseReservedIssuesForOrder(ctx, scope, f.accountID, orderID)
	require.Nil(t, apiErr)
	return released
}

// allocationCandidates asks FindReceiptsForAllocation what the item has to draw on, on a connection
// of its own so it is answering about committed state.
func (f *fixture) allocationCandidates(t *testing.T) ([]sqlc.FindReceiptsForAllocationRow, error) {
	t.Helper()
	return sqlc.New(f.db).FindReceiptsForAllocation(context.Background(), sqlc.FindReceiptsForAllocationParams{
		AccountID: f.accountID,
		ItemID:    f.itemID,
	})
}

// insertReservedIssueForOrder writes the reservation an issued sales order creates: an inventory
// issue at `reserved`, tagged with the order it was created for.
func (f *fixture) insertReservedIssueForOrder(t *testing.T, orderID, value, unitID string, createdAt time.Time) string {
	t.Helper()
	qID, iID := f.nextID("qy"), f.nextID("ivis")
	_, err := f.db.Exec(
		`INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		qID, value, unitID, createdAt, createdAt)
	require.NoError(t, err)
	_, err = f.db.Exec(
		`INSERT INTO inventory_issue (id, account_id, item_id, status_code, quantity_id, order_id, created_at, updated_at)
		 VALUES (?, ?, ?, 'reserved', ?, ?, ?, ?)`,
		iID, f.accountID, f.itemID, qID, orderID, createdAt, createdAt)
	require.NoError(t, err)
	return iID
}

func (f *fixture) setReceiptStatus(t *testing.T, receiptID, status string) {
	t.Helper()
	_, err := f.db.Exec(`UPDATE inventory_receipt SET status_code = ? WHERE id = ?`, status, receiptID)
	require.NoError(t, err)
}

func (f *fixture) receiptStatus(t *testing.T, receiptID string) string {
	t.Helper()
	var status string
	require.NoError(t, f.db.QueryRow(
		`SELECT status_code FROM inventory_receipt WHERE id = ?`, receiptID).Scan(&status))
	return status
}

func (f *fixture) reservedIssueCount(t *testing.T, orderID string) int {
	t.Helper()
	var n int
	require.NoError(t, f.db.QueryRow(
		`SELECT COUNT(*) FROM inventory_issue WHERE order_id = ? AND status_code = 'reserved'`,
		orderID).Scan(&n))
	return n
}

func (f *fixture) allocationIDsForIssue(t *testing.T, issueID string) []string {
	t.Helper()
	rows, err := f.db.Query(`SELECT id FROM inventory_allocation WHERE inventory_issue_id = ?`, issueID)
	require.NoError(t, err)
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		out = append(out, id)
	}
	require.NoError(t, rows.Err())
	return out
}

type allocationSatelliteIDs struct {
	quantityIDs []string
	rateID      string
}

func (f *fixture) allocationSatellites(t *testing.T, allocationID string) allocationSatelliteIDs {
	t.Helper()
	var quantityID, unitCostID, totalCostID string
	require.NoError(t, f.db.QueryRow(
		`SELECT quantity_id, unit_cost_id, total_cost_id FROM inventory_allocation WHERE id = ?`,
		allocationID).Scan(&quantityID, &unitCostID, &totalCostID))
	return allocationSatelliteIDs{quantityIDs: []string{quantityID, totalCostID}, rateID: unitCostID}
}

func (f *fixture) issueQuantityID(t *testing.T, issueID string) string {
	t.Helper()
	var quantityID string
	require.NoError(t, f.db.QueryRow(
		`SELECT quantity_id FROM inventory_issue WHERE id = ?`, issueID).Scan(&quantityID))
	return quantityID
}

func (f *fixture) quantityExists(t *testing.T, quantityID string) bool {
	t.Helper()
	return f.rowExists(t, `SELECT 1 FROM quantity WHERE id = ?`, quantityID)
}

func (f *fixture) rateExists(t *testing.T, rateID string) bool {
	t.Helper()
	return f.rowExists(t, `SELECT 1 FROM rate WHERE id = ?`, rateID)
}

func (f *fixture) rowExists(t *testing.T, stmt, id string) bool {
	t.Helper()
	var one int
	err := f.db.QueryRow(stmt, id).Scan(&one)
	if err == sql.ErrNoRows {
		return false
	}
	require.NoError(t, err)
	return true
}
