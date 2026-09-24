package db

import (
	"context"
	"sync"
)

type afterCommitKey struct{}

// AfterCommitScope collects callbacks registered with AfterCommit during one transaction attempt
// and runs them once that attempt commits. A transaction manager opens one per attempt with
// BeginAfterCommitScope; an attempt that rolls back (including one re-run after a lock conflict)
// simply drops its scope, so its callbacks never fire.
type AfterCommitScope struct {
	mu        sync.Mutex
	fns       []func()
	committed bool
}

// BeginAfterCommitScope returns a context carrying a fresh scope. Pass the returned context to
// the transaction's callback and call Committed on the scope after a successful commit.
func BeginAfterCommitScope(ctx context.Context) (context.Context, *AfterCommitScope) {
	s := &AfterCommitScope{}
	return context.WithValue(ctx, afterCommitKey{}, s), s
}

// Committed runs every callback registered so far. Callbacks registered afterwards (e.g. from a
// goroutine still holding the transaction's context) run immediately, since their writes are
// already visible.
func (s *AfterCommitScope) Committed() {
	s.mu.Lock()
	fns := s.fns
	s.fns = nil
	s.committed = true
	s.mu.Unlock()

	for _, fn := range fns {
		fn()
	}
}

// AfterCommit runs fn once the transaction carried by ctx commits, or immediately when ctx
// carries no transaction (an autocommit write is already visible). Use it for side effects that
// must not observe uncommitted state, such as waking a poller to read a just-written row.
func AfterCommit(ctx context.Context, fn func()) {
	s, _ := ctx.Value(afterCommitKey{}).(*AfterCommitScope)
	if s == nil {
		fn()
		return
	}

	s.mu.Lock()
	if s.committed {
		s.mu.Unlock()
		fn()
		return
	}
	s.fns = append(s.fns, fn)
	s.mu.Unlock()
}
