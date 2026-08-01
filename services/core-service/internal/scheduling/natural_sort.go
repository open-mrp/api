package scheduling

import "unicode"

// naturalLess compares strings with embedded numbers numerically, so "Merz 9" sorts before "Merz 10" rather than after it.
//
// This exists because machine assignment is least-loaded-first with the machine order as the tie-break, so lexical ordering would quietly bias load toward whichever machine happened to sort first. It matches the script's localeCompare(..., {numeric: true}).
func naturalLess(a, b string) bool {
	ai, bi := 0, 0
	ar, br := []rune(a), []rune(b)

	for ai < len(ar) && bi < len(br) {
		ac, bc := ar[ai], br[bi]

		if unicode.IsDigit(ac) && unicode.IsDigit(bc) {
			// Consume both digit runs and compare them as numbers. Leading zeros are ignored for magnitude, then used as a tie-break so ordering is total.
			aStart, bStart := ai, bi
			for ai < len(ar) && unicode.IsDigit(ar[ai]) {
				ai++
			}
			for bi < len(br) && unicode.IsDigit(br[bi]) {
				bi++
			}

			aDigits := trimLeadingZeros(ar[aStart:ai])
			bDigits := trimLeadingZeros(br[bStart:bi])

			if len(aDigits) != len(bDigits) {
				return len(aDigits) < len(bDigits)
			}
			for k := range aDigits {
				if aDigits[k] != bDigits[k] {
					return aDigits[k] < bDigits[k]
				}
			}
			continue
		}

		if ac != bc {
			return ac < bc
		}
		ai++
		bi++
	}

	return len(ar)-ai < len(br)-bi
}

func trimLeadingZeros(digits []rune) []rune {
	i := 0
	for i < len(digits)-1 && digits[i] == '0' {
		i++
	}
	return digits[i:]
}
