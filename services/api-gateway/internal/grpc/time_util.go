package grpc

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// TimestampToTime converts a protobuf timestamp to a time.Time.
func TimestampToTime(t *timestamppb.Timestamp) time.Time {
	return t.AsTime()
}

// TimestampToTimePtr converts a protobuf timestamp to a time.Time pointer.
func TimestampToTimePtr(t *timestamppb.Timestamp) *time.Time {
	if t != nil {
		t := t.AsTime()
		return &t
	}
	return nil
}
