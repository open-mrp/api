package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/lease"
	"github.com/open-mrp/api/shared/tracing"
	"go.opentelemetry.io/otel/attribute"
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

	// burnRateSweepBatchSize caps how many stale items one tick reads. This, not the poll interval, is the
	// guard against a thundering herd: even when every item is stale (e.g. first deploy), the backlog
	// drains a bounded batch per tick rather than flooding the consumer at once.
	burnRateSweepBatchSize = 500

	burnRateSweepMessageItems = 100

	burnRateWindow = 30 * 24 * time.Hour
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
	now            func() time.Time

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
		now:            time.Now,
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

// enqueueStaleRecalcs reads up to batchSize of the stalest items and enqueues a recompute, one command per batch of an account's items, for each whose result can have changed since its last recompute: one never recomputed, or one with a change log that has since left the 30-day window. A rate is computed only from the logs in that window, and new logs enqueue their own recompute, so for every other item a recompute would write the same rate; those are only marked computed.
//
// The tick only enqueues; the actual recompute runs on the shared consumer, one short transaction per item. It never recomputes inline, which would hold the lease for the length of the whole batch and serialize the sweep behind it.
func (s *burnRateScheduler) enqueueStaleRecalcs(ctx context.Context) {
	ctx, span := burnRateSweepTracer.Start(ctx, "service.burn_rate_scheduler.enqueue_stale")
	defer span.End()

	now := s.now().UTC()
	itemRepo := s.repos.NewItemRepo()
	items, apiErr := itemRepo.ListStaleBurnRateItems(ctx, now.Add(-s.staleThreshold), s.batchSize)
	if apiErr != nil {
		slog.Error("Burn rate sweep: failed to list stale items", "error", apiErr)
		return
	}

	var accounts []string
	byAccount := map[string][]domain.StaleBurnRateItem{}
	for _, item := range items {
		if _, ok := byAccount[item.AccountID]; !ok {
			accounts = append(accounts, item.AccountID)
		}
		byAccount[item.AccountID] = append(byAccount[item.AccountID], item)
	}

	windowStart := now.Add(-burnRateWindow)
	enqueued, skipped := 0, 0
	for _, accountID := range accounts {
		recompute, unchanged, apiErr := s.partitionStale(ctx, accountID, byAccount[accountID], windowStart)
		if apiErr != nil {
			slog.Error("Burn rate sweep: failed to read change logs", "account_id", accountID, "error", apiErr)
			continue
		}
		if len(unchanged) > 0 {
			if apiErr := itemRepo.MarkBurnRatesComputed(ctx, unchanged); apiErr != nil {
				slog.Error("Burn rate sweep: failed to mark unchanged items", "account_id", accountID, "error", apiErr)
			} else {
				skipped += len(unchanged)
			}
		}
		for start := 0; start < len(recompute); start += burnRateSweepMessageItems {
			batch := recompute[start:min(start+burnRateSweepMessageItems, len(recompute))]
			if apiErr := mediator.EnqueueRecalcBatch(ctx, s.repos, accountID, batch); apiErr != nil {
				// The items stay stale, so a later tick retries them.
				slog.Error("Burn rate sweep: failed to enqueue recalc", "account_id", accountID, "items", len(batch), "error", apiErr)
				continue
			}
			enqueued += len(batch)
		}
	}
	span.SetAttributes(attribute.Int("burn_rate.stale", len(items)), attribute.Int("burn_rate.enqueued", enqueued), attribute.Int("burn_rate.skipped", skipped))
}

func (s *burnRateScheduler) partitionStale(ctx context.Context, accountID string, items []domain.StaleBurnRateItem, windowStart time.Time) (recompute, unchanged []string, apiErr *apierror.APIError) {
	var since *time.Time
	var computed []string
	for _, item := range items {
		if item.ComputedAt == nil {
			recompute = append(recompute, item.ItemID)
			continue
		}
		computed = append(computed, item.ItemID)
		if from := item.ComputedAt.Add(-burnRateWindow); since == nil || from.Before(*since) {
			since = &from
		}
	}
	if len(computed) == 0 {
		return recompute, nil, nil
	}
	oldest, apiErr := s.repos.NewItemRepo().OldestChangeLogsSince(ctx, accountID, computed, *since)
	if apiErr != nil {
		return nil, nil, apiErr
	}
	for _, itemID := range computed {
		if at, ok := oldest[itemID]; ok && at.Before(windowStart) {
			recompute = append(recompute, itemID)
		} else {
			unchanged = append(unchanged, itemID)
		}
	}
	return recompute, unchanged, nil
}
