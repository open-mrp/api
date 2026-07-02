package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePickID = "pk_016452192feb7952d8393f0105"
const SamplePickNumber = "PK-001"

// A warehouse picking task for a sales order, tracking the quantities to pull from inventory and pack for shipment.
type Pick struct {
	// Pick ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick"`
	// Human-readable number that identifies the pick, distinct from the `id`.
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
	// Unset while the pick is still in progress. Set automatically when packing leaves no unpacked lines with a remaining quantity to pick, and cleared when the pick is voided; it can also be set or cleared directly via Update Pick.
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
	Priority:    SamplePriorityCode,
	Lines:       NewList([]PickLine{*SamplePickLine}, PageInfo{}),
	Departments: NewList([]Department{*SampleDepartment}, PageInfo{}),
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Pick) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePick)
}

// PackPickResponse is the result of packing a pick.
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

// PickShipmentsResponse is the result of getting shipments for a pick.
type PickShipmentsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick_shipments_response"`
	// Shipment numbers associated with the pick.
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
