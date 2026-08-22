// Package scheduler holds the background worker that delivers due scheduled messages. It mirrors the inbox purger / messaging reaper lease-guarded single-runner design: one pod per tick claims the lease and materializes any scheduled messages whose time has arrived.
package scheduler

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/lease"
)

// ScheduledMessageDeliverer materializes due scheduled messages into real messages.
type ScheduledMessageDeliverer interface {
	DeliverDueScheduledMessages(ctx context.Context, limit int32) (int, *apierror.APIError)
}

// ScheduledMessageWorkerConfig configures the delivery worker.
type ScheduledMessageWorkerConfig struct {
	// ServiceName (required) scopes the lease name so each service runs its own worker.
	ServiceName string
	// PlatformMode (optional) shortens the default PollInterval in test mode so e2e can observe delivery.
	PlatformMode constants.PlatformMode
	// PollInterval (optional; default 30s, 2s in test) controls how often due messages are delivered.
	PollInterval time.Duration
	// BatchSize (optional; default 100) bounds messages delivered per tick.
	BatchSize int32
	// LeaseTTL (optional; default 5m) bounds how long the worker holds its lease.
	LeaseTTL time.Duration
}

// WithDefaults fills zero-value fields with production defaults.
func (c *ScheduledMessageWorkerConfig) WithDefaults() *ScheduledMessageWorkerConfig {
	if c == nil {
		c = &ScheduledMessageWorkerConfig{}
	}
	pollInterval := c.PollInterval
	if pollInterval == 0 {
		if c.PlatformMode.IsTest() {
			pollInterval = 2 * time.Second
		} else {
			pollInterval = 30 * time.Second
		}
	}
	return &ScheduledMessageWorkerConfig{
		ServiceName:  c.ServiceName,
		PlatformMode: c.PlatformMode,
		PollInterval: pollInterval,
		BatchSize:    int32(cmp.Or(int(c.BatchSize), 100)), // #nosec G115 - small config value
		LeaseTTL:     cmp.Or(c.LeaseTTL, 5*time.Minute),
	}
}

func (c *ScheduledMessageWorkerConfig) validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("scheduled message worker: service name is required")
	}
	if c.PollInterval <= 0 {
		return fmt.Errorf("scheduled message worker: poll interval must be positive")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("scheduled message worker: batch size must be positive")
	}
	if c.LeaseTTL <= 0 {
		return fmt.Errorf("scheduled message worker: lease ttl must be positive")
	}
	return nil
}

// ScheduledMessageWorker runs a lease-guarded background goroutine that delivers due scheduled messages.
type ScheduledMessageWorker struct {
	config    ScheduledMessageWorkerConfig
	deliverer ScheduledMessageDeliverer
	lease     *lease.Lease
	leaseName string

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewScheduledMessageWorker constructs the worker. A non-nil lease ensures only one pod delivers per tick.
func NewScheduledMessageWorker(config *ScheduledMessageWorkerConfig, deliverer ScheduledMessageDeliverer, l *lease.Lease) (*ScheduledMessageWorker, error) {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	if l == nil {
		return nil, fmt.Errorf("scheduled message worker: lease is required")
	}
	return &ScheduledMessageWorker{
		config:    *config,
		deliverer: deliverer,
		lease:     l,
		leaseName: "scheduled-message-worker-" + config.ServiceName,
	}, nil
}

// Start launches the background delivery goroutine.
func (w *ScheduledMessageWorker) Start(ctx context.Context) error {
	ctx = appctx.WithNoTrace(ctx)
	w.ctx, w.cancel = context.WithCancel(ctx)

	w.wg.Add(1)
	go w.loop()

	slog.Info("Scheduled message worker started", "poll_interval", w.config.PollInterval)
	return nil
}

// Stop cancels the loop and waits for the goroutine to exit.
func (w *ScheduledMessageWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
	slog.Info("Scheduled message worker stopped")
}

func (w *ScheduledMessageWorker) loop() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			_ = w.lease.WithLease(w.ctx, w.leaseName, w.config.LeaseTTL, func(ctx context.Context) error {
				if n, apiErr := w.deliverer.DeliverDueScheduledMessages(ctx, w.config.BatchSize); apiErr != nil {
					slog.Error("Failed to deliver due scheduled messages", "error", apiErr.PublicMessage)
				} else if n > 0 {
					slog.Info("Delivered scheduled messages", "count", n)
				}
				return nil
			})
		}
	}
}
