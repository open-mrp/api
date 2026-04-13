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
	// Consumed item.
	ConsumedItem *ConsumptionItem `json:"consumed_item" expandable:"true"`
	// Instructions for how this material is consumed.
	Instructions *string `json:"instructions"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Item embedded within a consumption.
type ConsumptionItem struct {
	// Item ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// Stock keeping unit code.
	SKU string `json:"sku" validate:"required"`
	// Item description.
	Description *string `json:"description"`
	// Item type code.
	ItemTypeCode constants.ItemTypeCode `json:"item_type" validate:"required"`
}

var sampleConsumptionInstructions = "Mix with water before adding"

var SampleConsumptionItem = &ConsumptionItem{
	ID:           SampleItemID,
	Object:       constants.ObjectTypeItem,
	SKU:          SampleItemSKU,
	Description:  &sampleDescription,
	ItemTypeCode: constants.ItemTypeCodePart,
}

func (*ConsumptionItem) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleConsumptionItem)
}

var SampleConsumption = &Consumption{
	ID:            SampleConsumptionID,
	Object:        constants.ObjectTypeConsumption,
	Quantity:      SampleQuantity,
	WasteQuantity: SampleQuantity,
	ConsumedItem:  SampleConsumptionItem,
	Instructions:  &sampleConsumptionInstructions,
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Consumption) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleConsumption)
}
