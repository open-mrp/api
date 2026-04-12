package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleConsumptionID = "cp_01jm4r6700f8nwq3v5hx2d9ktp"

// Consumption represents a material consumed by a production step.
type Consumption struct {
	// The unique identifier for the consumption.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=consumption"`
	// The quantity consumed.
	Quantity *Quantity `json:"quantity" expandable:"true"`
	// The waste quantity.
	WasteQuantity *Quantity `json:"waste_quantity" expandable:"true"`
	// The consumed item.
	ConsumedItem *ConsumptionItem `json:"consumed_item" expandable:"true"`
	// Optional instructions for how this material is consumed.
	Instructions *string `json:"instructions"`
	// The timestamp when the consumption was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the consumption was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// ConsumptionItem is a lightweight item representation for consumption sub-resources.
type ConsumptionItem struct {
	// The unique identifier for the item.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// The stock keeping unit code.
	SKU string `json:"sku" validate:"required"`
	// A description of the item.
	Description *string `json:"description"`
	// The item type code.
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
