package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePickDetailID = "pk_01jm4r6700f8nwq3v5hx2d9ktp"
const SamplePickNumber = "PK-001"

// PickSalesOrder is a sales order sub-resource for picks.
type PickSalesOrder struct {
	// Sales order ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order"`
}

// PickDepartment is a department sub-resource for picks.
type PickDepartment struct {
	// Department ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=department"`
	// Display name.
	Name string `json:"name" validate:"required"`
}

// PickDetail is a full pick resource.
type PickDetail struct {
	// Pick ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick"`
	// Pick number.
	Number string `json:"number" validate:"required"`
	// Associated sales order.
	SalesOrder *PickSalesOrder `json:"sales_order"`
	// Associated customer.
	Customer *Customer `json:"customer"`
	// Pick priority.
	Priority *Priority `json:"priority"`
	// Pick lines.
	Lines *List[PickLineDetail] `json:"lines" expandable:"true"`
	// Associated departments.
	Departments []PickDepartment `json:"departments"`
	// Timestamp when the pick was finished.
	FinishedAt *time.Time `json:"finished_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SamplePickSalesOrder = &PickSalesOrder{
	ID:     SampleSalesOrderDetailID,
	Object: constants.ObjectTypeSalesOrder,
}

var SamplePickDepartment = PickDepartment{
	ID:     SampleDepartmentID,
	Object: constants.ObjectTypeDepartment,
	Name:   SampleDepartmentName,
}

var SamplePickDetail = &PickDetail{
	ID:         SamplePickDetailID,
	Object:     constants.ObjectTypePick,
	Number:     SamplePickNumber,
	SalesOrder: SamplePickSalesOrder,
	Customer: &Customer{
		ID:     SampleCustomerID,
		Object: constants.ObjectTypeCustomer,
		Name:   SampleCustomerName,
		Number: SampleCustomerNumber,
	},
	Priority:    SamplePriority,
	Lines:       NewList([]PickLineDetail{*SamplePickLineDetail}, PageInfo{}),
	Departments: []PickDepartment{SamplePickDepartment},
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PickDetail) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePickDetail)
}

// PickSummary is a pick resource for list views.
type PickSummary struct {
	// Pick ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick"`
	// Pick number.
	Number string `json:"number" validate:"required"`
	// Associated sales order.
	SalesOrder *PickSalesOrder `json:"sales_order"`
	// Associated customer.
	Customer *Customer `json:"customer"`
	// Pick priority.
	Priority *Priority `json:"priority"`
	// Timestamp when the pick was finished.
	FinishedAt *time.Time `json:"finished_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SamplePickSummary = &PickSummary{
	ID:         SamplePickDetailID,
	Object:     constants.ObjectTypePick,
	Number:     SamplePickNumber,
	SalesOrder: SamplePickSalesOrder,
	Customer: &Customer{
		ID:     SampleCustomerID,
		Object: constants.ObjectTypeCustomer,
		Name:   SampleCustomerName,
		Number: SampleCustomerNumber,
	},
	Priority:  SamplePriority,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PickSummary) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePickSummary)
}

// PackPickResponse is the result of packing a pick.
type PackPickResponse struct {
	// Updated pick.
	Pick *PickDetail `json:"pick" validate:"required"`
	// Created shipment number.
	ShipmentNumber string `json:"shipment_number" validate:"required"`
}

var SamplePackPickResponse = &PackPickResponse{
	Pick:           SamplePickDetail,
	ShipmentNumber: "SH-001",
}

func (*PackPickResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePackPickResponse)
}

// PickShipmentsResponse is the result of getting shipments for a pick.
type PickShipmentsResponse struct {
	// Shipment numbers associated with the pick.
	ShipmentNumbers []string `json:"shipment_numbers" validate:"required"`
	// Total count of matching shipment numbers.
	Count int32 `json:"count" validate:"required"`
}

var SamplePickShipmentsResponse = &PickShipmentsResponse{
	ShipmentNumbers: []string{"SH-001", "SH-002"},
	Count:           2,
}

func (*PickShipmentsResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePickShipmentsResponse)
}
