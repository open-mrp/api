package shippo

import "testing"

func TestNormalizeShippoDecimal(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"float accumulation garbage", "1.0499999999999998", "1.05"},
		{"clean integer", "5", "5"},
		{"clean integer with point", "13", "13"},
		{"clean decimal", "23.5", "23.5"},
		{"trims trailing zeros", "9.5000", "9.5"},
		{"rounds to four places", "0.30000000000000004", "0.3"},
		{"rounds long fraction", "12.123456789", "12.1235"},
		{"zero", "0", "0"},
		{"unparseable passes through", "abc", "abc"},
		{"empty passes through", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeShippoDecimal(tc.in); got != tc.want {
				t.Errorf("normalizeShippoDecimal(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
