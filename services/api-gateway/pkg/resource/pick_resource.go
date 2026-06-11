package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePickID = "pk_016452192feb7952d8393f0105"
const SamplePickNumber = "PK-001"

// Pick is a full pick resource.
type Pick struct {
	// Pick ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick"`
	// Pick number.
	Number string `json:"number" validate:"required"`
	// Associated sales order.
	//
	// Expandable via include[]=sales_order.
	SalesOrder *SalesOrder `json:"sales_order" expandable:"true"`
	// Associated customer.
	//
	// Expandable via include[]=customer.
	Customer *Customer `json:"customer" expandable:"true"`
	// Pick priority code, used to order picks for fulfillment.
	//
	// - `low`: low priority.
	// - `normal`: normal priority.
	// - `high`: high priority.
	Priority constants.PriorityCode `json:"priority" validate:"required"`
	// Pick lines.
	Lines *List[PickLine] `json:"lines" expandable:"true"`
	// Associated departments.
	//
	// Expandable via include[]=departments.
	Departments *List[Department] `json:"departments" expandable:"true"`
	// Timestamp when the pick was finished.
	//
	// `null` while the pick is still in progress.
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
	// Updated pick.
	Pick *Pick `json:"pick" validate:"required"`
	// Created shipment number.
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
	// Total count of matching shipment numbers.
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
