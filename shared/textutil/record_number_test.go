package textutil

import "testing"

func TestFormatRecordNumber(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"123":     "000123",
		"1":       "000001",
		"123456":  "123456",
		"1234567": "1234567", // already >= 6 digits, unchanged
		"123-45":  "000123-45",
		"SO-001":  "SO-001", // alpha prefix untouched
		"":        "",
	}
	for in, want := range cases {
		if got := FormatRecordNumber(in); got != want {
			t.Errorf("FormatRecordNumber(%q) = %q, want %q", in, got, want)
		}
	}
}
