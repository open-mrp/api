// Package ledgerlock owns the inventory ledger's ordering root.
//
// Scope is evidence that the root has been taken for a known set of items. Every ledger-writing
// repository method takes one, so a transaction that never called Acquire cannot compile a call to
// them. Acquire is the only constructor: the fields are unexported, so a forged &Scope{} holds nothing
// and every EnsureLocked below it takes the lock late and pages.
//
// This is the enforcement because a linter cannot be. tools/txaudit deliberately does not walk
// transitively — see its own header, which explains that a name-based walk into everything a closure
// calls collapses on method names like Create and Publish — and every ledger write in this service is
// reached indirectly from its WithTx closure. The compiler sees the call graph; the linter does not.
package ledgerlock

import (
	"context"
	"log/slog"
	"sort"
	"sync"

	apierror "github.com/open-mrp/api/shared/errors"
)

// Locker is the repository side of the root. Both the reservation and mutation repositories implement
// it, because both write the ledger and either may be the first to reach it in a given transaction.
type Locker interface {
	LockItemForLedger(ctx context.Context, itemID string) *apierror.APIError
}

// Scope records the items whose root this transaction holds.
type Scope struct {
	mu    sync.Mutex
	items map[string]struct{}
}

// Acquire takes the root for every item the transaction will write, in ascending id order, and must be
// the first statement of the WithTx callback.
//
// One key, one direction, so two transactions wanting the same two items can never take them in
// opposite orders — which is exactly what ranging a Go map does at receiving_order_svc,
// shipment_service and undo_batch_scan_consumer, whose iteration order is randomised per run.
//
// Corollary A: never call this while already holding a lock on a ledger row. Resolve the item set on
// the pool, before WithTx.
func Acquire(ctx context.Context, l Locker, itemIDs []string) (*Scope, *apierror.APIError) {
	s := &Scope{items: make(map[string]struct{}, len(itemIDs))}
	for _, itemID := range SortedUnique(itemIDs) {
		if apiErr := l.LockItemForLedger(ctx, itemID); apiErr != nil {
			return nil, apiErr
		}
		s.items[itemID] = struct{}{}
	}
	return s, nil
}

// Holds reports whether this scope already covers the item.
func (s *Scope) Holds(itemID string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, held := s.items[itemID]
	return held
}

// EnsureLocked is the backstop at every ledger-writing repository entry point: a no-op when the root is
// already held, a late acquisition when it is not.
//
// Corollary B: a late acquisition takes the root while ledger row locks are already held, which is
// itself an ordering inversion against any transaction that holds the root and is waiting for those
// rows. It is taken anyway rather than refused, because refusing dead-letters a shop-floor scan or
// 500s a ship, and because the worst case is a 1213 the transaction manager retries. It is NOT safe.
// Every occurrence means some flow's item pre-read is incomplete, and is logged at ERROR to be paged
// on.
func (s *Scope) EnsureLocked(ctx context.Context, l Locker, itemID string) *apierror.APIError {
	if itemID == "" {
		return nil
	}
	if s.Holds(itemID) {
		return nil
	}

	slog.ErrorContext(ctx, "inventory ledger lock acquired late",
		"item_id", itemID, "scope_present", s != nil)

	if apiErr := l.LockItemForLedger(ctx, itemID); apiErr != nil {
		return apiErr
	}
	if s != nil {
		s.mu.Lock()
		if s.items == nil {
			// A forged &Scope{} has no map. It still degrades to a paged late acquisition rather than a
			// panic, which is the whole point of the residual: forgery is possible, silence is not.
			s.items = make(map[string]struct{}, 1)
		}
		s.items[itemID] = struct{}{}
		s.mu.Unlock()
	}
	return nil
}

// SortedUnique drops blanks, deduplicates and sorts. The acquisition order is the whole point: see
// Acquire.
func SortedUnique(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
