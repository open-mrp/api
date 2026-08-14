package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
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

var SampleComputedRate = &ComputedRate{
	Object:       constants.ObjectTypeComputedRate,
	Value:        SampleRateValue,
	DisplayValue: "$25.50 / " + SampleUnitAbbreviation,
}

func (*ComputedRate) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleComputedRate)
}
