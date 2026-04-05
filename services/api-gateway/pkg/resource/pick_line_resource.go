package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePickLineDetailID = "pkln_01jm4r6700f8nwq3v5hx2d9ktp"

// PickSalesOrderLine is a minimal sales order line sub-resource for pick lines.
type PickSalesOrderLine struct {
	// The unique identifier for the sales order line.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_line"`
	// The line item number.
	LineItemNumber int32 `json:"line_item_number"`
	// The product SKU.
	ProductSKU string `json:"product_sku" validate:"required"`
	// The product description.
	ProductDescription *string `json:"product_description"`
}

// PickLineDetail represents a pick line resource.
type PickLineDetail struct {
	// The unique identifier for the pick line.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick_line"`
	// The quantity picked for this line.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// The ordered quantity for this line.
	OrderedQuantity *Quantity `json:"ordered_quantity" validate:"required"`
	// The sales order line info.
	SalesOrderLine *PickSalesOrderLine `json:"sales_order_line"`
	// The timestamp when the line was packed.
	PackedAt *time.Time `json:"packed_at"`
	// The timestamp when the line was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the line was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var samplePickLineProductDescription = "6061-T6 Aluminum Sheet 4x8"

var SamplePickSalesOrderLine = &PickSalesOrderLine{
	ID:                 SampleSalesOrderLineDetailID,
	Object:             constants.ObjectTypeSalesOrderLine,
	LineItemNumber:     1,
	ProductSKU:         SampleItemSKU,
	ProductDescription: &samplePickLineProductDescription,
}

var SamplePickLineDetail = &PickLineDetail{
	ID:              SamplePickLineDetailID,
	Object:          constants.ObjectTypePickLine,
	Quantity:        SampleQuantity,
	OrderedQuantity: SampleQuantity,
	SalesOrderLine:  SamplePickSalesOrderLine,
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PickLineDetail) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePickLineDetail)
}
