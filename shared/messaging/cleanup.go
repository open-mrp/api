package messaging

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/lease"
)

const (
	defaultCleanupInterval         = 24 * time.Hour
	defaultCleanupBatchSize        = 1000
	defaultCleanupMaxBatchesPerRun = 100
	defaultCleanupLeaseName        = "idempotency-cleanup"
	defaultCleanupLeaseTTL         = 5 * time.Minute
	defaultCleanupScheduleTZ       = "America/New_York"
)

// CleanupConfig holds the configuration for the cleanup worker, which deletes expired rows from the idempotency key tables and old deleted_record entries.
type CleanupConfig struct {
	// Interval (optional; default: 24h) is how often the cleanup loop fires after the first run. Each tick triggers a full run that processes up to MaxBatchesPerRun batches for each table.
	Interval time.Duration

	// BatchSize (optional; default: 1000) is the maximum number of expired rows deleted in a single SQL DELETE statement. Keeping this bounded prevents table-lock escalation on MySQL and limits the replication lag each batch introduces.
	BatchSize int

	// MaxBatchesPerRun (optional; default: 100) caps the total number of sequential DELETE batches executed for each table in a single cleanup run. The effective ceiling per run is BatchSize * MaxBatchesPerRun rows per table.
	MaxBatchesPerRun int

	// LeaseName (optional; default: "idempotency-cleanup") identifies the distributed lease this worker acquires before each run so only one pod does the DELETE work.
	LeaseName string

	// LeaseTTL (optional; default: 5m) is how long the lease is held before expiring. A full cleanup run typically completes within a few seconds; the TTL is only a safety net for a crashed holder.
	LeaseTTL time.Duration

	// ScheduleLocation (optional; default: America/New_York) is the timezone used to determine when "midnight" falls for the first scheduled run. The worker waits until the next midnight in this location before running for the first time, then repeats every Interval thereafter.
	ScheduleLocation *time.Location
}

// WithDefaults returns a new CleanupConfig with zero-value fields replaced by production defaults.
func (c *CleanupConfig) WithDefaults() *CleanupConfig {
	if c == nil {
		c = &CleanupConfig{}
	}

	loc := c.ScheduleLocation
	if loc == nil {
		var err error
		loc, err = time.LoadLocation(defaultCleanupScheduleTZ)
		if err != nil {
			slog.Warn("Cleanup worker: could not load schedule timezone, falling back to UTC", "tz", defaultCleanupScheduleTZ, "error", err)
			loc = time.UTC
		}
	}

	return &CleanupConfig{
		Interval:         cmp.Or(c.Interval, defaultCleanupInterval),
		BatchSize:        cmp.Or(c.BatchSize, defaultCleanupBatchSize),
		MaxBatchesPerRun: cmp.Or(c.MaxBatchesPerRun, defaultCleanupMaxBatchesPerRun),
		LeaseName:        cmp.Or(c.LeaseName, defaultCleanupLeaseName),
		LeaseTTL:         cmp.Or(c.LeaseTTL, defaultCleanupLeaseTTL),
		ScheduleLocation: loc,
	}
}

// validate checks that all CleanupConfig fields are valid after defaults are applied.
func (c *CleanupConfig) validate() error {
	if c.Interval <= 0 {
		return fmt.Errorf("cleanup: interval must be positive")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("cleanup: batch size must be positive")
	}
	if c.MaxBatchesPerRun <= 0 {
		return fmt.Errorf("cleanup: max batches per run must be positive")
	}
	if c.ScheduleLocation == nil {
		return fmt.Errorf("cleanup: schedule location must not be nil")
	}
	return nil
}

// CleanupRepo defines the persistence interface for deleting expired idempotency keys and old deleted records. The platform-service implements this interface. Methods accept a row limit so the caller can bound DELETE scope and iterate in batches.
type CleanupRepo interface {
	// DeleteExpiredIdempotencyKeys deletes up to `limit` rows from the idempotency_key table whose expires_at timestamp has passed. These are the api-gateway–level keys used to deduplicate inbound HTTP requests. Returns the number of rows actually deleted; the caller uses this to decide whether another batch is needed (deleted < limit means the table is caught up).
	DeleteExpiredIdempotencyKeys(ctx context.Context, limit int) (int64, error)

	// DeleteExpiredServiceIdempotencyKeys deletes up to `limit` rows from the service_idempotency_key table whose expires_at timestamp has passed. These are the service-level keys used by individual gRPC handlers to prevent duplicate side effects. Returns the number of rows actually deleted.
	DeleteExpiredServiceIdempotencyKeys(ctx context.Context, limit int) (int64, error)

	// DeleteExpiredDeletedRecords deletes up to `limit` rows from deleted_record older than the retention window. Returns the number of rows actually deleted.
	DeleteExpiredDeletedRecords(ctx context.Context, limit int) (int64, error)

	// DeleteExpiredRequestLogs deletes up to `limit` rows from request_log older than 7 years. Returns the number of rows actually deleted.
	DeleteExpiredRequestLogs(ctx context.Context, limit int) (int64, error)

	// DeleteExpiredAuditEvents deletes up to `limit` rows from audit_event older than 7 years. Returns the number of rows actually deleted.
	DeleteExpiredAuditEvents(ctx context.Context, limit int) (int64, error)
}

