//go:build ledger

package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Discovering open demand must not stop new demand being recorded.
//
// FindOpenIssuesForItemPaged is a FOR UPDATE range scan over inventory_issue_open_paging_idx
// (account_id, item_id, status_code, created_at). When the item has fewer than a page of open issues
// — which is nearly every item — the scan runs off the end of the range and takes the next-key lock
// past the last match. That gap is where every new open issue for the item lands, so for the whole
// life of the allocate transaction nothing can record demand for that item: not a batch scan's
// remainder issue, not a shipment's residual, not a reservation.
//
// The transaction holding it is not short. On 2026-08-31 one of them was killed by vttablet's
// 20-second transaction timeout mid-flight (message_inbox 327850, "for tx killer rollback").
//
// Nobody would see a deadlock here. They would see the terminal hang.
func TestOpenIssueDiscovery_DoesNotBlockNewDemand(t *testing.T) {
	f := newFixture(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)

	// Fewer than a page, so the scan reaches the end of the range and takes the trailing gap.
	for i := range 5 {
		f.insertIssue(t, "open", "10", f.each, base.Add(time.Duration(i)*time.Second))
	}
	f.insertReceipt(t, "available", "1000", f.each, base)

	scanner := f.actor(t, "discovery")
	_, _, _, apiErr := scanner.repo.AllocateOpenIssuesForItemPage(
		context.Background(), f.accountID, f.itemID, time.Time{}, "", 200)
	require.Nil(t, apiErr, "the page itself must succeed; this test is about what it holds afterwards")
	requireProductionPlan(t, scanner, f.db)

	// New demand always arrives at NOW(3), to the right of everything the scan saw — squarely in the
	// trailing gap.
	if !f.probeCanInsertOpenIssue(t, time.Now().UTC()) {
		t.Error("a new open issue could not be recorded while an allocation page was open: the paged " +
			"read is holding the trailing gap of inventory_issue_open_paging_idx, which stalls every " +
			"batch scan, shipment and reservation for this item until the allocate transaction commits")
	}
}

// The same gap, reached from the direction that made it a deadlock rather than a stall.
//
// AllocateReservationsForConsumption flips a reservation to 'open' in place. The reservation was
// created at order-entry time, so its created_at sorts to the LEFT of every recently created open
// issue — the flip materialises an index entry inside the part of the range a scan locks FIRST, not
// at the end. An insert-intention lock conflicts with a gap lock, so the consumption allocator waits
// on the discovery scan while holding the receipt locks the discovery scan is about to want.
func TestReservedToOpenFlip_IsNotBlockedByADiscoveryScan(t *testing.T) {
	f := newFixture(t)
	old := time.Now().UTC().Add(-72 * time.Hour).Truncate(time.Millisecond)
	recent := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)

	reservationID := f.insertIssue(t, "reserved", "4", f.each, old)
	for i := range 3 {
		f.insertIssue(t, "open", "10", f.each, recent.Add(time.Duration(i)*time.Second))
	}
	f.insertReceipt(t, "available", "1000", f.each, old)

	scanner := f.actor(t, "discovery")
	_, _, _, apiErr := scanner.repo.AllocateOpenIssuesForItemPage(
		context.Background(), f.accountID, f.itemID, time.Time{}, "", 200)
	require.Nil(t, apiErr)
	requireProductionPlan(t, scanner, f.db)

	flipper := f.actor(t, "consumption")
	_, err := flipper.tx.ExecContext(context.Background(),
		`UPDATE inventory_issue SET status_code = 'open', issued_at = NOW(3), updated_at = NOW(3)
		  WHERE id = ? AND status_code = 'reserved'`, reservationID)
	if isLockWaitTimeout(err) {
		t.Error("consuming a reservation blocked on an allocation page: the reserved→open flip inserts " +
			"an index entry at the reservation's original created_at, inside the range the paged " +
			"FOR UPDATE gap-locked — this is the third edge of the 2026-09-01 cycle")
		return
	}
	require.NoError(t, err)
}

// The footprint assertion behind both of the above, stated directly rather than inferred.
//
// Discovery is meant to name candidates and decide nothing. Anything it locks — a gap on the paging
// index, a clustered inventory_issue row, a `quantity` row shared with every other reader of that
// issue — is a lock held for the length of a walk that may draw on dozens of receipts.
func TestOpenIssueDiscovery_TakesNoLocks(t *testing.T) {
	f := newFixture(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	for i := range 5 {
		f.insertIssue(t, "open", "10", f.each, base.Add(time.Duration(i)*time.Second))
	}

	scanner := f.actor(t, "discovery")
	_, err := scanner.q.FindOpenIssuesForItemPaged(context.Background(), findOpenIssuesPagedParams(f, 200))
	require.NoError(t, err)
	requireProductionPlan(t, scanner, f.db)

	for _, l := range scanner.locksHeld(t, f.db) {
		if l.Type != "RECORD" {
			continue // an intention lock on the table is unavoidable and harmless
		}
		switch {
		case l.Object == "quantity":
			t.Errorf("discovery holds a %s lock on a quantity row (%s): the JOIN under FOR UPDATE locks "+
				"rows shared with every other reader of that issue", l.Mode, l.Data)
		case strings.Contains(l.Mode, "GAP"):
			t.Errorf("discovery holds a gap lock %s on %s.%s (%s): nothing can insert into that range "+
				"until this transaction commits", l.Mode, l.Object, l.Index, l.Data)
		case l.Object == "inventory_issue":
			t.Errorf("discovery holds a %s lock on an inventory_issue row (%s.%s): naming a candidate "+
				"must not lock it — the claim is a separate, primary-key read", l.Mode, l.Index, l.Data)
		}
	}
}
