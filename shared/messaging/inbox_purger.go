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

// InboxPurgerConfig holds the configuration for the inbox purger worker.
type InboxPurgerConfig struct {
	// RetentionHours (optional; default: 168 i.e. 7 days) is how long processed inbox
	// records are kept before the purge loop deletes them.
	RetentionHours int

	// PurgeInterval (optional; default: 1h) controls how frequently the purger runs
	// its purge loop to delete old processed records.
	PurgeInterval time.Duration

	// BatchSize (optional; default: 1000) is the maximum number of processed inbox
	// records to delete in a single SQL DELETE statement.
	BatchSize int32
}

// WithDefaults fills zero-value fields with production defaults and returns the config.
func (c *InboxPurgerConfig) WithDefaults() *InboxPurgerConfig {
	if c == nil {
		c = &InboxPurgerConfig{}
	}

	return &InboxPurgerConfig{
		RetentionHours: cmp.Or(c.RetentionHours, 168), // 7 days
		PurgeInterval:  cmp.Or(c.PurgeInterval, 1*time.Hour),
		BatchSize:      int32(cmp.Or(int(c.BatchSize), 1000)), // #nosec G115 - small config value
	}
}

func (c *InboxPurgerConfig) validate() error {
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

// InboxPurger runs a background goroutine that periodically deletes processed
// inbox records older than the configured retention period. This prevents the
// message_inbox table from growing unboundedly while preserving recent records
// for debugging and deduplication.
type InboxPurger struct {
	config InboxPurgerConfig
	repo   InboxPurgerRepo

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewInboxPurger creates a new inbox purger. Pass a config with desired settings;
// zero-value fields are filled with production defaults.
func NewInboxPurger(config *InboxPurgerConfig, repo InboxPurgerRepo) (*InboxPurger, error) {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	return &InboxPurger{
		config: *config,
		repo:   repo,
	}, nil
}

// Start launches the background purge goroutine. The provided context is used as
// the parent for all purge operations; cancelling it (or calling Stop) shuts down
// the loop.
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

// Stop cancels the background context and blocks until the purge goroutine has
// exited.
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
			p.purgeProcessed()
		}
	}
}

func (p *InboxPurger) purgeProcessed() {
	count, err := p.repo.PurgeProcessed(p.ctx, p.config.RetentionHours, p.config.BatchSize)
	if err != nil {
		slog.Error("Failed to purge processed inbox messages", "error", err)
		return
	}

	if count > 0 {
		slog.Info("Purged processed inbox messages", "count", count)
	}
}
