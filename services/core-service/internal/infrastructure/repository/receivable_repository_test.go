package repository

import (
	"testing"
	"time"
)

func TestRequirePositiveBalance(t *testing.T) {
	t.Parallel()

	if requirePositiveBalance(nil) {
		t.Fatal("a listing without a cutoff must keep every unpaid invoice, including zero-balance ones")
	}

	cutoff := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	if !requirePositiveBalance(&cutoff) {
		t.Fatal("an as-of run must drop entries already settled by the cutoff")
	}
}

func TestBuildAllocationCutoffParam(t *testing.T) {
	t.Parallel()

	// A null cutoff leaves the window open in SQL. Carrying a far-future sentinel here instead
	// silently matched nothing, because nanosecond precision overflows MySQL's DATETIME.
	open := buildAllocationCutoffParam(nil)
	if open.Valid {
		t.Fatalf("expected no cutoff so every funded allocation counts, got %s", open.Time)
	}

	cutoff := time.Date(2026, 3, 1, 12, 30, 0, 0, time.UTC)
	got := buildAllocationCutoffParam(&cutoff)
	if !got.Valid || !got.Time.Equal(cutoff) {
		t.Fatalf("expected the cutoff to pass through, got %+v", got)
	}
}
