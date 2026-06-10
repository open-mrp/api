// Package retry provides configurable retry logic with exponential backoff and
// jitter. It is used throughout the platform wherever a transient failure (network
// timeout, temporary database unavailability, etc.) can be recovered by simply
// re-attempting the operation after a short delay.
//
// The central entry point is [WithBackoff], which executes an operation up to
// MaxRetries+1 times (1 initial attempt + MaxRetries retries). Between attempts it
// sleeps for a duration computed by [CalculateDelay]:
//
//	delay = InitialWait * Multiplier^attempt   (capped at MaxWait)
//	delay ±= delay * JitterFraction            (floored at InitialWait)
//
// Context cancellation is checked before each sleep, so callers can abort a retry
// loop promptly during graceful shutdown.
//
// Configuration uses the WithDefaults/validate pattern: pass a partially filled
// [Config] (or nil) and zero-value fields are replaced with production defaults.
package retry

import (
	"cmp"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"log"
	"math"
	"time"
)

const (
	// DefaultMaxRetries is the number of retry attempts after the initial call fails.
	// Total attempts = 1 (initial) + DefaultMaxRetries. Exported so callers can
	// reference it when building retry-aware error messages.
	DefaultMaxRetries = 3

	// defaultInitialWait is the base delay before the first retry (attempt 0).
	// Subsequent attempts multiply this by Multiplier^attempt.
	defaultInitialWait = 1 * time.Second

	// defaultMaxWait caps the computed delay so that high attempt counts don't
	// produce unreasonably long waits.
	defaultMaxWait = 10 * time.Second

	// defaultMultiplier is the exponential growth factor applied per attempt.
	// A value of 2.0 produces the classic doubling backoff: 1s, 2s, 4s, 8s, …
	defaultMultiplier = 2.0

	// defaultJitterFraction adds ±10% randomness to each delay to prevent
	// thundering-herd effects when many processes retry the same operation
	// (e.g. a fleet of enqueuer instances all failing against the same DB).
	defaultJitterFraction = 0.1
)

// Config controls the retry behavior of [WithBackoff]. All fields have sensible
// production defaults (applied by [Config.WithDefaults]) so callers only need to
// set the values they want to override.
type Config struct {
	// MaxRetries (optional; default: 3) is the maximum number of retry attempts after
	// the initial call. Total invocations = MaxRetries + 1. The zero value is treated
	// as "unset" by WithDefaults and replaced with the default of 3; running exactly
	// once with no retries is not expressible via this config.
	MaxRetries int

	// InitialWait (optional; default: 1s) is the delay before the first retry. It also
	// serves as the absolute floor for the jittered delay — no sleep will ever be
	// shorter than this value.
	InitialWait time.Duration

	// MaxWait (optional; default: 10s) caps the computed exponential delay. Once
	// InitialWait * Multiplier^attempt exceeds MaxWait, the delay is pinned to MaxWait
	// (before jitter).
	MaxWait time.Duration

	// Multiplier (optional; default: 2.0) is the base of the exponential growth applied
	// per attempt. A value of 2.0 doubles the delay each retry; 1.0 produces
	// constant-interval retries. Must be >= 1.0.
	Multiplier float64

	// JitterFraction (optional; default: 0.1) controls the +/- random spread applied to
	// each delay. A value of 0.1 means the final delay is within +/-10% of the computed
	// exponential value. Must be in [0, 1.0]. The zero value is treated as "unset" by
	// WithDefaults and replaced with the default of 0.1, so disabling jitter is only
	// possible when bypassing WithDefaults (e.g. calling CalculateDelay directly).
	JitterFraction float64
}

// WithDefaults returns a new Config with all zero-value fields replaced by
// production defaults. It is safe to call on a nil receiver — a nil Config
// produces a fully-defaulted config. The original Config is not mutated; a
// copy is always returned.
func (c *Config) WithDefaults() *Config {
	if c == nil {
		c = &Config{}
	}

	return &Config{
		MaxRetries:     cmp.Or(c.MaxRetries, DefaultMaxRetries),
		InitialWait:    cmp.Or(c.InitialWait, defaultInitialWait),
		MaxWait:        cmp.Or(c.MaxWait, defaultMaxWait),
		Multiplier:     cmp.Or(c.Multiplier, defaultMultiplier),
		JitterFraction: cmp.Or(c.JitterFraction, defaultJitterFraction),
	}
}

