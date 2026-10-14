// Package backfill runs one-off data migrations that touch too many rows to run on the deploy path.
// A backfill walks its table in keyset batches from a saved cursor, so it resumes where it stopped.
// Every database statement it makes is timed, and the batch size adapts to keep each statement
// near TargetStatement, well inside the 50ms budget every query is held to. Between batches it
// sleeps in proportion to the time it spent in the database, so its share of the database stays
// bounded however long it runs.
package backfill

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/open-mrp/apikit/lease"
)

// Batch is one step of a backfill: process up to limit rows after cursor and return the cursor to
// resume from. done reports that nothing is left. A batch must be idempotent, because a crash after
// its writes but before its cursor is saved runs it again. Wrap every database call in m.Time.
type Batch func(ctx context.Context, m *Meter, cursor string, limit int) (next string, rows int, done bool, err error)

// Progress persists a backfill's cursor so it survives restarts and redeploys.
type Progress interface {
	// Load returns the saved cursor and whether the backfill already finished. A backfill never run
	// before returns ("", false, nil).
	Load(ctx context.Context, name string) (cursor string, completed bool, err error)
	// Save records the cursor after a batch, adds the batch's rows to the running count, and marks
	// the backfill finished when completed is set.
	Save(ctx context.Context, name, cursor string, rows int64, completed bool) error
}

// Config configures a Runner.
type Config struct {
	// Name (required) identifies the backfill in its progress row and logs.
	Name string
	// Batch (required) processes one batch.
	Batch Batch
	// Progress (required) stores the cursor.
	Progress Progress
	// TargetStatement (optional; default: 25ms) is the duration the slowest statement of a batch is
	// steered toward. Half the 50ms budget, so a slow page or a busy moment stays inside it.
	TargetStatement time.Duration
	// InitialBatch (optional; default: 10) is the first batch size.
	InitialBatch int
	// MaxBatch (optional; default: 500) caps the batch size. Lower it when a row carries large
	// values the batch holds in memory.
	MaxBatch int
	// DutyCycle (optional; default: 0.2) is the largest fraction of wall time the backfill spends in
	// the database. After a batch that spent d in the database it sleeps d·(1−DutyCycle)/DutyCycle.
	DutyCycle float64
	// MinPause (optional; default: 50ms) is the shortest sleep between batches.
	MinPause time.Duration
	// RetryPause (optional; default: 30s) is how long to wait after a failed batch before retrying it.
	RetryPause time.Duration
}

// WithDefaults fills unset optional fields.
func (c *Config) WithDefaults() *Config {
	if c == nil {
		c = &Config{}
	}
	if c.TargetStatement <= 0 {
		c.TargetStatement = 25 * time.Millisecond
	}
	if c.InitialBatch <= 0 {
		c.InitialBatch = 10
	}
	if c.MaxBatch <= 0 {
		c.MaxBatch = 500
	}
	if c.DutyCycle <= 0 || c.DutyCycle > 1 {
		c.DutyCycle = 0.2
	}
	if c.MinPause <= 0 {
		c.MinPause = 50 * time.Millisecond
	}
	if c.RetryPause <= 0 {
		c.RetryPause = 30 * time.Second
	}
	return c
}

func (c *Config) validate() error {
	if c.Name == "" {
		return fmt.Errorf("backfill: name is required")
	}
	if c.Batch == nil {
		return fmt.Errorf("backfill: %s: batch is required", c.Name)
	}
	if c.Progress == nil {
		return fmt.Errorf("backfill: %s: progress is required", c.Name)
	}
	if c.InitialBatch > c.MaxBatch {
		return fmt.Errorf("backfill: %s: initial batch %d exceeds max batch %d", c.Name, c.InitialBatch, c.MaxBatch)
	}
	return nil
}

// Runner drives one backfill.
type Runner struct {
	cfg   Config
	sleep func(context.Context, time.Duration) error
}

func New(cfg *Config) (*Runner, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &Runner{cfg: *cfg, sleep: sleepCtx}, nil
}

// statementBudget is the latency every query is held to. A statement over it is logged.
const statementBudget = 50 * time.Millisecond

