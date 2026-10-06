package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/shared/lease"
)

const (
	// accountFollowupLeaseName ensures exactly one pod runs the tick.
	accountFollowupLeaseName = "account-followup-scheduler"
	accountFollowupLeaseTTL  = 2 * time.Minute

	// defaultAccountFollowupPollInterval is how often due follow-ups are enqueued. A follow-up is a day out, so an hour of slack in when it is drafted changes nothing.
	defaultAccountFollowupPollInterval = time.Hour
)

// AccountFollowupSchedulerConfig configures the periodic enqueue of due account follow-ups.
type AccountFollowupSchedulerConfig struct {
	// Svc (required) enqueues the due follow-ups.
	Svc domain.AccountFollowupSvc

	// Lease (required) serializes the tick across pods.
	Lease *lease.Lease

	// PollInterval (optional; default: 1h) is the tick interval. Zero or negative values are treated as unset.
	PollInterval time.Duration
}

func (c *AccountFollowupSchedulerConfig) WithDefaults() *AccountFollowupSchedulerConfig {
	if c == nil {
		c = &AccountFollowupSchedulerConfig{}
	}
	if c.PollInterval <= 0 {
		c.PollInterval = defaultAccountFollowupPollInterval
	}
	return c
}

func (c *AccountFollowupSchedulerConfig) validate() error {
	if c.Svc == nil {
		return fmt.Errorf("account follow-up scheduler: service is required")
	}
	if c.Lease == nil {
		return fmt.Errorf("account follow-up scheduler: lease is required")
	}
	return nil
}

type AccountFollowupScheduler struct {
	svc          domain.AccountFollowupSvc
	lease        *lease.Lease
	pollInterval time.Duration

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewAccountFollowupScheduler(config *AccountFollowupSchedulerConfig) (*AccountFollowupScheduler, error) {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	return &AccountFollowupScheduler{
		svc:          config.Svc,
		lease:        config.Lease,
		pollInterval: config.PollInterval,
		stopCh:       make(chan struct{}),
	}, nil
}

func (s *AccountFollowupScheduler) Start(ctx context.Context) {
	s.wg.Add(1)
	go s.pollLoop(ctx)
	slog.Info("Account follow-up scheduler started", "poll_interval", s.pollInterval)
}

func (s *AccountFollowupScheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	slog.Info("Account follow-up scheduler stopped")
}

func (s *AccountFollowupScheduler) pollLoop(ctx context.Context) {
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
			_ = s.lease.WithLease(ctx, accountFollowupLeaseName, accountFollowupLeaseTTL, func(leaseCtx context.Context) error {
				if apiErr := s.svc.EnqueueDue(leaseCtx); apiErr != nil {
					slog.ErrorContext(leaseCtx, "Account follow-up scheduler: failed to enqueue due follow-ups", "error", apiErr)
				}
				return nil
			})
		}
	}
}
