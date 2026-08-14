package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// An amount calculated on demand rather than stored.
//
// The same shape as a quantity minus the ID, because nothing was written: it is derived per request, such as a total rolled up across invoiced lines for one analysis.
type ComputedQuantity struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=computed_quantity"`
	// Raw decimal value, as a string to preserve precision.
	//
	// This is the unformatted machine value; see `display_value` for the human-readable rendering.
	Value string `json:"value" validate:"required" format:"decimal"`
	// Formatted value with unit abbreviation (e.g. "1,200 pr").
	DisplayValue string `json:"display_value" validate:"required"`
	// Unit of measure for this value.
	Unit *Unit `json:"unit" expandable:"true"`
}

var SampleComputedQuantity = &ComputedQuantity{
	Object:       constants.ObjectTypeComputedQuantity,
	Value:        "1200",
	DisplayValue: "1,200 " + SampleUnitAbbreviation,
}

func (*ComputedQuantity) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleComputedQuantity)
}
