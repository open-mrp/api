package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleReceivingOrderID = "rcor_016911ec6c634a298b3dc1798e"
const SampleReceivingOrderLineID = "rcorln_01f2aca124f3f5add7c94d5e4f"

// A receiving order tracks inbound inventory against an issued purchase order.
//
// One receiving order is created automatically when a purchase order is issued, with one line per purchase order line. As goods arrive, line quantities are received and then stocked into inventory; the order is marked complete once every line is stocked.
//
// The list endpoint returns this same type as the retrieve endpoint, with the same fields available.
type ReceivingOrder struct {
	// Receiving order ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=receiving_order"`
	// Human-readable identifier for the receiving order, assigned when the originating purchase order is issued.
	//
	// It mirrors that purchase order's number (e.g. `PO-001`). Distinct from `id`; use it to reference the order in the UI and on documents.
	Number string `json:"number" validate:"required"`
	// Free-text note carried over from the originating purchase order.
	//
	// Present only on the retrieve response; it is not returned in list results.
	Note *string `json:"note"`
	// Purchase order associated with this receiving order.
	PurchaseOrder *PurchaseOrder `json:"purchase_order" expandable:"true"`
	// The supplier (seller) account the originating purchase order was placed with.
	Supplier *Supplier `json:"supplier" expandable:"true"`
	// Line items in this receiving order.
	Lines *List[ReceivingOrderLine] `json:"lines" expandable:"true"`
	// Total number of lines on this receiving order.
	//
	// Always populated, even when `lines` is not expanded.
	LineCount int32 `json:"line_count"`
	// Percentage of lines that have been stocked, from `0` to `100`, rounded to two decimal places.
	//
	// A line counts toward completion once its `stocked_at` is set. Reaches `100` when every line is stocked, at which point the order is marked complete.
	CompletionPercentage float64 `json:"completion_percentage"`
	// Timestamp when the receiving order was completed, set automatically once all of its lines have been stocked.
	CompletedAt *time.Time `json:"completed_at"`
	// Timestamp when the receiving order was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when the receiving order was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleReceivingOrder = &ReceivingOrder{
	ID:                   SampleReceivingOrderID,
	Object:               constants.ObjectTypeReceivingOrder,
	Number:               "RO-001",
	Lines:                NewList([]ReceivingOrderLine{*SampleReceivingOrderLine}, PageInfo{}),
	LineCount:            2,
	CompletionPercentage: 50.0,
	CreatedAt:            timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:            timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ReceivingOrder) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleReceivingOrder)
}

// Line item in a receiving order.
//
// One line is created per purchase order line when the purchase order is issued, with its quantity initialized to the full ordered quantity. When a line is stocked short of the ordered quantity, a new line is created automatically for the remainder.
type ReceivingOrderLine struct {
	// Receiving order line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=receiving_order_line"`
	// The quantity being received on this line.
	//
	// Initialized to the originating order line's full ordered quantity. Adjust it with Update Receiving Order Line, or use the receive actions to set it to the quantity still outstanding on the order line. Voiding the line resets it to `0`.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// Quantity rejected on inspection and not stocked into inventory, recorded when the line is stocked.
	RejectedQuantity *Quantity `json:"rejected_quantity"`
	// The purchase order line this receiving line was created from.
	OrderLine *SalesOrderLine `json:"order_line" expandable:"true"`
	// The item being received (the originating order line's item).
	Item *Item `json:"item"`
	// Timestamp when the received quantity was stocked into inventory.
	//
	// Once set, the line counts toward the order's `completion_percentage`.
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
	OrderLine: SampleSalesOrderLine,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ReceivingOrderLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleReceivingOrderLine)
}
