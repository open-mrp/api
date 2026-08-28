package retry

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"
)

func TestWithDefaults(t *testing.T) {
	t.Parallel()
	cfg := new(Config).WithDefaults()

	if cfg.MaxRetries != DefaultMaxRetries {
		t.Errorf("MaxRetries = %d, want %d", cfg.MaxRetries, DefaultMaxRetries)
	}
	if cfg.InitialWait != defaultInitialWait {
		t.Errorf("InitialWait = %v, want %v", cfg.InitialWait, defaultInitialWait)
	}
	if cfg.MaxWait != defaultMaxWait {
		t.Errorf("MaxWait = %v, want %v", cfg.MaxWait, defaultMaxWait)
	}
	if cfg.Multiplier != defaultMultiplier {
		t.Errorf("Multiplier = %f, want %f", cfg.Multiplier, defaultMultiplier)
	}
	if cfg.JitterFraction != defaultJitterFraction {
		t.Errorf("JitterFraction = %f, want %f", cfg.JitterFraction, defaultJitterFraction)
	}
}

func TestWithDefaults_PreservesExplicitValues(t *testing.T) {
	t.Parallel()
	cfg := (&Config{
		MaxRetries:  10,
		InitialWait: 500 * time.Millisecond,
		MaxWait:     30 * time.Second,
		Multiplier:  1.5,
	}).WithDefaults()

	if cfg.MaxRetries != 10 {
		t.Errorf("MaxRetries = %d, want 10", cfg.MaxRetries)
	}
	if cfg.Multiplier != 1.5 {
		t.Errorf("Multiplier = %f, want 1.5", cfg.Multiplier)
	}
	// JitterFraction was zero → should get default
	if cfg.JitterFraction != defaultJitterFraction {
		t.Errorf("JitterFraction = %f, want %f", cfg.JitterFraction, defaultJitterFraction)
	}
}

func TestWithDefaults_NilReceiver(t *testing.T) {
	t.Parallel()
	var cfg *Config
	got := cfg.WithDefaults()
	if got.MaxRetries != DefaultMaxRetries {
		t.Errorf("MaxRetries = %d, want %d", got.MaxRetries, DefaultMaxRetries)
	}
}

func TestValidate_ValidConfigs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  Config
	}{
		{"defaults", *new(Config).WithDefaults()},
		{"zero retries", Config{MaxRetries: 0, InitialWait: time.Second, MaxWait: time.Second, Multiplier: 1.0}},
		{"equal waits", Config{MaxRetries: 1, InitialWait: time.Second, MaxWait: time.Second, Multiplier: 1.0}},
		{"zero jitter", Config{MaxRetries: 1, InitialWait: time.Second, MaxWait: 10 * time.Second, Multiplier: 2.0, JitterFraction: 0}},
		{"max jitter", Config{MaxRetries: 1, InitialWait: time.Second, MaxWait: 10 * time.Second, Multiplier: 2.0, JitterFraction: 1.0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.validate(); err != nil {
				t.Errorf("validate() returned error for valid config: %v", err)
			}
		})
	}
}

