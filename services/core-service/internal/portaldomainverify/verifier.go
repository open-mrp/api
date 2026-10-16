// Package portaldomainverify re-checks pending portal domains in the background, so a domain moves to verified once
// its DNS records are published without anyone running the verify action.
package portaldomainverify

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/lease"
)

// Rechecker checks the domains that are due.
type Rechecker interface {
	RecheckPendingDomains(ctx context.Context, interval time.Duration, limit int32) (int, *apierror.APIError)
}

// Config configures the verifier.
type Config struct {
	// PlatformMode (optional) shortens the default PollInterval in test mode.
	PlatformMode constants.PlatformMode
	// PollInterval (optional; default 1m, 5s in test) controls how often the verifier looks for domains to check.
	PollInterval time.Duration
	// RecheckInterval (optional; default 5m) is how long a domain waits between checks. Creating or verifying a domain
	// counts as a check.
	RecheckInterval time.Duration
	// BatchSize (optional; default 20) bounds domains checked per tick, each costing one provider call.
	BatchSize int32
	// LeaseTTL (optional; default 5m) bounds how long the verifier holds its lease.
	LeaseTTL time.Duration
}

// WithDefaults fills zero-value fields with production defaults.
func (c *Config) WithDefaults() *Config {
	if c == nil {
		c = &Config{}
	}
	poll := time.Minute
	if c.PlatformMode.IsTest() {
		poll = 5 * time.Second
	}
	return &Config{
		PlatformMode:    c.PlatformMode,
		PollInterval:    cmp.Or(c.PollInterval, poll),
		RecheckInterval: cmp.Or(c.RecheckInterval, 5*time.Minute),
		BatchSize:       cmp.Or(c.BatchSize, 20),
		LeaseTTL:        lease.TTLOr(c.LeaseTTL, 5*time.Minute),
	}
}

func (c *Config) validate() error {
	if c.PollInterval <= 0 {
		return fmt.Errorf("portal domain verifier: poll interval must be positive")
	}
	if c.RecheckInterval <= 0 {
		return fmt.Errorf("portal domain verifier: recheck interval must be positive")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("portal domain verifier: batch size must be positive")
	}
	if c.LeaseTTL <= 0 {
		return fmt.Errorf("portal domain verifier: lease ttl must be positive")
	}
	return nil
}

const leaseName = "portal-domain-verifier"

// Verifier runs a lease-guarded goroutine that re-checks pending and securing portal domains.
type Verifier struct {
	config    Config
	rechecker Rechecker
	lease     *lease.Lease

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New constructs the verifier. The lease ensures only one pod checks per tick.
func New(config *Config, rechecker Rechecker, l *lease.Lease) (*Verifier, error) {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	if rechecker == nil {
		return nil, fmt.Errorf("portal domain verifier: rechecker is required")
	}
	if l == nil {
		return nil, fmt.Errorf("portal domain verifier: lease is required")
	}
	return &Verifier{config: *config, rechecker: rechecker, lease: l}, nil
}

// Start launches the background goroutine.
func (v *Verifier) Start(ctx context.Context) {
	ctx = appctx.WithNoTrace(ctx)
	v.ctx, v.cancel = context.WithCancel(ctx)

	v.wg.Add(1)
	go v.loop()

	slog.Info("Portal domain verifier started", "poll_interval", v.config.PollInterval)
}

// Stop cancels the loop and waits for the goroutine to exit.
func (v *Verifier) Stop() {
	if v.cancel != nil {
		v.cancel()
	}
	v.wg.Wait()
	slog.Info("Portal domain verifier stopped")
}

func (v *Verifier) loop() {
	defer v.wg.Done()

	ticker := time.NewTicker(v.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-v.ctx.Done():
			return
		case <-ticker.C:
			_ = v.lease.WithLease(v.ctx, leaseName, v.config.LeaseTTL, func(ctx context.Context) error {
				if n, apiErr := v.rechecker.RecheckPendingDomains(ctx, v.config.RecheckInterval, v.config.BatchSize); apiErr != nil {
					slog.Error("Failed to re-check pending portal domains", "error", apiErr.Error())
				} else if n > 0 {
					slog.Info("Verified pending portal domains", "count", n)
				}
				return nil
			})
		}
	}
}
