package ptrutil

import (
	"database/sql"
	"time"
)

func String(s string) *string {
	return &s
}

func Int(i int) *int {
	return &i
}

func Int32(i int32) *int32 {
	return &i
}

func Int64(i int64) *int64 {
	return &i
}

func Float32(f float32) *float32 {
	return &f
}

func Float64(f float64) *float64 {
	return &f
}

func Bool(b bool) *bool {
	return &b
}

func Uint(u uint) *uint {
	return &u
}

func Uint32(u uint32) *uint32 {
	return &u
}

func Uint64(u uint64) *uint64 {
	return &u
}

func Time(t time.Time) *time.Time {
	return &t
}

func TimestampToTimePtr(timestamp string) *time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z", timestamp)
	if err != nil {
		return nil
	}
	return &t
}

func TimestampToTime(timestamp string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z", timestamp)
	if err != nil {
		return time.Time{}
	}
	return t
}

func NullStringToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func NullTimeToPtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	return &nt.Time
}

func NullInt32ToPtr(ni sql.NullInt32) *int {
	if !ni.Valid {
		return nil
	}
	val := int(ni.Int32)
	return &val
}

func NullInt32ToTimePtr(ni sql.NullInt32) *time.Time {
	if !ni.Valid {
		return nil
	}
	t := time.Unix(int64(ni.Int32), 0)
	return &t
}
