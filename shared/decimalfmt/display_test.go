package decimalfmt

import "testing"

func TestDisplay(t *testing.T) {
	for _, tc := range []struct {
		value     string
		twoPlaces bool
		want      string
	}{
		{"1234.5", true, "1,234.50"},
		{"1234567.000000", false, "1,234,567"},
		{"-1234.5", true, "-1,234.50"},
		{"999", false, "999"},
		{"12.250000000000000000000000000000", false, "12.25"},
		{"", true, "0"},
	} {
		if got := Display(tc.value, tc.twoPlaces); got != tc.want {
			t.Errorf("Display(%q, %v) = %q, want %q", tc.value, tc.twoPlaces, got, tc.want)
		}
	}
}
