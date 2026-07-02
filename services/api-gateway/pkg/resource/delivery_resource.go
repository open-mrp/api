package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleDeliveryID = "dlv_0143cbea89e0f17c3d19828a3a"
const SampleDeliveryLineID = "dlvl_011663287f82d3a595acc18bcd"
const SampleLotID = "lot_01efb5e19625fdc035bb0670df"

// An inventory lot — a batch of an item received together and tracked under a single lot number.
type Lot struct {
	// Lot ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=lot"`
	// Lot number identifying the batch.
	LotNumber string `json:"lot_number" validate:"required"`
}

var SampleLot = &Lot{
	ID:        SampleLotID,
	Object:    constants.ObjectTypeLot,
	LotNumber: "LOT-001",
}

// A delivery of goods received against a purchase order.
//
// Each delivery records the items received and whether the delivery was accepted or rejected.
type Delivery struct {
	// Delivery ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=delivery"`
	// Human-readable delivery number.
	Number string `json:"number" validate:"required"`
	// The purchase order this delivery was received against.
	PurchaseOrder *PurchaseOrder `json:"purchase_order" expandable:"true"`
	// Whether the delivery was accepted or rejected on receipt.
	//
	// - `accepted`: the delivery was received and accepted (`accepted_at` is set).
	// - `rejected`: the delivery was refused (`rejected_at` is set).
	Status constants.DeliveryStatus `json:"status" validate:"required"`
	// Delivery line items.
	Lines *List[DeliveryLine] `json:"lines" expandable:"true"`
	// When the delivery was accepted, or null if it was rejected.
	AcceptedAt *time.Time `json:"accepted_at"`
	// When the delivery was rejected, or null if it was accepted.
	RejectedAt *time.Time `json:"rejected_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleDelivery = &Delivery{
	ID:            SampleDeliveryID,
	Object:        constants.ObjectTypeDelivery,
	Number:        "DLV-001",
	PurchaseOrder: SamplePurchaseOrder,
	Status:        constants.DeliveryStatusAccepted,
	Lines:         NewList([]DeliveryLine{*SampleDeliveryLine}, PageInfo{}),
	AcceptedAt:    timeutil.TimestampToTimePtr(sampleUpdatedAtTimestamp),
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Delivery) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleDelivery)
}

// Delivery line item.
type DeliveryLine struct {
	// Delivery line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=delivery_line"`
	// The item received on this line.
	//
	// Null if the item has been deleted.
	Item *Item `json:"item"`
	// Quantity received.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// Cost per unit of the received item.
	UnitCost *Rate `json:"unit_cost" validate:"required"`
	// Location the line was received into.
	Location *Location `json:"location"`
	// Lot the received inventory was assigned to, or null if the line is not lot-tracked.
	Lot *Lot `json:"lot"`
	// When this line was accepted.
	AcceptedAt *time.Time `json:"accepted_at"`
	// When this line was rejected.
	RejectedAt *time.Time `json:"rejected_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleDeliveryLine = &DeliveryLine{
	ID:     SampleDeliveryLineID,
	Object: constants.ObjectTypeDeliveryLine,
	Item: &Item{
		ID:     SampleItemID,
		Object: constants.ObjectTypeItem,
		SKU:    SampleItemSKU,
	},
	Quantity:   SampleQuantity,
	UnitCost:   SampleRate,
	Location:   SampleLocation,
	Lot:        SampleLot,
	AcceptedAt: timeutil.TimestampToTimePtr(sampleUpdatedAtTimestamp),
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*DeliveryLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleDeliveryLine)
}
