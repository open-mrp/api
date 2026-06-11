package apiresource

import (
	"fmt"
	"math/big"
	"strings"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

const SampleQuantityID = "qty_015a85becc1a6afdfb1afc27ff"

// Value with an associated unit.
type Quantity struct {
	// Quantity ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=quantity"`
	// Raw decimal value of the quantity, as a string to preserve precision.
	//
	// This is the unformatted machine value; see `display_value` for the human-readable rendering with unit and thousands separators.
	Value string `json:"value" validate:"required" format:"decimal"`
	// Formatted value with unit abbreviation (e.g. "$1,234.56" or "100 kg").
	DisplayValue string `json:"display_value" validate:"required"`
	// Unit of measure for this value (e.g. a currency, mass, or count unit).
	Unit *Unit `json:"unit" expandable:"true"`
}

var SampleQuantity = &Quantity{
	ID:           SampleQuantityID,
	Object:       constants.ObjectTypeQuantity,
	Value:        "1234.56",
	DisplayValue: "$1,234.56",
	Unit:         newSampleUnit("US Dollar", "$", constants.UnitTypeCurrency),
}

func (*Quantity) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleQuantity)
}

// NormalizeQuantityValue returns a canonical decimal string for API quantity values, trimming database fixed-point padding. Currency units use two fractional digits; other dimensions trim trailing fractional zeros.
func NormalizeQuantityValue(value, unitType string) string {
	return normalizeDecimalString(value, unitType == string(constants.UnitTypeCurrency))
}

// NormalizeMonetaryQuantityValue returns a decimal string with exactly two fractional digits. Use for price-like quantities whose unit dimension is not currency (e.g. shipping flat rates).
func NormalizeMonetaryQuantityValue(value string) string {
	return normalizeDecimalString(value, true)
}

// NormalizeRateValue returns a canonical decimal string for API rate values. It trims database fixed-point padding while preserving at least two fractional digits when the source value is fractional.
func NormalizeRateValue(value string) string {
	normalized := normalizeDecimalString(value, false)
	if !strings.Contains(value, ".") {
		return normalized
	}
	parts := strings.SplitN(normalized, ".", 2)
	if len(parts) == 1 {
		return normalized + ".00"
	}
	if len(parts[1]) == 0 {
		return normalized + "00"
	}
	if len(parts[1]) == 1 {
		return normalized + "0"
	}
	return normalized
}

// FormatDisplayValue formats a decimal value string with a unit abbreviation. For currency units, the abbreviation is placed before the value (e.g. "$1,234.56"). For other units, the abbreviation is placed after the value (e.g. "100 kg").
func FormatDisplayValue(value, unitAbbreviation, unitType string) string {
	formatted := formatDecimal(value, unitType == string(constants.UnitTypeCurrency))
	if unitType == string(constants.UnitTypeCurrency) {
		return unitAbbreviation + formatted
	}
	return formatted + " " + unitAbbreviation
}

func normalizeDecimalString(value string, isCurrency bool) string {
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

	if isCurrency {
		if len(decPart) >= 2 {
			decPart = decPart[:2]
		} else {
			decPart = (decPart + "00")[:2]
		}
	} else {
		decPart = strings.TrimRight(decPart, "0")
	}

	var result string
	if decPart != "" {
		result = intPart + "." + decPart
	} else {
		result = intPart
	}

	if negative {
		result = "-" + result
	}

	return result
}

// formatDecimal formats a decimal string with comma separators. For currency, always shows 2 decimal places. For other types, trims trailing zeros.
func formatDecimal(value string, isCurrency bool) string {
	s := normalizeDecimalString(value, isCurrency)
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	negative := false
	if strings.HasPrefix(intPart, "-") {
		negative = true
		intPart = intPart[1:]
	}
	intPart = addCommas(intPart)
	var result string
	if len(parts) == 2 {
		result = intPart + "." + parts[1]
	} else {
		result = intPart
	}
	if negative {
		result = "-" + result
	}
	return result
}

// addCommas inserts commas as thousands separators into an integer string.
func addCommas(s string) string {
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	remainder := len(s) % 3
	if remainder > 0 {
		b.WriteString(s[:remainder])
	}
	for i := remainder; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		fmt.Fprint(&b, s[i:i+3])
	}
	return b.String()
}
