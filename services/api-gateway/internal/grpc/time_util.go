package grpc

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// TimestampToTime converts a protobuf timestamp to a time.Time.
func TimestampToTime(t *timestamppb.Timestamp) time.Time {
	return t.AsTime()
}

// ParseDateString parses a date string in YYYY-MM-DD format (a UTC day) to a time.Time. A full
// RFC 3339 timestamp is also accepted and used as given, so a caller can bound the range by its own
// time zone's midnight.
func ParseDateString(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

// Parses an inclusive end date (YYYY-MM-DD) as the last microsecond of that day, so rows created
// during the day still match `<= end`. Microseconds, not nanoseconds — DATETIME(6) stores no more.
// A full RFC 3339 timestamp is taken as the exact inclusive end.
func ParseEndDateString(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return t, err
	}
	return t.Add(24*time.Hour - time.Microsecond), nil
}

// TimestampToTimePtr converts a protobuf timestamp to a time.Time pointer.
func TimestampToTimePtr(t *timestamppb.Timestamp) *time.Time {
	if t != nil {
		t := t.AsTime()
		return &t
	}
	return nil
}
