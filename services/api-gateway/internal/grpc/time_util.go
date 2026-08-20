package grpc

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// TimestampToTime converts a protobuf timestamp to a time.Time.
func TimestampToTime(t *timestamppb.Timestamp) time.Time {
	return t.AsTime()
}

// ParseDateString parses a date string in YYYY-MM-DD format to a time.Time.
func ParseDateString(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

// Parses an inclusive end date (YYYY-MM-DD) as the last microsecond of that day, so rows created
// during the day still match `<= end`. Microseconds, not nanoseconds — DATETIME(6) stores no more.
func ParseEndDateString(s string) (time.Time, error) {
	t, err := ParseDateString(s)
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
