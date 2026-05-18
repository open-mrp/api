package db

import (
	"database/sql"
)

func NullInt64Ptr(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{Int64: 0, Valid: false}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}

// Int64FromInterface extracts an int64 from an interface{} value.
// MySQL CASE expressions are typed as interface{} by sqlc and may arrive
// as int64, int, or []byte depending on the driver. Returns (0, false) for nil.
func Int64FromInterface(v any) (int64, bool) {
	if v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case []byte:
		// MySQL may return numeric columns as []byte in some contexts.
		var val int64
		for _, b := range n {
			if b < '0' || b > '9' {
				return 0, false
			}
			val = val*10 + int64(b-'0')
		}
		return val, len(n) > 0
	default:
		return 0, false
	}
}
