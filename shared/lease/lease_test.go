package lease

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// memRepo is an in-memory Repo simulating a task_leases table across any number of
// holders. Safe for concurrent use.
type memRepo struct {
	mu    sync.Mutex
	now   func() time.Time
	rows  map[string]memRow
	calls struct {
		acquire atomic.Int32
		renew   atomic.Int32
		release atomic.Int32
	}
	failAcquire error
	failRenew   error
	failRelease error
}

type memRow struct {
	holder    string
	expiresAt time.Time
}

func newMemRepo() *memRepo {
	return &memRepo{
		now:  time.Now,
		rows: make(map[string]memRow),
	}
}

func (r *memRepo) Acquire(_ context.Context, name, holder string, ttl time.Duration) (bool, error) {
	r.calls.acquire.Add(1)
	if r.failAcquire != nil {
		return false, r.failAcquire
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	row, ok := r.rows[name]
	if !ok || now.After(row.expiresAt) || row.holder == holder {
		r.rows[name] = memRow{holder: holder, expiresAt: now.Add(ttl)}
		return true, nil
	}
	return false, nil
}

func (r *memRepo) Renew(_ context.Context, name, holder string, ttl time.Duration) (bool, error) {
	r.calls.renew.Add(1)
	if r.failRenew != nil {
		return false, r.failRenew
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[name]
	if !ok || row.holder != holder {
		return false, nil
	}
	row.expiresAt = r.now().Add(ttl)
	r.rows[name] = row
	return true, nil
}

func (r *memRepo) Release(_ context.Context, name, holder string) error {
	r.calls.release.Add(1)
	if r.failRelease != nil {
		return r.failRelease
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if row, ok := r.rows[name]; ok && row.holder == holder {
		delete(r.rows, name)
	}
	return nil
}

func TestWithLease_AcquiresAndReleases(t *testing.T) {
	repo := newMemRepo()
	l := NewWithHolder(repo, "pod-a")

	var ran bool
	err := l.WithLease(context.Background(), "test", 500*time.Millisecond, func(ctx context.Context) error {
		ran = true
		return nil
	})
	require.NoError(t, err)
	require.True(t, ran)
	require.Empty(t, repo.rows, "release should delete the row")
}

func TestWithLease_SecondHolderSkips(t *testing.T) {
	repo := newMemRepo()
	a := NewWithHolder(repo, "pod-a")
	b := NewWithHolder(repo, "pod-b")

	// A acquires and is still running when B attempts.
	done := make(chan struct{})
	proceed := make(chan struct{})

	go func() {
		_ = a.WithLease(context.Background(), "test", 2*time.Second, func(ctx context.Context) error {
			close(proceed)
			<-done
			return nil
		})
	}()
	<-proceed

	var bRan bool
	err := b.WithLease(context.Background(), "test", 2*time.Second, func(ctx context.Context) error {
		bRan = true
		return nil
	})
	require.NoError(t, err)
	require.False(t, bRan, "second holder must not run while first holds the lease")

	close(done)
}

func TestWithLease_AcquireErrorIsSwallowed(t *testing.T) {
	repo := newMemRepo()
	repo.failAcquire = errors.New("db down")
	l := NewWithHolder(repo, "pod-a")

	var ran bool
	err := l.WithLease(context.Background(), "test", time.Second, func(ctx context.Context) error {
		ran = true
		return nil
	})
	require.NoError(t, err, "acquire errors must not propagate (the next tick will retry)")
	require.False(t, ran)
}

func TestWithLease_RenewalKeepsLeaseAlive(t *testing.T) {
	repo := newMemRepo()
	l := NewWithHolder(repo, "pod-a")

	ttl := 90 * time.Millisecond // renew every 30ms
	err := l.WithLease(context.Background(), "test", ttl, func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond) // spans multiple renew ticks
		return nil
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, repo.calls.renew.Load(), int32(3), "expected several renewals within the work window")
}

func TestWithLease_RenewalFailureCancelsWork(t *testing.T) {
	repo := newMemRepo()
	l := NewWithHolder(repo, "pod-a")

	ttl := 60 * time.Millisecond

	var sawCancel bool
	err := l.WithLease(context.Background(), "test", ttl, func(ctx context.Context) error {
		// Simulate the lease being yanked out from under us after the first renew fires.
		time.AfterFunc(10*time.Millisecond, func() {
			repo.mu.Lock()
			delete(repo.rows, "test")
			repo.mu.Unlock()
		})
		select {
		case <-ctx.Done():
			sawCancel = true
			return ctx.Err()
		case <-time.After(2 * time.Second):
			return errors.New("ctx was never cancelled")
		}
	})
	require.Error(t, err)
	require.True(t, sawCancel)
}

func TestWithLease_ReleasesWhenParentCancelled(t *testing.T) {
	repo := newMemRepo()
	l := NewWithHolder(repo, "pod-a")

	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})

	done := make(chan error, 1)
	go func() {
		done <- l.WithLease(ctx, "test", 500*time.Millisecond, func(workCtx context.Context) error {
			close(started)
			<-workCtx.Done()
			return workCtx.Err()
		})
	}()
	<-started
	cancel()

	err := <-done
	require.Error(t, err)
	require.Equal(t, int32(1), repo.calls.release.Load(), "release must still run even when parent ctx is cancelled")
	require.Empty(t, repo.rows)
}

