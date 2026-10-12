// Package decimalfmt renders stored fixed-point decimals the way the public API presents them.
package decimalfmt

import (
	"math/big"
	"strings"
)

// Normalize trims database fixed-point padding from a decimal string. twoPlaces truncates or pads
// the fraction to exactly two digits (money); otherwise trailing fractional zeros are dropped.
func Normalize(value string, twoPlaces bool) string {
	if value == "" {
		return "0"
	}

	parts := strings.SplitN(value, ".", 2)
	intPart := parts[0]
	decPart := ""
	if len(parts) == 2 {
		decPart = parts[1]
	}

	negative := false
	if strings.HasPrefix(intPart, "-") {
		negative = true
		intPart = intPart[1:]
	}

	n := new(big.Int)
	if _, ok := n.SetString(intPart, 10); !ok {
		intPart = "0"
	} else {
		intPart = n.String()
	}

	if twoPlaces {
		if len(decPart) >= 2 {
			decPart = decPart[:2]
		} else {
			decPart = (decPart + "00")[:2]
		}
	} else {
		decPart = strings.TrimRight(decPart, "0")
	}

	result := intPart
	if decPart != "" {
		result = intPart + "." + decPart
	}
	if negative {
		result = "-" + result
	}
	return result
}

// Rate is the public rate value: padding trimmed, but at least two fractional digits when the
// source value is fractional.
func Rate(value string) string {
	normalized := Normalize(value, false)
	if !strings.Contains(value, ".") {
		return normalized
	}
	parts := strings.SplitN(normalized, ".", 2)
	if len(parts) == 1 {
		return normalized + ".00"
	}
	if len(parts[1]) == 1 {
		return normalized + "0"
	}
	return normalized
}
