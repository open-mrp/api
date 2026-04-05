package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSalesOrderLineDetailID = "orln_01jm4r6700f8nwq3v5hx2d9ktp"

// SalesOrderLineDetail represents a full sales order line resource.
type SalesOrderLineDetail struct {
	// The unique identifier for the sales order line.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_line"`
	// The line item number.
	LineItemNumber int32 `json:"line_item_number" validate:"required"`
	// The product SKU.
	ProductSKU string `json:"product_sku" validate:"required"`
	// The product description.
	ProductDescription *string `json:"product_description"`
	// The item associated with this line.
	Item *Item `json:"item"`
	// The quantity ordered.
	QuantityOrdered *Quantity `json:"quantity_ordered" validate:"required"`
	// The quantity picked.
	QuantityPicked *Quantity `json:"quantity_picked"`
	// The quantity packed.
	QuantityPacked *Quantity `json:"quantity_packed"`
	// The quantity invoiced.
	QuantityInvoiced *Quantity `json:"quantity_invoiced"`
	// The unit price for this line.
	UnitPrice *Rate `json:"unit_price" validate:"required"`
	// The unit cost for this line.
	UnitCost *Rate `json:"unit_cost"`
	// The EDI line item ID.
	EdiLineItemID *string `json:"edi_line_item_id"`
	// The timestamp when the line was completed.
	CompletedAt *time.Time `json:"completed_at"`
	// The timestamp when the line was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the line was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleProductDescription = "6061-T6 Aluminum Sheet 4x8"

var SampleSalesOrderLineDetail = &SalesOrderLineDetail{
	ID:                 SampleSalesOrderLineDetailID,
	Object:             constants.ObjectTypeSalesOrderLine,
	LineItemNumber:     1,
	ProductSKU:         SampleItemSKU,
	ProductDescription: &sampleProductDescription,
	QuantityOrdered:    SampleQuantity,
	UnitPrice:          SampleRate,
	CreatedAt:          timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:          timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SalesOrderLineDetail) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSalesOrderLineDetail)
}
