package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleUnitID = "un_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleUnitName = "Kilogram"
const SampleUnitAbbreviation = "kg"

// Unit represents a unit of measurement used for conversions and product quantities.
type Unit struct {
	// The unique identifier for the unit.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=unit"`
	// The display name of the unit (e.g. "Gram", "Kilogram").
	Name string `json:"name" validate:"required"`
	// The short abbreviation for the unit (e.g. "g", "kg").
	Abbreviation string `json:"abbreviation" validate:"required"`
	// The unit dimension.
	Type constants.UnitType `json:"type" validate:"required"`
	// The conversion ratio numerator relative to the base unit in the same dimension.
	RatioNumerator string `json:"ratio_numerator" validate:"required" format:"decimal"`
	// The conversion ratio denominator relative to the base unit in the same dimension.
	RatioDenominator string `json:"ratio_denominator" validate:"required" format:"decimal"`
	// The conversion offset numerator, used for temperature-like conversions. Zero for most unit types.
	OffsetNumerator string `json:"offset_numerator" validate:"required" format:"decimal"`
	// The conversion offset denominator. Typically 1.
	OffsetDenominator string `json:"offset_denominator" validate:"required" format:"decimal"`
	// Whether this unit is the base unit for its dimension. Conversion ratios are relative to this unit.
	IsBaseUnit bool `json:"is_base_unit"`
	// The owner of this resource.
	Owner *Owner `json:"owner" expandable:"true"`
	// When this unit was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this unit was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleUnit = &Unit{
	ID:                SampleUnitID,
	Object:            constants.ObjectTypeUnit,
	Name:              SampleUnitName,
	Abbreviation:      SampleUnitAbbreviation,
	Type:              constants.UnitTypeMass,
	RatioNumerator:    "1000",
	RatioDenominator:  "1",
	OffsetNumerator:   "0",
	OffsetDenominator: "1",
	IsBaseUnit:        false,
	Owner:             SampleOwnerSystem,
	CreatedAt:         timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:         timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Unit) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUnit)
}

// ValidateUnitsResponse represents the response for the validate units endpoint.
type ValidateUnitsResponse struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=map"`
	// The validated units keyed by the original map key.
	Units map[string]*Unit `json:"units" validate:"required"`
}

var SampleValidateUnitsResponse = &ValidateUnitsResponse{
	Object: constants.ObjectTypeMap,
	Units:  map[string]*Unit{"0": SampleUnit},
}

func (*ValidateUnitsResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleValidateUnitsResponse)
}
