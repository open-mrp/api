// Package lease provides a distributed single-holder lease primitive backed by a SQL
// table. It lets multiple service pods coordinate so that a periodic task runs on
// exactly one pod at a time, while the other pods skip the tick.
//
// The typical flow is:
//
//	l := lease.New(repo)
//	err := l.WithLease(ctx, "agent-scheduler", 90*time.Second, func(ctx context.Context) error {
//	    return scheduler.checkSchedules(ctx)
//	})
//
// WithLease acquires the lease, runs fn with a context that is cancelled if the
// lease is lost, renews the lease at ttl/3 while fn runs, and releases it on exit.
// If another pod currently holds the lease, WithLease returns nil without running fn.
package lease

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Repo abstracts the persistence backend for a task_leases table. MySQL and
// PostgreSQL each have per-service implementations wrapping sqlc-generated queries.
type Repo interface {
	// Acquire attempts to claim `name` for `holder`, setting expires_at to now()+ttl.
	// Returns true iff the caller now holds the lease. The claim succeeds when either
	// no row exists for `name`, the existing row has expired, or the existing row is
	// already held by the same `holder` (idempotent re-acquire).
	Acquire(ctx context.Context, name, holder string, ttl time.Duration) (bool, error)

	// Renew extends expires_at to now()+ttl iff (name, holder) matches. Returns
	// true iff the renewal succeeded. A false return means the lease was lost
	// (expired and taken by someone else, or manually released) and the caller
	// should stop its in-flight work.
	Renew(ctx context.Context, name, holder string, ttl time.Duration) (bool, error)

	// Release deletes the lease row iff (name, holder) matches. Safe to call even
	// if the caller no longer holds the lease.
	Release(ctx context.Context, name, holder string) error
}

// Lease acquires leases on behalf of a single holder identity.
type Lease struct {
	repo   Repo
	holder string
}

// New returns a Lease that identifies itself as "{hostname}-{pid}", matching the
// outbox enqueuer's LockOwner convention.
func New(repo Repo) *Lease {
	hostname, _ := os.Hostname()
	return &Lease{
		repo:   repo,
		holder: fmt.Sprintf("%s-%d", hostname, os.Getpid()),
	}
}

// NewWithHolder is like New but uses an explicit holder identity. Intended for tests
// that need to simulate multiple pods within a single process.
func NewWithHolder(repo Repo, holder string) *Lease {
	return &Lease{repo: repo, holder: holder}
}

// Holder returns the identity this Lease uses when acquiring leases.
func (l *Lease) Holder() string { return l.holder }

// WithLease runs fn iff the lease `name` can be acquired. While fn runs the lease
// is renewed at ttl/3 intervals; if renewal fails (the lease was lost), the
// context passed to fn is cancelled so fn can bail out. On exit the lease is released.
//
// Returns fn's error if fn ran. Returns nil if another pod holds the lease or
// acquisition errored — these are not treated as caller-visible failures because
// the tick will simply re-attempt on the next interval.
func (l *Lease) WithLease(ctx context.Context, name string, ttl time.Duration, fn func(context.Context) error) error {
	acquired, err := l.repo.Acquire(ctx, name, l.holder, ttl)
	if err != nil {
		slog.Warn("Lease acquire failed", "lease", name, "holder", l.holder, "error", err)
		return nil
	}
	if !acquired {
		return nil
	}

	workCtx, cancelWork := context.WithCancel(ctx)
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
					cancelWork()
					return
				}
			}
		}
	})

	fnErr := fn(workCtx)
	cancelWork()
	wg.Wait()

	// Release on a detached context so a cancelled parent still reaches the DB.
	// Bound it with a short timeout so shutdown isn't blocked by a slow DB.
	releaseCtx, cancelRelease := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelRelease()
	if err := l.repo.Release(releaseCtx, name, l.holder); err != nil {
		slog.Warn("Lease release failed", "lease", name, "holder", l.holder, "error", err)
	}

	return fnErr
}
