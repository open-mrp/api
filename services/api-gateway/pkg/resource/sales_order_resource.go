package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSalesOrderDetailID = "or_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleSalesOrderNumber = "SO-001"

// SalesOrderType represents a sales order type sub-resource.
type SalesOrderType struct {
	// The type code.
	Code string `json:"code" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_type"`
	// The display name of the type.
	Name string `json:"name" validate:"required"`
}

var SampleSalesOrderType = &SalesOrderType{
	Code:   "standard",
	Object: constants.ObjectTypeSalesOrderType,
	Name:   "Standard",
}

// SalesOrderStatusDetail represents a sales order status sub-resource.
type SalesOrderStatusDetail struct {
	// The status code.
	Code string `json:"code" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_status"`
	// The display name of the status.
	Name string `json:"name" validate:"required"`
}

var SampleSalesOrderStatusDetail = &SalesOrderStatusDetail{
	Code:   "estimate",
	Object: constants.ObjectTypeSalesOrderStatus,
	Name:   "Estimate",
}

// Pick represents a minimal pick sub-resource.
type Pick struct {
	// The unique identifier for the pick.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object string `json:"object" validate:"required"`
}

// SalesOrderDetail represents a full sales order resource.
type SalesOrderDetail struct {
	// The unique identifier for the sales order.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order"`
	// The sales order number.
	Number string `json:"number" validate:"required"`
	// The customer purchase order number.
	CustomerPO *string `json:"customer_po"`
	// A note attached to this sales order.
	Note *string `json:"note"`
	// Whether the acknowledgment has been sent.
	IsAcknowledgmentSent bool `json:"is_acknowledgment_sent"`
	// The customer associated with this order.
	Customer *Customer `json:"customer" expandable:"true"`
	// The billing address.
	BillToAddress *Address `json:"bill_to_address" expandable:"true"`
	// The shipping address.
	ShipToAddress *Address `json:"ship_to_address" expandable:"true"`
	// The carrier for this order.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// The service level for this order.
	ServiceLevel *ServiceLevel `json:"service_level" expandable:"true"`
	// The carrier billing type.
	CarrierBillingType *string `json:"carrier_billing_type"`
	// The carrier billing account number.
	CarrierBillingAccount *string `json:"carrier_billing_account"`
	// The sales representative. Uses Actor sub-resource.
	SalesRep *Actor `json:"sales_rep"`
	// The order status.
	Status *SalesOrderStatusDetail `json:"status" validate:"required"`
	// The order type.
	Type *SalesOrderType `json:"type" validate:"required"`
	// The priority.
	Priority *Priority `json:"priority" validate:"required"`
	// The payment term.
	PaymentTerm *PaymentTerm `json:"payment_term" expandable:"true"`
	// The shipping term.
	ShippingTerm *ShippingTerm `json:"shipping_term" expandable:"true"`
	// The order discount.
	OrderDiscount *OrderDiscount `json:"order_discount" expandable:"true"`
	// The production run associated with this order.
	ProductionRun *ProductionRun `json:"production_run"`
	// The pick associated with this order.
	Pick *Pick `json:"pick"`
	// The order lines.
	Lines *List[SalesOrderLineDetail] `json:"lines" expandable:"true"`
	// The timestamp when the order was issued.
	IssuedAt *time.Time `json:"issued_at"`
	// The timestamp when the order was completed/fulfilled.
	CompletedAt *time.Time `json:"completed_at"`
	// The timestamp of the first shipment.
	FirstShipAt *time.Time `json:"first_ship_at"`
	// The timestamp when the order expired.
	ExpiredAt *time.Time `json:"expired_at"`
	// The timestamp when the order is promised.
	PromisedAt *time.Time `json:"promised_at"`
	// The timestamp when the order was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the order was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleCustomerPO = "PO-12345"
var sampleNote = "Rush order"

var SampleSalesOrderDetail = &SalesOrderDetail{
	ID:                   SampleSalesOrderDetailID,
	Object:               constants.ObjectTypeSalesOrder,
	Number:               SampleSalesOrderNumber,
	CustomerPO:           &sampleCustomerPO,
	Note:                 &sampleNote,
	IsAcknowledgmentSent: false,
	Customer: &Customer{
		ID:     SampleCustomerID,
		Object: constants.ObjectTypeCustomer,
		Name:   SampleCustomerName,
		Number: SampleCustomerNumber,
	},
	BillToAddress: SampleAddress,
	ShipToAddress: SampleAddress,
	Status:        SampleSalesOrderStatusDetail,
	Type:          SampleSalesOrderType,
	Priority:      SamplePriority,
	Lines:         NewList([]SalesOrderLineDetail{*SampleSalesOrderLineDetail}, PageInfo{}),
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SalesOrderDetail) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSalesOrderDetail)
}

// SalesOrderSummary represents a lightweight sales order for list views.
type SalesOrderSummary struct {
	// The unique identifier for the sales order.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order"`
	// The sales order number.
	Number string `json:"number" validate:"required"`
	// The customer purchase order number.
	CustomerPO *string `json:"customer_po"`
	// The customer associated with this order.
	Customer *Customer `json:"customer" validate:"required"`
	// The order status.
	Status *SalesOrderStatusDetail `json:"status" validate:"required"`
	// The order type.
	Type *SalesOrderType `json:"type" validate:"required"`
	// The priority.
	Priority *Priority `json:"priority" validate:"required"`
	// The number of line items.
	LineCount int32 `json:"line_count"`
	// Whether the acknowledgment has been sent.
	IsAcknowledgmentSent bool `json:"is_acknowledgment_sent"`
	// The timestamp when the order was issued.
	IssuedAt *time.Time `json:"issued_at"`
	// The timestamp when the order was completed.
	CompletedAt *time.Time `json:"completed_at"`
	// The timestamp when the order was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the order was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleSalesOrderSummary = &SalesOrderSummary{
	ID:     SampleSalesOrderDetailID,
	Object: constants.ObjectTypeSalesOrder,
	Number: SampleSalesOrderNumber,
	Customer: &Customer{
		ID:     SampleCustomerID,
		Object: constants.ObjectTypeCustomer,
		Name:   SampleCustomerName,
		Number: SampleCustomerNumber,
	},
	Status:    SampleSalesOrderStatusDetail,
	Type:      SampleSalesOrderType,
	Priority:  SamplePriority,
	LineCount: 3,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SalesOrderSummary) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSalesOrderSummary)
}
