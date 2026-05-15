package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleConsumptionID = "cp_01jm4r6700f8nwq3v5hx2d9ktp"

// Material consumed by a production step.
type Consumption struct {
	// Consumption ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=consumption"`
	// Quantity consumed.
	Quantity *Quantity `json:"quantity" expandable:"true"`
	// Waste quantity.
	WasteQuantity *Quantity `json:"waste_quantity" expandable:"true"`
	// Consumed item. Expandable via include[]=consumed_item.
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
