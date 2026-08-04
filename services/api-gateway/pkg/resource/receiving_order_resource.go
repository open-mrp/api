package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleReceivingOrderID = "rcor_iy0usuxcrjj8"
const SampleReceivingOrderLineID = "rcorln_7f39n28j00fr"

// A receiving order tracks inbound inventory against an issued purchase order.
//
// One receiving order is created automatically when a purchase order is issued, with one line per purchase order line. As goods arrive, line quantities are received and then stocked into inventory; the order is marked complete once every line is stocked. Unissuing the purchase order deletes the receiving order and its lines.
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
	// Not returned in list results.
	Note *string `json:"note"`
	// The purchase order whose issuance created this receiving order.
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
	// A line counts toward completion once its `stocked_at` is set, and the order is marked complete when the figure reaches `100`. It is calculated for list results only; on responses that return a single receiving order it is `0`, and progress is best read from the lines' `stocked_at` values.
	CompletionPercentage float64 `json:"completion_percentage"`
	// Timestamp when the receiving order was completed.
	//
	// Set automatically once every line has been stocked, and also when the originating purchase order is closed. It is cleared again when the receiving order is voided or that purchase order is re-opened.
	CompletedAt *time.Time `json:"completed_at"`
	// Timestamp when the receiving order was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when the receiving order was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleReceivingOrderNote = "Please expedite"

var SampleReceivingOrder = &ReceivingOrder{
	ID:                   SampleReceivingOrderID,
	Object:               constants.ObjectTypeReceivingOrder,
	Number:               "RO-001",
	Note:                 &sampleReceivingOrderNote,
	PurchaseOrder:        SamplePurchaseOrder,
	Supplier:             SampleSupplier,
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
	// Quantity refused on inspection and never taken into inventory.
	//
	// Accumulated from the rejected quantities recorded against this line each time the order is stocked.
	RejectedQuantity *Quantity `json:"rejected_quantity"`
	// The purchase order line this receiving line was created from.
	OrderLine *SalesOrderLine `json:"order_line" expandable:"true"`
	// The item being received (the originating order line's item).
	Item *Item `json:"item"`
	// Timestamp when the received quantity was stocked into inventory.
	//
	// Once set, the line counts toward the order's `completion_percentage`. Voiding the line or the whole order clears it, but does not reverse the inventory that was already received.
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
	Item:      SampleItem,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ReceivingOrderLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleReceivingOrderLine)
}
