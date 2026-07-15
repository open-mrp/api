package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleRateID = "ra_015aa0a9522cf222024fd21d1a"
const SampleRateValue = "25.50"

// Value expressed as a ratio of two units, such as a price per kilogram or a throughput per hour.
type Rate struct {
	// Rate ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=rate"`
	// Decimal value of the rate, as a string to preserve precision.
	//
	// Expressed as the amount of the numerator unit per one denominator unit.
	Value string `json:"value" validate:"required" format:"decimal"`
	// Unit of the rate's numerator (e.g. the currency of a price).
	NumeratorUnit *Unit `json:"numerator_unit" expandable:"true"`
	// Unit of the rate's denominator (the per-unit basis, e.g. kilograms for a price per kilogram).
	DenominatorUnit *Unit `json:"denominator_unit" expandable:"true"`
	// Human-readable formatted value (e.g. "$25.50 / kg" or "100 kg / hr").
	DisplayValue string `json:"display_value" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleRate = &Rate{
	ID:              SampleRateID,
	Object:          constants.ObjectTypeRate,
	Value:           SampleRateValue,
	NumeratorUnit:   newSampleUnit("US Dollar", "USD", constants.UnitTypeCurrency),
	DenominatorUnit: newSampleUnit(SampleUnitName, SampleUnitAbbreviation, constants.UnitTypeMass),
	DisplayValue:    "$25.50 / " + SampleUnitAbbreviation,
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

// FormatRateDisplayValue formats a rate as "numerator / denominator" (e.g. "$25.50 / kg").
func FormatRateDisplayValue(value, numeratorAbbreviation, numeratorUnitType, denominatorAbbreviation string) string {
	numerator := FormatDisplayValue(value, numeratorAbbreviation, numeratorUnitType)
	if denominatorAbbreviation == "" {
		return numerator
	}
	return numerator + " / " + denominatorAbbreviation
}

func (*Rate) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRate)
}
