package apiresource

import (
	"github.com/shopspring/decimal"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
)

// A rate calculated on demand rather than stored.
//
// The same shape as a rate minus the fields only a persisted row can have: it carries no ID and no timestamps because nothing was written. Used where a figure is derived per request, such as an analysis comparing one customer's price against the median other customers pay.
type ComputedRate struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=computed_rate"`
	// Decimal value of the rate, as a string to preserve precision.
	//
	// Expressed as the amount of the numerator unit per one denominator unit.
	Value string `json:"value" validate:"required" format:"decimal"`
	// Unit of the rate's numerator (e.g. the currency of a price).
	NumeratorUnit *Unit `json:"numerator_unit" expandable:"true"`
	// Unit of the rate's denominator (the per-unit basis, e.g. pairs for a price per pair).
	DenominatorUnit *Unit `json:"denominator_unit" expandable:"true"`
	// Human-readable formatted value (e.g. "$25.50 / pr").
	DisplayValue string `json:"display_value" validate:"required"`
}

// FormatRateDisplay renders a rate the way a person reads it, e.g. "$25.50 / pr". Either abbreviation may be empty, in which case that half is simply left off.
func FormatRateDisplay(value, numeratorAbbr, denominatorAbbr string) string {
	amount, err := decimal.NewFromString(value)
	if err != nil {
		amount = decimal.Zero
	}
	display := amount.StringFixed(2)
	if numeratorAbbr != "" {
		display = numeratorAbbr + display
	}
	if denominatorAbbr != "" {
		display += " / " + denominatorAbbr
	}
	return display
}

// NewComputedRate builds a computed rate with its units already attached, for the endpoints that resolve units eagerly rather than behind an include.
//
// The value is carried through exactly as the caller computed it. A quoted price has to equal the price the order will actually charge, and rounding it here to a fixed number of places would make the two disagree by a cent; callers that want a normalized scale apply it themselves.
func NewComputedRate(value string, numeratorUnit, denominatorUnit *Unit) *ComputedRate {
	var numeratorAbbr, denominatorAbbr string
	if numeratorUnit != nil {
		numeratorAbbr = numeratorUnit.Abbreviation
	}
	if denominatorUnit != nil {
		denominatorAbbr = denominatorUnit.Abbreviation
	}
	return &ComputedRate{
		Object:          constants.ObjectTypeComputedRate,
		Value:           value,
		NumeratorUnit:   numeratorUnit,
		DenominatorUnit: denominatorUnit,
		DisplayValue:    FormatRateDisplay(value, numeratorAbbr, denominatorAbbr),
	}
}

var SampleComputedRate = &ComputedRate{
	Object:       constants.ObjectTypeComputedRate,
	Value:        SampleRateValue,
	DisplayValue: "$25.50 / " + SampleUnitAbbreviation,
}

func (*ComputedRate) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleComputedRate)
}