// CleanupWorker runs a single background goroutine that periodically purges expired idempotency keys and old deleted records. It waits until the next midnight in the configured ScheduleLocation before its first run, then repeats every CleanupConfig.Interval.
//
// The worker uses appctx.WithNoTrace to suppress trace spans for its background operations, and respects context cancellation — calling Stop blocks until the goroutine exits cleanly.
type CleanupWorker struct {
	config CleanupConfig
	repo   CleanupRepo
	lease  *lease.Lease

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewCleanupWorker creates a new cleanup worker. A non-nil lease is required — each run claims the configured lease so that only one pod across the cluster runs the DELETEs per tick.
func NewCleanupWorker(config *CleanupConfig, repo CleanupRepo, l *lease.Lease) (*CleanupWorker, error) {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	if l == nil {
		return nil, fmt.Errorf("cleanup: lease is required")
	}

	return &CleanupWorker{
		config: *config,
		repo:   repo,
		lease:  l,
	}, nil
}

// Start launches the background cleanup goroutine. The provided context is used as the parent for all cleanup operations; cancelling it (or calling Stop) shuts down the loop. Tracing is disabled on the derived context so cleanup polling does not generate trace spans.
func (w *CleanupWorker) Start(ctx context.Context) error {
	ctx = appctx.WithNoTrace(ctx)
	w.ctx, w.cancel = context.WithCancel(ctx)

	w.wg.Add(1)
	go w.cleanupLoop()

	nextRun := nextMidnight(w.config.ScheduleLocation)
	slog.Info("Cleanup worker started",
		"interval", w.config.Interval,
		"batch_size", w.config.BatchSize,
		"max_batches_per_run", w.config.MaxBatchesPerRun,
		"schedule_tz", w.config.ScheduleLocation.String(),
		"next_run_at", nextRun,
	)

	return nil
}

// Stop cancels the background context and blocks until the cleanup goroutine has exited. It is safe to call from a deferred shutdown path.
func (w *CleanupWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
	slog.Info("Cleanup worker stopped")
}

// nextMidnight returns the next midnight (00:00:00) in loc, always in the future.
func nextMidnight(loc *time.Location) time.Time {
	now := time.Now().In(loc)
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	if !midnight.After(now) {
		midnight = midnight.AddDate(0, 0, 1)
	}
	return midnight
}

// cleanupLoop waits until the next midnight in the configured timezone before the first run, then repeats every Interval. Each run is wrapped in a distributed lease so only one pod across the cluster executes the DELETEs; other pods skip the tick without error. It exits when the worker's context is cancelled.
func (w *CleanupWorker) cleanupLoop() {
	defer w.wg.Done()

	initialTimer := time.NewTimer(time.Until(nextMidnight(w.config.ScheduleLocation)))
	defer initialTimer.Stop()

	select {
	case <-w.ctx.Done():
		return
	case <-initialTimer.C:
	}

	w.runUnderLease()

	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.runUnderLease()
		}
	}
}

func (w *CleanupWorker) runUnderLease() {
	_ = w.lease.WithLease(w.ctx, w.config.LeaseName, w.config.LeaseTTL, func(ctx context.Context) error {
		w.runCleanup(ctx)
		return nil
	})
}

// runCleanup performs a full cleanup pass across all cleanup targets. It delegates batch iteration to cleanupTable and logs a summary when at least one row was deleted.
func (w *CleanupWorker) runCleanup(ctx context.Context) {
	apiDeleted := w.cleanupTable(ctx, "idempotency_key", w.repo.DeleteExpiredIdempotencyKeys)
	serviceDeleted := w.cleanupTable(ctx, "service_idempotency_key", w.repo.DeleteExpiredServiceIdempotencyKeys)
	deletedRecordsDeleted := w.cleanupTable(ctx, "deleted_record", w.repo.DeleteExpiredDeletedRecords)
	requestLogsDeleted := w.cleanupTable(ctx, "request_log", w.repo.DeleteExpiredRequestLogs)
	auditEventsDeleted := w.cleanupTable(ctx, "audit_event", w.repo.DeleteExpiredAuditEvents)

	if apiDeleted > 0 || serviceDeleted > 0 || deletedRecordsDeleted > 0 || requestLogsDeleted > 0 || auditEventsDeleted > 0 {
		slog.Info("Cleanup completed",
			"api_keys_deleted", apiDeleted,
			"service_keys_deleted", serviceDeleted,
			"deleted_records_deleted", deletedRecordsDeleted,
			"request_logs_deleted", requestLogsDeleted,
			"audit_events_deleted", auditEventsDeleted,
		)
	}
}

// cleanupTable iterates up to MaxBatchesPerRun batches for a single table, calling deleteFunc (which issues a bounded DELETE) on each iteration. It stops early when the context is cancelled, an error occurs, or the last batch returned fewer rows than BatchSize (indicating the table is caught up). Returns the cumulative count of deleted rows.
func (w *CleanupWorker) cleanupTable(ctx context.Context, tableName string, deleteFunc func(ctx context.Context, limit int) (int64, error)) int64 {
	var totalDeleted int64

	for batch := 0; batch < w.config.MaxBatchesPerRun; batch++ {
		if ctx.Err() != nil {
			break
		}

		deleted, err := deleteFunc(ctx, w.config.BatchSize)
		if err != nil {
			slog.Error("Failed cleanup delete",
				"table", tableName,
				"error", err,
			)
			break
		}

		totalDeleted += deleted

		if deleted < int64(w.config.BatchSize) {
			break
		}
	}

	return totalDeleted
}
