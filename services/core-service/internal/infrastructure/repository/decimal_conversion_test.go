package repository

import (
	"database/sql"
	"testing"
)

// sqlc types the same column differently depending on the query it appears in — a ratio column is a
// bare string where its table is inner-joined and a sql.NullString where it is LEFT JOINed. The
// unknown-type default returns 0, so a caller that passed the wrapper lost the number silently.
//
// That is not hypothetical: the rate-denominator ratio arrived wrapped in GetItemRunRateHistory,
// read as zero, and skipped the quantity conversion, leaving every pair-rated step at twice its true
// seconds per unit — the exact bug the surrounding change exists to fix.
func TestDecimalToFloat64_HandlesNullString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   any
		want float64
	}{
		{"bare string", "2.5", 2.5},
		{"valid NullString converts, not zero", sql.NullString{String: "2.5", Valid: true}, 2.5},
		{"null NullString is zero", sql.NullString{Valid: false}, 0},
		{"null NullString ignores its string", sql.NullString{String: "2.5", Valid: false}, 0},
		{"the ratio that bit us", sql.NullString{String: "2.000000000000000000000000000000", Valid: true}, 2},
		{"negative", sql.NullString{String: "-1.25", Valid: true}, -1.25},
		{"bytes", []uint8("3.5"), 3.5},
		{"unknown type is still zero", struct{}{}, 0},
	}

	for _, c := range cases {
		if got := decimalToFloat64(c.in); got != c.want {
			t.Errorf("%s: decimalToFloat64(%#v) = %v, want %v", c.name, c.in, got, c.want)
		}
	}
}