func TestWithLease_ReturnsFnError(t *testing.T) {
	repo := newMemRepo()
	l := NewWithHolder(repo, "pod-a")

	wantErr := errors.New("boom")
	err := l.WithLease(context.Background(), "test", time.Second, func(ctx context.Context) error {
		return wantErr
	})
	require.ErrorIs(t, err, wantErr)
}

func TestNew_DerivesHolderFromHostnameAndPid(t *testing.T) {
	l := New(newMemRepo())
	require.NotEmpty(t, l.Holder())
	require.Contains(t, l.Holder(), "-", "holder should look like hostname-pid")
}

// secondGranularityRepo mirrors the production repos, which persist the TTL as int64(ttl/time.Second) into a whole-second SQL column.
type secondGranularityRepo struct {
	*memRepo
}

func (r *secondGranularityRepo) Acquire(ctx context.Context, name, holder string, ttl time.Duration) (bool, error) {
	return r.memRepo.Acquire(ctx, name, holder, ttl.Truncate(time.Second))
}

func (r *secondGranularityRepo) Renew(ctx context.Context, name, holder string, ttl time.Duration) (bool, error) {
	return r.memRepo.Renew(ctx, name, holder, ttl.Truncate(time.Second))
}

func TestAcquire_SecondGranularityExcludesSecondHolder(t *testing.T) {
	t.Parallel()
	repo := &secondGranularityRepo{memRepo: newMemRepo()}

	ok, err := repo.Acquire(context.Background(), "test", "pod-a", 2*time.Second)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = repo.Acquire(context.Background(), "test", "pod-b", 2*time.Second)
	require.NoError(t, err)
	require.False(t, ok, "a live lease must exclude every other holder")
}

func TestWithLease_TransientRenewErrorCancelsWorkAndLeavesRow(t *testing.T) {
	t.Parallel()
	repo := newMemRepo()
	repo.failRenew = errors.New("db blip")
	repo.failRelease = errors.New("db blip")
	l := NewWithHolder(repo, "pod-a")

	// A renew *error* is treated exactly like a lost lease: the work context is cancelled even though the row is still ours and nowhere near expiry.
	err := l.WithLease(context.Background(), "test", 30*time.Millisecond, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	require.ErrorIs(t, err, context.Canceled)
	require.GreaterOrEqual(t, repo.calls.renew.Load(), int32(1))
	require.Equal(t, int32(1), repo.calls.release.Load())
	// Release failed too, so the claim outlives the work: no pod runs the task until the TTL lapses.
	require.Equal(t, "pod-a", repo.rows["test"].holder, "a failed release leaves the lease row orphaned")
}

func TestWithLease_ReleaseErrorDoesNotMaskFnError(t *testing.T) {
	t.Parallel()
	repo := newMemRepo()
	repo.failRelease = errors.New("db blip")
	l := NewWithHolder(repo, "pod-a")

	wantErr := errors.New("boom")
	err := l.WithLease(context.Background(), "test", time.Second, func(ctx context.Context) error {
		return wantErr
	})
	require.ErrorIs(t, err, wantErr, "release failures are logged, not returned")
}

// A non-positive TTL used to reach time.NewTicker inside the renewal goroutine, panicking the whole process where no caller could recover. It must be rejected before the lease is ever acquired.
func TestWithLease_RejectsNonPositiveTTL(t *testing.T) {
	t.Parallel()
	for _, ttl := range []time.Duration{0, -time.Second, -time.Millisecond} {
		t.Run(ttl.String(), func(t *testing.T) {
			t.Parallel()
			repo := newMemRepo()
			l := NewWithHolder(repo, "pod-a")

			ran := false
			err := l.WithLease(t.Context(), "test", ttl, func(context.Context) error {
				ran = true
				return nil
			})

			require.Error(t, err)
			require.Contains(t, err.Error(), "ttl must be positive")
			require.False(t, ran, "fn must not run without a usable lease")
			require.Zero(t, repo.calls.acquire.Load(), "must reject before touching the repo")
			require.Zero(t, repo.calls.release.Load())
		})
	}
}

// The smallest positive TTL still divides to a zero renew interval, which must fall back to the TTL rather than reaching NewTicker with zero.
func TestWithLease_TinyPositiveTTLDoesNotPanic(t *testing.T) {
	t.Parallel()
	repo := newMemRepo()
	l := NewWithHolder(repo, "pod-a")

	ran := false
	err := l.WithLease(t.Context(), "test", 1, func(context.Context) error {
		ran = true
		return nil
	})

	require.NoError(t, err)
	require.True(t, ran)
}

func TestTTLOr(t *testing.T) {
	t.Parallel()
	fallback := 5 * time.Minute
	tests := []struct {
		name string
		ttl  time.Duration
		want time.Duration
	}{
		{name: "positive is kept", ttl: 90 * time.Second, want: 90 * time.Second},
		{name: "zero takes the fallback", ttl: 0, want: fallback},
		{name: "negative takes the fallback", ttl: -time.Second, want: fallback},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, TTLOr(tt.ttl, fallback))
		})
	}
}
