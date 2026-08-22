package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/lease"
	"github.com/open-mrp/api/shared/tracing"
	"github.com/robfig/cron/v3"
)

var scheduleCadenceTracer = tracing.GetTracer("core-service.production_schedule_scheduler")

const (
	// scheduleCadenceLeaseName ensures exactly one pod fires the cadence per tick.
	scheduleCadenceLeaseName = "production-schedule-scheduler"
	scheduleCadenceLeaseTTL  = 2 * time.Minute

	// A cron cadence is measured in days, so polling every minute is ample and keeps the query load negligible.
	scheduleCadencePollInterval = 1 * time.Minute

	// stalledGenerationThreshold is how long a version may sit in `generating` before it is treated as orphaned. A real solve is minutes; an hour means the process that was solving it died.
	stalledGenerationThreshold = 1 * time.Hour
)

// ProductionScheduleSchedulerConfig configures the generation cadence.
type ProductionScheduleSchedulerConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// Lease (required) serializes the tick across pods.
	Lease *lease.Lease

	// Svc (required) performs the transactional enqueue. The tick never solves in process; it reserves a version and queues the work.
	Svc domain.ProductionScheduleSvc

	// PollInterval (optional; default: 1m) overrides the tick interval, for tests. Zero or negative values are treated as unset.
	PollInterval time.Duration
}

// WithDefaults returns the config with default values applied where unset.
func (c *ProductionScheduleSchedulerConfig) WithDefaults() *ProductionScheduleSchedulerConfig {
	if c == nil {
		c = &ProductionScheduleSchedulerConfig{}
	}
	if c.PollInterval <= 0 {
		c.PollInterval = scheduleCadencePollInterval
	}
	return c
}

func (c *ProductionScheduleSchedulerConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("production schedule scheduler: repos is required")
	}
	if c.Lease == nil {
		return fmt.Errorf("production schedule scheduler: lease is required")
	}
	if c.Svc == nil {
		return fmt.Errorf("production schedule scheduler: svc is required")
	}
	return nil
}

type productionScheduleScheduler struct {
	repos        domain.RepoFactory
	lease        *lease.Lease
	svc          domain.ProductionScheduleSvc
	pollInterval time.Duration

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewProductionScheduleScheduler(config *ProductionScheduleSchedulerConfig) *productionScheduleScheduler {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &productionScheduleScheduler{
		repos:        config.Repos,
		lease:        config.Lease,
		svc:          config.Svc,
		pollInterval: config.PollInterval,
		stopCh:       make(chan struct{}),
	}
}

func (s *productionScheduleScheduler) Start(ctx context.Context) error {
	s.wg.Add(1)
	go s.pollLoop(ctx)
	slog.Info("Production schedule cadence started", "poll_interval", s.pollInterval)
	return nil
}

func (s *productionScheduleScheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	slog.Info("Production schedule cadence stopped")
}

func (s *productionScheduleScheduler) pollLoop(ctx context.Context) {
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
			_ = s.lease.WithLease(ctx, scheduleCadenceLeaseName, scheduleCadenceLeaseTTL, func(leaseCtx context.Context) error {
				s.enqueueDueAccounts(leaseCtx)
				s.reapStalledGenerations(leaseCtx)
				return nil
			})
		}
	}
}

// enqueueDueAccounts fires the cadence for every account whose cron says it is due.
//
// The tick only enqueues. A solve takes minutes on a real tenant, and doing it here would hold the lease and block every other account behind whichever one is currently solving — the cadence would silently degrade to one account per solve duration.
func (s *productionScheduleScheduler) enqueueDueAccounts(ctx context.Context) {
	ctx, span := scheduleCadenceTracer.Start(ctx, "service.production_schedule_scheduler.enqueue_due")
	defer span.End()

	repo := s.repos.NewProductionScheduleRepo()

	cadences, apiErr := repo.ListGenerationCadences(ctx)
	if apiErr != nil {
		slog.Error("Schedule cadence: failed to list cadences", "error", apiErr)
		return
	}

	now := time.Now().UTC()

	for _, cadence := range cadences {
		schedule, err := cron.ParseStandard(cadence.Cron)
		if err != nil {
			// A bad expression is a configuration problem, not a reason to stop the whole cadence for every other account.
			slog.Error("Schedule cadence: invalid cron expression",
				"account_id", cadence.AccountID, "cron", cadence.Cron, "error", err)
			continue
		}

		// The merchant's timezone decides when "every Wednesday" happens. Falling back to UTC silently would fire a European merchant's Monday plan on Sunday night.
		location := time.UTC
		if cadence.Timezone != "" {
			if loaded, loadErr := time.LoadLocation(cadence.Timezone); loadErr == nil {
				location = loaded
			} else {
				slog.Warn("Schedule cadence: unknown timezone, falling back to UTC",
					"account_id", cadence.AccountID, "timezone", cadence.Timezone)
			}
		}

		// Measure from the last fire, or from when the cadence was configured if it has never fired. Using "now" as the baseline would mean a cadence never fires.
		since := cadence.CreatedAt
		if cadence.LastGeneratedAt != nil {
			since = *cadence.LastGeneratedAt
		}

		if next := schedule.Next(since.In(location)); next.After(now.In(location)) {
			continue
		}

		if apiErr := s.svc.EnqueueScheduledGeneration(ctx, domain.EnqueueGenerationParams{
			AccountID:    cadence.AccountID,
			PlanningAsOf: now,
			AutoPublish:  cadence.AutoPublish,
		}); apiErr != nil {
			slog.Error("Schedule cadence: failed to enqueue generation",
				"account_id", cadence.AccountID, "error", apiErr)
			continue
		}

		// Stamped only after a successful enqueue, so a failed publish is retried on the next tick rather than skipped until the next cron window.
		if apiErr := repo.StampGenerationRun(ctx, cadence.AccountID, now); apiErr != nil {
			slog.Error("Schedule cadence: failed to stamp generation run",
				"account_id", cadence.AccountID, "error", apiErr)
		}
	}
}

// reapStalledGenerations fails versions orphaned by a process that died mid-solve.
//
// Without this they sit in `generating` forever and the account looks like it has a schedule when what it has is an empty shell.
func (s *productionScheduleScheduler) reapStalledGenerations(ctx context.Context) {
	ctx, span := scheduleCadenceTracer.Start(ctx, "service.production_schedule_scheduler.reap_stalled")
	defer span.End()

	cutoff := time.Now().UTC().Add(-stalledGenerationThreshold)
	if apiErr := s.repos.NewProductionScheduleRepo().ReapStalledGenerations(ctx, cutoff); apiErr != nil {
		slog.Error("Schedule cadence: failed to reap stalled generations", "error", apiErr)
	}
}
