package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleProductionID = "pn_01jm4r6700f8nwq3v5hx2d9ktp"

// Production output of a production step.
type ProductionOutput struct {
	// Production ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production"`
	// Produced item.
	ProducedItem *ConsumptionItem `json:"produced_item"`
	// Quantity produced.
	Quantity *Quantity `json:"quantity"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
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
