package grpc

import (
	"testing"
	"time"
)

// An end date must cover the whole day: a row stamped at 14:30 still falls on or before that date.
func TestParseEndDateString_CoversTheWholeDay(t *testing.T) {
	end, err := ParseEndDateString("2026-08-11")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got := end.Format("2006-01-02 15:04:05.000000"); got != "2026-08-11 23:59:59.999999" {
		t.Errorf("end of day = %q, want 2026-08-11 23:59:59.999999", got)
	}

	// Nanosecond precision would overflow DATETIME(6) and round up into the next day.
	if end.Nanosecond()%1000 != 0 {
		t.Errorf("end carries sub-microsecond precision (%d ns), which DATETIME(6) cannot store", end.Nanosecond())
	}

	if _, err := ParseEndDateString("not-a-date"); err == nil {
		t.Error("a malformed date must still error")
	}
}

func TestParseDateStringAcceptsTimestamps(t *testing.T) {
	start, err := ParseDateString("2026-09-01T04:00:00Z")
	if err != nil || !start.Equal(time.Date(2026, 9, 1, 4, 0, 0, 0, time.UTC)) {
		t.Fatalf("ParseDateString(timestamp) = %v, %v", start, err)
	}
	end, err := ParseEndDateString("2026-09-30T03:59:59.999-00:00")
	if err != nil || !end.Equal(time.Date(2026, 9, 30, 3, 59, 59, 999_000_000, time.UTC)) {
		t.Fatalf("ParseEndDateString(timestamp) = %v, %v", end, err)
	}
	day, err := ParseEndDateString("2026-09-30")
	if err != nil || !day.Equal(time.Date(2026, 9, 30, 23, 59, 59, 999_999_000, time.UTC)) {
		t.Fatalf("ParseEndDateString(day) = %v, %v", day, err)
	}
}
