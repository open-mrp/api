package decimalfmt

import "testing"

func TestRate(t *testing.T) {
	for in, want := range map[string]string{
		"12.500000000000000000000000000000": "12.50",
		"3.250000000000000000000000000000":  "3.25",
		"7.000000000000000000000000000000":  "7.00",
		"0.123400000000000000000000000000":  "0.1234",
		"-2.5":                              "-2.50",
		"25":                                "25",
		"":                                  "0",
	} {
		if got := Rate(in); got != want {
			t.Errorf("Rate(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeTwoPlaces(t *testing.T) {
	for in, want := range map[string]string{
		"1234.567000": "1234.56",
		"5":           "5.00",
		"007.1":       "7.10",
	} {
		if got := Normalize(in, true); got != want {
			t.Errorf("Normalize(%q, true) = %q, want %q", in, got, want)
		}
	}
}
