package messaging

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
)

// mockEnqueuerRepo serves scripted batches of outbox messages and records the
// MarkPublished / MarkFailed calls it receives.
type mockEnqueuerRepo struct {
	mu      sync.Mutex
	batches [][]*OutboxMessage

	acquireCalls   int
	publishedCalls [][]int64
	failedIDs      []int64
}

func (m *mockEnqueuerRepo) AcquireAndLock(_ context.Context, _ string, _ int, _ int) ([]*OutboxMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.acquireCalls++
	if len(m.batches) == 0 {
		return nil, nil
	}
	batch := m.batches[0]
	m.batches = m.batches[1:]
	return batch, nil
}

func (m *mockEnqueuerRepo) MarkPublished(_ context.Context, ids []int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishedCalls = append(m.publishedCalls, ids)
	return nil
}

// pushBatch appends a batch the poll loop can later acquire, safe to call while the loop runs.
func (m *mockEnqueuerRepo) pushBatch(b []*OutboxMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.batches = append(m.batches, b)
}

func (m *mockEnqueuerRepo) MarkFailed(_ context.Context, id int64, _ string, _ int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedIDs = append(m.failedIDs, id)
	return nil
}

func (m *mockEnqueuerRepo) CleanupExpiredLocks(context.Context, int32) (int64, error) {
	return 0, nil
}

func (m *mockEnqueuerRepo) PurgePublished(context.Context, int, int32) (int64, error) {
	return 0, nil
}

// mockEnqueuerBroker records published routing keys and fails for routing keys
// in failKeys.
type mockEnqueuerBroker struct {
	mu        sync.Mutex
	published []string
	failKeys  map[string]bool
}

func (b *mockEnqueuerBroker) PublishMessage(_ context.Context, _, routingKey string, _ contracts.AmqpMessage) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failKeys[routingKey] {
		return errors.New("broker unavailable")
	}
	b.published = append(b.published, routingKey)
	return nil
}

func (b *mockEnqueuerBroker) ConsumeMessages(context.Context, string, MessageHandler, ...ConsumeOption) error {
	return nil
}

func (b *mockEnqueuerBroker) IsReady() bool { return true }
func (b *mockEnqueuerBroker) Close()        {}

func outboxBatch(start, count int) []*OutboxMessage {
	batch := make([]*OutboxMessage, count)
	for i := range count {
		batch[i] = &OutboxMessage{
			ID:          int64(start + i),
			MessageID:   fmt.Sprintf("msg_%d", start+i),
			MessageType: "test.event",
			Destination: ApplicationExchange,
			RoutingKey:  fmt.Sprintf("rk_%d", start+i),
		}
	}
	return batch
}

func newTestEnqueuer(t *testing.T, repo OutboxEnqueuerRepo, broker MessageBroker, batchSize int) *Enqueuer {
	t.Helper()
	cfg := (&EnqueuerConfig{
		ServiceName:  "test-service",
		PlatformMode: constants.PlatformModeTest,
		BatchSize:    batchSize,
	}).WithDefaults()

	e := &Enqueuer{config: *cfg, repo: repo, broker: broker, notify: make(chan struct{}, 1)}
	e.ctx, e.cancel = context.WithCancel(context.Background())
	t.Cleanup(e.cancel)
	return e
}

// drainPending must keep acquiring batches until one comes back smaller than
// BatchSize — otherwise throughput is capped at BatchSize per poll tick.
func TestEnqueuerDrainPendingProcessesUntilShortBatch(t *testing.T) {
	t.Parallel()

	repo := &mockEnqueuerRepo{batches: [][]*OutboxMessage{
		outboxBatch(0, 3),  // full batch
		outboxBatch(10, 3), // full batch
		outboxBatch(20, 1), // short batch — drain stops here
	}}
	broker := &mockEnqueuerBroker{}
	e := newTestEnqueuer(t, repo, broker, 3)

	e.drainPending()

	if repo.acquireCalls != 3 {
		t.Errorf("expected 3 acquire calls (drain until short batch), got %d", repo.acquireCalls)
	}
	if len(broker.published) != 7 {
		t.Errorf("expected all 7 messages published, got %d", len(broker.published))
	}
}

// A short (or empty) first batch must end the drain immediately.
func TestEnqueuerDrainPendingStopsOnEmpty(t *testing.T) {
	t.Parallel()

	repo := &mockEnqueuerRepo{}
	e := newTestEnqueuer(t, repo, &mockEnqueuerBroker{}, 3)

	e.drainPending()

	if repo.acquireCalls != 1 {
		t.Errorf("expected exactly 1 acquire call on empty backlog, got %d", repo.acquireCalls)
	}
}

