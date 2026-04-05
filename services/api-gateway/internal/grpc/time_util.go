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

// TimestampToTimePtr converts a protobuf timestamp to a time.Time pointer.
func TimestampToTimePtr(t *timestamppb.Timestamp) *time.Time {
	if t != nil {
		t := t.AsTime()
		return &t
	}
	return nil
}
