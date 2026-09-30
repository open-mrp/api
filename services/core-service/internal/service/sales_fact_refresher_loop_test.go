package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/lease"
)

// memLeaseRepo is an in-memory task_leases table shared by every "pod" in a test.
type memLeaseRepo struct {
	mu      sync.Mutex
	holders map[string]string
	expires map[string]time.Time
}

func newMemLeaseRepo() *memLeaseRepo {
	return &memLeaseRepo{holders: map[string]string{}, expires: map[string]time.Time{}}
}

func (r *memLeaseRepo) Acquire(_ context.Context, name, holder string, ttl time.Duration) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if h, ok := r.holders[name]; ok && h != holder && time.Now().Before(r.expires[name]) {
		return false, nil
	}
	r.holders[name], r.expires[name] = holder, time.Now().Add(ttl)
	return true, nil
}

func (r *memLeaseRepo) Renew(_ context.Context, name, holder string, ttl time.Duration) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.holders[name] != holder {
		return false, nil
	}
	r.expires[name] = time.Now().Add(ttl)
	return true, nil
}

func (r *memLeaseRepo) Release(_ context.Context, name, holder string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.holders[name] == holder {
		delete(r.holders, name)
		delete(r.expires, name)
	}
	return nil
}

// loopRefresher is a refresher whose Tick is replaced by tick, with its runs counted.
type loopRefresher struct {
	*SalesFactRefresher
	runs atomic.Int32
}

func startLoopRefresher(t *testing.T, leases lease.Repo, holder string, cfg SalesFactRefresherConfig, tick func(run int32) bool) *loopRefresher {
	t.Helper()
	cfg.Repos = nil
	cfg.Lease = lease.NewWithHolder(leases, holder)
	r := &loopRefresher{}
	r.SalesFactRefresher = &SalesFactRefresher{cfg: cfg.WithDefaults(), wake: make(chan struct{}, 1), stopCh: make(chan struct{})}
	r.tick = func(context.Context) bool { return tick(r.runs.Add(1)) }
	require.NoError(t, r.Start(context.Background()))
	t.Cleanup(r.Stop)
	return r
}

// quietLoop starts one pod with a short startup delay and an idle interval no test outlasts.
func quietLoop(t *testing.T, tick func(run int32) bool) *loopRefresher {
	return startLoopRefresher(t, newMemLeaseRepo(), "pod-a", SalesFactRefresherConfig{
		TickInterval: 20 * time.Millisecond,
		IdleInterval: time.Hour,
		WakeDelay:    10 * time.Millisecond,
	}, tick)
}

func idle(int32) bool { return false }

func waitRuns(t *testing.T, r *loopRefresher, want int32) {
	t.Helper()
	require.Eventually(t, func() bool { return r.runs.Load() >= want }, 2*time.Second, 2*time.Millisecond, "refresher ran %d times, want %d", r.runs.Load(), want)
}

func requireRunsStay(t *testing.T, r *loopRefresher, want int32, over time.Duration) {
	t.Helper()
	time.Sleep(over)
	require.Equal(t, want, r.runs.Load())
}

func TestRefresherRunsOnceAtStartupThenIdles(t *testing.T) {
	r := quietLoop(t, idle)
	waitRuns(t, r, 1)
	requireRunsStay(t, r, 1, 200*time.Millisecond)
}

func TestRefresherRunsAtTheIdleIntervalWithoutAWake(t *testing.T) {
	r := startLoopRefresher(t, newMemLeaseRepo(), "pod-a", SalesFactRefresherConfig{
		TickInterval: 10 * time.Millisecond,
		IdleInterval: 50 * time.Millisecond,
	}, idle)
	waitRuns(t, r, 4)
}

func TestAWakeRunsTheRefresherWithinTheWakeDelay(t *testing.T) {
	r := quietLoop(t, idle)
	waitRuns(t, r, 1)

	r.Wake()

	waitRuns(t, r, 2)
	requireRunsStay(t, r, 2, 100*time.Millisecond)
}