// Successfully published messages must be marked in a single set-based call per
// batch, and failures must be marked individually without polluting the
// published set.
func TestEnqueuerProcessBatchMarksPublishedAsBatch(t *testing.T) {
	t.Parallel()

	repo := &mockEnqueuerRepo{batches: [][]*OutboxMessage{outboxBatch(0, 3)}}
	broker := &mockEnqueuerBroker{failKeys: map[string]bool{"rk_1": true}}
	e := newTestEnqueuer(t, repo, broker, 3)

	acquired := e.processBatch()

	if acquired != 3 {
		t.Errorf("expected processBatch to report 3 acquired, got %d", acquired)
	}
	if len(repo.publishedCalls) != 1 {
		t.Fatalf("expected exactly 1 MarkPublished call for the batch, got %d", len(repo.publishedCalls))
	}
	if got := repo.publishedCalls[0]; len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Errorf("expected published ids [0 2], got %v", got)
	}
	if len(repo.failedIDs) != 1 || repo.failedIDs[0] != 1 {
		t.Errorf("expected failed ids [1], got %v", repo.failedIDs)
	}
}

// A batch where every publish fails must not call MarkPublished at all.
func TestEnqueuerProcessBatchAllFailed(t *testing.T) {
	t.Parallel()

	repo := &mockEnqueuerRepo{batches: [][]*OutboxMessage{outboxBatch(0, 2)}}
	broker := &mockEnqueuerBroker{failKeys: map[string]bool{"rk_0": true, "rk_1": true}}
	e := newTestEnqueuer(t, repo, broker, 2)

	e.processBatch()

	if len(repo.publishedCalls) != 0 {
		t.Errorf("expected no MarkPublished calls, got %d", len(repo.publishedCalls))
	}
	if len(repo.failedIDs) != 2 {
		t.Errorf("expected 2 failed ids, got %v", repo.failedIDs)
	}
}

// Cancelling the enqueuer context must stop the drain between batches.
func TestEnqueuerDrainPendingStopsOnContextCancel(t *testing.T) {
	t.Parallel()

	// Endless full batches; without the ctx check the drain would never stop.
	repo := &endlessRepo{}
	e := newTestEnqueuer(t, repo, &mockEnqueuerBroker{}, 2)
	e.cancel()

	e.drainPending()

	if repo.acquireCalls() != 0 {
		t.Errorf("expected no acquire calls after cancel, got %d", repo.acquireCalls())
	}
}

// Notify must be non-blocking and coalescing: repeated kicks collapse to a single pending wake-up
// (the drain that follows handles every available row), and a kick on a nil enqueuer is a safe no-op.
func TestEnqueuerNotifyIsNonBlockingAndCoalescing(t *testing.T) {
	t.Parallel()

	e := newTestEnqueuer(t, &mockEnqueuerRepo{}, &mockEnqueuerBroker{}, 1)

	// Several kicks with no reader draining must not block and must leave exactly one pending signal.
	for range 5 {
		e.Notify()
	}
	if got := len(e.notify); got != 1 {
		t.Fatalf("expected 1 coalesced pending notify, got %d", got)
	}

	// nil receiver must not panic.
	var nilEnq *Enqueuer
	nilEnq.Notify()
}

// A Notify kick must wake the poll loop to drain at once, even when the poll timer is set so far out
// that it would never fire on its own — this is the property that lets a chat run start immediately
// instead of waiting out the idle poll backoff.
func TestEnqueuerNotifyWakesIdlePollLoop(t *testing.T) {
	t.Parallel()

	repo := &mockEnqueuerRepo{}
	broker := &mockEnqueuerBroker{}

	// Production mode (no test-mode initial drain) + an hour-long interval, so the ONLY thing that can
	// trigger a drain during this test is a Notify kick.
	cfg := (&EnqueuerConfig{
		ServiceName:     "test-service",
		PlatformMode:    constants.PlatformModeProduction,
		BatchSize:       10,
		PollInterval:    time.Hour,
		MaxPollInterval: time.Hour,
	}).WithDefaults()
	e := &Enqueuer{config: *cfg, repo: repo, broker: broker, notify: make(chan struct{}, 1)}
	e.ctx, e.cancel = context.WithCancel(context.Background())

	e.wg.Add(1)
	go e.pollLoop()
	defer func() {
		e.cancel()
		e.wg.Wait()
	}()

	// Enqueue work while the loop idles on its hour-long timer, then kick it.
	repo.pushBatch(outboxBatch(0, 3))
	e.Notify()

	deadline := time.Now().Add(2 * time.Second)
	for {
		broker.mu.Lock()
		n := len(broker.published)
		broker.mu.Unlock()
		if n == 3 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("Notify did not wake the poll loop: published %d/3 within deadline", n)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// endlessRepo always returns a full batch.
type endlessRepo struct {
	mu    sync.Mutex
	calls int
}

func (m *endlessRepo) AcquireAndLock(_ context.Context, _ string, limit int, _ int) ([]*OutboxMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	return outboxBatch(m.calls*100, limit), nil
}

func (m *endlessRepo) acquireCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func (m *endlessRepo) MarkPublished(context.Context, []int64) error         { return nil }
func (m *endlessRepo) MarkFailed(context.Context, int64, string, int) error { return nil }
func (m *endlessRepo) CleanupExpiredLocks(context.Context, int32) (int64, error) {
	return 0, nil
}
func (m *endlessRepo) PurgePublished(context.Context, int, int32) (int64, error) {
	return 0, nil
}
