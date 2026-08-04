package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleConsumptionID = "cp_blst8ze24dy3"

// Material consumed by a production step.
//
// Each consumption records one input item and how much of it the step uses. Consumptions also determine the production flow: when another step produces the consumed item, the two steps are linked upstream/downstream automatically.
//
// The quantities are stated against the step's own output, so a step producing 100 pairs and consuming 5 kg of yarn needs 5 kg per 100 pairs. Material requirements for an order scale every consumption in the flow by how much of the finished item is wanted.
type Consumption struct {
	// Consumption ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=consumption"`
	// Quantity of the material consumed by the production step.
	Quantity *Quantity `json:"quantity" expandable:"true"`
	// Quantity of the material expected to be lost as waste.
	//
	// Tracked separately from the consumed quantity, but added to it when material requirements are worked out, since the waste has to be bought as well.
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
