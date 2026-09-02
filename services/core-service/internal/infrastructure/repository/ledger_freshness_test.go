//go:build ledger

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/ledgerlock"
)

// The 2026-08-26 over-draw, reproduced: a transaction that queues on a receipt lock must not compute
// what is left from a snapshot older than the winner's commit.
//
// Both sides allocate a different issue against the same receipt, which holds enough for one of them.
// The loser opens its transaction and reads before the winner commits, then blocks on the receipt
// lock — so its REPEATABLE READ view, once opened, says the receipt is untouched. Whether it then
// draws 40 or 60 is the whole question.
//
// On HEAD it draws 60. The paged read is a locking read and does not open the view, but the caller
// resolved unit ratios next — a plain read — and the view opened there, before any receipt lock was
// held. The FOR UPDATE bought ordering and not freshness, which is the opposite of what its comment
// claimed.
//
// Two changes make it draw 40: the ratios move behind the receipt lock so the view opens after it,
// and what the receipt has left is read currently rather than from the view at all.
func TestFreshness_SecondAllocatorSeesTheFirstsDraw(t *testing.T) {
	f := newFixture(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)

	issueA := f.insertIssue(t, "open", "60", f.each, base)
	issueB := f.insertIssue(t, "open", "60", f.each, base.Add(time.Second))
	receipt := f.insertReceipt(t, "available", "100", f.each, base) // covers one of them, not both

	f.seedItemLockRow(t, f.itemID)

	winner := f.actor(t, "winner")
	loser := f.actor(t, "loser")
	ctx := context.Background()

	// The winner covers the first issue, drawing 60 of the receipt's 100. Not committed yet, so the
	// loser cannot see it by any means other than a current read taken after the winner commits.
	require.Nil(t, winner.repo.AllocateOneOpenIssue(ctx, winner.scope(t, f), f.accountID, f.itemID, issueA),
		"winner: allocating the first issue")

	// The loser starts on the second issue and blocks — now on the item's ordering root rather than on
	// the receipt, since both sides take it first. Everything it goes on to read therefore happens after
	// the winner committed, which is the freshness the ordering rule buys and the FOR UPDATE never did.
	//
	// The whole sequence runs inside the goroutine: acquiring the root is itself the blocking statement,
	// so doing it on the test's own goroutine would wedge the test rather than the actor.
	loserDone := make(chan error, 1)
	go func() {
		scope, apiErr := ledgerlock.Acquire(ctx, &inventoryReservationRepo{queries: loser.q}, []string{f.itemID})
		if apiErr != nil {
			loserDone <- apiErr
			return
		}
		if apiErr := loser.repo.AllocateOneOpenIssue(ctx, scope, f.accountID, f.itemID, issueB); apiErr != nil {
			loserDone <- apiErr
			return
		}
		loserDone <- nil
	}()
	loser.waitUntilBlockedOr(t, f.db, loserDone)

	winner.commit(t)

	select {
	case err := <-loserDone:
		require.NoError(t, err, "loser: allocating the second issue after the winner committed")
	case <-time.After(20 * time.Second):
		t.Fatal("the loser never finished after the winner committed")
	}
	loser.commit(t)

	// The receipt held 100. The winner took 60. Whatever the loser decided, the two together may not
	// exceed what was there.
	assertReceiptNotOverDrawn(t, f, receipt)

	// And it must have drawn the 40 that was actually left, not nothing: a transaction that reads
	// correctly and then refuses to draw is a different bug with the same symptom downstream.
	require.Equal(t, "40", f.baseAllocatedForIssue(t, issueB).String(),
		"the loser should have covered the 40 the winner left, in base units")
}

// The half of the design that does not depend on anybody taking a lock — which is the half that has
// to hold, because dashboard/apps/api writes these same four tables from Prisma with no locking read
// anywhere, on live invoice-delete and order-release paths.
//
// A writer we never serialize against exhausts the receipt while our transaction is open. Our own
// arithmetic then has to notice, and it can only notice through a current read: our snapshot predates
// their commit and always will.
//
// What it does about it is the delta rule. Their breach is not ours to fail on — aborting here would
// charge this transaction for a row it did not write, and the issue behind it would then fail on
// every pass forever, which is a permanent inbox failure for somebody else's corruption. So we skip
// the receipt, alarm, and leave the ledger no worse than we found it.
func TestVerification_UnlockedWriterDoesNotCauseAnOverDraw(t *testing.T) {
	f := newFixture(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)

	// `other` exists only to hang the other writer's allocation off, since an allocation needs an issue
	// to point at.
	issue := f.insertIssue(t, "open", "100", f.each, base)
	other := f.insertIssue(t, "open", "100", f.each, base.Add(time.Second))
	receipt := f.insertReceipt(t, "available", "100", f.each, base)

	f.seedItemLockRow(t, f.itemID)

	ours := f.actor(t, "go-allocator")
	ctx := context.Background()

	// Open this transaction's REPEATABLE READ view before the other writer commits. A consistent read
	// is what creates it, and from here on this transaction's snapshot can never show the row written
	// below — which is the point: no amount of locking makes a snapshot newer.
	//
	// A real allocate transaction opens its view the same way, a statement or two in, and then sits on
	// a receipt lock for as long as the draw takes. The window is not narrow.
	var seen int
	require.NoError(t, ours.tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM inventory_allocation WHERE inventory_receipt_id = ?`, receipt).Scan(&seen))
	require.Equal(t, 0, seen, "nothing is allocated off the receipt yet")

	// Now the dashboard draws the whole receipt against a different issue, committed, holding no lock
	// and having taken no locking read. Our snapshot still says the receipt is untouched.
	f.writeRawAllocation(t, other, receipt, "100", f.each)

	require.Nil(t, ours.repo.AllocateOneOpenIssue(ctx, ours.scope(t, f), f.accountID, f.itemID, issue),
		"a receipt exhausted by an unlocked writer must be skipped, not fail the transaction: failing "+
			"would poison this issue on every later pass for a row we did not write")
	ours.commit(t)

	assertReceiptNotOverDrawn(t, f, receipt)
	require.Equal(t, "0", f.baseAllocatedForIssue(t, issue).String(),
		"nothing was left on the receipt, so the issue should have drawn nothing at all")
}
