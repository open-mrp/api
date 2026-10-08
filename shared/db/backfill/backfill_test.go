package backfill

import (
	"context"
	"errors"
	"testing"
	"time"
)

type memProgress struct {
	cursor    string
	rows      int64
	completed bool
	saves     int
}

func (p *memProgress) Load(context.Context, string) (string, bool, error) {
	return p.cursor, p.completed, nil
}

func (p *memProgress) Save(_ context.Context, _, cursor string, rows int64, completed bool) error {
	p.cursor, p.completed = cursor, completed
	p.rows += rows
	p.saves++
	return nil
}

func newTestRunner(t *testing.T, cfg *Config) (*Runner, *[]time.Duration) {
	t.Helper()
	r, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var pauses []time.Duration
	r.sleep = func(_ context.Context, d time.Duration) error {
		pauses = append(pauses, d)
		return nil
	}
	return r, &pauses
}

func TestRun_ResumesFromSavedCursorAndFinishes(t *testing.T) {
	t.Parallel()
	progress := &memProgress{cursor: "3"}
	var seen []string
	batch := func(_ context.Context, m *Meter, cursor string, _ int) (string, int, bool, error) {
		seen = append(seen, cursor)
		_ = m.Time(func() error { return nil })
		if cursor == "5" {
			return "5", 0, true, nil
		}
		next := map[string]string{"3": "4", "4": "5"}[cursor]
		return next, 1, false, nil
	}
	r, _ := newTestRunner(t, &Config{Name: "test", Batch: batch, Progress: progress})

	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if want := []string{"3", "4", "5"}; len(seen) != 3 || seen[0] != want[0] || seen[2] != want[2] {
		t.Errorf("cursors: got %v want %v", seen, want)
	}
	if !progress.completed || progress.rows != 2 {
		t.Errorf("progress: completed=%v rows=%d", progress.completed, progress.rows)
	}
}

func TestRun_SkipsACompletedBackfill(t *testing.T) {
	t.Parallel()
	progress := &memProgress{completed: true}
	r, _ := newTestRunner(t, &Config{Name: "test", Progress: progress, Batch: func(context.Context, *Meter, string, int) (string, int, bool, error) {
		t.Fatal("a completed backfill must not run")
		return "", 0, false, nil
	}})
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

// A failed batch is retried from the same cursor, never skipped, and smaller.
func TestRun_RetriesAFailedBatchFromTheSameCursor(t *testing.T) {
	t.Parallel()
	progress := &memProgress{}
	var limits []int
	var cursors []string
	calls := 0
	batch := func(_ context.Context, _ *Meter, cursor string, limit int) (string, int, bool, error) {
		calls++
		limits = append(limits, limit)
		cursors = append(cursors, cursor)
		if calls == 1 {
			return "", 0, false, errors.New("lock wait timeout")
		}
		return "a", 1, true, nil
	}
	r, _ := newTestRunner(t, &Config{Name: "test", Batch: batch, Progress: progress, InitialBatch: 10})

	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if cursors[0] != "" || cursors[1] != "" {
		t.Errorf("retry must reuse the cursor, got %v", cursors)
	}
	if limits[1] >= limits[0] {
		t.Errorf("retry must shrink the batch: %v", limits)
	}
	if progress.saves != 1 {
		t.Errorf("a failed batch must not save progress; saves=%d", progress.saves)
	}
}

func TestNextLimit_SteersTowardTarget(t *testing.T) {
	t.Parallel()
	target := 25 * time.Millisecond
	if got := nextLimit(100, 100*time.Millisecond, target, 500); got >= 25 {
		t.Errorf("a 4x-slow batch must shrink below a quarter, got %d", got)
	}
	if got := nextLimit(100, 5*time.Millisecond, target, 500); got != 150 {
		t.Errorf("a fast batch grows by half, got %d", got)
	}
	if got := nextLimit(400, time.Millisecond, target, 500); got != 500 {
		t.Errorf("growth is capped at max, got %d", got)
	}
	if got := nextLimit(1, time.Second, target, 500); got != 1 {
		t.Errorf("the batch never drops below one row, got %d", got)
	}
	if got := nextLimit(100, 20*time.Millisecond, target, 500); got != 100 {
		t.Errorf("a batch near target holds, got %d", got)
	}
}

// The pause holds the backfill to its duty cycle of database time.
func TestRun_PausesInProportionToDatabaseTime(t *testing.T) {
	t.Parallel()
	calls := 0
	batch := func(_ context.Context, m *Meter, _ string, _ int) (string, int, bool, error) {
		calls++
		_ = m.Time(func() error { time.Sleep(20 * time.Millisecond); return nil })
		return "x", 1, calls == 2, nil
	}
	r, pauses := newTestRunner(t, &Config{Name: "test", Batch: batch, Progress: &memProgress{}, DutyCycle: 0.2})

	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(*pauses) != 1 || (*pauses)[0] < 80*time.Millisecond {
		t.Errorf("expected one pause of at least 4x the 20ms of database time, got %v", *pauses)
	}
}

func TestRun_StopsWhenCancelled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	batch := func(context.Context, *Meter, string, int) (string, int, bool, error) {
		cancel()
		return "x", 1, false, nil
	}
	r, _ := newTestRunner(t, &Config{Name: "test", Batch: batch, Progress: &memProgress{}})
	if err := r.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
