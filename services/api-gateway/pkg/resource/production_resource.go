package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleProductionID = "pn_01jm4r6700f8nwq3v5hx2d9ktp"

// ProductionOutput represents the production output of a production step.
type ProductionOutput struct {
	// The unique identifier for the production.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production"`
	// The produced item.
	ProducedItem *ConsumptionItem `json:"produced_item"`
	// The quantity produced.
	Quantity *Quantity `json:"quantity"`
	// The timestamp when the production was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the production was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProductionOutput = &ProductionOutput{
	ID:     SampleProductionID,
	Object: constants.ObjectTypeProduction,
	ProducedItem: &ConsumptionItem{
		ID:           SampleItemID,
		Object:       constants.ObjectTypeItem,
		SKU:          SampleItemSKU,
		ItemTypeCode: constants.ItemTypeCodeProduct,
	},
	Quantity:  SampleQuantity,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductionOutput) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionOutput)
}
