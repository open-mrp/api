package messaging

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/augno/api/shared/lease"
	"github.com/stretchr/testify/require"
)

// alwaysGrantLeaseRepo satisfies lease.Repo and always grants, never loses, the lease.
// Used by tests that want to exercise the cleanup/purge body without standing up a DB.
type alwaysGrantLeaseRepo struct{}

func (alwaysGrantLeaseRepo) Acquire(context.Context, string, string, time.Duration) (bool, error) {
	return true, nil
}
func (alwaysGrantLeaseRepo) Renew(context.Context, string, string, time.Duration) (bool, error) {
	return true, nil
}
func (alwaysGrantLeaseRepo) Release(context.Context, string, string) error { return nil }

func testLease() *lease.Lease { return lease.NewWithHolder(alwaysGrantLeaseRepo{}, "test-holder") }

type mockCleanupRepo struct {
	mu                 sync.Mutex
	apiDeleteCalls     int
	serviceDeleteCalls int
	deletedRecordCalls int
	requestLogCalls    int
	auditEventCalls    int
	apiDeleteFunc      func(ctx context.Context, limit int) (int64, error)
	serviceDeleteFunc  func(ctx context.Context, limit int) (int64, error)
	deletedRecordFunc  func(ctx context.Context, limit int) (int64, error)
	requestLogFunc     func(ctx context.Context, limit int) (int64, error)
	auditEventFunc     func(ctx context.Context, limit int) (int64, error)
}

func (m *mockCleanupRepo) DeleteExpiredIdempotencyKeys(ctx context.Context, limit int) (int64, error) {
	m.mu.Lock()
	m.apiDeleteCalls++
	m.mu.Unlock()
	if m.apiDeleteFunc != nil {
		return m.apiDeleteFunc(ctx, limit)
	}
	return 0, nil
}

func (m *mockCleanupRepo) DeleteExpiredServiceIdempotencyKeys(ctx context.Context, limit int) (int64, error) {
	m.mu.Lock()
	m.serviceDeleteCalls++
	m.mu.Unlock()
	if m.serviceDeleteFunc != nil {
		return m.serviceDeleteFunc(ctx, limit)
	}
	return 0, nil
}

func (m *mockCleanupRepo) DeleteExpiredDeletedRecords(ctx context.Context, limit int) (int64, error) {
	m.mu.Lock()
	m.deletedRecordCalls++
	m.mu.Unlock()
	if m.deletedRecordFunc != nil {
		return m.deletedRecordFunc(ctx, limit)
	}
	return 0, nil
}

func (m *mockCleanupRepo) DeleteExpiredRequestLogs(ctx context.Context, limit int) (int64, error) {
	m.mu.Lock()
	m.requestLogCalls++
	m.mu.Unlock()
	if m.requestLogFunc != nil {
		return m.requestLogFunc(ctx, limit)
	}
	return 0, nil
}

func (m *mockCleanupRepo) DeleteExpiredAuditEvents(ctx context.Context, limit int) (int64, error) {
	m.mu.Lock()
	m.auditEventCalls++
	m.mu.Unlock()
	if m.auditEventFunc != nil {
		return m.auditEventFunc(ctx, limit)
	}
	return 0, nil
}

func (m *mockCleanupRepo) getCallCounts() (int, int, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.apiDeleteCalls, m.serviceDeleteCalls, m.deletedRecordCalls
}

func TestCleanupWorkerStartStop(t *testing.T) {
	t.Parallel()
	repo := &mockCleanupRepo{}
	worker, err := NewCleanupWorker(&CleanupConfig{
		Interval:         time.Hour,
		BatchSize:        100,
		MaxBatchesPerRun: 10,
	}, repo, testLease())
	require.NoError(t, err)

	err = worker.Start(context.Background())
	require.NoError(t, err)

	// Give it a moment to run the initial cleanup
	time.Sleep(50 * time.Millisecond)

	worker.Stop()

	// Verify cleanup was called at least once (initial run on startup)
	apiCalls, serviceCalls, deletedRecordCalls := repo.getCallCounts()
	require.GreaterOrEqual(t, apiCalls, 1)
	require.GreaterOrEqual(t, serviceCalls, 1)
	require.GreaterOrEqual(t, deletedRecordCalls, 1)
}

