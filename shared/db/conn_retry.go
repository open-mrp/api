package db

import (
	"context"
	"log/slog"
	"time"

	"github.com/open-mrp/api/shared/retry"
)

// ConnRetryConfig returns the short retry policy used by WithConnRetry. The waits are deliberately small: a dropped database connection (e.g. a Vitess tablet failover) is either recovered by the next pooled connection almost immediately or not at all, and callers sit on hot request paths.
func ConnRetryConfig() *retry.Config {
	return (&retry.Config{
		MaxRetries:     2,
		InitialWait:    25 * time.Millisecond,
		MaxWait:        250 * time.Millisecond,
		Multiplier:     2.0,
		JitterFraction: 0.2,
	}).WithDefaults()
}

// WithConnRetry retries operation only for transient connection failures (see IsRetryableConnectionError). Callers must only use it for idempotent operations — typically pure reads — because a connection lost mid-write leaves the write's outcome unknown. A nil cfg uses ConnRetryConfig.
func WithConnRetry(ctx context.Context, cfg *retry.Config, operation string, fn func() error) error {
	if cfg == nil {
		cfg = ConnRetryConfig()
	}

	var lastErr error
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			sleep := retry.CalculateDelay(cfg, attempt-1)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(sleep):
			}
		}

		err := fn()
		if err == nil {
			return nil
		}
		if !IsRetryableConnectionError(err) {
			return err
		}

		lastErr = err
		if attempt < cfg.MaxRetries {
			slog.Warn("Retrying DB operation after connection failure",
				"operation", operation,
				"attempt", attempt+1,
				"max_retries", cfg.MaxRetries,
				"error", err,
			)
		}
	}

	return lastErr
}
