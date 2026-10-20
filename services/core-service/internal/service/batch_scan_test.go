package service

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The expected strings are what the dashboard's scanning service printed for the same float64, so the
// material check reads the same after the move to Go. They include the cases a naive port gets wrong:
// exact binary ties (1.03125), values whose decimal looks like a tie but whose float is not (7.00005),
// and results that round to zero.
func TestFormatQuantityMatchesTheDashboard(t *testing.T) {
	t.Parallel()

	tenth, fifth := 0.1, 0.2
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{math.Copysign(0, -1), "0"},
		{12, "12"},
		{-12, "-12"},
		{12.5, "12.5"},
		{1.03125, "1.0313"},
		{-1.03125, "-1.0313"},
		{0.09375, "0.0938"},
		{1.00005, "1.0001"},
		{7.00005, "7"},
		{33.333333333333336, "33.3333"},
		{2.0 / 3, "0.6667"},
		{-2.0 / 3, "-0.6667"},
		{0.00004, "0"},
		{-0.00004, "0"},
		{0.00005, "0.0001"},
		{99999.99995, "99999.9999"},
		{tenth + fifth, "0.3"},
		{1234567890.12345, "1234567890.1235"},
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
		{math.NaN(), "NaN"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, formatQuantity(c.in), "formatQuantity(%v)", c.in)
	}
}
