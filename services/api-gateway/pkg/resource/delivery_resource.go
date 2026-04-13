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

// Sales order sub-resource.
type SalesOrder struct {
	// Sales order ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order"`
	// Sales order number.
	Number string `json:"number" validate:"required"`
}

var SampleSalesOrder = &SalesOrder{
	ID:     SampleSalesOrderID,
	Object: constants.ObjectTypeSalesOrder,
	Number: "PO-001",
}

// Lot sub-resource.
type Lot struct {
	// Lot ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=lot"`
	// Lot number.
	LotNumber string `json:"lot_number" validate:"required"`
}

var SampleLot = &Lot{
	ID:        SampleLotID,
	Object:    constants.ObjectTypeLot,
	LotNumber: "LOT-001",
}

// Delivery summary with line count.
type DeliverySummary struct {
	// Delivery ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=delivery"`
	// Delivery number.
	Number string `json:"number" validate:"required"`
	// Associated purchase order.
	PurchaseOrder *SalesOrder `json:"purchase_order" validate:"required"`
	// Delivery status (accepted or rejected).
	Status constants.DeliveryStatus `json:"status" validate:"required"`
	// Number of delivery lines.
	LineCount int32 `json:"line_count"`
	// Accepted timestamp.
	AcceptedAt *time.Time `json:"accepted_at"`
	// Rejected timestamp.
	RejectedAt *time.Time `json:"rejected_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
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

// Delivery with line items.
type Delivery struct {
	// Delivery ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=delivery"`
	// Delivery number.
	Number string `json:"number" validate:"required"`
	// Associated purchase order.
	PurchaseOrder *SalesOrder `json:"purchase_order" validate:"required"`
	// Delivery status (accepted or rejected).
	Status constants.DeliveryStatus `json:"status" validate:"required"`
	// Delivery line items.
	Lines *List[DeliveryLine] `json:"lines"`
	// Accepted timestamp.
	AcceptedAt *time.Time `json:"accepted_at"`
	// Rejected timestamp.
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
	PurchaseOrder: SampleSalesOrder,
	Status:        constants.DeliveryStatusAccepted,
	Lines:         NewList([]DeliveryLine{*SampleDeliveryLine}, PageInfo{}),
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
	// Associated item. Null if the item has been deleted.
	Item *Item `json:"item"`
	// Quantity received.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// Unit cost.
	UnitCost *Rate `json:"unit_cost" validate:"required"`
	// Receiving location.
	Location *Location `json:"location"`
	// Associated lot.
	Lot *Lot `json:"lot"`
	// Accepted timestamp.
	AcceptedAt *time.Time `json:"accepted_at"`
	// Rejected timestamp.
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
	Quantity:  SampleQuantity,
	UnitCost:  SampleRate,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*DeliveryLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleDeliveryLine)
}
