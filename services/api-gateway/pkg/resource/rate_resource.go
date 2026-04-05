package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleRateID = "ra_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleRateValue = "25.500000000000000000000000000000"

// Rate represents a ratio between two quantities with different units.
type Rate struct {
	// The unique identifier for the rate.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=rate"`
	// The rate value as a decimal string.
	Value string `json:"value" validate:"required" format:"decimal"`
	// The numerator unit for this rate.
	NumeratorUnit *Unit `json:"numerator_unit" expandable:"true"`
	// The denominator unit for this rate.
	DenominatorUnit *Unit `json:"denominator_unit" expandable:"true"`
	// A human-readable formatted value including the unit (e.g. "$25.50 / kg" or "100 kg / hr").
	DisplayValue string `json:"display_value" validate:"required"`
	// When this rate was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this rate was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleRate = &Rate{
	ID:     SampleRateID,
	Object: constants.ObjectTypeRate,
	Value:  SampleRateValue,
	NumeratorUnit: &Unit{
		ID:           SampleUnitID,
		Object:       constants.ObjectTypeUnit,
		Name:         "US Dollar",
		Abbreviation: "USD",
		Type:         constants.UnitTypeCurrency,
	},
	DenominatorUnit: &Unit{
		ID:           SampleUnitID,
		Object:       constants.ObjectTypeUnit,
		Name:         SampleUnitName,
		Abbreviation: SampleUnitAbbreviation,
		Type:         constants.UnitTypeMass,
	},
	DisplayValue: "$25.50 / " + SampleUnitAbbreviation,
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:    timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
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
