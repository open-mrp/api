package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleUnitID = "un_82bd37dae5po"
const SampleUnitName = "Kilogram"
const SampleUnitAbbreviation = "kg"

// Unit of measurement used for conversions and product quantities.
type Unit struct {
	// Unit ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=unit"`
	// Display name of the unit (e.g. "Gram", "Kilogram").
	Name string `json:"name" validate:"required"`
	// Short abbreviation for the unit (e.g. "g", "kg").
	Abbreviation string `json:"abbreviation" validate:"required"`
	// The dimension this unit measures, such as mass, volume, or currency.
	//
	// A unit can only be converted to another unit of the same dimension. The `quantity` dimension is for discrete countable items rather than a physical measure.
	Type constants.UnitType `json:"type" validate:"required"`
	// Numerator of the ratio that converts a quantity in this unit into the dimension's base unit.
	//
	// A quantity is converted with `value × (ratio_numerator / ratio_denominator) + (offset_numerator / offset_denominator)`, so a kilogram in a gram-based dimension has a numerator of `1000` and a denominator of `1`.
	RatioNumerator string `json:"ratio_numerator" validate:"required" format:"decimal"`
	// Denominator of the ratio that converts a quantity in this unit into the dimension's base unit.
	//
	// Cannot be zero.
	RatioDenominator string `json:"ratio_denominator" validate:"required" format:"decimal"`
	// Numerator of the conversion offset, applied after the ratio for scales that do not share a zero point, such as temperature.
	//
	// Zero for units that convert by ratio alone.
	OffsetNumerator string `json:"offset_numerator" validate:"required" format:"decimal"`
	// Denominator of the conversion offset applied after the ratio.
	//
	// Never zero; a unit with no offset carries a numerator of `0` over a denominator of `1`.
	OffsetDenominator string `json:"offset_denominator" validate:"required" format:"decimal"`
	// Whether this is the base unit for its dimension.
	//
	// Every other unit's conversion ratio is expressed relative to the base unit. Base units are platform-defined; units created through the API are never base units.
	IsBaseUnit bool `json:"is_base_unit"`
	// Owner of this resource.
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

// SampleCurrencyUnit is a fully presented US Dollar unit for embedding in rate examples as a numerator (price) unit.
var SampleCurrencyUnit = newSampleUnit("US Dollar", "$", constants.UnitTypeCurrency)

// SampleEachUnit is a fully presented discrete-count unit for embedding in rate examples as a per-each denominator.
var SampleEachUnit = newSampleUnit("Each", "ea", constants.UnitTypeQuantity)

func (*Unit) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUnit)
}

// newSampleUnit builds a complete sample Unit for embedding in other resources' examples. The conversion fields default to an identity base unit and the audit fields to the shared sample timestamps, so every embedded unit is a fully populated, schema-valid example (the exact ratio is immaterial for a nested reference). Use this instead of a partial &Unit{...} literal, which would leave required fields empty in the generated example.
func newSampleUnit(name, abbreviation string, unitType constants.UnitType) *Unit {
	return &Unit{
		ID:                SampleUnitID,
		Object:            constants.ObjectTypeUnit,
		Name:              name,
		Abbreviation:      abbreviation,
		Type:              unitType,
		RatioNumerator:    "1",
		RatioDenominator:  "1",
		OffsetNumerator:   "0",
		OffsetDenominator: "1",
		IsBaseUnit:        true,
		Owner:             SampleOwnerSystem,
		CreatedAt:         timeutil.TimestampToTime(sampleCreatedAtTimestamp),
		UpdatedAt:         timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
	}
}

// Result of unit abbreviation validation.
type ValidateUnitsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=map"`
	// Validated units keyed by the original map key.
	//
	// Abbreviations are matched case-insensitively; keys whose abbreviation did not match any unit are omitted.
	Units map[string]*Unit `json:"units" validate:"required"`
}

var SampleValidateUnitsResponse = &ValidateUnitsResponse{
	Object: constants.ObjectTypeMap,
	Units:  map[string]*Unit{"0": SampleUnit},
}

func (*ValidateUnitsResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleValidateUnitsResponse)
}