// validate checks that the Config fields form a coherent retry policy. It
// returns a descriptive error for any constraint violation:
//   - MaxRetries must be >= 0 (WithDefaults replaces 0 with the default of 3
//     before WithBackoff validates, so the effective value is always >= 1).
//   - InitialWait and MaxWait must be positive, and MaxWait >= InitialWait.
//   - Multiplier must be >= 1.0 (otherwise delays would shrink over time).
//   - JitterFraction must be in [0, 1.0].
func (c *Config) validate() error {
	if c.MaxRetries < 0 {
		return errors.New("retry: max retries must be non-negative")
	}
	if c.InitialWait <= 0 {
		return errors.New("retry: initial wait must be positive")
	}
	if c.MaxWait <= 0 {
		return errors.New("retry: max wait must be positive")
	}
	if c.MaxWait < c.InitialWait {
		return errors.New("retry: max wait must be greater than or equal to initial wait")
	}
	if c.Multiplier < 1.0 {
		return errors.New("retry: multiplier must be greater than or equal to 1.0")
	}
	if c.JitterFraction < 0 || c.JitterFraction > 1.0 {
		return errors.New("retry: jitter fraction must be between 0 and 1.0")
	}
	return nil
}

// cryptoRandFloat64 returns a cryptographically random float64 in [0, 1). It uses
// crypto/rand instead of math/rand to avoid global-lock contention and to eliminate
// the need for seeding. If the entropy source fails (extremely unlikely), it falls
// back to 0.5 to provide a neutral jitter value rather than panicking.
func cryptoRandFloat64() float64 {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0.5
	}
	return float64(binary.BigEndian.Uint64(buf[:])) / float64(^uint64(0))
}

// CalculateDelay computes the backoff sleep duration for the given 0-indexed attempt.
// The formula is:
//
//	base   = InitialWait * Multiplier^attempt
//	capped = min(base, MaxWait)
//	jitter = capped * JitterFraction * uniform(-1, 1)
//	delay  = max(capped + jitter, InitialWait)
//
// The delay is always at least InitialWait, even after negative jitter is applied.
// When JitterFraction is 0 the result is fully deterministic. A nil Config is
// promoted to defaults via WithDefaults before computation.
func CalculateDelay(cfg *Config, attempt int) time.Duration {
	if cfg == nil {
		cfg = cfg.WithDefaults()
	}

	delay := float64(cfg.InitialWait) * math.Pow(cfg.Multiplier, float64(attempt))

	if delay > float64(cfg.MaxWait) {
		delay = float64(cfg.MaxWait)
	}

	if cfg.JitterFraction > 0 {
		jitterRange := delay * cfg.JitterFraction
		// Random value in [-1, 1) scaled by jitterRange
		jitter := (cryptoRandFloat64() - 0.5) * 2 * jitterRange
		delay += jitter

		if delay < float64(cfg.InitialWait) {
			delay = float64(cfg.InitialWait)
		}
	}

	return time.Duration(delay)
}

// WithBackoff executes operation and retries it up to MaxRetries times on failure,
// sleeping with exponential backoff between attempts. The retry loop:
//
//  1. Calls operation(). On success (nil error), returns nil immediately.
//  2. On failure, computes the backoff delay via [CalculateDelay] and sleeps. If the
//     context is cancelled during the sleep, returns ctx.Err() instead of continuing.
//  3. After exhausting all retries, returns the last error from operation.
//
// A nil Config is promoted to defaults. If the (defaulted) config fails validation,
// the validation error is returned without ever calling operation.
func WithBackoff(ctx context.Context, cfg *Config, operation func() error) error {
	cfg = cfg.WithDefaults()

	if err := cfg.validate(); err != nil {
		return err
	}

	var err error
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			sleep := CalculateDelay(cfg, attempt-1)
			log.Printf("retry: attempt %d/%d after %v", attempt, cfg.MaxRetries, sleep)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(sleep):
			}
		}

		if err = operation(); err == nil {
			return nil
		}

		log.Printf("retry: operation failed (attempt %d/%d): %v", attempt+1, cfg.MaxRetries, err)
	}

	return err
}
