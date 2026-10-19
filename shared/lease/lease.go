// Package lease provides a distributed single-holder lease primitive backed by a SQL table. It lets multiple service pods coordinate so that a periodic task runs on exactly one pod at a time, while the other pods skip the tick.
//
// The typical flow is:
//
//	l := lease.New(repo)
//	err := l.WithLease(ctx, "agent-scheduler", 90*time.Second, func(ctx context.Context) error {
//	    return scheduler.checkSchedules(ctx)
//	})
//
// WithLease acquires the lease, runs fn with a context that is cancelled if the lease is lost, renews the lease at ttl/3 while fn runs, and releases it on exit. If another pod currently holds the lease, WithLease returns nil without running fn.
//
// A scheduler that ticks more often than its TTL should call WithHeldLease instead, which keeps the lease between ticks so the pod that holds it pays one renewal per tick rather than an acquire, a holder read and a release.
package lease

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/open-mrp/api/shared/querytag"
)

// Repo abstracts the persistence backend for a task_leases table. MySQL and PostgreSQL each have per-service implementations wrapping sqlc-generated queries.
type Repo interface {
	// Acquire attempts to claim `name` for `holder`, setting expires_at to now()+ttl. Returns true iff the caller now holds the lease. The claim succeeds when either no row exists for `name`, the existing row has expired, or the existing row is already held by the same `holder` (idempotent re-acquire).
	Acquire(ctx context.Context, name, holder string, ttl time.Duration) (bool, error)

	// Renew extends expires_at to now()+ttl iff (name, holder) matches. Returns true iff the renewal succeeded. A false return means the lease was lost (expired and taken by someone else, or manually released) and the caller should stop its in-flight work.
	Renew(ctx context.Context, name, holder string, ttl time.Duration) (bool, error)

	// Release deletes the lease row iff (name, holder) matches. Safe to call even if the caller no longer holds the lease.
	Release(ctx context.Context, name, holder string) error
}

// Lease acquires leases on behalf of a single holder identity.
type Lease struct {
	repo   Repo
	holder string

	mu sync.Mutex
	// held maps each lease WithHeldLease is keeping between calls to the stop func of its release-on-cancel hook.
	held map[string]func() bool
}

// New returns a Lease that identifies itself as "{hostname}-{pid}", matching the outbox enqueuer's LockOwner convention.
func New(repo Repo) *Lease {
	hostname, _ := os.Hostname()
	return &Lease{
		repo:   repo,
		holder: fmt.Sprintf("%s-%d", hostname, os.Getpid()),
	}
}

// NewWithHolder is like New but uses an explicit holder identity. Intended for tests that need to simulate multiple pods within a single process.
func NewWithHolder(repo Repo, holder string) *Lease {
	return &Lease{repo: repo, holder: holder}
}

// TTLOr returns ttl when it is positive and fallback otherwise. Config structs reach for cmp.Or to default a TTL, but that only replaces a zero value: a negative one survives, and WithLease rejects it, silently disabling the periodic task for callers that discard the error.
func TTLOr(ttl, fallback time.Duration) time.Duration {
	if ttl > 0 {
		return ttl
	}
	return fallback
}

// Holder returns the identity this Lease uses when acquiring leases.
func (l *Lease) Holder() string { return l.holder }

// WithLease runs fn iff the lease `name` can be acquired. While fn runs the lease is renewed at ttl/3 intervals; if renewal fails (the lease was lost), the context passed to fn is cancelled so fn can bail out. On exit the lease is released.
//
// Returns fn's error if fn ran. Returns nil if another pod holds the lease or acquisition errored — these are not treated as caller-visible failures because the tick will simply re-attempt on the next interval. Returns an error for a non-positive ttl, which no retry can fix.
func (l *Lease) WithLease(ctx context.Context, name string, ttl time.Duration, fn func(context.Context) error) error {
	// A non-positive ttl expires the row the instant it is written, so no caller would ever hold exclusive access. Reject it here rather than let it reach time.NewTicker below, which panics from inside the renewal goroutine where no caller can recover.
	if ttl <= 0 {
		return fmt.Errorf("lease %q: ttl must be positive, got %s", name, ttl)
	}

	acquired, err := l.repo.Acquire(ctx, name, l.holder, ttl)
	if err != nil {
		slog.Warn("Lease acquire failed", "lease", name, "holder", l.holder, "error", err)
		return nil
	}
	if !acquired {
		return nil
	}

	fnErr, _ := l.run(ctx, name, ttl, fn)
	l.release(ctx, name)
	return fnErr
}

