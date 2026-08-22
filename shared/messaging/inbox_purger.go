package messaging

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/lease"
)

// InboxPurgerConfig holds the configuration for the inbox purger worker.
type InboxPurgerConfig struct {
	// ServiceName (required) identifies which service owns this purger. It's included in the lease name so each service scopes its purger independently.
	ServiceName string

	// PlatformMode (optional; default: "") shortens the default PurgeInterval to 1m when test (see WithDefaults); otherwise the production 1h default applies.
	PlatformMode constants.PlatformMode

	// RetentionHours (optional; default: 168 i.e. 7 days) is how long processed inbox records are kept before the purge loop deletes them.
	RetentionHours int

	// PurgeInterval (optional; default: 1h) controls how frequently the purger runs its purge loop to delete old processed records.
	PurgeInterval time.Duration

	// BatchSize (optional; default: 1000) is the maximum number of processed inbox records to delete in a single SQL DELETE statement.
	BatchSize int32

	// LeaseTTL (optional; default: 5m) bounds how long the purger holds its lease before a crashed holder's claim expires.
	LeaseTTL time.Duration
}

// WithDefaults fills zero-value fields with production defaults and returns the config.
func (c *InboxPurgerConfig) WithDefaults() *InboxPurgerConfig {
	if c == nil {
		c = &InboxPurgerConfig{}
	}

	purgeInterval := c.PurgeInterval
	if purgeInterval == 0 {
		if c.PlatformMode.IsTest() {
			purgeInterval = 1 * time.Minute
		} else {
			purgeInterval = 1 * time.Hour
		}
	}

	return &InboxPurgerConfig{
		ServiceName:    c.ServiceName,
		PlatformMode:   c.PlatformMode,
		RetentionHours: cmp.Or(c.RetentionHours, 168), // 7 days
		PurgeInterval:  purgeInterval,
		BatchSize:      int32(cmp.Or(int(c.BatchSize), 1000)), // #nosec G115 - small config value
		LeaseTTL:       cmp.Or(c.LeaseTTL, 5*time.Minute),
	}
}

func (c *InboxPurgerConfig) validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("inbox purger: service name is required")
	}
	if c.RetentionHours <= 0 {
		return fmt.Errorf("inbox purger: retention hours must be positive")
	}
	if c.PurgeInterval <= 0 {
		return fmt.Errorf("inbox purger: purge interval must be positive")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("inbox purger: batch size must be positive")
	}
	return nil
}

// InboxPurger runs a background goroutine that periodically deletes processed inbox records older than the configured retention period. This prevents the message_inbox table from growing unboundedly while preserving recent records for debugging and deduplication.
type InboxPurger struct {
	config    InboxPurgerConfig
	repo      InboxPurgerRepo
	lease     *lease.Lease
	leaseName string

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewInboxPurger creates a new inbox purger. A non-nil lease is required so that only one pod per service deletes processed inbox rows each tick.
func NewInboxPurger(config *InboxPurgerConfig, repo InboxPurgerRepo, l *lease.Lease) (*InboxPurger, error) {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	if l == nil {
		return nil, fmt.Errorf("inbox purger: lease is required")
	}
	return &InboxPurger{
		config:    *config,
		repo:      repo,
		lease:     l,
		leaseName: "inbox-purger-" + config.ServiceName,
	}, nil
}

// Start launches the background purge goroutine. The provided context is used as the parent for all purge operations; cancelling it (or calling Stop) shuts down the loop.
func (p *InboxPurger) Start(ctx context.Context) error {
	ctx = appctx.WithNoTrace(ctx)
	p.ctx, p.cancel = context.WithCancel(ctx)

	p.wg.Add(1)
	go p.purgeLoop()

	slog.Info("Inbox purger started",
		"retention_hours", p.config.RetentionHours,
		"purge_interval", p.config.PurgeInterval,
	)

	return nil
}

// Stop cancels the background context and blocks until the purge goroutine has exited.
func (p *InboxPurger) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	slog.Info("Inbox purger stopped")
}

func (p *InboxPurger) purgeLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.PurgeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			_ = p.lease.WithLease(p.ctx, p.leaseName, p.config.LeaseTTL, func(ctx context.Context) error {
				p.purgeProcessed(ctx)
				return nil
			})
		}
	}
}

func (p *InboxPurger) purgeProcessed(ctx context.Context) {
	count, err := p.repo.PurgeProcessed(ctx, p.config.RetentionHours, p.config.BatchSize)
	if err != nil {
		slog.Error("Failed to purge processed inbox messages", "error", err)
		return
	}

	if count > 0 {
		slog.Info("Purged processed inbox messages", "count", count)
	}
}