func TestABurstOfWakesIsServedByOneRun(t *testing.T) {
	r := startLoopRefresher(t, newMemLeaseRepo(), "pod-a", SalesFactRefresherConfig{
		TickInterval: 20 * time.Millisecond,
		IdleInterval: time.Hour,
		WakeDelay:    100 * time.Millisecond,
	}, idle)
	waitRuns(t, r, 1)

	for range 50 {
		r.Wake()
	}

	waitRuns(t, r, 2)
	requireRunsStay(t, r, 2, 250*time.Millisecond)
}

func TestAWakeCannotPostponeAnEarlierRun(t *testing.T) {
	r := startLoopRefresher(t, newMemLeaseRepo(), "pod-a", SalesFactRefresherConfig{
		TickInterval: 30 * time.Millisecond,
		IdleInterval: time.Hour,
		WakeDelay:    time.Hour,
	}, func(run int32) bool { return run == 1 })
	waitRuns(t, r, 1)

	// The backlog left by run 1 is due in 30ms; an hour-long wake delay must not push it back.
	r.Wake()

	waitRuns(t, r, 2)
}

func TestABacklogKeepsTheRefresherRunningUntilItIsDrained(t *testing.T) {
	r := quietLoop(t, func(run int32) bool { return run < 4 })

	waitRuns(t, r, 4)
	requireRunsStay(t, r, 4, 200*time.Millisecond)
}

func TestAWakeDuringARunIsServedByTheNextRun(t *testing.T) {
	inRun, release := make(chan struct{}), make(chan struct{})
	r := quietLoop(t, func(run int32) bool {
		if run == 2 {
			close(inRun)
			<-release
		}
		return false
	})
	waitRuns(t, r, 1)
	r.Wake()
	<-inRun

	// A mark written now may be newer than the marks run 2 listed.
	r.Wake()
	close(release)

	waitRuns(t, r, 3)
	requireRunsStay(t, r, 3, 100*time.Millisecond)
}

func TestAWakeIsRetriedWhileAnotherPodHoldsTheLease(t *testing.T) {
	leases := newMemLeaseRepo()
	r := startLoopRefresher(t, leases, "pod-a", SalesFactRefresherConfig{
		TickInterval: 20 * time.Millisecond,
		IdleInterval: time.Hour,
		WakeDelay:    10 * time.Millisecond,
	}, idle)
	waitRuns(t, r, 1)

	// Pod B is mid-run: it may have listed the dirty marks before this wake's mark was written.
	ok, err := leases.Acquire(context.Background(), salesFactLeaseName, "pod-b", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	r.Wake()
	requireRunsStay(t, r, 1, 100*time.Millisecond)

	require.NoError(t, leases.Release(context.Background(), salesFactLeaseName, "pod-b"))

	waitRuns(t, r, 2)
	requireRunsStay(t, r, 2, 100*time.Millisecond)
}

func TestOnlyOnePodRunsAtATime(t *testing.T) {
	leases := newMemLeaseRepo()
	var running, overlaps atomic.Int32
	tick := func(int32) bool {
		if running.Add(1) > 1 {
			overlaps.Add(1)
		}
		time.Sleep(5 * time.Millisecond)
		running.Add(-1)
		return true
	}
	cfg := SalesFactRefresherConfig{TickInterval: time.Millisecond, IdleInterval: time.Hour, WakeDelay: time.Millisecond}
	a := startLoopRefresher(t, leases, "pod-a", cfg, tick)
	b := startLoopRefresher(t, leases, "pod-b", cfg, tick)

	// Either pod may win every race; what matters is that their runs never overlap.
	require.Eventually(t, func() bool { return a.runs.Load()+b.runs.Load() >= 20 }, 2*time.Second, 2*time.Millisecond)
	require.Zero(t, overlaps.Load())
}

func TestStopEndsTheLoop(t *testing.T) {
	r := &SalesFactRefresher{
		cfg:    (&SalesFactRefresherConfig{Lease: lease.NewWithHolder(newMemLeaseRepo(), "pod-a"), TickInterval: time.Millisecond}).WithDefaults(),
		wake:   make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
	r.tick = func(context.Context) bool { return true }
	require.NoError(t, r.Start(context.Background()))

	done := make(chan struct{})
	go func() { r.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not return")
	}
	r.Wake() // never blocks, even with no loop to receive it
}
