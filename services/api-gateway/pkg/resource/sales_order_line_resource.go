package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSalesOrderLineID = "orln_0142f9b74268973450b3a76ce3"

// Full sales order line resource.
type SalesOrderLine struct {
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
	// Associated product.
	Product *Product `json:"product" expandable:"true"`
	// Quantity ordered.
	QuantityOrdered *Quantity `json:"quantity_ordered" expandable:"true"`
	// Unit price.
	UnitPrice *Rate `json:"unit_price" expandable:"true"`
	// Unit cost.
	UnitCost *Rate `json:"unit_cost" expandable:"true"`
	// Derived monetary totals for this line.
	Totals *SalesOrderTotals `json:"totals" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleProductDescription = "6061-T6 Aluminum Sheet 4x8"

var SampleSalesOrderLine = &SalesOrderLine{
	ID:                 SampleSalesOrderLineID,
	Object:             constants.ObjectTypeSalesOrderLine,
	LineItemNumber:     1,
	ProductSKU:         SampleItemSKU,
	ProductDescription: &sampleProductDescription,
	QuantityOrdered:    SampleQuantity,
	UnitPrice:          SampleRate,
	Totals:             SampleSalesOrderTotals,
	CreatedAt:          timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:          timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SalesOrderLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSalesOrderLine)
}
