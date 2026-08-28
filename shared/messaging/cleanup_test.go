package messaging

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/lease"
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

	worker.Stop()
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

	worker.runCleanup(context.Background())

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

	worker.runCleanup(context.Background())

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

	worker.runCleanup(context.Background())

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

func TestCleanupConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  CleanupConfig
		wantErr string
	}{
		{name: "valid", config: CleanupConfig{Interval: time.Hour, BatchSize: 100, MaxBatchesPerRun: 10, ScheduleLocation: time.UTC}},
		{name: "non-positive interval", config: CleanupConfig{Interval: -time.Hour, BatchSize: 100, MaxBatchesPerRun: 10, ScheduleLocation: time.UTC}, wantErr: "interval must be positive"},
		{name: "non-positive batch size", config: CleanupConfig{Interval: time.Hour, BatchSize: -1, MaxBatchesPerRun: 10, ScheduleLocation: time.UTC}, wantErr: "batch size must be positive"},
		{name: "non-positive max batches per run", config: CleanupConfig{Interval: time.Hour, BatchSize: 100, MaxBatchesPerRun: -1, ScheduleLocation: time.UTC}, wantErr: "max batches per run must be positive"},
		{name: "nil schedule location", config: CleanupConfig{Interval: time.Hour, BatchSize: 100, MaxBatchesPerRun: 10}, wantErr: "schedule location must not be nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestNewCleanupWorkerRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	// Defaults only fill zero values, so a negative interval survives into validate.
	worker, err := NewCleanupWorker(&CleanupConfig{Interval: -time.Hour}, &mockCleanupRepo{}, testLease())
	require.ErrorContains(t, err, "interval must be positive")
	require.Nil(t, worker)
}

func TestNewCleanupWorkerRequiresLease(t *testing.T) {
	t.Parallel()

	worker, err := NewCleanupWorker(&CleanupConfig{}, &mockCleanupRepo{}, nil)
	require.ErrorContains(t, err, "lease is required")
	require.Nil(t, worker)
}

func TestCleanupConfigWithDefaultsLoadsScheduleTimezone(t *testing.T) {
	t.Parallel()

	config := new(CleanupConfig).WithDefaults()
	require.Equal(t, defaultCleanupScheduleTZ, config.ScheduleLocation.String())
	require.Equal(t, defaultCleanupLeaseName, config.LeaseName)
	require.Equal(t, defaultCleanupLeaseTTL, config.LeaseTTL)

	explicit := (&CleanupConfig{ScheduleLocation: time.UTC}).WithDefaults()
	require.Equal(t, time.UTC, explicit.ScheduleLocation)
}

func TestNextMidnight(t *testing.T) {
	t.Parallel()

	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	for _, loc := range []*time.Location{time.UTC, newYork, time.FixedZone("half-hour", 30*60)} {
		t.Run(loc.String(), func(t *testing.T) {
			t.Parallel()

			now := time.Now().In(loc)
			next := nextMidnight(loc)

			require.True(t, next.After(now), "next midnight must always be in the future")
			require.Equal(t, 0, next.Hour())
			require.Equal(t, 0, next.Minute())
			require.Equal(t, 0, next.Second())
			require.Equal(t, 0, next.Nanosecond())
			require.Equal(t, loc, next.Location())
			require.LessOrEqual(t, next.Sub(now), 24*time.Hour, "next midnight is at most a day away")
		})
	}
}

func TestCleanupLoopWaitsForMidnightThenRepeatsEveryInterval(t *testing.T) {
	t.Parallel()

	runs := make(chan struct{}, 8)
	repo := &mockCleanupRepo{
		apiDeleteFunc: func(context.Context, int) (int64, error) {
			select {
			case runs <- struct{}{}:
			default:
			}
			return 0, nil
		},
	}

	loc := zoneWhereMidnightIsNear(2 * time.Second)
	untilMidnight := time.Until(nextMidnight(loc))
	require.Greater(t, untilMidnight, 500*time.Millisecond, "the fixture must leave a measurable wait before midnight")

	worker, err := NewCleanupWorker(&CleanupConfig{
		Interval:         100 * time.Millisecond,
		BatchSize:        100,
		MaxBatchesPerRun: 1,
		ScheduleLocation: loc,
	}, repo, testLease())
	require.NoError(t, err)

	start := time.Now()
	require.NoError(t, worker.Start(context.Background()))
	defer worker.Stop()

	select {
	case <-runs:
	case <-time.After(30 * time.Second):
		t.Fatal("cleanup never ran at the scheduled midnight")
	}
	require.GreaterOrEqual(t, time.Since(start), untilMidnight-50*time.Millisecond, "the first run must wait for midnight, not fire immediately")

	select {
	case <-runs:
	case <-time.After(30 * time.Second):
		t.Fatal("cleanup did not repeat on the configured interval")
	}
}

// zoneWhereMidnightIsNear builds a whole-second fixed-offset location whose local clock sits just under `d` before midnight, so a scheduling test can exercise the real midnight-then-interval path without waiting for a real day boundary.
func zoneWhereMidnightIsNear(d time.Duration) *time.Location {
	now := time.Now().UTC()
	sinceUTCMidnight := now.Hour()*3600 + now.Minute()*60 + now.Second()

	offset := 86400 - int(d.Seconds()) - sinceUTCMidnight
	for offset <= -43200 {
		offset += 86400
	}
	for offset > 43200 {
		offset -= 86400
	}

	return time.FixedZone("test-midnight", offset)
}
