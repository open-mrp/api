package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleUnitID = "unit_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleUnitName = "Kilogram"
const SampleUnitAbbreviation = "kg"

var SampleUnit = &Unit{
	ID:                SampleUnitID,
	Object:            constants.ObjectTypeUnit,
	Name:              SampleUnitName,
	Abbreviation:      SampleUnitAbbreviation,
	Type:              constants.UnitTypeMass,
	RatioNumerator:    "1000.000000000000000000000000000000",
	RatioDenominator:  "1.000000000000000000000000000000",
	OffsetNumerator:   "0.000000000000000000000000000000",
	OffsetDenominator: "1.000000000000000000000000000000",
	IsBaseUnit:        false,
	IsInternal:        false,
	CreatedAt:         timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:         timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

// Unit represents a unit of measurement used for conversions and product quantities.
type Unit struct {
	// The unique identifier for the unit.
	ID string `json:"id" validate:"required"`
	// The object type.
	Object constants.ObjectType `json:"object" validate:"required,enum=unit"`
	// The display name of the unit (e.g. "Gram", "Kilogram").
	Name string `json:"name" validate:"required"`
	// The short abbreviation for the unit (e.g. "g", "kg").
	Abbreviation string `json:"abbreviation" validate:"required"`
	// The unit dimension (e.g. "quantity", "mass", "time", "currency").
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
	// Whether this unit belongs to the requesting account. False for system/global units.
	IsInternal bool `json:"is_internal"`
	// When this unit was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this unit was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

func (*Unit) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUnit)
}
