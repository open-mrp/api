package scheduler

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

// PendingDomainRechecker verifies pending email domains in the background.
type PendingDomainRechecker interface {
	RecheckPendingDomains(ctx context.Context, interval time.Duration, limit int32) (int, *apierror.APIError)
}

// EmailDomainVerifierConfig configures the background domain verifier.
type EmailDomainVerifierConfig struct {
	// ServiceName (required) scopes the lease name so each service runs its own verifier.
	ServiceName string
	// PlatformMode (optional) shortens the default PollInterval in test mode.
	PlatformMode constants.PlatformMode
	// PollInterval (optional; default 1m, 5s in test) controls how often the verifier looks for domains to re-check.
	PollInterval time.Duration
	// RecheckInterval (optional; default 5m) is how long a domain waits between checks, so one just registered or just checked is left alone.
	RecheckInterval time.Duration
	// BatchSize (optional; default 20) bounds domains checked per tick, each costing one SES call.
	BatchSize int32
	// LeaseTTL (optional; default 5m) bounds how long the verifier holds its lease.
	LeaseTTL time.Duration
}

// WithDefaults fills zero-value fields with production defaults.
func (c *EmailDomainVerifierConfig) WithDefaults() *EmailDomainVerifierConfig {
	if c == nil {
		c = &EmailDomainVerifierConfig{}
	}
	pollInterval := c.PollInterval
	if pollInterval == 0 {
		if c.PlatformMode.IsTest() {
			pollInterval = 5 * time.Second
		} else {
			pollInterval = time.Minute
		}
	}
	return &EmailDomainVerifierConfig{
		ServiceName:     c.ServiceName,
		PlatformMode:    c.PlatformMode,
		PollInterval:    pollInterval,
		RecheckInterval: cmp.Or(c.RecheckInterval, 5*time.Minute),
		BatchSize:    int32(cmp.Or(int(c.BatchSize), 20)), // #nosec G115 - small config value
		LeaseTTL:     lease.TTLOr(c.LeaseTTL, 5*time.Minute),
	}
}

func (c *EmailDomainVerifierConfig) validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("email domain verifier: service name is required")
	}
	if c.PollInterval <= 0 {
		return fmt.Errorf("email domain verifier: poll interval must be positive")
	}
	if c.RecheckInterval <= 0 {
		return fmt.Errorf("email domain verifier: recheck interval must be positive")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("email domain verifier: batch size must be positive")
	}
	if c.LeaseTTL <= 0 {
		return fmt.Errorf("email domain verifier: lease ttl must be positive")
	}
	return nil
}

// EmailDomainVerifier runs a lease-guarded background goroutine that re-checks pending email domains, so a domain verifies once its DNS records are published without anyone running the verify action.
type EmailDomainVerifier struct {
	config    EmailDomainVerifierConfig
	rechecker PendingDomainRechecker
	lease     *lease.Lease
	leaseName string

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewEmailDomainVerifier constructs the verifier. A non-nil lease ensures only one pod checks per tick.
func NewEmailDomainVerifier(config *EmailDomainVerifierConfig, rechecker PendingDomainRechecker, l *lease.Lease) (*EmailDomainVerifier, error) {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	if rechecker == nil {
		return nil, fmt.Errorf("email domain verifier: rechecker is required")
	}
	if l == nil {
		return nil, fmt.Errorf("email domain verifier: lease is required")
	}
	return &EmailDomainVerifier{
		config:    *config,
		rechecker: rechecker,
		lease:     l,
		leaseName: "email-domain-verifier-" + config.ServiceName,
	}, nil
}

// Start launches the background goroutine.
func (v *EmailDomainVerifier) Start(ctx context.Context) error {
	ctx = appctx.WithNoTrace(ctx)
	v.ctx, v.cancel = context.WithCancel(ctx)

	v.wg.Add(1)
	go v.loop()

	slog.Info("Email domain verifier started", "poll_interval", v.config.PollInterval)
	return nil
}

// Stop cancels the loop and waits for the goroutine to exit.
func (v *EmailDomainVerifier) Stop() {
	if v.cancel != nil {
		v.cancel()
	}
	v.wg.Wait()
	slog.Info("Email domain verifier stopped")
}

func (v *EmailDomainVerifier) loop() {
	defer v.wg.Done()

	ticker := time.NewTicker(v.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-v.ctx.Done():
			return
		case <-ticker.C:
			_ = v.lease.WithLease(v.ctx, v.leaseName, v.config.LeaseTTL, func(ctx context.Context) error {
				if n, apiErr := v.rechecker.RecheckPendingDomains(ctx, v.config.RecheckInterval, v.config.BatchSize); apiErr != nil {
					slog.Error("Failed to re-check pending email domains", "error", apiErr.PublicMessage)
				} else if n > 0 {
					slog.Info("Verified pending email domains", "count", n)
				}
				return nil
			})
		}
	}
}
