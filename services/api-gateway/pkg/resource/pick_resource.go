package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePickDetailID = "pk_01jm4r6700f8nwq3v5hx2d9ktp"
const SamplePickNumber = "PK-001"

// PickSalesOrder is a minimal sales order sub-resource for picks.
type PickSalesOrder struct {
	// The unique identifier for the sales order.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order"`
}

// PickDepartment is a minimal department sub-resource for picks.
type PickDepartment struct {
	// The unique identifier for the department.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=department"`
	// The display name of the department.
	Name string `json:"name" validate:"required"`
}

// PickDetail represents a full pick resource.
type PickDetail struct {
	// The unique identifier for the pick.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick"`
	// The pick number.
	Number string `json:"number" validate:"required"`
	// The sales order associated with this pick.
	SalesOrder *PickSalesOrder `json:"sales_order"`
	// The customer associated with this pick.
	Customer *Customer `json:"customer"`
	// The priority of this pick.
	Priority *Priority `json:"priority"`
	// The pick lines.
	Lines *List[PickLineDetail] `json:"lines" expandable:"true"`
	// The departments associated with this pick.
	Departments []PickDepartment `json:"departments"`
	// The timestamp when the pick was finished.
	FinishedAt *time.Time `json:"finished_at"`
	// The timestamp when the pick was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the pick was last updated.
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

// PickSummary represents a pick in list views.
type PickSummary struct {
	// The unique identifier for the pick.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick"`
	// The pick number.
	Number string `json:"number" validate:"required"`
	// The sales order associated with this pick.
	SalesOrder *PickSalesOrder `json:"sales_order"`
	// The customer associated with this pick.
	Customer *Customer `json:"customer"`
	// The priority of this pick.
	Priority *Priority `json:"priority"`
	// The timestamp when the pick was finished.
	FinishedAt *time.Time `json:"finished_at"`
	// The timestamp when the pick was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the pick was last updated.
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

// PackPickResponse represents the response from packing a pick.
type PackPickResponse struct {
	// The updated pick.
	Pick *PickDetail `json:"pick" validate:"required"`
	// The shipment number created.
	ShipmentNumber string `json:"shipment_number" validate:"required"`
}

var SamplePackPickResponse = &PackPickResponse{
	Pick:           SamplePickDetail,
	ShipmentNumber: "SH-001",
}

func (*PackPickResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePackPickResponse)
}

// PickShipmentsResponse represents the response from getting shipments for a pick.
type PickShipmentsResponse struct {
	// The shipment numbers associated with the pick.
	ShipmentNumbers []string `json:"shipment_numbers" validate:"required"`
	// The total count of matching shipment numbers.
	Count int32 `json:"count" validate:"required"`
}

var SamplePickShipmentsResponse = &PickShipmentsResponse{
	ShipmentNumbers: []string{"SH-001", "SH-002"},
	Count:           2,
}

func (*PickShipmentsResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePickShipmentsResponse)
}
