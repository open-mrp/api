package analyticsep

import (
	"github.com/shopspring/decimal"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
)

// computedRate renders a price as an unpersisted rate. The unit sub-objects are left for the include resolver; display_value carries the readable form so a caller that does not expand them still sees the basis.
func computedRate(value, numeratorAbbr, denominatorAbbr string) *apiresource.ComputedRate {
	amount, err := decimal.NewFromString(value)
	if err != nil {
		amount = decimal.Zero
	}
	return &apiresource.ComputedRate{
		Object:       constants.ObjectTypeComputedRate,
		Value:        amount.StringFixed(4),
		DisplayValue: apiresource.FormatRateDisplay(value, numeratorAbbr, denominatorAbbr),
	}
}

// computedQuantity renders an amount as an unpersisted quantity.
func computedQuantity(value, unitAbbr string) *apiresource.ComputedQuantity {
	amount, err := decimal.NewFromString(value)
	if err != nil {
		amount = decimal.Zero
	}
	display := amount.StringFixed(2)
	if unitAbbr != "" {
		display += " " + unitAbbr
	}
	return &apiresource.ComputedQuantity{
		Object:       constants.ObjectTypeComputedQuantity,
		Value:        amount.StringFixed(4),
		DisplayValue: display,
	}
}
