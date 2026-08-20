package mediator

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestThrottle(now *time.Time) *lastUsedThrottle {
	return &lastUsedThrottle{
		window: lastUsedMarkWindow,
		seen:   make(map[string]time.Time),
		now:    func() time.Time { return *now },
	}
}

func TestLastUsedThrottle_MarksFirstSightThenSuppressesInsideTheWindow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	throttle := newTestThrottle(&now)

	assert.True(t, throttle.shouldMark("acus_1"), "the first request must record a last-used time")
	assert.False(t, throttle.shouldMark("acus_1"), "an immediate repeat must not write again")

	now = now.Add(lastUsedMarkWindow - time.Second)
	assert.False(t, throttle.shouldMark("acus_1"), "still inside the window")

	now = now.Add(2 * time.Second)
	assert.True(t, throttle.shouldMark("acus_1"), "past the window, the timestamp is due again")
}

// Switching accounts must be recorded at once, because that is the case last_used_at exists to
// answer. A different account carries a different account-user id, so it is never suppressed.
func TestLastUsedThrottle_DoesNotSuppressADifferentAccountUser(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	throttle := newTestThrottle(&now)

	require.True(t, throttle.shouldMark("acus_account_a"))

	now = now.Add(time.Second)
	assert.True(t, throttle.shouldMark("acus_account_b"), "switching accounts must be written through")
}

func TestLastUsedThrottle_IgnoresAnEmptyID(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	assert.False(t, newTestThrottle(&now).shouldMark(""))
}

func TestLastUsedThrottle_SweepsEntriesOlderThanTheWindow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	throttle := newTestThrottle(&now)

	for i := range lastUsedSweepThreshold {
		require.True(t, throttle.shouldMark(string(rune('a'))+string(rune(i))))
	}
	require.Len(t, throttle.seen, lastUsedSweepThreshold)

	// Everything recorded above is now stale, so the next insert collects it.
	now = now.Add(lastUsedMarkWindow + time.Second)
	require.True(t, throttle.shouldMark("acus_fresh"))

	assert.Len(t, throttle.seen, 1, "the sweep must drop every entry past the window")
}

// Concurrent requests for one account user must produce exactly one write, not one per goroutine.
func TestLastUsedThrottle_OnlyOneConcurrentCallerWins(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	throttle := newTestThrottle(&now)

	const callers = 64
	var wg sync.WaitGroup
	var mu sync.Mutex
	marked := 0

	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if throttle.shouldMark("acus_1") {
				mu.Lock()
				marked++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, marked, "exactly one of %d concurrent callers may write", callers)
}
