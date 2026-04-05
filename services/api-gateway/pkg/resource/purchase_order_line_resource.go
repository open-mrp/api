package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePurchaseOrderLineDetailID = "poln_01jm4r6700f8nwq3v5hx2d9ktp"

// PurchaseOrderLineDetail represents a full purchase order line resource.
type PurchaseOrderLineDetail struct {
	// The unique identifier for the purchase order line.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=purchase_order_line"`
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
	// The quantity received.
	QuantityReceived *Quantity `json:"quantity_received"`
	// The unit price for this line.
	UnitPrice *Rate `json:"unit_price" validate:"required"`
	// The unit cost for this line.
	UnitCost *Rate `json:"unit_cost"`
	// The timestamp when the line was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the line was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var samplePurchaseOrderLineProductDescription = "6061-T6 Aluminum Sheet 4x8"

var SamplePurchaseOrderLineDetail = &PurchaseOrderLineDetail{
	ID:                 SamplePurchaseOrderLineDetailID,
	Object:             constants.ObjectTypePurchaseOrderLine,
	LineItemNumber:     1,
	ProductSKU:         SampleItemSKU,
	ProductDescription: &samplePurchaseOrderLineProductDescription,
	QuantityOrdered:    SampleQuantity,
	UnitPrice:          SampleRate,
	CreatedAt:          timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:          timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PurchaseOrderLineDetail) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePurchaseOrderLineDetail)
}
