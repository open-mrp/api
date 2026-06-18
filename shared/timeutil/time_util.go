// Package timeutil provides helpers for parsing timestamp strings used throughout the platform's data layer. All functions expect the ISO 8601 / RFC 3339 format YYYY-MM-DDTHH:MM:SSZ (UTC, no fractional seconds). This is the canonical format returned by MySQL's DATETIME columns when serialized to JSON or gRPC string fields.
package timeutil

import (
	"time"
)

// TimestampToTimePtr parses a UTC timestamp string (YYYY-MM-DDTHH:MM:SSZ) into a *time.Time. Returns nil if the string does not match the expected format, making it safe to use directly in struct assignments where a nil pointer represents "not set" (e.g. optional processed_at or expires_at fields).
func TimestampToTimePtr(timestamp string) *time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z", timestamp)
	if err != nil {
		return nil
	}
	return &t
}

// TimestampToTime parses a UTC timestamp string (YYYY-MM-DDTHH:MM:SSZ) into a time.Time value. Returns the zero time (time.Time{}) if the string does not match the expected format. Use TimestampToTimePtr when you need to distinguish between "missing" (nil) and "zero time".
func TimestampToTime(timestamp string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z", timestamp)
	if err != nil {
		return time.Time{}
	}
	return t
}
