package repository

import (
	"testing"
	"time"
)

func TestParseDateString_AcceptsBothDocumentedFormats(t *testing.T) {
	t.Parallel()

	want := time.Date(2026, time.May, 10, 0, 0, 0, 0, time.UTC)

	for _, in := range []string{"2026-05-10", "2026-05-10T00:00:00Z"} {
		got := parseDateString(&in)
		if !got.Valid {
			t.Fatalf("%q produced no filter; a documented format that parses as nothing silently widens the query", in)
		}
		if !got.Time.Equal(want) {
			t.Fatalf("%q = %s, want %s", in, got.Time, want)
		}
	}
}

// The RFC3339 form carries a time of day, and dropping it would move an ends_at filter to midnight and quietly exclude the rest of that day.
func TestParseDateString_KeepsTheTimeOfDayFromAnInstant(t *testing.T) {
	t.Parallel()

	in := "2026-05-10T00:23:00Z"
	got := parseDateString(&in)
	if !got.Valid {
		t.Fatal("expected a filter")
	}
	if want := time.Date(2026, time.May, 10, 0, 23, 0, 0, time.UTC); !got.Time.Equal(want) {
		t.Fatalf("got %s, want %s", got.Time, want)
	}
}

func TestParseDateString_AbsentAndEmptyProduceNoFilter(t *testing.T) {
	t.Parallel()

	if got := parseDateString(nil); got.Valid {
		t.Fatal("nil should produce no filter")
	}
	empty := ""
	if got := parseDateString(&empty); got.Valid {
		t.Fatal("empty should produce no filter")
	}
}

// Documented, not endorsed: an unparseable value is dropped rather than rejected, so the caller gets an unfiltered list instead of a 400. Pinned so the behavior is a decision rather than an accident, and so changing it is a deliberate edit to this test.
func TestParseDateString_GarbageIsSilentlyDropped(t *testing.T) {
	t.Parallel()

	in := "last tuesday"
	if got := parseDateString(&in); got.Valid {
		t.Fatal("expected no filter from an unparseable value")
	}
}

// parseDateFilter is the pick/shipment name for the same parser; the two drifting apart is what let one of them reject a format the other accepted.
func TestParseDateFilter_MatchesParseDateString(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"2026-05-10", "2026-05-10T00:23:00Z", "", "nonsense"} {
		value := in
		a, b := parseDateString(&value), parseDateFilter(&value)
		if a.Valid != b.Valid || !a.Time.Equal(b.Time) {
			t.Fatalf("%q: parseDateString = %v/%s, parseDateFilter = %v/%s", in, a.Valid, a.Time, b.Valid, b.Time)
		}
	}
}
