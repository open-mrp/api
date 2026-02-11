package messaging

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/augno/api/shared/appctx"
)

const (
	defaultCleanupInterval         = 24 * time.Hour
	defaultCleanupBatchSize        = 1000
	defaultCleanupMaxBatchesPerRun = 100
)

// CleanupConfig holds the configuration for the idempotency key cleanup worker.
// The worker deletes expired rows from both the api-gateway-level idempotency_key
// table and the service-level service_idempotency_key table.
type CleanupConfig struct {
	// Interval (optional; default: 24h) is how often the cleanup loop fires. Each tick
	// triggers a full run that processes up to MaxBatchesPerRun batches for each table.
	Interval time.Duration

	// BatchSize (optional; default: 1000) is the maximum number of expired rows deleted
	// in a single SQL DELETE statement. Keeping this bounded prevents table-lock escalation
	// on MySQL and limits the replication lag each batch introduces.
	BatchSize int

	// MaxBatchesPerRun (optional; default: 100) caps the total number of sequential DELETE
	// batches executed for each table in a single cleanup run. The effective ceiling per run
	// is BatchSize * MaxBatchesPerRun rows per table.
	MaxBatchesPerRun int
}

// WithDefaults returns a new CleanupConfig with zero-value fields replaced by production defaults.
func (c *CleanupConfig) WithDefaults() *CleanupConfig {
	if c == nil {
		c = &CleanupConfig{}
	}

	return &CleanupConfig{
		Interval:         cmp.Or(c.Interval, defaultCleanupInterval),
		BatchSize:        cmp.Or(c.BatchSize, defaultCleanupBatchSize),
		MaxBatchesPerRun: cmp.Or(c.MaxBatchesPerRun, defaultCleanupMaxBatchesPerRun),
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
	return nil
}

// CleanupRepo defines the persistence interface for deleting expired idempotency
// keys. The platform-service implements this interface against the idempotency_key
// and service_idempotency_key tables. Both methods accept a row limit so the caller
// can bound the DELETE scope and iterate in batches.
type CleanupRepo interface {
	// DeleteExpiredIdempotencyKeys deletes up to `limit` rows from the
	// idempotency_key table whose expires_at timestamp has passed. These are the
	// api-gateway–level keys used to deduplicate inbound HTTP requests. Returns
	// the number of rows actually deleted; the caller uses this to decide whether
	// another batch is needed (deleted < limit means the table is caught up).
	DeleteExpiredIdempotencyKeys(ctx context.Context, limit int) (int64, error)

	// DeleteExpiredServiceIdempotencyKeys deletes up to `limit` rows from the
	// service_idempotency_key table whose expires_at timestamp has passed. These
	// are the service-level keys used by individual gRPC handlers to prevent
	// duplicate side effects. Returns the number of rows actually deleted.
	DeleteExpiredServiceIdempotencyKeys(ctx context.Context, limit int) (int64, error)
}

// CleanupWorker runs a single background goroutine that periodically purges
// expired idempotency keys from both the api-gateway and service-level tables.
// It fires immediately on Start (so a fresh deployment doesn't wait a full
// Interval before its first cleanup) and then repeats at CleanupConfig.Interval.
//
// The worker uses appctx.WithNoTrace to suppress trace spans for its background
// operations, and respects context cancellation — calling Stop blocks until the
// goroutine exits cleanly.
type CleanupWorker struct {
	config CleanupConfig
	repo   CleanupRepo

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewCleanupWorker creates a new cleanup worker with the given configuration and
// repository. The worker does not start until Start is called.
func NewCleanupWorker(config *CleanupConfig, repo CleanupRepo) (*CleanupWorker, error) {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}

	return &CleanupWorker{
		config: *config,
		repo:   repo,
	}, nil
}

// Start launches the background cleanup goroutine. The provided context is used
// as the parent for all cleanup operations; cancelling it (or calling Stop) shuts
// down the loop. Tracing is disabled on the derived context so cleanup polling does
// not generate trace spans.
func (w *CleanupWorker) Start(ctx context.Context) error {
	// Disable tracing for background cleanup operations to avoid cluttering traces
	ctx = appctx.WithNoTrace(ctx)
	w.ctx, w.cancel = context.WithCancel(ctx)

	w.wg.Add(1)
	go w.cleanupLoop()

	slog.Info("Idempotency key cleanup worker started",
		"interval", w.config.Interval,
		"batch_size", w.config.BatchSize,
		"max_batches_per_run", w.config.MaxBatchesPerRun,
	)

	return nil
}

// Stop cancels the background context and blocks until the cleanup goroutine has
// exited. It is safe to call from a deferred shutdown path.
func (w *CleanupWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
	slog.Info("Idempotency key cleanup worker stopped")
}

// cleanupLoop runs cleanup immediately on startup and then on every Interval tick.
// It exits when the worker's context is cancelled.
func (w *CleanupWorker) cleanupLoop() {
	defer w.wg.Done()

	// Run cleanup immediately on startup
	w.runCleanup()

	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.runCleanup()
		}
	}
}

// runCleanup performs a full cleanup pass across both idempotency tables. It
// delegates batch iteration to cleanupTable and logs a summary when at least one
// row was deleted.
func (w *CleanupWorker) runCleanup() {
	apiDeleted := w.cleanupTable("idempotency_key", w.repo.DeleteExpiredIdempotencyKeys)
	serviceDeleted := w.cleanupTable("service_idempotency_key", w.repo.DeleteExpiredServiceIdempotencyKeys)

	if apiDeleted > 0 || serviceDeleted > 0 {
		slog.Info("Idempotency key cleanup completed",
			"api_keys_deleted", apiDeleted,
			"service_keys_deleted", serviceDeleted,
		)
	}
}

// cleanupTable iterates up to MaxBatchesPerRun batches for a single table,
// calling deleteFunc (which issues a bounded DELETE) on each iteration. It stops
// early when the context is cancelled, an error occurs, or the last batch
// returned fewer rows than BatchSize (indicating the table is caught up). Returns
// the cumulative count of deleted rows.
func (w *CleanupWorker) cleanupTable(tableName string, deleteFunc func(ctx context.Context, limit int) (int64, error)) int64 {
	var totalDeleted int64

	for batch := 0; batch < w.config.MaxBatchesPerRun; batch++ {
		if w.ctx.Err() != nil {
			break
		}

		deleted, err := deleteFunc(w.ctx, w.config.BatchSize)
		if err != nil {
			slog.Error("Failed to delete expired idempotency keys",
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
