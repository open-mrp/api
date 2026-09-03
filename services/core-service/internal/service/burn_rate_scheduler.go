package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	"github.com/open-mrp/api/shared/lease"
	"github.com/open-mrp/api/shared/tracing"
)

var burnRateSweepTracer = tracing.GetTracer("core-service.burn_rate_scheduler")

const (
	// burnRateSweepLeaseName ensures exactly one pod runs the sweep per tick.
	burnRateSweepLeaseName = "burn-rate-sweeper"
	burnRateSweepLeaseTTL  = 2 * time.Minute

	// burnRateSweepPollInterval is how often the sweeper looks for stale items. It is kept well above
	// the time a batch takes to drain through the recompute consumer, so a batch is recomputed — and
	// its items marked fresh — before the next tick, and the sweep advances instead of re-selecting.
	burnRateSweepPollInterval = 10 * time.Minute

	// burnRateStaleThreshold is how long an item's burn rate may go without a recompute before the
	// sweep refreshes it. Items with ongoing consumption are recomputed by the write path long before
	// this and never surface; this bounds only how stale an idle item's rate can get.
	burnRateStaleThreshold = 24 * time.Hour

	// burnRateSweepBatchSize caps how many items one tick enqueues. This, not the poll interval, is the
	// guard against a thundering herd: even when every item is stale (e.g. first deploy), the backlog
	// drains a bounded batch per tick rather than flooding the consumer at once.
	burnRateSweepBatchSize = 500
)

// BurnRateSchedulerConfig configures the periodic burn-rate sweep.
type BurnRateSchedulerConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// Lease (required) serializes the tick across pods.
	Lease *lease.Lease

	// PollInterval (optional; default: 10m) overrides the tick interval, for tests. Zero or negative values are treated as unset.
	PollInterval time.Duration

	// StaleThreshold (optional; default: 24h) overrides how old a burn rate must be to be swept, for tests. Zero or negative values are treated as unset.
	StaleThreshold time.Duration

	// BatchSize (optional; default: 500) overrides how many items a tick enqueues, for tests. Zero or negative values are treated as unset.
	BatchSize int32
}

// WithDefaults returns the config with default values applied where unset.
func (c *BurnRateSchedulerConfig) WithDefaults() *BurnRateSchedulerConfig {
	if c == nil {
		c = &BurnRateSchedulerConfig{}
	}
	if c.PollInterval <= 0 {
		c.PollInterval = burnRateSweepPollInterval
	}
	if c.StaleThreshold <= 0 {
		c.StaleThreshold = burnRateStaleThreshold
	}
	if c.BatchSize <= 0 {
		c.BatchSize = burnRateSweepBatchSize
	}
	return c
}

func (c *BurnRateSchedulerConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("burn rate scheduler: repos is required")
	}
	if c.Lease == nil {
		return fmt.Errorf("burn rate scheduler: lease is required")
	}
	return nil
}

type burnRateScheduler struct {
	repos          domain.RepoFactory
	lease          *lease.Lease
	pollInterval   time.Duration
	staleThreshold time.Duration
	batchSize      int32

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewBurnRateScheduler(config *BurnRateSchedulerConfig) *burnRateScheduler {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &burnRateScheduler{
		repos:          config.Repos,
		lease:          config.Lease,
		pollInterval:   config.PollInterval,
		staleThreshold: config.StaleThreshold,
		batchSize:      config.BatchSize,
		stopCh:         make(chan struct{}),
	}
}

func (s *burnRateScheduler) Start(ctx context.Context) error {
	s.wg.Add(1)
	go s.pollLoop(ctx)
	slog.Info("Burn rate sweeper started",
		"poll_interval", s.pollInterval, "stale_threshold", s.staleThreshold, "batch_size", s.batchSize)
	return nil
}

func (s *burnRateScheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	slog.Info("Burn rate sweeper stopped")
}

func (s *burnRateScheduler) pollLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			_ = s.lease.WithLease(ctx, burnRateSweepLeaseName, burnRateSweepLeaseTTL, func(leaseCtx context.Context) error {
				s.enqueueStaleRecalcs(leaseCtx)
				return nil
			})
		}
	}
}

// enqueueStaleRecalcs re-enqueues a burn-rate recompute for up to batchSize of the stalest items.
//
// The tick only enqueues; the actual recompute runs on the shared consumer, one short transaction per
// item. It never recomputes inline, which would hold the lease for the length of the whole batch and
// serialize the sweep behind it.
func (s *burnRateScheduler) enqueueStaleRecalcs(ctx context.Context) {
	ctx, span := burnRateSweepTracer.Start(ctx, "service.burn_rate_scheduler.enqueue_stale")
	defer span.End()

	staleBefore := time.Now().UTC().Add(-s.staleThreshold)
	items, apiErr := s.repos.NewItemRepo().ListStaleBurnRateItems(ctx, staleBefore, s.batchSize)
	if apiErr != nil {
		slog.Error("Burn rate sweep: failed to list stale items", "error", apiErr)
		return
	}

	for _, item := range items {
		if apiErr := mediator.EnqueueRecalc(ctx, s.repos, item.AccountID, item.ItemID); apiErr != nil {
			// One bad enqueue should not abort the batch; it is retried on the next tick, since the
			// item stays stale until a recompute lands.
			slog.Error("Burn rate sweep: failed to enqueue recalc",
				"account_id", item.AccountID, "item_id", item.ItemID, "error", apiErr)
		}
	}
}
