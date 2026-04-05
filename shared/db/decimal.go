package db

import "strings"

// TrimDecimal removes unnecessary trailing zeros from a MySQL DECIMAL string.
// "1.000000000000000000000000000000" → "1", "10.500000..." → "10.5".
func TrimDecimal(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}
