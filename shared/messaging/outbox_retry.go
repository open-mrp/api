package messaging

import (
	"context"
	"log/slog"
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	"github.com/augno/api/shared/retry"
)

// OutboxDBRetryConfig returns the short retry policy used for small outbox
// database operations that can safely be re-attempted after a lock conflict.
func OutboxDBRetryConfig(platformMode constants.PlatformMode) *retry.Config {
	if platformMode.IsTest() {
		return (&retry.Config{
			MaxRetries:     4,
			InitialWait:    5 * time.Millisecond,
			MaxWait:        100 * time.Millisecond,
			Multiplier:     2.0,
			JitterFraction: 0.2,
		}).WithDefaults()
	}

	return (&retry.Config{
		MaxRetries:     3,
		InitialWait:    25 * time.Millisecond,
		MaxWait:        500 * time.Millisecond,
		Multiplier:     2.0,
		JitterFraction: 0.2,
	}).WithDefaults()
}

// WithOutboxDBLockRetry retries operation only for transient database lock
// conflicts. Callers should use it for small outbox operations whose retry is
// idempotent, not for broader business transactions.
func WithOutboxDBLockRetry(ctx context.Context, cfg *retry.Config, operation string, fn func() error) error {
	cfg = cfg.WithDefaults()

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
		if !db.IsRetryableLockConflict(err) {
			return err
		}

		lastErr = err
		if attempt < cfg.MaxRetries {
			slog.Warn("Retrying outbox DB operation after lock conflict",
				"operation", operation,
				"attempt", attempt+1,
				"max_retries", cfg.MaxRetries,
				"error", err,
			)
		}
	}

	return lastErr
}
