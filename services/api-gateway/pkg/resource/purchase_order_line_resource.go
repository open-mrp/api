package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SamplePurchaseOrderLineID = "poln_lechc7ak8sp9"

// A single line item on a purchase order.
type PurchaseOrderLine struct {
	// Purchase order line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=purchase_order_line"`
	// Sequence number of this line within the order, starting at 1.
	//
	// Assigned automatically as one past the highest number currently on the order, so deleting a line can leave a gap in the numbering.
	LineItemNumber int32 `json:"line_item_number" validate:"required"`
	// SKU of the ordered product, copied onto the line at order time.
	ProductSKU string `json:"product_sku" validate:"required"`
	// Free-text description of the ordered product.
	ProductDescription *string `json:"product_description"`
	// Catalog item this line references, if it is linked to one.
	Item *Item `json:"item" expandable:"true"`
	// Quantity ordered from the supplier.
	QuantityOrdered *Quantity `json:"quantity_ordered" validate:"required"`
	// Quantity booked against this line on the order's receiving order.
	//
	// Rolled up from the receiving order lines linked to this line, so it stays at zero until the order is issued and receiving lines are created for it. Summed at read time rather than stored: it carries no id, and arrives with the unit it was summed in.
	QuantityReceived *ComputedQuantity `json:"quantity_received"`
	// Agreed purchase price per unit for this line.
	UnitPrice *Rate `json:"unit_price" validate:"required"`
	// Recorded cost per unit, if captured separately from the purchase price.
	UnitCost *Rate `json:"unit_cost"`
	// Created timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var samplePurchaseOrderLineProductDescription = "6061-T6 Aluminum Sheet 4x8"

var SamplePurchaseOrderLine = &PurchaseOrderLine{
	ID:                 SamplePurchaseOrderLineID,
	Object:             constants.ObjectTypePurchaseOrderLine,
	LineItemNumber:     1,
	ProductSKU:         SampleItemSKU,
	ProductDescription: &samplePurchaseOrderLineProductDescription,
	Item:               SampleItem,
	QuantityOrdered:    SampleQuantity,
	QuantityReceived:   SampleComputedQuantity,
	UnitPrice:          SampleRate,
	UnitCost:           SampleRate,
	CreatedAt:          timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:          timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PurchaseOrderLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePurchaseOrderLine)
}
