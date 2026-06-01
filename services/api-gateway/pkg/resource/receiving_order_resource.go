package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleReceivingOrderID = "rcor_016911ec6c634a298b3dc1798e"
const SampleReceivingOrderLineID = "rcorln_01f2aca124f3f5add7c94d5e4f"

// Receiving order summary for list views.
type ReceivingOrderSummary struct {
	// Receiving order ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=receiving_order"`
	// Receiving order number.
	Number string `json:"number" validate:"required"`
	// Purchase order associated with this receiving order. Expandable via include[]=purchase_order.
	PurchaseOrder *SalesOrderDetail `json:"purchase_order" expandable:"true"`
	// Supplier associated with this receiving order.
	Supplier *Account `json:"supplier"`
	// Number of lines in this receiving order.
	LineCount int32 `json:"line_count"`
	// Completion percentage of this receiving order.
	CompletionPercentage float64 `json:"completion_percentage"`
	// Timestamp when the receiving order was completed.
	CompletedAt *time.Time `json:"completed_at"`
	// Timestamp when the receiving order was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when the receiving order was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleReceivingOrderSummary = &ReceivingOrderSummary{
	ID:     SampleReceivingOrderID,
	Object: constants.ObjectTypeReceivingOrder,
	Number: "RO-001",
	PurchaseOrder: &SalesOrderDetail{
		ID:     SampleSalesOrderDetailID,
		Object: constants.ObjectTypeSalesOrder,
		Number: SampleSalesOrderNumber,
	},
	LineCount:            2,
	CompletionPercentage: 50.0,
	CreatedAt:            timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:            timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ReceivingOrderSummary) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleReceivingOrderSummary)
}

// Receiving order with lines.
type ReceivingOrder struct {
	// Receiving order ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=receiving_order"`
	// Receiving order number.
	Number string `json:"number" validate:"required"`
	// Purchase order associated with this receiving order. Expandable via include[]=purchase_order.
	PurchaseOrder *SalesOrderDetail `json:"purchase_order" expandable:"true"`
	// Supplier associated with this receiving order.
	Supplier *Account `json:"supplier"`
	// Note on the receiving order.
	Note *string `json:"note"`
	// Line items in this receiving order.
	Lines *List[ReceivingOrderLine] `json:"lines"`
	// Timestamp when the receiving order was completed.
	CompletedAt *time.Time `json:"completed_at"`
	// Timestamp when the receiving order was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when the receiving order was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleReceivingOrder = &ReceivingOrder{
	ID:     SampleReceivingOrderID,
	Object: constants.ObjectTypeReceivingOrder,
	Number: "RO-001",
	PurchaseOrder: &SalesOrderDetail{
		ID:     SampleSalesOrderDetailID,
		Object: constants.ObjectTypeSalesOrder,
		Number: SampleSalesOrderNumber,
	},
	Lines:     NewList([]ReceivingOrderLine{*SampleReceivingOrderLine}, PageInfo{}),
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ReceivingOrder) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleReceivingOrder)
}

// Line item in a receiving order.
type ReceivingOrderLine struct {
	// Receiving order line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=receiving_order_line"`
	// Quantity received.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// Rejected quantity.
	RejectedQuantity *Quantity `json:"rejected_quantity"`
	// Order line associated with this receiving order line.
	OrderLine *SalesOrderLineDetail `json:"order_line" validate:"required"`
	// Timestamp when the line was stocked.
	StockedAt *time.Time `json:"stocked_at"`
	// Timestamp when the line was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when the line was last updated.
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
