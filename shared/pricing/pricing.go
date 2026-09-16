// Package pricing turns a line's quantity and unit price into money, the way the dashboard does.
//
// A line keeps its quantity in its own unit while its price is a rate per the rate's denominator
// unit, and the two can differ: three cartons of twelve pairs priced at $22.50 a pair are $810.00,
// not the $67.50 that multiplying the stored values gives. Every extended price, line total and
// order or invoice total goes through ExtendedPrice so that conversion cannot be forgotten.
//
// ExtendedPrice is a port of the dashboard's QuantityUtils.multiplyRate, which both the legacy API
// and the frontend total with: the quantity is normalized to its base unit (times the unit's ratio
// numerator, divided by its ratio denominator), the price is divided by its denominator unit's base
// factor, and the two are multiplied. Every step is rounded to 40 significant digits, half away from
// zero, as the dashboard's decimal.js is configured, so the Go totals agree with the dashboard's to
// the last digit rather than only where the arithmetic happens to terminate.
//
// The dashboard also adds unit offsets and normalizes a non-base currency numerator. No unit carries
// an offset and every currency unit is a base unit, so the queries that supply a UnitConversion
// return none for such a line instead (see the line-pricing skill), and pricing it fails loudly
// rather than silently disagreeing with the dashboard.
package pricing

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// precision is the dashboard's decimal.js precision, in significant digits.
const precision = 40

// UnitConversion carries the base ratios pricing converts between: the ratio of the line's quantity
// unit, and the ratio of the unit its price is quoted per. A unit's ratio is its size in its
// dimension's base unit, kept as the stored numerator/denominator pair because the dashboard
// multiplies and divides by the pair rather than by a reduced ratio.
//
// The zero value is not valid; use Identity for a line priced per its own unit.
type UnitConversion struct {
	QuantityRatioNumerator   decimal.Decimal
	QuantityRatioDenominator decimal.Decimal
	PriceRatioNumerator      decimal.Decimal
	PriceRatioDenominator    decimal.Decimal
}

var one = decimal.NewFromInt(1)

// Identity is the conversion of a line whose quantity is in the unit its price is quoted per.
var Identity = UnitConversion{QuantityRatioNumerator: one, QuantityRatioDenominator: one, PriceRatioNumerator: one, PriceRatioDenominator: one}

// Between is the conversion for a quantity in a unit with the first ratio, priced per a unit with the
// second (each unit.ratio_numerator / unit.ratio_denominator).
func Between(quantityRatioNumerator, quantityRatioDenominator, priceRatioNumerator, priceRatioDenominator decimal.Decimal) UnitConversion {
	return UnitConversion{
		QuantityRatioNumerator:   quantityRatioNumerator,
		QuantityRatioDenominator: quantityRatioDenominator,
		PriceRatioNumerator:      priceRatioNumerator,
		PriceRatioDenominator:    priceRatioDenominator,
	}
}

// ParseUnitConversion reads the four ratio terms a query or message carries as decimal strings. An
// empty or zero term is an error rather than a silent identity: the query leaves the terms empty for a
// line the dashboard would not price the same way, and a zero ratio cannot be divided by.
func ParseUnitConversion(quantityRatioNumerator, quantityRatioDenominator, priceRatioNumerator, priceRatioDenominator string) (UnitConversion, error) {
	terms := [4]string{quantityRatioNumerator, quantityRatioDenominator, priceRatioNumerator, priceRatioDenominator}
	var parsed [4]decimal.Decimal
	for i, term := range terms {
		d, err := decimal.NewFromString(term)
		if err != nil {
			return UnitConversion{}, fmt.Errorf("unit conversion %v: term %q: %w", terms, term, err)
		}
		if d.IsZero() {
			return UnitConversion{}, fmt.Errorf("unit conversion %v has a zero term", terms)
		}
		parsed[i] = d
	}
	return Between(parsed[0], parsed[1], parsed[2], parsed[3]), nil
}

// ExtendedPrice is qty (in the line's unit) times unitPrice (per the price's unit), unrounded, as the
// dashboard's multiplyRate computes it.
func ExtendedPrice(qty, unitPrice decimal.Decimal, conv UnitConversion) decimal.Decimal {
	// normalizeQuantity: measure.times(ratioNumerator).div(ratioDenominator). A unit whose ratio is
	// one is a base unit, which the dashboard leaves untouched.
	normalizedQty := qty
	if !isUnitRatio(conv.QuantityRatioNumerator, conv.QuantityRatioDenominator) {
		normalizedQty = div(mul(qty, conv.QuantityRatioNumerator), conv.QuantityRatioDenominator)
	}

	// normalizeRate: the measure divided by normalizeQuantity(1, denominatorUnit).
	normalizedPrice := unitPrice
	if !isUnitRatio(conv.PriceRatioNumerator, conv.PriceRatioDenominator) {
		factor := div(mul(one, conv.PriceRatioNumerator), conv.PriceRatioDenominator)
		normalizedPrice = div(unitPrice, factor)
	}

	return mul(normalizedQty, normalizedPrice)
}

// LineTotal is ExtendedPrice rounded to the cent, half away from zero, as the dashboard's
// calculateTotalOrdered and calculateTotalInvoiced round each line before summing.
func LineTotal(qty, unitPrice decimal.Decimal, conv UnitConversion) decimal.Decimal {
	return ExtendedPrice(qty, unitPrice, conv).Round(2)
}

func isUnitRatio(numerator, denominator decimal.Decimal) bool {
	return numerator.Equal(denominator)
}

// mul is decimal.js times at the dashboard's precision.
func mul(a, b decimal.Decimal) decimal.Decimal {
	return significant(a.Mul(b))
}

// div is decimal.js div at the dashboard's precision. The quotient is truncated a few digits past
// the precision and then rounded, which rounds the same way the exact quotient would: truncation only
// moves a value toward zero, and never far enough to cross the halfway point it is rounded against.
func div(a, b decimal.Decimal) decimal.Decimal {
	if a.IsZero() {
		return a
	}
	places := precision - leadingExponent(a) + leadingExponent(b) + 5
	q, _ := a.QuoRem(b, places)
	return significant(q)
}

// significant rounds d to the dashboard's precision in significant digits, half away from zero.
func significant(d decimal.Decimal) decimal.Decimal {
	if d.IsZero() {
		return d
	}
	return d.Round(precision - 1 - leadingExponent(d))
}

// leadingExponent is the power of ten of d's leading digit.
func leadingExponent(d decimal.Decimal) int32 {
	return int32(d.NumDigits()) + d.Exponent() - 1
}
