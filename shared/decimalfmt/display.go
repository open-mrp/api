package decimalfmt

import "strings"

// Display formats a decimal for reading: normalized as Normalize does, with commas between thousands
// ("1234.5" with twoPlaces reads "1,234.50").
func Display(value string, twoPlaces bool) string {
	s := Normalize(value, twoPlaces)
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign, s = "-", s[1:]
	}
	intPart, frac, hasFrac := strings.Cut(s, ".")
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if hasFrac {
		return sign + b.String() + "." + frac
	}
	return sign + b.String()
}