// WithHeldLease is WithLease for a task that runs on a timer shorter than ttl. It keeps the lease after fn returns instead of releasing it, so the next call renews it (one write) instead of acquiring it again (an upsert and a holder read) and releasing it after (a delete). The pod that wins keeps running the task until it stops calling or loses the lease; the others skip, as with WithLease, and take over once its row expires.
//
// The lease is released when ctx is cancelled, so a pod that shuts down hands the task over at once rather than after ttl. Calls for one name must not overlap.
func (l *Lease) WithHeldLease(ctx context.Context, name string, ttl time.Duration, fn func(context.Context) error) error {
	if ttl <= 0 {
		return fmt.Errorf("lease %q: ttl must be positive, got %s", name, ttl)
	}

	if !l.claim(ctx, name, ttl) {
		return nil
	}
	fnErr, lost := l.run(ctx, name, ttl, fn)
	if lost {
		l.forget(name)
	}
	return fnErr
}

// claim renews a lease this Lease is already holding, or acquires it otherwise, and reports whether the caller now holds it. A renewal that finds the row taken falls back to acquiring, which also covers a row that expired and was deleted.
func (l *Lease) claim(ctx context.Context, name string, ttl time.Duration) bool {
	l.mu.Lock()
	_, holding := l.held[name]
	l.mu.Unlock()

	if holding {
		ok, err := l.repo.Renew(ctx, name, l.holder, ttl)
		if err != nil {
			// Keep the record: the row is most likely still ours, and the next call renews it.
			slog.Warn("Lease renew failed", "lease", name, "holder", l.holder, "error", err)
			return false
		}
		if ok {
			return true
		}
		l.forget(name)
	}

	acquired, err := l.repo.Acquire(ctx, name, l.holder, ttl)
	if err != nil {
		slog.Warn("Lease acquire failed", "lease", name, "holder", l.holder, "error", err)
		return false
	}
	if !acquired {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.held == nil {
		l.held = make(map[string]func() bool)
	}
	if _, ok := l.held[name]; !ok {
		l.held[name] = context.AfterFunc(ctx, func() {
			l.mu.Lock()
			delete(l.held, name)
			l.mu.Unlock()
			l.release(ctx, name)
		})
	}
	return true
}

// forget drops a lease this Lease no longer holds, so its release-on-cancel hook doesn't fire.
func (l *Lease) forget(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if stop, ok := l.held[name]; ok {
		stop()
		delete(l.held, name)
	}
}

// run calls fn under a held lease, renewing it at ttl/3 and cancelling fn's context if a renewal fails. lost reports that a renewal found the lease taken.
func (l *Lease) run(ctx context.Context, name string, ttl time.Duration, fn func(context.Context) error) (fnErr error, lost bool) {
	workCtx, cancelWork := context.WithCancel(querytag.With(ctx, querytag.Job, name))
	defer cancelWork()

	renewInterval := ttl / 3
	if renewInterval <= 0 {
		renewInterval = ttl
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		ticker := time.NewTicker(renewInterval)
		defer ticker.Stop()
		for {
			select {
			case <-workCtx.Done():
				return
			case <-ticker.C:
				ok, err := l.repo.Renew(workCtx, name, l.holder, ttl)
				if err != nil {
					slog.Warn("Lease renew failed", "lease", name, "holder", l.holder, "error", err)
					cancelWork()
					return
				}
				if !ok {
					slog.Warn("Lease lost during renewal", "lease", name, "holder", l.holder)
					lost = true
					cancelWork()
					return
				}
			}
		}
	})

	fnErr = fn(workCtx)
	cancelWork()
	wg.Wait()
	return fnErr, lost
}

// release deletes the lease row if this holder still owns it.
func (l *Lease) release(ctx context.Context, name string) {
	// Release on a detached context so a cancelled parent still reaches the DB. Bound it with a short timeout so shutdown isn't blocked by a slow DB.
	releaseCtx, cancelRelease := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelRelease()
	if err := l.repo.Release(releaseCtx, name, l.holder); err != nil {
		slog.Warn("Lease release failed", "lease", name, "holder", l.holder, "error", err)
	}
}
