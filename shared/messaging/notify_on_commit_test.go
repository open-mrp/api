package messaging

import (
	"context"
	"testing"

	"github.com/open-mrp/api/shared/db"
	"github.com/stretchr/testify/assert"
)

// NotifyOnCommit uses the process-wide running enqueuer, so these tests are not parallel.

func pendingKick(e *Enqueuer) bool {
	select {
	case <-e.notify:
		return true
	default:
		return false
	}
}

func withRunningEnqueuer(t *testing.T) *Enqueuer {
	t.Helper()
	e := newTestEnqueuer(t, &mockEnqueuerRepo{}, &mockEnqueuerBroker{}, 1)
	runningEnqueuer.Store(e)
	t.Cleanup(func() { runningEnqueuer.CompareAndSwap(e, nil) })
	return e
}

func TestNotifyOnCommit_WaitsForCommit(t *testing.T) {
	e := withRunningEnqueuer(t)
	ctx, scope := db.BeginAfterCommitScope(context.Background())

	NotifyOnCommit(ctx, OutboxMessageInput{})
	assert.False(t, pendingKick(e), "must not wake the poller before the row is committed")

	scope.Committed()
	assert.True(t, pendingKick(e))
}

func TestNotifyOnCommit_AutocommitWakesImmediately(t *testing.T) {
	e := withRunningEnqueuer(t)

	NotifyOnCommit(context.Background(), OutboxMessageInput{})
	assert.True(t, pendingKick(e))
}

func TestNotifyOnCommit_SkipsDelayedRows(t *testing.T) {
	e := withRunningEnqueuer(t)

	NotifyOnCommit(context.Background(), OutboxMessageInput{DelaySeconds: 30})
	assert.False(t, pendingKick(e))
}

func TestNotifyOnCommit_NoRunningEnqueuerIsHarmless(t *testing.T) {
	prev := runningEnqueuer.Swap(nil)
	t.Cleanup(func() { runningEnqueuer.CompareAndSwap(nil, prev) })

	assert.NotPanics(t, func() { NotifyOnCommit(context.Background(), OutboxMessageInput{}) })
}
