package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
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
	// The supplier (seller) account the originating purchase order was placed with.
	Supplier *Supplier `json:"supplier" expandable:"true"`
	// Line items in this receiving order.
	Lines *List[ReceivingOrderLine] `json:"lines" expandable:"true"`
	// Total number of lines on this receiving order.
	//
	// Always populated, even when `lines` is not expanded.
	LineCount int32 `json:"line_count"`
	// What the order is worth and how far it has been put away.
	Totals *ReceivingOrderTotals `json:"totals" expandable:"true"`
	// The records this receiving order sits between.
	Related *ReceivingOrderRelated `json:"related" expandable:"true"`
	// Timestamp when the receiving order was completed.
	//
	// Set automatically once every line has been stocked, and also when the originating purchase order is closed. It is cleared again when the receiving order is voided or that purchase order is re-opened.
	CompletedAt *time.Time `json:"completed_at"`
	// Timestamp when the receiving order was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when the receiving order was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// ReceivingOrderTotals is what the order is worth and how far it has been put away.
//
// A receiving order's lines can each count in a different unit, so the amounts — the purchase order's agreed unit price times a quantity — are what make the stages comparable, and completion is a ratio of two of them.
type ReceivingOrderTotals struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=receiving_order_totals"`
	// Total value the purchase order asked for across this order's lines, as a decimal string.
	//
	// This is the baseline the stage completions are measured against.
	Ordered string `json:"ordered" validate:"required" format:"decimal"`
	// Value taken into inventory, and how far stocking has progressed.
	Stocked ReceivingOrderStageTotal `json:"stocked"`
	// Value refused on inspection, and how much of the order that accounts for.
	Rejected ReceivingOrderStageTotal `json:"rejected"`
}

var SampleReceivingOrderTotals = &ReceivingOrderTotals{
	Object:   constants.ObjectTypeReceivingOrderTotals,
	Ordered:  "12480.00",
	Stocked:  ReceivingOrderStageTotal{Object: constants.ObjectTypeReceivingOrderStageTotal, Amount: "6240.00", Completion: 0.5},
	Rejected: ReceivingOrderStageTotal{Object: constants.ObjectTypeReceivingOrderStageTotal, Amount: "0.00", Completion: 0},
}

func (*ReceivingOrderTotals) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleReceivingOrderTotals)
}

// ReceivingOrderStageTotal is how much of a receiving order has reached one stage.
type ReceivingOrderStageTotal struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=receiving_order_stage_total"`
	// Value that has reached this stage, as a decimal string.
	Amount string `json:"amount" validate:"required" format:"decimal"`
	// Progress through this stage, as a fraction between 0 and 1.
	//
	// Calculated as this stage's amount divided by `totals.ordered`, so `1` means the whole order has cleared the stage. It is a ratio of amounts rather than of quantities because a receiving order's lines can each count in a different unit.
	Completion float64 `json:"completion"`
}

func (*ReceivingOrderStageTotal) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleReceivingOrderTotals.Stocked)
}

// ReceivingOrderRelated names the records a receiving order sits between.
type ReceivingOrderRelated struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=receiving_order_related"`
	// The purchase order whose issuance created this receiving order.
	PurchaseOrder *Record `json:"purchase_order" expandable:"true"`
	// The deliveries booked against this order, oldest first.
	Deliveries *List[Record] `json:"deliveries" expandable:"true"`
}

var SampleReceivingOrderRelated = &ReceivingOrderRelated{
	Object: constants.ObjectTypeReceivingOrderRelated,
}

func (*ReceivingOrderRelated) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleReceivingOrderRelated)
}

var sampleReceivingOrderNote = "Please expedite"

var SampleReceivingOrder = &ReceivingOrder{
	ID:        SampleReceivingOrderID,
	Object:    constants.ObjectTypeReceivingOrder,
	Number:    "RO-001",
	Note:      &sampleReceivingOrderNote,
	Supplier:  SampleSupplier,
	Lines:     NewList([]ReceivingOrderLine{*SampleReceivingOrderLine}, PageInfo{}),
	LineCount: 2,
	Totals:    SampleReceivingOrderTotals,
	Related:   SampleReceivingOrderRelated,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
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
	// Accumulated from the rejected quantities recorded against this line each time the order is stocked, so it is summed at read time rather than stored: it carries no id, and arrives with the unit it was summed in.
	RejectedQuantity *ComputedQuantity `json:"rejected_quantity"`
	// Position of the originating purchase order line within its order, starting at 1.
	LineItemNumber *int32 `json:"line_item_number"`
	// The item being received.
	Item *Item `json:"item" expandable:"true"`
	// Quantity the purchase order asked for on this line, which `quantity` is measured against.
	QuantityOrdered *Quantity `json:"quantity_ordered" expandable:"true"`
	// The purchase order line this line receives against.
	//
	// The receiving line is raised from a purchase order line and carries that line's item and ordered quantity directly; expand this to read the rest of it, including the agreed unit price.
	OrderLine *PurchaseOrderLine `json:"order_line" expandable:"true"`
	// Timestamp when the received quantity was stocked into inventory.
	//
	// Once set, the line counts toward the order's `totals.stocked.completion`. Voiding the line or the whole order clears it, but does not reverse the inventory that was already received.
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
	Item:      SampleItem,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ReceivingOrderLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleReceivingOrderLine)
}