func TestValidate_NegativeMaxRetries(t *testing.T) {
	t.Parallel()
	cfg := Config{MaxRetries: -1, InitialWait: time.Second, MaxWait: time.Second, Multiplier: 1.0}
	err := cfg.validate()
	if err == nil {
		t.Fatal("expected error for negative MaxRetries")
	}
	if err.Error() != "retry: max retries must be non-negative" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_ZeroInitialWait(t *testing.T) {
	t.Parallel()
	cfg := Config{MaxRetries: 1, InitialWait: 0, MaxWait: time.Second, Multiplier: 1.0}
	err := cfg.validate()
	if err == nil {
		t.Fatal("expected error for zero InitialWait")
	}
	if err.Error() != "retry: initial wait must be positive" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_NegativeInitialWait(t *testing.T) {
	t.Parallel()
	cfg := Config{MaxRetries: 1, InitialWait: -time.Second, MaxWait: time.Second, Multiplier: 1.0}
	err := cfg.validate()
	if err == nil {
		t.Fatal("expected error for negative InitialWait")
	}
}

func TestValidate_ZeroMaxWait(t *testing.T) {
	t.Parallel()
	cfg := Config{MaxRetries: 1, InitialWait: time.Second, MaxWait: 0, Multiplier: 1.0}
	err := cfg.validate()
	if err == nil {
		t.Fatal("expected error for zero MaxWait")
	}
	if err.Error() != "retry: max wait must be positive" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_MaxWaitLessThanInitialWait(t *testing.T) {
	t.Parallel()
	cfg := Config{MaxRetries: 1, InitialWait: 5 * time.Second, MaxWait: time.Second, Multiplier: 1.0}
	err := cfg.validate()
	if err == nil {
		t.Fatal("expected error for MaxWait < InitialWait")
	}
	if err.Error() != "retry: max wait must be greater than or equal to initial wait" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_MultiplierBelowOne(t *testing.T) {
	t.Parallel()
	cfg := Config{MaxRetries: 1, InitialWait: time.Second, MaxWait: 10 * time.Second, Multiplier: 0.5}
	err := cfg.validate()
	if err == nil {
		t.Fatal("expected error for Multiplier < 1.0")
	}
	if err.Error() != "retry: multiplier must be greater than or equal to 1.0" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_NegativeJitterFraction(t *testing.T) {
	t.Parallel()
	cfg := Config{MaxRetries: 1, InitialWait: time.Second, MaxWait: 10 * time.Second, Multiplier: 2.0, JitterFraction: -0.1}
	err := cfg.validate()
	if err == nil {
		t.Fatal("expected error for negative JitterFraction")
	}
	if err.Error() != "retry: jitter fraction must be between 0 and 1.0" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_JitterFractionAboveOne(t *testing.T) {
	t.Parallel()
	cfg := Config{MaxRetries: 1, InitialWait: time.Second, MaxWait: 10 * time.Second, Multiplier: 2.0, JitterFraction: 1.5}
	err := cfg.validate()
	if err == nil {
		t.Fatal("expected error for JitterFraction > 1.0")
	}
}

func TestCalculateDelay_NoJitter(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		InitialWait:    100 * time.Millisecond,
		MaxWait:        10 * time.Second,
		Multiplier:     2.0,
		JitterFraction: 0,
	}

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
	}

	for _, tt := range tests {
		got := CalculateDelay(cfg, tt.attempt)
		if got != tt.want {
			t.Errorf("CalculateDelay(attempt=%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestCalculateDelay_CapsAtMaxWait(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		InitialWait:    100 * time.Millisecond,
		MaxWait:        500 * time.Millisecond,
		Multiplier:     2.0,
		JitterFraction: 0,
	}

	got := CalculateDelay(cfg, 10)
	if got != 500*time.Millisecond {
		t.Errorf("CalculateDelay(attempt=10) = %v, want %v (MaxWait)", got, 500*time.Millisecond)
	}
}

func TestCalculateDelay_CustomMultiplier(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		InitialWait:    100 * time.Millisecond,
		MaxWait:        10 * time.Second,
		Multiplier:     1.5,
		JitterFraction: 0,
	}

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 150 * time.Millisecond},
		{2, time.Duration(float64(100*time.Millisecond) * math.Pow(1.5, 2))},
	}

	for _, tt := range tests {
		got := CalculateDelay(cfg, tt.attempt)
		if got != tt.want {
			t.Errorf("CalculateDelay(multiplier=1.5, attempt=%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestCalculateDelay_NilConfig(t *testing.T) {
	t.Parallel()
	got := CalculateDelay(nil, 0)
	// nil config uses defaults which include 10% jitter, so the result
	// should be within ±10% of defaultInitialWait.
	lo := time.Duration(float64(defaultInitialWait) * (1 - defaultJitterFraction))
	hi := time.Duration(float64(defaultInitialWait) * (1 + defaultJitterFraction))
	if got < lo || got > hi {
		t.Errorf("CalculateDelay(nil, 0) = %v, want in [%v, %v]", got, lo, hi)
	}
}

func TestCalculateDelay_JitterWithinBounds(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		InitialWait:    1 * time.Second,
		MaxWait:        10 * time.Second,
		Multiplier:     2.0,
		JitterFraction: 0.1,
	}

	for attempt := range 4 {
		baseDelay := float64(cfg.InitialWait) * math.Pow(cfg.Multiplier, float64(attempt))
		if baseDelay > float64(cfg.MaxWait) {
			baseDelay = float64(cfg.MaxWait)
		}
		jitterRange := baseDelay * cfg.JitterFraction
		lower := time.Duration(baseDelay - jitterRange)
		upper := time.Duration(baseDelay + jitterRange)

		// Floor at InitialWait
		if lower < cfg.InitialWait {
			lower = cfg.InitialWait
		}

		for range 100 {
			got := CalculateDelay(cfg, attempt)
			if got < lower || got > upper {
				t.Errorf("CalculateDelay(attempt=%d) = %v, want in [%v, %v]", attempt, got, lower, upper)
			}
		}
	}
}

func TestCalculateDelay_ZeroJitterIsDeterministic(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		InitialWait:    100 * time.Millisecond,
		MaxWait:        10 * time.Second,
		Multiplier:     2.0,
		JitterFraction: 0,
	}

	first := CalculateDelay(cfg, 2)
	for range 50 {
		got := CalculateDelay(cfg, 2)
		if got != first {
			t.Fatalf("expected deterministic result %v, got %v", first, got)
		}
	}
}

func TestWithBackoff_SucceedsFirstAttempt(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		MaxRetries:     3,
		InitialWait:    1 * time.Millisecond,
		MaxWait:        10 * time.Millisecond,
		Multiplier:     2.0,
		JitterFraction: 0,
	}

	calls := 0
	err := WithBackoff(context.Background(), cfg, func() error {
		calls++
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestWithBackoff_SucceedsAfterRetries(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		MaxRetries:     5,
		InitialWait:    1 * time.Millisecond,
		MaxWait:        10 * time.Millisecond,
		Multiplier:     2.0,
		JitterFraction: 0,
	}

	calls := 0
	err := WithBackoff(context.Background(), cfg, func() error {
		calls++
		if calls < 3 {
			return errors.New("transient error")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestWithBackoff_ExhaustsRetries(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		MaxRetries:     2,
		InitialWait:    1 * time.Millisecond,
		MaxWait:        10 * time.Millisecond,
		Multiplier:     2.0,
		JitterFraction: 0,
	}

	sentinel := errors.New("persistent error")
	calls := 0
	err := WithBackoff(context.Background(), cfg, func() error {
		calls++
		return sentinel
	})

	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
	// 1 initial + 2 retries = 3 total
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestWithBackoff_ContextCancellation(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		MaxRetries:     10,
		InitialWait:    1 * time.Second,
		MaxWait:        10 * time.Second,
		Multiplier:     2.0,
		JitterFraction: 0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := WithBackoff(ctx, cfg, func() error {
		calls++
		cancel()
		return errors.New("fail")
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestWithBackoff_NilConfig(t *testing.T) {
	t.Parallel()
	calls := 0
	err := WithBackoff(context.Background(), nil, func() error {
		calls++
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestWithBackoff_ValidationError(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		MaxRetries:  1,
		InitialWait: time.Second,
		MaxWait:     time.Second,
		Multiplier:  0.5, // invalid: < 1.0
	}

	err := WithBackoff(context.Background(), cfg, func() error {
		t.Fatal("operation should not be called when config is invalid")
		return nil
	})

	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCalculateDelay_NonNilConfigIsNotDefaulted(t *testing.T) {
	t.Parallel()
	// Only a nil Config is promoted by WithDefaults; a struct literal reaches the formula exactly as the caller wrote it.
	tests := []struct {
		name    string
		cfg     *Config
		attempt int
		want    time.Duration
	}{
		{"zero value, first attempt", &Config{}, 0, 0},
		{"zero value, later attempt", &Config{}, 5, 0},
		{"partial config keeps zero jitter", &Config{InitialWait: 100 * time.Millisecond, MaxWait: 10 * time.Second, Multiplier: 2.0}, 2, 400 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateDelay(tt.cfg, tt.attempt)
			if got != tt.want {
				t.Errorf("CalculateDelay(%+v, %d) = %v, want %v", tt.cfg, tt.attempt, got, tt.want)
			}
		})
	}
}

func TestCalculateDelay_JitterExceedsMaxWait(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		InitialWait:    1 * time.Second,
		MaxWait:        2 * time.Second,
		Multiplier:     2.0,
		JitterFraction: 0.5,
	}

	upper := time.Duration(float64(cfg.MaxWait) * (1 + cfg.JitterFraction))
	sawAboveMaxWait := false
	for range 200 {
		got := CalculateDelay(cfg, 10)
		if got < cfg.InitialWait || got > upper {
			t.Fatalf("CalculateDelay(attempt=10) = %v, want in [%v, %v]", got, cfg.InitialWait, upper)
		}
		if got > cfg.MaxWait {
			sawAboveMaxWait = true
		}
	}

	// MaxWait caps the exponential, not the jittered result — callers sizing a shutdown budget off MaxWait need the jitter headroom too.
	if !sawAboveMaxWait {
		t.Errorf("no delay exceeded MaxWait %v; jitter is expected to overshoot it", cfg.MaxWait)
	}
}

func TestWithBackoff_PreCancelledContext(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		MaxRetries:     3,
		InitialWait:    1 * time.Second,
		MaxWait:        10 * time.Second,
		Multiplier:     2.0,
		JitterFraction: 0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	err := WithBackoff(ctx, cfg, func() error {
		calls++
		return errors.New("fail")
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	// Cancellation is only checked before a sleep, so the first attempt runs on a context that is already dead.
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}
