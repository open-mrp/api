package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleDeliveryID = "dlv_9xsjlqx5753y"
const SampleDeliveryLineID = "dlvl_9vn001g1rc2t"
const SampleLotID = "lot_t1ge2m2qt3cw"

// An inventory lot — a batch of an item received together and tracked under a single lot number.
type Lot struct {
	// Lot ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=lot"`
	// Lot number identifying the batch.
	//
	// Unique per item within the account: stocking goods under a lot number that already exists for that item records them into the existing lot rather than creating a new one.
	LotNumber string `json:"lot_number" validate:"required"`
}

var SampleLot = &Lot{
	ID:        SampleLotID,
	Object:    constants.ObjectTypeLot,
	LotNumber: "LOT-001",
}

// A delivery of goods received against a purchase order.
//
// Deliveries are not created directly. One is recorded each time a receiving order is stocked, capturing what arrived in that shipment, where it was put away, and what was refused on inspection. A purchase order received in several shipments therefore has several deliveries.
type Delivery struct {
	// Delivery ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=delivery"`
	// Human-readable delivery number.
	//
	// The first delivery against a purchase order takes that order's number; each later delivery appends a sequence suffix, such as `PO-001-2`.
	Number string `json:"number" validate:"required"`
	// The purchase order this delivery was received against.
	PurchaseOrder *PurchaseOrder `json:"purchase_order" expandable:"true"`
	// Whether any of the delivered goods were accepted into inventory.
	//
	// - `accepted`: at least part of the shipment was put into inventory. Quantities refused on inspection can still appear on the delivery's lines.
	// - `rejected`: nothing on the delivery entered inventory.
	Status constants.DeliveryStatus `json:"status" validate:"required"`
	// The goods recorded on this delivery.
	Lines *List[DeliveryLine] `json:"lines" expandable:"true"`
	// When goods on this delivery were accepted into inventory.
	//
	// A delivery that also had quantities refused has both this and `rejected_at` set.
	AcceptedAt *time.Time `json:"accepted_at"`
	// When goods on this delivery were refused on inspection.
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

// A quantity of one item recorded on a delivery.
//
// Stocking a receiving order creates one line for each storage allocation of accepted goods, plus one further line for any quantity refused on inspection. Exactly one of `accepted_at` and `rejected_at` is set on each line, so a single receiving order line can produce several delivery lines.
type DeliveryLine struct {
	// Delivery line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=delivery_line"`
	// The item received on this line.
	Item *Item `json:"item"`
	// Quantity recorded on this line.
	//
	// On a refused line this is the quantity rejected rather than the quantity taken into inventory.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// Cost per unit of the goods on this line.
	//
	// Copied from the originating purchase order line's unit price at the moment of stocking, so later price changes on the purchase order leave it untouched.
	UnitCost *Rate `json:"unit_cost" validate:"required"`
	// Storage location the goods on this line were put away at.
	//
	// Not set on refused lines, or when the quantity was stocked without naming a location.
	Location *Location `json:"location"`
	// Lot the goods on this line were assigned to.
	//
	// Set only when a lot number was supplied while stocking, and applied to every line produced from that receiving order line, including the refused one.
	Lot *Lot `json:"lot"`
	// When the goods on this line were accepted into inventory.
	AcceptedAt *time.Time `json:"accepted_at"`
	// When the goods on this line were refused on inspection.
	RejectedAt *time.Time `json:"rejected_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleDeliveryLine = &DeliveryLine{
	ID:         SampleDeliveryLineID,
	Object:     constants.ObjectTypeDeliveryLine,
	Item:       SampleItem,
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
