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
	// Associated purchase order. Expandable via include[]=purchase_order.
	PurchaseOrder *SalesOrderDetail `json:"purchase_order" expandable:"true"`
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
	ID:     SampleDeliveryID,
	Object: constants.ObjectTypeDelivery,
	Number: "DLV-001",
	PurchaseOrder: &SalesOrderDetail{
		ID:     SampleSalesOrderDetailID,
		Object: constants.ObjectTypeSalesOrder,
		Number: SampleSalesOrderNumber,
	},
	Status:    constants.DeliveryStatusAccepted,
	LineCount: 2,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
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
	// Associated purchase order. Expandable via include[]=purchase_order.
	PurchaseOrder *SalesOrderDetail `json:"purchase_order" expandable:"true"`
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
	ID:     SampleDeliveryID,
	Object: constants.ObjectTypeDelivery,
	Number: "DLV-001",
	PurchaseOrder: &SalesOrderDetail{
		ID:     SampleSalesOrderDetailID,
		Object: constants.ObjectTypeSalesOrder,
		Number: SampleSalesOrderNumber,
	},
	Status:    constants.DeliveryStatusAccepted,
	Lines:     NewList([]DeliveryLine{*SampleDeliveryLine}, PageInfo{}),
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
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
