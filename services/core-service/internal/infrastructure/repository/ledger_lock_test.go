//go:build ledger

package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The ordering root does the one job it exists for: two transactions are never both inside the
// section for the same item.
//
// Everything else in the design follows from this and nothing else. Once it holds, the order in which
// a flow reaches inventory_issue, inventory_receipt, inventory_allocation and their satellites stops
// mattering — which is the only way to reconcile an allocator that must hold demand before choosing
// receipts with a reversal that must free receipts before restoring demand.
func TestLedgerLock_SerialisesTwoAcquirers(t *testing.T) {
	f := newFixture(t)
	f.seedItemLockRow(t, f.itemID)

	holder := f.actor(t, "holder")
	contender := f.actor(t, "contender")

	require.NoError(t, holder.acquireItemLock(f.itemID), "holder: acquire")

	done := make(chan error, 1)
	go func() { done <- contender.acquireItemLock(f.itemID) }()

	// The server says it is waiting, rather than the test assuming it by now.
	contender.waitUntilBlockedOr(t, f.db, done)

	holder.commit(t)

	select {
	case err := <-done:
		require.NoError(t, err, "the contender must acquire once the holder commits, not fail")
	case <-time.After(10 * time.Second):
		t.Fatal("the contender never acquired the root after the holder committed")
	}
	contender.commit(t)
}

// The cold branch, which is the one the obvious designs get wrong (NEW-A).
//
// Two transactions creating lock rows for two DIFFERENT new items must not deadlock on this table's
// own primary key. Items are created continuously, including by the dashboard, so this branch is not a
// rare startup case — it is the normal path for every newly received item until its row exists.
//
// It is safe because INSERT ... ON DUPLICATE KEY UPDATE is the only statement ever issued here, so no
// access path takes a gap lock and there is nothing for an insert-intention lock to conflict with.
// TestLedgerLock_RejectedShapeIsUnsafe below is the measurement behind that claim.
func TestLedgerLock_ConcurrentColdAcquire(t *testing.T) {
	f := newFixture(t)

	// Two ids that sort adjacently, so any gap lock one takes would cover the other.
	a, b := f.itemID+"_aa", f.itemID+"_ab"

	first, second := f.actor(t, "cold-a"), f.actor(t, "cold-b")
	errs := make(chan error, 2)
	go func() { errs <- first.acquireItemLock(a) }()
	go func() { errs <- second.acquireItemLock(b) }()

	for range 2 {
		select {
		case err := <-errs:
			require.NoError(t, err,
				"acquiring the root for a brand-new item deadlocked or timed out against a different "+
					"brand-new item: the mechanism built to stop deadlocks is causing one")
		case <-time.After(10 * time.Second):
			t.Fatal("cold acquisition hung")
		}
	}
	first.commit(t)
	second.commit(t)
}

// Corollary D as a measurement rather than an assertion: the rejected shape really does deadlock.
//
// A locking read that matches no row takes a GAP lock; the INSERT that would follow takes an
// insert-intention lock, which conflicts with another transaction's gap lock over the same gap. Two
// transactions creating two different new items then deadlock — on their first statement, which is the
// 2026-09-01 incident's own signature.
//
// This is here because "never add a SELECT to this table" is the kind of rule that gets relaxed by
// someone who cannot see what it costs. If MySQL ever makes this shape safe, this test fails and says
// so, and the reasoning in 00017_inventory_item_lock should be re-read before anything is changed.
func TestLedgerLock_RejectedShapeIsUnsafe(t *testing.T) {
	f := newFixture(t)
	a, b := f.itemID+"_ra", f.itemID+"_rb"

	first, second := f.actor(t, "rejected-a"), f.actor(t, "rejected-b")

	// Both take the gap lock the missing row leaves behind.
	var scratch string
	require.ErrorIs(t, first.tx.QueryRow(
		`SELECT item_id FROM inventory_item_lock WHERE item_id = ? FOR UPDATE`, a).Scan(&scratch),
		sql.ErrNoRows, "the row must not exist, or this is not the cold path")
	require.ErrorIs(t, second.tx.QueryRow(
		`SELECT item_id FROM inventory_item_lock WHERE item_id = ? FOR UPDATE`, b).Scan(&scratch),
		sql.ErrNoRows)

	errs := make(chan error, 2)
	go func() {
		_, err := first.tx.Exec(`INSERT INTO inventory_item_lock (item_id, created_at) VALUES (?, NOW(3))`, a)
		errs <- err
	}()
	go func() {
		_, err := second.tx.Exec(`INSERT INTO inventory_item_lock (item_id, created_at) VALUES (?, NOW(3))`, b)
		errs <- err
	}()

	var conflicts int
	for range 2 {
		select {
		case err := <-errs:
			if isDeadlock(err) || isLockWaitTimeout(err) {
				conflicts++
			}
		case <-time.After(15 * time.Second):
			t.Fatal("the rejected shape neither completed nor conflicted")
		}
	}
	if conflicts == 0 {
		t.Error("SELECT ... FOR UPDATE followed by INSERT no longer conflicts on two different new keys. " +
			"That is the shape 00017_inventory_item_lock rejects, and the whole reason the acquisition " +
			"must stay a bare INSERT ... ON DUPLICATE KEY UPDATE. Re-read that migration's note before " +
			"relaxing anything.")
	}
}
