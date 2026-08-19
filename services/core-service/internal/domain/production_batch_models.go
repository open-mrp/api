package domain

import "github.com/shopspring/decimal"

// ProductionBatchLineInput identifies one order line's produced item and ordered
// quantity, the input to the production-run batch computation.
type ProductionBatchLineInput struct {
	ProducedItemID string
	OrderedMeasure decimal.Decimal
	OrderedUnitID  string
}

// ProductionBatchItem is a computed production batch: a material-only-block item and
// the quantity to produce, expressed in UnitID.
type ProductionBatchItem struct {
	ItemID  string
	Measure decimal.Decimal
	UnitID  string
}

// UnitFactors carries a unit's linear conversion to/from its dimension's base unit:
// base = (value * ratioNum / ratioDen) + (offsetNum / offsetDen).
type UnitFactors struct {
	RatioNum   decimal.Decimal
	RatioDen   decimal.Decimal
	OffsetNum  decimal.Decimal
	OffsetDen  decimal.Decimal
	IsBaseUnit bool
	// DimensionCode is what the unit measures, so a caller that needs a duration can reject a unit that measures socks.
	DimensionCode string
}

// ToBase converts a measure in this unit to its dimension's base measure.
func (f UnitFactors) ToBase(v decimal.Decimal) decimal.Decimal {
	if f.IsBaseUnit {
		return v
	}
	return v.Mul(f.ratio()).Add(f.offset())
}

// FromBase converts a base measure to a measure in this unit.
func (f UnitFactors) FromBase(base decimal.Decimal) decimal.Decimal {
	if f.IsBaseUnit {
		return base
	}
	r := f.ratio()
	if r.IsZero() {
		return base.Sub(f.offset())
	}
	return base.Sub(f.offset()).Div(r)
}

func (f UnitFactors) ratio() decimal.Decimal {
	if f.RatioDen.IsZero() {
		return decimal.NewFromInt(1)
	}
	return f.RatioNum.Div(f.RatioDen)
}

func (f UnitFactors) offset() decimal.Decimal {
	if f.OffsetDen.IsZero() {
		return decimal.Zero
	}
	return f.OffsetNum.Div(f.OffsetDen)
}
