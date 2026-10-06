package domain

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	apierror "github.com/open-mrp/api/shared/errors"
)

// ConvertScanQuantity restates a measure in another unit of the same type, the way the scanning
// floor's quantity math always has: the same unit is returned untouched, and a unit of another type is
// an error rather than a guess.
func ConvertScanQuantity(ctx context.Context, conv UnitConversionRepo, measure decimal.Decimal, from, to LightUnit) (decimal.Decimal, *apierror.APIError) {
	if from.ID == to.ID {
		return measure, nil
	}
	if from.Type != "" && to.Type != "" && from.Type != to.Type {
		return decimal.Zero, apierror.NewValidationError(fmt.Sprintf(
			"Cannot convert between quantity unit %q and new unit %q. Units must be in the same unit group.",
			unitLabel(from), unitLabel(to),
		))
	}
	return conv.ConvertValue(ctx, measure, from.ID, to.ID)
}

func unitLabel(u LightUnit) string {
	if u.Name != "" {
		return u.Name
	}
	if u.Abbreviation != "" {
		return u.Abbreviation
	}
	return u.ID
}

// AsScanNumber passes a computed quantity through a float64, which is what the dashboard's scanning
// math has always stored: equal inputs give equal outputs on both sides, and comparisons between
// outputs behave the same.
func AsScanNumber(d decimal.Decimal) decimal.Decimal {
	return decimal.NewFromFloat(d.InexactFloat64())
}
