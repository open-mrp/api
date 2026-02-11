package messaging

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type mockCleanupRepo struct {
	mu                 sync.Mutex
	apiDeleteCalls     int
	serviceDeleteCalls int
	apiDeleteFunc      func(ctx context.Context, limit int) (int64, error)
	serviceDeleteFunc  func(ctx context.Context, limit int) (int64, error)
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

func (m *mockCleanupRepo) getCallCounts() (int, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.apiDeleteCalls, m.serviceDeleteCalls
}

func TestCleanupWorkerStartStop(t *testing.T) {
	repo := &mockCleanupRepo{}
	worker, err := NewCleanupWorker(&CleanupConfig{
		Interval:         time.Hour,
		BatchSize:        100,
		MaxBatchesPerRun: 10,
	}, repo)
	require.NoError(t, err)

	err = worker.Start(context.Background())
	require.NoError(t, err)

	// Give it a moment to run the initial cleanup
	time.Sleep(50 * time.Millisecond)

	worker.Stop()

	// Verify cleanup was called at least once (initial run on startup)
	apiCalls, serviceCalls := repo.getCallCounts()
	require.GreaterOrEqual(t, apiCalls, 1)
	require.GreaterOrEqual(t, serviceCalls, 1)
}

func TestCleanupWorkerDeletesBatches(t *testing.T) {
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
	}

	worker, err := NewCleanupWorker(&CleanupConfig{
		Interval:         time.Hour,
		BatchSize:        100,
		MaxBatchesPerRun: 10,
	}, repo)
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
	apiBatchesDeleted := 0
	repo := &mockCleanupRepo{
		apiDeleteFunc: func(ctx context.Context, limit int) (int64, error) {
			apiBatchesDeleted++
			return int64(limit), nil // Always return full batch
		},
		serviceDeleteFunc: func(ctx context.Context, limit int) (int64, error) {
			return 0, nil
		},
	}

	worker, err := NewCleanupWorker(&CleanupConfig{
		Interval:         time.Hour,
		BatchSize:        100,
		MaxBatchesPerRun: 5, // Limit to 5 batches
	}, repo)
	require.NoError(t, err)

	err = worker.Start(context.Background())
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	worker.Stop()

	// Should have stopped at max batches
	require.Equal(t, 5, apiBatchesDeleted)
}

func TestCleanupWorkerHandlesErrors(t *testing.T) {
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
	}

	worker, err := NewCleanupWorker(&CleanupConfig{
		Interval:         time.Hour,
		BatchSize:        100,
		MaxBatchesPerRun: 10,
	}, repo)
	require.NoError(t, err)

	err = worker.Start(context.Background())
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	worker.Stop()

	// Should have stopped after the error
	require.Equal(t, 1, apiBatchesDeleted)
}

func TestCleanupConfigWithDefaults(t *testing.T) {
	config := new(CleanupConfig).WithDefaults()

	require.Equal(t, 24*time.Hour, config.Interval)
	require.Equal(t, 1000, config.BatchSize)
	require.Equal(t, 100, config.MaxBatchesPerRun)
}
