package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleConsumptionID = "cp_0152c5d4330f178ebe1158f910"

// Material consumed by a production step.
//
// Each consumption records one input item and how much of it the step uses. Consumptions also determine the production flow: when another step produces the consumed item, the two steps are linked upstream/downstream automatically.
type Consumption struct {
	// Consumption ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=consumption"`
	// Quantity of the material consumed by the production step.
	Quantity *Quantity `json:"quantity" expandable:"true"`
	// Quantity of the material accounted for as waste, separate from the consumed quantity.
	WasteQuantity *Quantity `json:"waste_quantity" expandable:"true"`
	// The item consumed by the production step.
	ConsumedItem *Item `json:"consumed_item" expandable:"true"`
	// Instructions for how this material is consumed.
	Instructions *string `json:"instructions"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleConsumptionInstructions = "Mix with water before adding"

var SampleConsumption = &Consumption{
	ID:            SampleConsumptionID,
	Object:        constants.ObjectTypeConsumption,
	Quantity:      SampleQuantity,
	WasteQuantity: SampleQuantity,
	ConsumedItem:  SampleItem,
	Instructions:  &sampleConsumptionInstructions,
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Consumption) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleConsumption)
}