func TestCleanupWorkerDeletesBatches(t *testing.T) {
	t.Parallel()
	apiBatchesDeleted := 0
	serviceBatchesDeleted := 0
	repo := &mockCleanupRepo{
		apiDeleteFunc: func(ctx context.Context, limit int) (int64, error) {
			apiBatchesDeleted++
			if apiBatchesDeleted <= 2 {
				return int64(limit), nil // Full batch, continue
			}
			return 50, nil // Partial batch, stop
		},
		serviceDeleteFunc: func(ctx context.Context, limit int) (int64, error) {
			serviceBatchesDeleted++
			return 0, nil // No records to delete
		},
		deletedRecordFunc: func(ctx context.Context, limit int) (int64, error) {
			return 0, nil
		},
	}

	worker, err := NewCleanupWorker(&CleanupConfig{
		Interval:         time.Hour,
		BatchSize:        100,
		MaxBatchesPerRun: 10,
	}, repo, testLease())
	require.NoError(t, err)

	err = worker.Start(context.Background())
	require.NoError(t, err)

	// Wait for initial cleanup to complete
	time.Sleep(50 * time.Millisecond)

	worker.Stop()

	// Should have deleted 3 batches for API (2 full + 1 partial)
	require.Equal(t, 3, apiBatchesDeleted)
	// Should have called service delete once (returned 0, so no more batches)
	require.Equal(t, 1, serviceBatchesDeleted)
}

func TestCleanupWorkerRespectsMaxBatches(t *testing.T) {
	t.Parallel()
	apiBatchesDeleted := 0
	repo := &mockCleanupRepo{
		apiDeleteFunc: func(ctx context.Context, limit int) (int64, error) {
			apiBatchesDeleted++
			return int64(limit), nil // Always return full batch
		},
		serviceDeleteFunc: func(ctx context.Context, limit int) (int64, error) {
			return 0, nil
		},
		deletedRecordFunc: func(ctx context.Context, limit int) (int64, error) {
			return 0, nil
		},
	}

	worker, err := NewCleanupWorker(&CleanupConfig{
		Interval:         time.Hour,
		BatchSize:        100,
		MaxBatchesPerRun: 5, // Limit to 5 batches
	}, repo, testLease())
	require.NoError(t, err)

	err = worker.Start(context.Background())
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	worker.Stop()

	// Should have stopped at max batches
	require.Equal(t, 5, apiBatchesDeleted)
}

func TestCleanupWorkerHandlesErrors(t *testing.T) {
	t.Parallel()
	expectedErr := errors.New("database error")
	apiBatchesDeleted := 0
	repo := &mockCleanupRepo{
		apiDeleteFunc: func(ctx context.Context, limit int) (int64, error) {
			apiBatchesDeleted++
			return 0, expectedErr
		},
		serviceDeleteFunc: func(ctx context.Context, limit int) (int64, error) {
			return 0, nil
		},
		deletedRecordFunc: func(ctx context.Context, limit int) (int64, error) {
			return 0, nil
		},
	}

	worker, err := NewCleanupWorker(&CleanupConfig{
		Interval:         time.Hour,
		BatchSize:        100,
		MaxBatchesPerRun: 10,
	}, repo, testLease())
	require.NoError(t, err)

	err = worker.Start(context.Background())
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	worker.Stop()

	// Should have stopped after the error
	require.Equal(t, 1, apiBatchesDeleted)
}

func TestCleanupConfigWithDefaults(t *testing.T) {
	t.Parallel()
	config := new(CleanupConfig).WithDefaults()

	require.Equal(t, 24*time.Hour, config.Interval)
	require.Equal(t, 1000, config.BatchSize)
	require.Equal(t, 100, config.MaxBatchesPerRun)
}
