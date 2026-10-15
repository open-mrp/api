package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleUnitID = "un_82bd37dae5po"
const SampleUnitName = "Kilogram"
const SampleUnitAbbreviation = "kg"

// A unit of measure that quantities and rates are stated in, such as `kg` or `ea`.
type Unit struct {
	// Unit ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=unit"`
	// Display name, such as `Kilogram`.
	//
	// Unique (case-insensitive) among your account's units and the system units.
	Name string `json:"name" validate:"required"`
	// Short label, such as `kg`.
	//
	// Unique (case-insensitive) among your account's units and the system units.
	Abbreviation string `json:"abbreviation" validate:"required"`
	// What the unit measures.
	//
	// Values convert only between units of the same dimension. `quantity` is for countable things, such as each, pair or case.
	Dimension constants.UnitDimension `json:"dimension" validate:"required"`
	// The ratio that converts a value in this unit to its dimension's base unit.
	//
	// A value converts as `value × ratio_numerator ÷ ratio_denominator + offset_numerator ÷ offset_denominator`. A kilogram, with grams as the base, is `1000` / `1`. The denominator can't be zero.
	RatioNumerator string `json:"ratio_numerator" validate:"required" format:"decimal"`
	// The ratio that converts a value in this unit to its dimension's base unit.
	//
	// A value converts as `value × ratio_numerator ÷ ratio_denominator + offset_numerator ÷ offset_denominator`. A kilogram, with grams as the base, is `1000` / `1`. The denominator can't be zero.
	RatioDenominator string `json:"ratio_denominator" validate:"required" format:"decimal"`
	// An offset added after the ratio, for scales with different zero points, such as temperature.
	//
	// `0` / `1` for units with no offset. The denominator can't be zero.
	OffsetNumerator string `json:"offset_numerator" validate:"required" format:"decimal"`
	// An offset added after the ratio, for scales with different zero points, such as temperature.
	//
	// `0` / `1` for units with no offset. The denominator can't be zero.
	OffsetDenominator string `json:"offset_denominator" validate:"required" format:"decimal"`
	// Who owns this unit.
	//
	// `system` units are shared by every account and are read-only; `account` units belong to your account.
	Owner *Owner `json:"owner" expandable:"true"`
	// Time the unit was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Time the unit was last changed.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleUnit = &Unit{
	ID:                SampleUnitID,
	Object:            constants.ObjectTypeUnit,
	Name:              SampleUnitName,
	Abbreviation:      SampleUnitAbbreviation,
	Dimension:         constants.UnitDimensionMass,
	RatioNumerator:    "1000",
	RatioDenominator:  "1",
	OffsetNumerator:   "0",
	OffsetDenominator: "1",
	Owner:             SampleOwnerSystem,
	CreatedAt:         timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:         timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

// SampleCurrencyUnit is the dollar unit the retiring transaction, settlement and cost resources still state money in.
var SampleCurrencyUnit = newSampleUnit("US Dollar", "$", constants.UnitDimension(constants.UnitTypeCurrency))

// SampleEachUnit is a fully presented discrete-count unit for embedding in rate examples as a per-each denominator.
var SampleEachUnit = newSampleUnit("Each", "ea", constants.UnitDimensionQuantity)

func (*Unit) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUnit)
}

// newSampleUnit builds a complete sample Unit for embedding in other resources' examples. The conversion fields default to an identity base unit and the audit fields to the shared sample timestamps, so every embedded unit is a fully populated, schema-valid example (the exact ratio is immaterial for a nested reference). Use this instead of a partial &Unit{...} literal, which would leave required fields empty in the generated example.
func newSampleUnit(name, abbreviation string, dimension constants.UnitDimension) *Unit {
	return &Unit{
		ID:                SampleUnitID,
		Object:            constants.ObjectTypeUnit,
		Name:              name,
		Abbreviation:      abbreviation,
		Dimension:         dimension,
		RatioNumerator:    "1",
		RatioDenominator:  "1",
		OffsetNumerator:   "0",
		OffsetDenominator: "1",
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
