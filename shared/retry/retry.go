/*
Package retry provides a simple retry mechanism with exponential backoff.
It is as abstract as possible to allow for different retry strategies.
*/
package retry

import (
	"context"
	"log"
	"math/rand"
	"time"
)

type Config struct {
	MaxRetries  int
	InitialWait time.Duration
	MaxWait     time.Duration
}

// DefaultConfig returns a Config with sensible default values
func DefaultConfig() Config {
	return Config{
		MaxRetries:  3,
		InitialWait: 1 * time.Second,
		MaxWait:     10 * time.Second,
	}
}

// jitterDuration applies "full jitter" to the base duration, returning a random
// duration in the range [0, base). If base is non-positive, it returns 0.
func jitterDuration(base time.Duration) time.Duration {
	if base <= 0 {
		return 0
	}

	return time.Duration(rand.Int63n(int64(base))) // #nosec G404
}

// WithBackoff executes the given operation with exponential backoff retry logic
func WithBackoff(ctx context.Context, cfg Config, operation func() error) error {
	var err error
	wait := cfg.InitialWait

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			sleep := jitterDuration(wait)
			log.Printf("Retry attempt %d/%d after %v", attempt, cfg.MaxRetries, sleep)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(sleep):
			}

			// Exponential backoff with max wait cap
			wait *= 2
			if wait > cfg.MaxWait {
				wait = cfg.MaxWait
			}
		}

		if err = operation(); err == nil {
			return nil
		}

		log.Printf("Operation failed (attempt %d/%d): %v", attempt+1, cfg.MaxRetries, err)
	}

	return err
}
