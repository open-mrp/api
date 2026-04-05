// Package safeconv provides safe integer conversion functions that clamp
// values at the target type's boundaries instead of silently overflowing.
package safeconv

import "math"

// Int64ToInt32 converts an int64 to int32, clamping at math.MaxInt32 / math.MinInt32.
func Int64ToInt32(v int64) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v) // #nosec G115 -- bounds checked above
}

// IntToInt32 converts an int to int32, clamping at math.MaxInt32 / math.MinInt32.
func IntToInt32(v int) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v) // #nosec G115 -- bounds checked above
}
