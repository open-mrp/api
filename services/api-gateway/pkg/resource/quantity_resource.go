package apiresource

import (
	"fmt"
	"strings"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/decimalfmt"
	apiexample "github.com/open-mrp/apikit/example"
)

const SampleQuantityID = "qty_8hnxbigf0bod"

// A measured amount: a numeric value together with the unit it is expressed in.
//
// Quantities are shared building blocks rather than standalone records — other resources point at them to report stock levels, ordered and packed amounts, money, weights, and durations.
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
	return decimalfmt.Rate(value)
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
	return decimalfmt.Normalize(value, isCurrency)
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
