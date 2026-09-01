//go:build ledger

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// The lock-order inversion that has survived three patches.
//
// The async allocator goes issues → receipts: FindOpenIssuesForItemPaged FOR UPDATE, then
// FindReceiptsForAllocation FOR UPDATE. The receipt-first flows — stocking a receiving order,
// voiding a shipment, undoing a batch scan — go the other way: they reach FindReceiptsForAllocation
// FOR UPDATE first (their own open-issue read has been non-locking since 3e99b962) and then X-lock
// inventory_issue at CloseFullyAllocatedInventoryIssue with the receipt locks still held. Same two
// tables, opposite order.
//
// The interleaving is forced, not raced: each side takes its first lock, and the test waits for
// InnoDB itself to report the other blocked before letting either proceed.
func TestNoDeadlock_AsyncVsReceiptFirst(t *testing.T) {
	f := newFixture(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	issueID := f.insertIssue(t, "open", "10", f.each, base)
	receiptID := f.insertReceipt(t, "available", "100", f.each, base)

	async := f.actor(t, "allocate-consumer")
	sync := f.actor(t, "receiving-order-stock")
	ctx := context.Background()

	// --- Step 1: each side takes its FIRST lock, and only its first. -------------------------

	// Async: the issue page. On HEAD a FOR UPDATE range scan; after move 1 a primary-key locking read
	// of exactly one row.
	_, err := async.q.FindOpenIssuesForItemPaged(ctx, findOpenIssuesPagedParams(f, 200))
	require.NoError(t, err, "async: page read")
	requireProductionPlan(t, async, f.db)

	// Sync: the receipts, which is where a stocking transaction is by the time it allocates.
	_, err = sync.q.FindReceiptsForAllocation(ctx, sqlc.FindReceiptsForAllocationParams{
		AccountID: f.accountID, ItemID: f.itemID,
	})
	require.NoError(t, err, "sync: receipt read")

	// --- Step 2: each side reaches for what the other holds. --------------------------------

	asyncErr := make(chan error, 1)
	syncErr := make(chan error, 1)

	go func() {
		// The async side's second lock: the receipts the sync side is holding.
		_, err := async.q.FindReceiptsForAllocation(ctx, sqlc.FindReceiptsForAllocationParams{
			AccountID: f.accountID, ItemID: f.itemID,
		})
		asyncErr <- err
	}()
	async.waitUntilBlockedOr(t, f.db, asyncErr)

	go func() {
		// The sync side's second lock: an X lock on the issue, taken while it still holds receipts.
		// This is CloseFullyAllocatedInventoryIssue, the statement 3e99b962 left untouched.
		syncErr <- sync.q.CloseFullyAllocatedInventoryIssue(ctx, issueID)
	}()

	// --- Step 3: whichever way InnoDB breaks it, neither side may see 1213. -----------------

	assertNeitherDeadlocked(t, asyncErr, syncErr,
		"the ABBA inversion between the async allocator (issues → receipts) and the receipt-first sync "+
			"flows (receipts → CloseFullyAllocatedInventoryIssue) is live: receipt "+receiptID+", issue "+issueID)

	async.commit(t)
	sync.commit(t)
}

// Cycle 2: the paged scan walks the open set in created_at order; every WHERE id IN (...) writer
// walks the same rows in clustered PK order, and primary keys are 12-char nanoids
// (shared/id/utils.go), so the two orders are uncorrelated. With two overlapping rows the orders
// invert about half the time — which is why this test names the ids and forces the bad direction
// rather than hoping for it.
func TestNoDeadlock_ScanVersusPKOrderedRestore(t *testing.T) {
	f := newFixture(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)

	// created_at order is deliberately the reverse of id order: `late` is read first by the scan,
	// `early` is read first by the PK-ordered writer.
	late := f.insertIssueWithID(t, f.itemID+"_zzzz", "open", "10", f.each, base.Add(time.Second))
	early := f.insertIssueWithID(t, f.itemID+"_aaaa", "open", "10", f.each, base.Add(2*time.Second))
	f.insertReceipt(t, "available", "100", f.each, base)

	scanner := f.actor(t, "discovery")      // reaches late, then early (created_at)
	restorer := f.actor(t, "shipment-void") // reaches early, then late (id IN (...))
	ctx := context.Background()

	_, err := scanner.q.FindOpenIssuesForItemPaged(ctx, findOpenIssuesPagedParams(f, 1)) // locks `late` only
	require.NoError(t, err)
	requireProductionPlan(t, scanner, f.db)
	require.NoError(t, restorer.q.RestoreIssuesToReserved(ctx, []string{early})) // locks `early` only

	scanErr, restoreErr := make(chan error, 1), make(chan error, 1)
	go func() {
		_, err := scanner.q.FindOpenIssuesForItemPaged(ctx,
			findOpenIssuesPagedParamsAfter(f, base.Add(time.Second), late, 1))
		scanErr <- err
	}()
	scanner.waitUntilBlockedOr(t, f.db, scanErr)
	go func() { restoreErr <- restorer.q.RestoreIssuesToReserved(ctx, []string{late}) }()

	assertNeitherDeadlocked(t, scanErr, restoreErr,
		"the created_at-ordered page scan and a PK-ordered RestoreIssuesToReserved deadlocked over the "+
			"same two inventory_issue rows")
}

// Cycle 3: the clustered/secondary inversion on a single row, with no gap and no second row involved.
//
// A locking range read over inventory_issue_open_paging_idx takes the secondary-index entry and then
// the clustered row; UPDATE ... WHERE id = ? takes the clustered row and then maintains the secondary
// entry, because migration 00004 put status_code — the column every status writer changes — into the
// index the scan locks. One shared row is a cycle.
func TestNoDeadlock_ScanVersusSingleRowStatusUpdate(t *testing.T) {
	f := newFixture(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	shared := f.insertIssue(t, "open", "10", f.each, base)
	other := f.insertIssue(t, "open", "10", f.each, base.Add(time.Second))
	f.insertReceipt(t, "available", "100", f.each, base)

	scanner := f.actor(t, "discovery")
	closer := f.actor(t, "sync-allocator")
	ctx := context.Background()

	// The scanner holds the first row; the closer holds the second.
	_, err := scanner.q.FindOpenIssuesForItemPaged(ctx, findOpenIssuesPagedParams(f, 1))
	require.NoError(t, err)
	requireProductionPlan(t, scanner, f.db)
	require.NoError(t, closer.q.CloseFullyAllocatedInventoryIssue(ctx, other))

	scanErr, closeErr := make(chan error, 1), make(chan error, 1)
	go func() {
		_, err := scanner.q.FindOpenIssuesForItemPaged(ctx,
			findOpenIssuesPagedParamsAfter(f, base, shared, 1))
		scanErr <- err
	}()
	scanner.waitUntilBlockedOr(t, f.db, scanErr)
	go func() { closeErr <- closer.q.CloseFullyAllocatedInventoryIssue(ctx, shared) }()

	assertNeitherDeadlocked(t, scanErr, closeErr,
		"the paged scan and CloseFullyAllocatedInventoryIssue deadlocked reaching the clustered row and "+
			"its inventory_issue_open_paging_idx entry in opposite orders")
}
