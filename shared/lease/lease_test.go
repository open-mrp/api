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

func TestWithHeldLease_RenewsInsteadOfReacquiring(t *testing.T) {
	t.Parallel()
	repo := newMemRepo()
	l := NewWithHolder(repo, "pod-a")

	runs := 0
	for range 3 {
		err := l.WithHeldLease(t.Context(), "test", time.Minute, func(context.Context) error {
			runs++
			return nil
		})
		require.NoError(t, err)
	}

	require.Equal(t, 3, runs)
	require.Equal(t, int32(1), repo.calls.acquire.Load(), "only the first call acquires")
	require.Equal(t, int32(2), repo.calls.renew.Load(), "later calls renew")
	require.Zero(t, repo.calls.release.Load(), "the lease is kept between calls")
	require.Equal(t, "pod-a", repo.rows["test"].holder)
}

func TestWithHeldLease_OtherHoldersSkipUntilItExpires(t *testing.T) {
	t.Parallel()
	repo := newMemRepo()
	now := time.Now()
	repo.now = func() time.Time { return now }
	a := NewWithHolder(repo, "pod-a")
	b := NewWithHolder(repo, "pod-b")

	run := func(l *Lease) bool {
		ran := false
		require.NoError(t, l.WithHeldLease(t.Context(), "test", time.Minute, func(context.Context) error {
			ran = true
			return nil
		}))
		return ran
	}

	require.True(t, run(a))
	require.False(t, run(b), "pod-b must skip while pod-a holds the lease between calls")
	require.True(t, run(a))

	// pod-a stops renewing (crashed): once the row expires pod-b takes over, and pod-a's next renewal finds it gone.
	now = now.Add(2 * time.Minute)
	require.True(t, run(b))
	require.False(t, run(a))
	require.True(t, run(b))
}

func TestWithHeldLease_ReacquiresAfterRowIsDeleted(t *testing.T) {
	t.Parallel()
	repo := newMemRepo()
	l := NewWithHolder(repo, "pod-a")

	noop := func(context.Context) error { return nil }
	require.NoError(t, l.WithHeldLease(t.Context(), "test", time.Minute, noop))
	repo.mu.Lock()
	delete(repo.rows, "test")
	repo.mu.Unlock()

	ran := false
	require.NoError(t, l.WithHeldLease(t.Context(), "test", time.Minute, func(context.Context) error {
		ran = true
		return nil
	}))
	require.True(t, ran, "a failed renewal falls back to acquiring")
	require.Equal(t, int32(2), repo.calls.acquire.Load())
}

func TestWithHeldLease_RenewErrorSkipsTickAndKeepsLease(t *testing.T) {
	t.Parallel()
	repo := newMemRepo()
	l := NewWithHolder(repo, "pod-a")

	noop := func(context.Context) error { return nil }
	require.NoError(t, l.WithHeldLease(t.Context(), "test", time.Minute, noop))

	repo.mu.Lock()
	repo.failRenew = errors.New("db blip")
	repo.mu.Unlock()
	ran := false
	require.NoError(t, l.WithHeldLease(t.Context(), "test", time.Minute, func(context.Context) error {
		ran = true
		return nil
	}))
	require.False(t, ran)
	require.Equal(t, int32(1), repo.calls.acquire.Load(), "a renew error must not fall through to acquiring")

	repo.mu.Lock()
	repo.failRenew = nil
	repo.mu.Unlock()
	require.NoError(t, l.WithHeldLease(t.Context(), "test", time.Minute, noop))
	require.Equal(t, int32(1), repo.calls.acquire.Load(), "the next call renews the lease it kept")
}

func TestWithHeldLease_ReleasesWhenContextCancelled(t *testing.T) {
	t.Parallel()
	repo := newMemRepo()
	l := NewWithHolder(repo, "pod-a")

	ctx, cancel := context.WithCancel(t.Context())
	require.NoError(t, l.WithHeldLease(ctx, "test", time.Minute, func(context.Context) error { return nil }))
	cancel()

	require.Eventually(t, func() bool { return repo.calls.release.Load() == 1 }, time.Second, 5*time.Millisecond)
	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.Empty(t, repo.rows, "shutdown hands the lease over at once")
}

func TestWithHeldLease_LostDuringRunIsForgotten(t *testing.T) {
	t.Parallel()
	repo := newMemRepo()
	l := NewWithHolder(repo, "pod-a")

	ctx, cancel := context.WithCancel(t.Context())
	err := l.WithHeldLease(ctx, "test", 30*time.Millisecond, func(ctx context.Context) error {
		repo.mu.Lock()
		repo.rows["test"] = memRow{holder: "pod-b", expiresAt: time.Now().Add(time.Hour)}
		repo.mu.Unlock()
		<-ctx.Done()
		return ctx.Err()
	})
	require.ErrorIs(t, err, context.Canceled)

	// pod-a no longer holds it, so cancelling must not release pod-b's row.
	cancel()
	time.Sleep(20 * time.Millisecond)
	require.Zero(t, repo.calls.release.Load())
	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.Equal(t, "pod-b", repo.rows["test"].holder)
}

func TestWithHeldLease_RejectsNonPositiveTTL(t *testing.T) {
	t.Parallel()
	repo := newMemRepo()
	l := NewWithHolder(repo, "pod-a")

	err := l.WithHeldLease(t.Context(), "test", 0, func(context.Context) error { return nil })
	require.ErrorContains(t, err, "ttl must be positive")
	require.Zero(t, repo.calls.acquire.Load())
}
