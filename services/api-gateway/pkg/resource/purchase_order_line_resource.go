package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePurchaseOrderLineID = "poln_01466ec5a2737c7b871e2a756f"

// A single line item on a purchase order.
type PurchaseOrderLine struct {
	// Purchase order line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=purchase_order_line"`
	// Position of this line within the order, starting at 1.
	LineItemNumber int32 `json:"line_item_number" validate:"required"`
	// SKU of the ordered product, copied onto the line at order time.
	ProductSKU string `json:"product_sku" validate:"required"`
	// Free-text description of the ordered product.
	ProductDescription *string `json:"product_description"`
	// Catalog item this line references, if it is linked to one.
	Item *Item `json:"item"`
	// Quantity ordered from the supplier.
	QuantityOrdered *Quantity `json:"quantity_ordered" validate:"required"`
	// Quantity received against this line so far.
	QuantityReceived *Quantity `json:"quantity_received"`
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
	QuantityOrdered:    SampleQuantity,
	UnitPrice:          SampleRate,
	CreatedAt:          timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:          timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PurchaseOrderLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePurchaseOrderLine)
}
