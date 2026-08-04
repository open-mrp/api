package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePickID = "pk_6eilj488bq8d"
const SamplePickNumber = "PK-001"

// A warehouse picking task for a sales order, tracking the quantities to pull from inventory and pack for shipment.
//
// A pick is created automatically when a sales order is issued, with one line for each order line whose product is of type `sale` — service, shipping, tax, credit and return lines are skipped — and nothing picked yet. There is no endpoint that creates a pick directly.
type Pick struct {
	// Pick ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick"`
	// Human-readable number that identifies the pick, distinct from the `id`.
	//
	// Copied from the sales order's number when the pick is created, and can be renamed with Update Pick.
	Number string `json:"number" validate:"required"`
	// The sales order this pick fulfills.
	SalesOrder *SalesOrder `json:"sales_order" expandable:"true"`
	// The customer the associated sales order is for.
	Customer *Customer `json:"customer" expandable:"true"`
	// Priority used to order picks for fulfillment, inherited from the associated sales order.
	Priority constants.PriorityCode `json:"priority" validate:"required"`
	// The pick's lines, each tracking the quantity picked against one sales order line.
	Lines *List[PickLine] `json:"lines" expandable:"true"`
	// Departments assigned to this pick.
	Departments *List[Department] `json:"departments" expandable:"true"`
	// Timestamp when the pick was finished.
	//
	// Set automatically once every line on the pick has been packed, and cleared whenever picking work reopens — when the pick is voided, when a shipment for the order is deleted, or when the order is reopened or its lines change so quantity is outstanding again. It can also be set or cleared directly with Update Pick.
	FinishedAt *time.Time `json:"finished_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SamplePick = &Pick{
	ID:          SamplePickID,
	Object:      constants.ObjectTypePick,
	Number:      SamplePickNumber,
	SalesOrder:  SampleSalesOrder,
	Customer:    SampleCustomer,
	Priority:    SamplePriorityCode,
	Lines:       NewList([]PickLine{*SamplePickLine}, PageInfo{}),
	Departments: NewList([]Department{*SampleDepartment}, PageInfo{}),
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Pick) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePick)
}

// The result of packing a pick: the pick as it stands after packing, plus the number of the shipment that packing created.
type PackPickResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pack_pick_response"`
	// The pick after the pack operation, including its updated lines.
	Pick *Pick `json:"pick" validate:"required"`
	// Number of the shipment created by the pack operation.
	//
	// Derived from the sales order number: the first shipment for an order uses the order number itself; later shipments append a sequence suffix (e.g. `SO-123-2`).
	ShipmentNumber string `json:"shipment_number" validate:"required"`
}

var SamplePackPickResponse = &PackPickResponse{
	Object:         constants.ObjectTypePackPickResponse,
	Pick:           SamplePick,
	ShipmentNumber: "SH-001",
}

func (*PackPickResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePackPickResponse)
}

// The shipment numbers for the sales order a pick belongs to.
type PickShipmentsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick_shipments_response"`
	// Shipment numbers associated with the pick, oldest first.
	ShipmentNumbers []string `json:"shipment_numbers" validate:"required"`
	// Total number of matching shipments, ignoring `limit` and `offset`.
	Count int32 `json:"count" validate:"required"`
}

var SamplePickShipmentsResponse = &PickShipmentsResponse{
	Object:          constants.ObjectTypePickShipmentsResponse,
	ShipmentNumbers: []string{"SH-001", "SH-002"},
	Count:           2,
}

func (*PickShipmentsResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePickShipmentsResponse)
}
