package textutil

import "strings"

// FormatRecordNumber left-pads the numeric first segment of a record number to
// six digits, mirroring the dashboard's formatRecordNumber. A first segment that
// begins with an ASCII letter (e.g. an "SO-" prefix) is left untouched.
//
// Examples: "123" -> "000123", "123-45" -> "000123-45", "SO-001" -> "SO-001".
func FormatRecordNumber(number string) string {
	parts := strings.Split(number, "-")
	if len(parts) > 0 && parts[0] != "" && !startsWithASCIILetter(parts[0]) {
		parts[0] = leftPadZeros(parts[0], 6)
	}
	return strings.Join(parts, "-")
}

func startsWithASCIILetter(s string) bool {
	c := s[0]
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func leftPadZeros(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat("0", width-len(s)) + s
}
