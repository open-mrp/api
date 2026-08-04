package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSalesOrderLineID = "orln_la01fxgrwcnr"
const SampleSalesOrderLineID2 = "orln_vwp43e1rq2zb"

// A single line item on a sales order.
type SalesOrderLine struct {
	// Sales order line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_line"`
	// Position of the line on the order.
	//
	// Assigned automatically in sequence, starting at `1`. Product lines are numbered first and the automatically generated freight and discount lines always sit at the bottom; removing a line renumbers the rest so the sequence stays contiguous.
	LineItemNumber int32 `json:"line_item_number" validate:"required"`
	// SKU recorded on this line.
	//
	// Taken from the product unless the line supplies its own, and editable afterwards, so it preserves what was sold even if the product's SKU later changes.
	ProductSKU string `json:"product_sku" validate:"required"`
	// Description recorded on this line, taken from the product unless the line supplies its own.
	ProductDescription *string `json:"product_description"`
	// The product being sold on this line.
	//
	// The product's `type` tells you what kind of line this is: the freight and discount lines an order generates for itself reference the account's built-in shipping and credit products, so their type is `shipping` or `credit` rather than `sale`.
	Product *Product `json:"product" expandable:"true"`
	// Quantity ordered.
	QuantityOrdered *Quantity `json:"quantity_ordered" expandable:"true"`
	// Price charged per unit.
	UnitPrice *Rate `json:"unit_price" expandable:"true"`
	// Internal cost per unit.
	//
	// Reflects what the business pays for the item, not what the customer is charged, and is used to derive line profitability.
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
	Product:            SampleProduct,
	QuantityOrdered:    SampleQuantity,
	UnitPrice:          SampleRate,
	UnitCost:           SampleRate,
	Totals:             SampleSalesOrderTotals,
	CreatedAt:          timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:          timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SalesOrderLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSalesOrderLine)
}
