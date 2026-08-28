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

// The caller-side non-empty check on parts[0] is the only thing keeping startsWithASCIILetter from indexing an empty string.
func TestFormatRecordNumber_EmptyFirstSegment(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"-":     "-",
		"--":    "--",
		"-45":   "-45",
		"-45-6": "-45-6",
		"-SO":   "-SO",
	}
	for in, want := range cases {
		if got := FormatRecordNumber(in); got != want {
			t.Errorf("FormatRecordNumber(%q) = %q, want %q", in, got, want)
		}
	}
}

// Parity with the dashboard's formatRecordNumber (packages/objects/src/utils/format-record-number.ts):
// the same inputs its own tests cover must produce the same output here, or a PDF and the UI disagree.
func TestFormatRecordNumber_DashboardParity(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"A123":     "A123",
		"B1":       "B1",
		"Z999":     "Z999",
		"A123-01":  "A123-01",
		"B1-99":    "B1-99",
		"a123":     "a123",
		"b1-99":    "b1-99",
		"123":      "000123",
		"123-45":   "000123-45",
		"00123":    "000123",
		"00001-99": "000001-99",
	}
	for in, want := range cases {
		if got := FormatRecordNumber(in); got != want {
			t.Errorf("FormatRecordNumber(%q) = %q, want %q", in, got, want)
		}
	}
}

// Padding is byte-based, so a multibyte first segment pads to six bytes rather than six characters.
// The dashboard's padStart counts UTF-16 units and yields "00über" for the same input.
func TestFormatRecordNumber_NonASCIIFirstSegment(t *testing.T) {
	t.Parallel()
	if got := FormatRecordNumber("über"); got != "0über" {
		t.Errorf("FormatRecordNumber(%q) = %q, want %q", "über", got, "0über")
	}
}
