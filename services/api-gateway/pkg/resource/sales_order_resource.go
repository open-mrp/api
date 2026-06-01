package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSalesOrderDetailID = "or_01d5034136c3ccc048abecc312"
const SampleSalesOrderNumber = "SO-001"

// Sales order type sub-resource.
type SalesOrderType struct {
	// Type code.
	Code string `json:"code" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_type"`
	// Display name.
	Name string `json:"name" validate:"required"`
}

var SampleSalesOrderType = &SalesOrderType{
	Code:   "standard",
	Object: constants.ObjectTypeSalesOrderType,
	Name:   "Standard",
}

// Sales order status sub-resource.
type SalesOrderStatusDetail struct {
	// Status code.
	Code string `json:"code" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_status"`
	// Display name.
	Name string `json:"name" validate:"required"`
}

var SampleSalesOrderStatusDetail = &SalesOrderStatusDetail{
	Code:   "estimate",
	Object: constants.ObjectTypeSalesOrderStatus,
	Name:   "Estimate",
}

// Minimal pick sub-resource.
type Pick struct {
	// Pick ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick"`
}

// Full sales order resource.
type SalesOrderDetail struct {
	// Sales order ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order"`
	// Sales order number.
	Number string `json:"number" validate:"required"`
	// Customer purchase order number.
	CustomerPO *string `json:"customer_po"`
	// Order note.
	Note *string `json:"note"`
	// Whether the acknowledgment has been sent.
	IsAcknowledgmentSent bool `json:"is_acknowledgment_sent"`
	// Associated customer.
	Customer *Customer `json:"customer" expandable:"true"`
	// Billing address.
	BillToAddress *Address `json:"bill_to_address" expandable:"true"`
	// Shipping address.
	ShipToAddress *Address `json:"ship_to_address" expandable:"true"`
	// Carrier.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// Service level.
	ServiceLevel *ServiceLevel `json:"service_level" expandable:"true"`
	// Carrier billing type.
	CarrierBillingType *string `json:"carrier_billing_type"`
	// Carrier billing account number.
	CarrierBillingAccount *string `json:"carrier_billing_account"`
	// Sales representative. Uses Actor sub-resource.
	SalesRep *Actor `json:"sales_rep"`
	// Order status.
	Status *SalesOrderStatusDetail `json:"status" validate:"required"`
	// Order type.
	Type *SalesOrderType `json:"type" validate:"required"`
	// Priority.
	Priority *Priority `json:"priority" validate:"required"`
	// Payment term.
	PaymentTerm *PaymentTerm `json:"payment_term" expandable:"true"`
	// Shipping term.
	ShippingTerm *ShippingTerm `json:"shipping_term" expandable:"true"`
	// Order discount.
	OrderDiscount *OrderDiscount `json:"order_discount" expandable:"true"`
	// Associated production run.
	ProductionRun *ProductionRun `json:"production_run"`
	// Associated pick.
	Pick *Pick `json:"pick"`
	// Order lines.
	Lines *List[SalesOrderLineDetail] `json:"lines" expandable:"true"`
	// Count of order lines. Always populated in list responses.
	LineCount int32 `json:"line_count"`
	// Issued timestamp.
	IssuedAt *time.Time `json:"issued_at"`
	// Completed timestamp.
	CompletedAt *time.Time `json:"completed_at"`
	// First shipment timestamp.
	FirstShipAt *time.Time `json:"first_ship_at"`
	// Expiration timestamp.
	ExpiredAt *time.Time `json:"expired_at"`
	// Promised timestamp.
	PromisedAt *time.Time `json:"promised_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
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

// Lightweight sales order for list views.
type SalesOrderSummary struct {
	// Sales order ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order"`
	// Sales order number.
	Number string `json:"number" validate:"required"`
	// Customer purchase order number.
	CustomerPO *string `json:"customer_po"`
	// Associated customer.
	Customer *Customer `json:"customer" validate:"required"`
	// Order status.
	Status *SalesOrderStatusDetail `json:"status" validate:"required"`
	// Order type.
	Type *SalesOrderType `json:"type" validate:"required"`
	// Priority.
	Priority *Priority `json:"priority" validate:"required"`
	// Line item count.
	LineCount int32 `json:"line_count"`
	// Whether the acknowledgment has been sent.
	IsAcknowledgmentSent bool `json:"is_acknowledgment_sent"`
	// Issued timestamp.
	IssuedAt *time.Time `json:"issued_at"`
	// Completed timestamp.
	CompletedAt *time.Time `json:"completed_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
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

func ExpandableSalesOrderStub(id, number string, ts time.Time) *SalesOrderDetail {
	if id == "" {
		id = SampleSalesOrderDetailID
	}
	if number == "" {
		number = SampleSalesOrderNumber
	}
	if ts.IsZero() {
		ts = time.Unix(0, 0).UTC()
	}
	return &SalesOrderDetail{
		ID:     id,
		Object: constants.ObjectTypeSalesOrder,
		Number: number,
		Status: &SalesOrderStatusDetail{
			Code:   string(constants.SalesOrderStatusCodeIssued),
			Object: constants.ObjectTypeSalesOrderStatus,
			Name:   "Issued",
		},
		Type: &SalesOrderType{
			Code:   "standard",
			Object: constants.ObjectTypeSalesOrderType,
			Name:   "Standard",
		},
		Priority:  ExpandablePriorityStub("", constants.PriorityCodeNormal, SamplePriorityName, ts),
		CreatedAt: ts,
		UpdatedAt: ts,
	}
}