// Run processes batches until the backfill finishes or ctx is done. A failed batch is retried after
// RetryPause from the same cursor; Run returns nil once the backfill is complete, and ctx's error if
// it is cancelled first.
func (r *Runner) Run(ctx context.Context) error {
	cursor, completed, err := r.cfg.Progress.Load(ctx, r.cfg.Name)
	if err != nil {
		return fmt.Errorf("backfill %s: load progress: %w", r.cfg.Name, err)
	}
	if completed {
		return nil
	}

	log := slog.With("backfill", r.cfg.Name)
	log.InfoContext(ctx, "Backfill starting", "cursor", cursor)

	limit := r.cfg.InitialBatch
	var total int64
	lastReport := time.Now()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		m := &Meter{}
		next, rows, done, err := r.cfg.Batch(ctx, m, cursor, limit)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			log.WarnContext(ctx, "Backfill batch failed; retrying", "error", err, "cursor", cursor, "limit", limit)
			// A failure may be the database pushing back: retry smaller.
			limit = max(1, limit/2)
			if err := r.sleep(ctx, r.cfg.RetryPause); err != nil {
				return err
			}
			continue
		}

		total += int64(rows)
		if err := r.cfg.Progress.Save(ctx, r.cfg.Name, next, int64(rows), done); err != nil {
			log.WarnContext(ctx, "Backfill progress save failed; retrying", "error", err)
			if err := r.sleep(ctx, r.cfg.RetryPause); err != nil {
				return err
			}
			continue
		}
		cursor = next

		slowest := m.Slowest()
		if slowest > statementBudget {
			log.WarnContext(ctx, "Backfill statement exceeded the query budget", "duration", slowest, "limit", limit)
		}
		if time.Since(lastReport) > time.Minute || done {
			log.InfoContext(ctx, "Backfill progress", "rows", total, "batch", limit, "slowest_statement", slowest, "cursor", cursor)
			lastReport = time.Now()
		}
		if done {
			log.InfoContext(ctx, "Backfill complete", "rows", total)
			return nil
		}

		limit = nextLimit(limit, slowest, r.cfg.TargetStatement, r.cfg.MaxBatch)
		if err := r.sleep(ctx, r.pause(m.Total())); err != nil {
			return err
		}
	}
}

// leaseTTL is how long a pod holds a backfill's lease between renewals.
const leaseTTL = 2 * time.Minute

// Keep runs the backfill on whichever pod holds its lease, so exactly one pod works on it at a time.
// It tries again every interval — after a deploy restarts the holder, or a batch keeps failing —
// and returns once the backfill is complete or ctx is done. Start it in its own goroutine.
func (r *Runner) Keep(ctx context.Context, l *lease.Lease, interval time.Duration) {
	for {
		if err := l.WithLease(ctx, "backfill:"+r.cfg.Name, leaseTTL, r.Run); err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "Backfill stopped; will retry", "backfill", r.cfg.Name, "error", err)
		}
		if _, completed, err := r.cfg.Progress.Load(ctx, r.cfg.Name); err == nil && completed {
			return
		}
		if err := r.sleep(ctx, interval); err != nil {
			return
		}
	}
}

// nextLimit steers the batch size so the slowest statement lands near target: shrink in proportion
// when over it, grow gently when well under it.
func nextLimit(limit int, slowest, target time.Duration, maxBatch int) int {
	switch {
	case slowest > target:
		scaled := int(float64(limit) * float64(target) / float64(slowest) * 0.8)
		return max(1, min(scaled, limit-1))
	case slowest < target/2:
		return min(maxBatch, max(limit+1, limit*3/2))
	default:
		return limit
	}
}

func (r *Runner) pause(dbTime time.Duration) time.Duration {
	d := time.Duration(float64(dbTime) * (1 - r.cfg.DutyCycle) / r.cfg.DutyCycle)
	return max(d, r.cfg.MinPause)
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Meter times a batch's database statements.
type Meter struct {
	mu      sync.Mutex
	total   time.Duration
	slowest time.Duration
}

// Time runs one database statement and records how long it took.
func (m *Meter) Time(fn func() error) error {
	start := time.Now()
	err := fn()
	elapsed := time.Since(start)

	m.mu.Lock()
	m.total += elapsed
	m.slowest = max(m.slowest, elapsed)
	m.mu.Unlock()
	return err
}

// Total is the time the batch spent in the database.
func (m *Meter) Total() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.total
}

// Slowest is the batch's longest statement.
func (m *Meter) Slowest() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.slowest
}
