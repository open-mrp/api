package cache

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const (
	defaultMonitorInterval    = 30 * time.Second
	defaultMonitorRepeatEvery = 5 * time.Minute
)

// MonitorConfig configures RedisStore.Monitor.
type MonitorConfig struct {
	// Logger (required) receives reachability changes.
	Logger *slog.Logger

	// Interval (optional; default: 30s) is the time between reachability checks.
	Interval time.Duration

	// RepeatEvery (optional; default: 5m) is how often an ongoing outage is logged again, so it stays visible in recent logs.
	RepeatEvery time.Duration
}

// WithDefaults returns a copy of the config with unset fields filled. It is safe to call on a nil receiver.
func (c *MonitorConfig) WithDefaults() *MonitorConfig {
	if c == nil {
		c = &MonitorConfig{}
	}
	out := *c
	if out.Interval == 0 {
		out.Interval = defaultMonitorInterval
	}
	if out.RepeatEvery == 0 {
		out.RepeatEvery = defaultMonitorRepeatEvery
	}
	return &out
}

func (c *MonitorConfig) validate() error {
	if c.Logger == nil {
		return fmt.Errorf("cache: monitor logger is required")
	}
	if c.Interval <= 0 || c.RepeatEvery <= 0 {
		return fmt.Errorf("cache: monitor interval and repeat must be positive")
	}
	return nil
}

// Monitor logs when Redis becomes unreachable, again every RepeatEvery while it stays so, and when it recovers. The cache fails open, so without this an outage costs only latency and never shows up in logs. It blocks until ctx is done.
func (s *RedisStore) Monitor(ctx context.Context, cfg *MonitorConfig) error {
	cfg = cfg.WithDefaults()
	if err := cfg.validate(); err != nil {
		return err
	}

	var downSince, lastWarned time.Time
	check := func() {
		err := s.Ping(ctx)
		now := time.Now()
		switch {
		case err == nil && downSince.IsZero() && lastWarned.IsZero():
			return
		case err == nil:
			cfg.Logger.Info("Redis reachable again; analytics cache resumed", "down_for", now.Sub(downSince).Round(time.Second).String())
			downSince, lastWarned = time.Time{}, time.Time{}
		case ctx.Err() != nil:
			return
		case downSince.IsZero():
			downSince, lastWarned = now, now
			cfg.Logger.Warn("Redis unreachable; analytics reports are computed uncached", "error", err)
		case now.Sub(lastWarned) >= cfg.RepeatEvery:
			lastWarned = now
			cfg.Logger.Warn("Redis still unreachable; analytics reports are computed uncached", "down_for", now.Sub(downSince).Round(time.Second).String(), "error", err)
		}
	}

	check()
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			check()
		}
	}
}
