package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSalesOrderLineDetailID = "orln_01jm4r6700f8nwq3v5hx2d9ktp"

// Full sales order line resource.
type SalesOrderLineDetail struct {
	// Sales order line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_line"`
	// Line item number.
	LineItemNumber int32 `json:"line_item_number" validate:"required"`
	// Product SKU.
	ProductSKU string `json:"product_sku" validate:"required"`
	// Product description.
	ProductDescription *string `json:"product_description"`
	// Associated item. Expandable.
	Item *Item `json:"item" expandable:"true"`
	// Quantity ordered.
	QuantityOrdered *Quantity `json:"quantity_ordered" validate:"required"`
	// Quantity picked.
	QuantityPicked *Quantity `json:"quantity_picked"`
	// Quantity packed.
	QuantityPacked *Quantity `json:"quantity_packed"`
	// Quantity invoiced.
	QuantityInvoiced *Quantity `json:"quantity_invoiced"`
	// Unit price.
	UnitPrice *Rate `json:"unit_price" validate:"required"`
	// Unit cost.
	UnitCost *Rate `json:"unit_cost"`
	// EDI line item ID.
	EdiLineItemID *string `json:"edi_line_item_id"`
	// Completed timestamp.
	CompletedAt *time.Time `json:"completed_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
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
