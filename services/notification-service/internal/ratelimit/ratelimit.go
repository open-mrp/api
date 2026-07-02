// Package ratelimit provides a small in-memory token-bucket limiter used to throttle abusive messaging actors (message send, conversation creation). It is per-process — adequate as an anti-abuse backstop with generous defaults; it is not a distributed quota.
package ratelimit

import (
	"fmt"
	"sync"
	"time"
)

// Limiter is a keyed token-bucket rate limiter. Each key (e.g. an account_user id) gets its own bucket that refills continuously up to capacity.
type Limiter struct {
	mu           sync.Mutex
	buckets      map[string]*bucket
	capacity     float64
	refillPerSec float64
	now          func() time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// Config configures a token-bucket Limiter.
type Config struct {
	// Capacity (required; > 0) is the burst capacity — the maximum number of tokens a bucket holds, and the number of requests an idle key may make back-to-back before being throttled.
	Capacity float64
	// RefillPerSec (required; > 0) is the steady-state refill rate in tokens per second.
	RefillPerSec float64
	// Now (optional; default: time.Now) is the clock used to measure refill, overridable in tests.
	Now func() time.Time
}

// WithDefaults fills zero-value optional fields with production defaults. It is safe to call with a nil receiver.
func (c *Config) WithDefaults() *Config {
	if c == nil {
		c = &Config{}
	}
	now := c.Now
	if now == nil {
		now = time.Now
	}
	return &Config{
		Capacity:     c.Capacity,
		RefillPerSec: c.RefillPerSec,
		Now:          now,
	}
}

func (c *Config) validate() error {
	if c.Capacity <= 0 {
		return fmt.Errorf("ratelimit: capacity must be positive")
	}
	if c.RefillPerSec <= 0 {
		return fmt.Errorf("ratelimit: refill per second must be positive")
	}
	return nil
}

// New returns a limiter for the given config, applying defaults then validating it.
func New(cfg *Config) (*Limiter, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &Limiter{
		buckets:      make(map[string]*bucket),
		capacity:     cfg.Capacity,
		refillPerSec: cfg.RefillPerSec,
		now:          cfg.Now,
	}, nil
}

// Allow consumes one token for key, returning false when the bucket is empty (the caller should respond 429). An empty key is never limited.
func (l *Limiter) Allow(key string) bool {
	if key == "" {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	b := l.buckets[key]
	if b == nil {
		b = &bucket{tokens: l.capacity, last: now}
		l.buckets[key] = b
	} else {
		elapsed := now.Sub(b.last).Seconds()
		if elapsed > 0 {
			b.tokens += elapsed * l.refillPerSec
			if b.tokens > l.capacity {
				b.tokens = l.capacity
			}
			b.last = now
		}
	}
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}
