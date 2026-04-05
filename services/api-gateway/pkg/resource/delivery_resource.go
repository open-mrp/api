package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleDeliveryID = "dlv_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleDeliveryLineID = "dlvl_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleSalesOrderID = "so_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleLotID = "lot_01jm4r6700f8nwq3v5hx2d9ktp"

// SalesOrder represents a sales order sub-resource.
type SalesOrder struct {
	// The unique identifier for the sales order.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order"`
	// The sales order number.
	Number string `json:"number" validate:"required"`
}

var SampleSalesOrder = &SalesOrder{
	ID:     SampleSalesOrderID,
	Object: constants.ObjectTypeSalesOrder,
	Number: "PO-001",
}

// Lot represents a lot sub-resource.
type Lot struct {
	// The unique identifier for the lot.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=lot"`
	// The lot number.
	LotNumber string `json:"lot_number" validate:"required"`
}

var SampleLot = &Lot{
	ID:        SampleLotID,
	Object:    constants.ObjectTypeLot,
	LotNumber: "LOT-001",
}

// DeliverySummary represents a delivery with a line count.
type DeliverySummary struct {
	// The unique identifier for the delivery.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=delivery"`
	// The delivery number.
	Number string `json:"number" validate:"required"`
	// The purchase order associated with this delivery.
	PurchaseOrder *SalesOrder `json:"purchase_order" validate:"required"`
	// The delivery status (accepted or rejected).
	Status constants.DeliveryStatus `json:"status" validate:"required"`
	// The number of lines in this delivery.
	LineCount int32 `json:"line_count"`
	// The timestamp when the delivery was accepted.
	AcceptedAt *time.Time `json:"accepted_at"`
	// The timestamp when the delivery was rejected.
	RejectedAt *time.Time `json:"rejected_at"`
	// The timestamp when the delivery was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the delivery was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleDeliverySummary = &DeliverySummary{
	ID:            SampleDeliveryID,
	Object:        constants.ObjectTypeDelivery,
	Number:        "DLV-001",
	PurchaseOrder: SampleSalesOrder,
	Status:        constants.DeliveryStatusAccepted,
	LineCount:     2,
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*DeliverySummary) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleDeliverySummary)
}

// Delivery represents a full delivery with lines.
type Delivery struct {
	// The unique identifier for the delivery.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=delivery"`
	// The delivery number.
	Number string `json:"number" validate:"required"`
	// The purchase order associated with this delivery.
	PurchaseOrder *SalesOrder `json:"purchase_order" validate:"required"`
	// The delivery status (accepted or rejected).
	Status constants.DeliveryStatus `json:"status" validate:"required"`
	// The line items in this delivery.
	Lines *List[DeliveryLine] `json:"lines"`
	// The timestamp when the delivery was accepted.
	AcceptedAt *time.Time `json:"accepted_at"`
	// The timestamp when the delivery was rejected.
	RejectedAt *time.Time `json:"rejected_at"`
	// The timestamp when the delivery was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the delivery was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleDelivery = &Delivery{
	ID:            SampleDeliveryID,
	Object:        constants.ObjectTypeDelivery,
	Number:        "DLV-001",
	PurchaseOrder: SampleSalesOrder,
	Status:        constants.DeliveryStatusAccepted,
	Lines:         NewList([]DeliveryLine{*SampleDeliveryLine}, PageInfo{}),
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Delivery) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleDelivery)
}

// DeliveryLine represents a line item in a delivery.
type DeliveryLine struct {
	// The unique identifier for the delivery line.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=delivery_line"`
	// The item associated with this line. Nullable if the item has been deleted.
	Item *Item `json:"item"`
	// The quantity received.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// The unit cost for this line.
	UnitCost *Rate `json:"unit_cost" validate:"required"`
	// The location where this delivery was received.
	Location *Location `json:"location"`
	// The lot associated with this line.
	Lot *Lot `json:"lot"`
	// The timestamp when the line was accepted.
	AcceptedAt *time.Time `json:"accepted_at"`
	// The timestamp when the line was rejected.
	RejectedAt *time.Time `json:"rejected_at"`
	// The timestamp when the line was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the line was last updated.
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
	Quantity:  SampleQuantity,
	UnitCost:  SampleRate,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*DeliveryLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleDeliveryLine)
}
