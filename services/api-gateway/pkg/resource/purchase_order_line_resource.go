package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePurchaseOrderLineID = "poln_01466ec5a2737c7b871e2a756f"

// Full purchase order line resource.
type PurchaseOrderLine struct {
	// Purchase order line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=purchase_order_line"`
	// Line item number.
	LineItemNumber int32 `json:"line_item_number" validate:"required"`
	// Product SKU.
	ProductSKU string `json:"product_sku" validate:"required"`
	// Product description.
	ProductDescription *string `json:"product_description"`
	// Item.
	Item *Item `json:"item"`
	// Quantity ordered.
	QuantityOrdered *Quantity `json:"quantity_ordered" validate:"required"`
	// Quantity received.
	QuantityReceived *Quantity `json:"quantity_received"`
	// Unit price.
	UnitPrice *Rate `json:"unit_price" validate:"required"`
	// Unit cost.
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
