// Package ledgerlock ensures inventory ledger writes acquire locks in a consistent order.
//
// Scope represents the set of inventory items that have already been locked for a transaction. Every repository method that writes to the inventory ledger requires a *Scope. This makes it difficult to write to the ledger without first calling Acquire.
//
// Acquire is the intended way to create a Scope. Its fields are private, so creating &Scope{} manually does not mark any items as locked. If that happens, EnsureLocked will detect the missing lock, acquire it late, and log an error so the problem is visible.
package ledgerlock

import (
	"context"
	"log/slog"
	"sort"
	"sync"

	apierror "github.com/open-mrp/api/shared/errors"
)

// Locker is the repository side of the root. Both the reservation and mutation repositories implement it, because both write the ledger and either may be the first to reach it in a given transaction.
type Locker interface {
	LockItemForLedger(ctx context.Context, itemID string) *apierror.APIError
}

// Scope records the items whose root this transaction holds.
type Scope struct {
	mu    sync.Mutex
	items map[string]struct{}
}

// Acquire locks all the inventory items this transaction plans to change in a consistent sorted order. These items are recorded in a `Scope` so two transactions cannot touch the same item.
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

// EnsureLocked checks whether the current transaction already holds the item's inventory lock. If not, it acquires the lock late and logs an error.
//
// An error here means that we failed to acquire a lock when we should have, meaning a deadlock is possible.
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
			// A forged &Scope{} has no map. It still degrades to a paged late acquisition rather than a panic, which is the whole point of the residual: forgery is possible, silence is not.
			s.items = make(map[string]struct{}, 1)
		}
		s.items[itemID] = struct{}{}
		s.mu.Unlock()
	}
	return nil
}

// SortedUnique drops blanks, deduplicates and sorts.
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
