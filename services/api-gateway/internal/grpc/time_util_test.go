package grpc

import "testing"

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
