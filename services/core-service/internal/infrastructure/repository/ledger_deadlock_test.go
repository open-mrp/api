//go:build ledger

package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// The lock-order inversion that has survived every fix so far, in the shape it still has.
//
// The allocator goes issue then receipts: it claims the issue by primary key, then takes
// FindReceiptsForAllocation FOR UPDATE. The reversal flows go the other way — ReverseInventoryForOrderItem
// reads the issues, deletes their allocations, frees the receipts those released
// (FreeReleasedReceipts), and only then writes the issues back with RestoreIssuesToReserved. Receipts
// written before issues, against issues held before receipts. Same two tables, opposite order.
//
// The pairing used to be the allocator against the INLINE sync allocator, which reached
// FindReceiptsForAllocation and then X-locked inventory_issue at CloseFullyAllocatedInventoryIssue.
// That flow is gone: stocking, voiding and undo now enqueue, and CloseFullyAllocatedInventoryIssue is
// reached only from the allocator itself, which holds the issue first. What is left is the reversal,
// which has to write both tables and cannot be reordered into agreement with the allocator — a
// reversal must free the receipts before it can restore the demand, and an allocator must hold the
// demand before it can decide which receipts to draw.
//
// No amount of reordering inside either flow fixes this one, which is the whole argument for the
// ordering root: both sides now take inventory_item_lock for the item as their first statement, so the
// second simply waits for the first instead of meeting it head-on.
//
// The interleaving is still forced rather than raced. The contender is driven to the point where the
// server reports it blocked, and only then is the holder allowed to commit — so this passes because
// the root serialises them, not because the test was too quick to catch the overlap.
func TestNoDeadlock_ReversalVersusAllocator(t *testing.T) {
	f := newFixture(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	issueID := f.insertIssue(t, "open", "10", f.each, base)
	receiptID := f.insertReceipt(t, "available", "100", f.each, base)

	async := f.actor(t, "allocate-consumer")
	reversal := f.actor(t, "shipment-void-reversal")
	ctx := context.Background()

	f.seedItemLockRow(t, f.itemID)

	// --- Step 1: the allocator takes the root, then its own locks. ---------------------------

	require.NoError(t, async.acquireItemLock(f.itemID), "async: acquire the item's ordering root")
	require.NoError(t, async.claim(t, f, issueID), "async: claim the issue it is about to cover")

	// --- Step 2: the reversal tries to enter the same section. -------------------------------

	reversalErr := make(chan error, 1)
	go func() {
		if err := reversal.acquireItemLock(f.itemID); err != nil {
			reversalErr <- err
			return
		}
		// Never reached until the allocator commits. Once it is, these run against a ledger the
		// allocator has finished with, in whatever order the reversal likes — which is the point.
		if err := reversal.q.FreeReleasedReceipts(ctx, []string{receiptID}); err != nil {
			reversalErr <- err
			return
		}
		reversalErr <- reversal.q.RestoreIssuesToReserved(ctx, []string{issueID})
	}()
	reversal.waitUntilBlockedOr(t, f.db, reversalErr)

	// --- Step 3: the allocator finishes the work that used to deadlock. ----------------------

	_, err := async.q.FindReceiptsForAllocation(ctx, sqlc.FindReceiptsForAllocationParams{
		AccountID: f.accountID, ItemID: f.itemID,
	})
	require.NoError(t, err,
		"the allocator must reach the receipts freely: it is alone inside the section for this item")
	async.commit(t)

	select {
	case err := <-reversalErr:
		checkNotDeadlock(t, err,
			"the reversal (receipts → issues) and the allocator (issue → receipts) still collided over "+
				"receipt "+receiptID+" and issue "+issueID+" despite both taking the item's ordering root")
	case <-time.After(20 * time.Second):
		t.Fatal("the reversal never entered the section after the allocator committed")
	}
	reversal.commit(t)
}

// Cycle 2, closed by the unit of work rather than by an ordering rule.
//
// The paged scan walked the open set in created_at order while every WHERE id IN (...) writer walks
// the same rows in clustered primary-key order, and primary keys are 12-char nanoids
// (shared/id/utils.go) so the two orders are uncorrelated. Two overlapping rows inverted about half
// the time.
//
// An allocate transaction now claims exactly one issue, by primary key, so it can never hold two
// issue rows in any order at all. The multi-row writer waits for the one it wants and then proceeds.
func TestNoDeadlock_ClaimVersusPKOrderedRestore(t *testing.T) {
	f := newFixture(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)

	// created_at order deliberately the reverse of id order, which is the case that used to invert.
	late := f.insertIssueWithID(t, f.itemID+"_zzzz", "open", "10", f.each, base.Add(time.Second))
	early := f.insertIssueWithID(t, f.itemID+"_aaaa", "open", "10", f.each, base.Add(2*time.Second))
	f.insertReceipt(t, "available", "100", f.each, base)

	claimer := f.actor(t, "allocate-consumer")
	restorer := f.actor(t, "shipment-void")
	ctx := context.Background()

	require.NoError(t, claimer.claim(t, f, late), "the claim must not block: it is one row by primary key")

	restoreErr := make(chan error, 1)
	go func() { restoreErr <- restorer.q.RestoreIssuesToReserved(ctx, []string{early, late}) }()
	restorer.waitUntilBlockedOr(t, f.db, restoreErr)

	// The claimer holds one row and wants nothing, so there is no cycle to break: it commits and the
	// writer completes.
	claimer.commit(t)
	select {
	case err := <-restoreErr:
		checkNotDeadlock(t, err, "a PK-ordered RestoreIssuesToReserved deadlocked against an allocate "+
			"transaction that holds a single issue row")
	case <-time.After(20 * time.Second):
		t.Fatal("the restore never completed after the claimer committed")
	}
	restorer.commit(t)
}

// Cycle 3: the clustered/secondary inversion on one row, closed the same way.
//
// A locking range read over inventory_issue_open_paging_idx took the secondary-index entry and then
// the clustered row, while UPDATE ... WHERE id = ? takes the clustered row and then maintains the
// secondary entry — because migration 00004 put status_code, the column every status writer changes,
// into the index the scan locked. One shared row was a cycle.
//
// Reaching the row by primary key takes the locks in the same direction as every status writer, so
// there is no inversion left to exploit.
func TestNoDeadlock_ClaimVersusSingleRowStatusUpdate(t *testing.T) {
	f := newFixture(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	shared := f.insertIssue(t, "open", "10", f.each, base)
	other := f.insertIssue(t, "open", "10", f.each, base.Add(time.Second))
	f.insertReceipt(t, "available", "100", f.each, base)

	claimer := f.actor(t, "allocate-consumer")
	closer := f.actor(t, "sync-allocator")
	ctx := context.Background()

	require.NoError(t, claimer.claim(t, f, shared))
	require.NoError(t, closer.q.CloseFullyAllocatedInventoryIssue(ctx, other))

	closeErr := make(chan error, 1)
	go func() { closeErr <- closer.q.CloseFullyAllocatedInventoryIssue(ctx, shared) }()
	closer.waitUntilBlockedOr(t, f.db, closeErr)

	claimer.commit(t)
	select {
	case err := <-closeErr:
		checkNotDeadlock(t, err, "CloseFullyAllocatedInventoryIssue deadlocked against a primary-key "+
			"claim of the same row")
	case <-time.After(20 * time.Second):
		t.Fatal("the close never completed after the claimer committed")
	}
	closer.commit(t)
}

// The footprint that closes cycles 2, 3 and 7 at once, asserted directly rather than through any one
// cycle: an allocate transaction holds exactly one inventory_issue row and no gap anywhere.
//
// The page used to be the transaction, so it held up to 200 issues, every index gap between and after
// them, and 200 shared `quantity` rows, for the length of a walk over all of them.
func TestClaim_LocksExactlyOneIssueAndNoGaps(t *testing.T) {
	f := newFixture(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	var issues []string
	for i := range 5 {
		issues = append(issues, f.insertIssue(t, "open", "10", f.each, base.Add(time.Duration(i)*time.Second)))
	}

	claimer := f.actor(t, "allocate-consumer")
	require.NoError(t, claimer.claim(t, f, issues[2]))

	var issueRows int
	for _, l := range claimer.locksHeld(t, f.db) {
		if l.Type != "RECORD" {
			continue
		}
		if strings.Contains(l.Mode, "GAP") && !strings.Contains(l.Mode, "REC_NOT_GAP") {
			t.Errorf("the claim holds a gap lock %s on %s.%s (%s): new demand for this item cannot be "+
				"recorded until the transaction commits", l.Mode, l.Object, l.Index, l.Data)
		}
		if l.Object == "inventory_issue" {
			issueRows++
		}
	}
	if issueRows > 2 {
		// The clustered row plus its secondary-index entry is the whole expected footprint.
		t.Errorf("the claim holds %d inventory_issue record locks; one issue is one clustered row and at "+
			"most one index entry", issueRows)
	}
}
