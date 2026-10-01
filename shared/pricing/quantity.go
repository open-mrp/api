package pricing

import "github.com/shopspring/decimal"

// UnitRatio is a unit's size in its dimension's base unit, kept as the stored
// unit.ratio_numerator / unit.ratio_denominator pair.
type UnitRatio struct {
	Numerator   decimal.Decimal
	Denominator decimal.Decimal
}

// ParseUnitRatio reads a unit's ratio terms as a query returns them. A zero or unparseable term is an
// error: a ratio cannot be divided by zero, and guessing one would silently misstate a quantity.
func ParseUnitRatio(numerator, denominator string) (UnitRatio, error) {
	conv, err := ParseUnitConversion(numerator, denominator, "1", "1")
	if err != nil {
		return UnitRatio{}, err
	}
	return UnitRatio{Numerator: conv.QuantityRatioNumerator, Denominator: conv.QuantityRatioDenominator}, nil
}

// ConvertQuantity re-expresses qty, counted in a unit of ratio from, in a unit of ratio to, as the
// dashboard's UnitGroupUtils.convertValue does: two units of the same size convert exactly, and
// otherwise the quantity is taken to the base unit and then divided by the target's size.
//
// Offsets are not applied. They exist only for interval scales such as temperature, which nothing
// is ever stocked or received in.
func ConvertQuantity(qty decimal.Decimal, from, to UnitRatio) decimal.Decimal {
	if from.Numerator.Mul(to.Denominator).Equal(to.Numerator.Mul(from.Denominator)) {
		return qty
	}
	return div(mul(div(mul(qty, from.Numerator), from.Denominator), to.Denominator), to.Numerator)
}
