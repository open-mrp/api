package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleReceivingOrderID = "rcor_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleReceivingOrderLineID = "rcorln_01jm4r6700f8nwq3v5hx2d9ktp"

// ReceivingOrderSummary represents a receiving order in list views.
type ReceivingOrderSummary struct {
	// The unique identifier for the receiving order.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=receiving_order"`
	// The receiving order number.
	Number string `json:"number" validate:"required"`
	// The purchase order associated with this receiving order.
	PurchaseOrder *SalesOrder `json:"purchase_order" validate:"required"`
	// The supplier associated with this receiving order.
	Supplier *Account `json:"supplier"`
	// The number of lines in this receiving order.
	LineCount int32 `json:"line_count"`
	// The completion percentage of this receiving order.
	CompletionPercentage float64 `json:"completion_percentage"`
	// The timestamp when the receiving order was completed.
	CompletedAt *time.Time `json:"completed_at"`
	// The timestamp when the receiving order was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the receiving order was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleReceivingOrderSummary = &ReceivingOrderSummary{
	ID:                   SampleReceivingOrderID,
	Object:               constants.ObjectTypeReceivingOrder,
	Number:               "RO-001",
	PurchaseOrder:        SampleSalesOrder,
	LineCount:            2,
	CompletionPercentage: 50.0,
	CreatedAt:            timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:            timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ReceivingOrderSummary) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleReceivingOrderSummary)
}

// ReceivingOrder represents a full receiving order with lines.
type ReceivingOrder struct {
	// The unique identifier for the receiving order.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=receiving_order"`
	// The receiving order number.
	Number string `json:"number" validate:"required"`
	// The purchase order associated with this receiving order.
	PurchaseOrder *SalesOrder `json:"purchase_order" validate:"required"`
	// The supplier associated with this receiving order.
	Supplier *Account `json:"supplier"`
	// A note on the receiving order.
	Note *string `json:"note"`
	// The line items in this receiving order.
	Lines *List[ReceivingOrderLine] `json:"lines"`
	// The timestamp when the receiving order was completed.
	CompletedAt *time.Time `json:"completed_at"`
	// The timestamp when the receiving order was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the receiving order was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleReceivingOrder = &ReceivingOrder{
	ID:            SampleReceivingOrderID,
	Object:        constants.ObjectTypeReceivingOrder,
	Number:        "RO-001",
	PurchaseOrder: SampleSalesOrder,
	Lines:         NewList([]ReceivingOrderLine{*SampleReceivingOrderLine}, PageInfo{}),
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ReceivingOrder) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleReceivingOrder)
}

// ReceivingOrderLine represents a line item in a receiving order.
type ReceivingOrderLine struct {
	// The unique identifier for the receiving order line.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=receiving_order_line"`
	// The quantity received.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// The rejected quantity.
	RejectedQuantity *Quantity `json:"rejected_quantity"`
	// The order line associated with this receiving order line.
	OrderLine *SalesOrderLineDetail `json:"order_line" validate:"required"`
	// The timestamp when the line was stocked.
	StockedAt *time.Time `json:"stocked_at"`
	// The timestamp when the line was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the line was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleReceivingOrderLine = &ReceivingOrderLine{
	ID:        SampleReceivingOrderLineID,
	Object:    constants.ObjectTypeReceivingOrderLine,
	Quantity:  SampleQuantity,
	OrderLine: SampleSalesOrderLineDetail,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ReceivingOrderLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleReceivingOrderLine)
}
