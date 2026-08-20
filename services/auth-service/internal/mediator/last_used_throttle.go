package mediator

import (
	"sync"
	"time"
)

// lastUsedMarkWindow is how long a written last_used_at is treated as still current. It bounds
// how stale the value can be, which matters only for picking the account a returning user lands
// on: a user who switches accounts is marked at once (a different account user, so a different
// key), and only switching *back* inside the window can leave the older timestamp standing.
const lastUsedMarkWindow = 5 * time.Minute

// lastUsedSweepThreshold is the entry count above which a sweep runs before inserting. Entries are
// account-user IDs seen in the last window, so the steady-state size is the process's concurrent
// user count; the threshold only has to be high enough that sweeps are rare.
const lastUsedSweepThreshold = 4096

// lastUsedMarks throttles last_used_at writes across every mediator built in this process.
//
// Mediators are rebuilt per request (see mediatorFactoryImpl.Build), so this deliberately lives at
// package scope: per-instance state would never see a second request and would throttle nothing.
// Replicas each keep their own map, so the write rate falls to at most one per account user per
// window per replica rather than one per request.
var lastUsedMarks = &lastUsedThrottle{
	window: lastUsedMarkWindow,
	seen:   make(map[string]time.Time),
	now:    time.Now,
}

type lastUsedThrottle struct {
	mu     sync.Mutex
	window time.Duration
	seen   map[string]time.Time
	now    func() time.Time
}

// shouldMark reports whether this account user's last_used_at is due for a write, recording the
// write as having happened when it returns true.
func (t *lastUsedThrottle) shouldMark(accountUserID string) bool {
	if accountUserID == "" {
		return false
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now()
	if marked, ok := t.seen[accountUserID]; ok && now.Sub(marked) < t.window {
		return false
	}

	if len(t.seen) >= lastUsedSweepThreshold {
		for id, marked := range t.seen {
			if now.Sub(marked) >= t.window {
				delete(t.seen, id)
			}
		}
	}

	t.seen[accountUserID] = now
	return true
}
