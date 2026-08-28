package safeconv

import (
	"math"
	"testing"
)

func TestInt64ToInt32(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   int64
		want int32
	}{
		{"zero", 0, 0},
		{"small positive", 42, 42},
		{"small negative", -42, -42},
		{"one", 1, 1},
		{"negative one", -1, -1},
		{"max int32 passes through unchanged", math.MaxInt32, math.MaxInt32},
		{"min int32 passes through unchanged", math.MinInt32, math.MinInt32},
		{"just below max int32", math.MaxInt32 - 1, math.MaxInt32 - 1},
		{"just above min int32", math.MinInt32 + 1, math.MinInt32 + 1},
		{"max int32 plus one clamps", math.MaxInt32 + 1, math.MaxInt32},
		{"min int32 minus one clamps", math.MinInt32 - 1, math.MinInt32},
		{"max int64 clamps instead of wrapping", math.MaxInt64, math.MaxInt32},
		{"min int64 clamps instead of wrapping", math.MinInt64, math.MinInt32},
		{"large positive clamps", 1 << 40, math.MaxInt32},
		{"large negative clamps", -(1 << 40), math.MinInt32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Int64ToInt32(tt.in); got != tt.want {
				t.Errorf("Int64ToInt32(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestIntToInt32(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   int
		want int32
	}{
		{"zero", 0, 0},
		{"small positive", 42, 42},
		{"small negative", -42, -42},
		{"max int32 passes through unchanged", math.MaxInt32, math.MaxInt32},
		{"min int32 passes through unchanged", math.MinInt32, math.MinInt32},
		{"just below max int32", math.MaxInt32 - 1, math.MaxInt32 - 1},
		{"just above min int32", math.MinInt32 + 1, math.MinInt32 + 1},
		{"max int clamps instead of wrapping", math.MaxInt, math.MaxInt32},
		{"min int clamps instead of wrapping", math.MinInt, math.MinInt32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IntToInt32(tt.in); got != tt.want {
				t.Errorf("IntToInt32(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

// A clamped conversion must never flip sign, which is what a silent truncation does.
func TestClampPreservesSign(t *testing.T) {
	t.Parallel()
	if got := Int64ToInt32(math.MaxInt32 + 1); got < 0 {
		t.Errorf("Int64ToInt32(MaxInt32+1) = %d, wrapped negative", got)
	}
	if got := Int64ToInt32(math.MinInt32 - 1); got > 0 {
		t.Errorf("Int64ToInt32(MinInt32-1) = %d, wrapped positive", got)
	}
	if got := IntToInt32(math.MaxInt32 + 1); got < 0 {
		t.Errorf("IntToInt32(MaxInt32+1) = %d, wrapped negative", got)
	}
}
