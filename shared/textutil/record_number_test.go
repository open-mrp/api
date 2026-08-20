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

func TestFormatAccountNumber(t *testing.T) {
	cases := map[string]string{
		"8841":   "08841", // pads to five, not six
		"08841":  "08841", // a leading zero means it is already formatted
		"C-8841": "C-8841",
		"123456": "123456", // longer than five is left alone
		"":       "",
	}
	for in, want := range cases {
		if got := FormatAccountNumber(in); got != want {
			t.Errorf("FormatAccountNumber(%q) = %q, want %q", in, got, want)
		}
	}
}
