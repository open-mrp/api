package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSalesOrderID = "or_01d5034136c3ccc048abecc312"
const SampleSalesOrderNumber = "SO-001"

// Sales order type sub-resource.
//
// NOTE: retained for purchase orders, which still embed the full status/type sub-resources. Sales orders now expose status/priority as plain codes.
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
//
// NOTE: retained for purchase orders (see SalesOrderType note).
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

// SalesOrderTotals holds the derived monetary totals for a sales order or one of its lines, following the lifecycle ordered -> packed -> invoiced.
type SalesOrderTotals struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_totals"`
	// Total ordered amount as a decimal string (unit price x quantity ordered).
	Ordered string `json:"ordered" validate:"required" format:"decimal"`
	// Total packed amount as a decimal string (unit price x quantity packed).
	Packed string `json:"packed" validate:"required" format:"decimal"`
	// Total invoiced amount as a decimal string (unit price x quantity invoiced).
	Invoiced string `json:"invoiced" validate:"required" format:"decimal"`
}

// SalesOrderRelated groups the records related to a sales order.
//
// The members are individually expandable (e.g. include[]=related.pick); the group itself is always present.
type SalesOrderRelated struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_related"`
	// Associated pick, as a lightweight record reference.
	Pick *Record `json:"pick" expandable:"true"`
	// Associated production run, as a lightweight record reference.
	ProductionRun *Record `json:"production_run" expandable:"true"`
	// Associated shipments, as lightweight record references.
	Shipments *List[Record] `json:"shipments" expandable:"true"`
}

// Full sales order resource.
type SalesOrder struct {
	// Sales order ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order"`
	// Human-readable order number, e.g. `SO-001`.
	//
	// Assigned automatically when the order is created; unique within your account.
	Number string `json:"number" validate:"required"`
	// The customer's own purchase order number, for cross-referencing.
	//
	// Unique among this customer's orders.
	CustomerPurchaseOrderNumber *string `json:"customer_purchase_order_number"`
	// Order note.
	Note *string `json:"note"`
	// Order lifecycle status.
	//
	// - `estimate`: a draft quote that has not yet been committed; not counted as a real order.
	// - `issued`: the order has been issued and is being fulfilled.
	// - `fulfilled`: the order has been completed and closed.
	//
	// Status changes are made through the issue, unissue, close, and reopen action endpoints rather than by updating this field.
	Status constants.SalesOrderStatusCode `json:"status" validate:"required"`
	// Fulfillment priority, used to rank orders on the shop floor.
	Priority constants.PriorityCode `json:"priority" validate:"required"`
	// Payment state of the order.
	//
	// Payment tracking is not yet wired up, so this currently always reports `unpaid`.
	PaymentStatus constants.SalesOrderPaymentStatus `json:"payment_status" validate:"required"`
	// Whether an order acknowledgment has been sent to the customer.
	AcknowledgmentStatus constants.AcknowledgmentStatus `json:"acknowledgment_status" validate:"required"`
	// Associated customer.
	Customer *Customer `json:"customer" expandable:"true"`
	// Sales representative.
	SalesRep *Actor `json:"sales_rep" expandable:"true"`
	// Who created this order, and their relation (internal/customer/system).
	CreatedBy *CreatedBy `json:"created_by" expandable:"true"`
	// Billing address.
	BillToAddress *Address `json:"bill_to_address" expandable:"true"`
	// Shipping address.
	ShipToAddress *Address `json:"ship_to_address" expandable:"true"`
	// Carrier selection and freight billing for this order.
	Freight *Freight `json:"freight" expandable:"true"`
	// Payment term.
	PaymentTerm *PaymentTerm `json:"payment_term" expandable:"true"`
	// Shipping term.
	ShippingTerm *ShippingTerm `json:"shipping_term" expandable:"true"`
	// Order discount.
	OrderDiscount *OrderDiscount `json:"order_discount" expandable:"true"`
	// Order lines.
	Lines *List[SalesOrderLine] `json:"lines" expandable:"true"`
	// Number of order lines on this order, returned even when the `lines` list itself is not expanded.
	LineCount int32 `json:"line_count"`
	// Derived monetary totals.
	Totals *SalesOrderTotals `json:"totals" expandable:"true"`
	// Records related to this order (pick, production run, shipments).
	Related *SalesOrderRelated `json:"related"`
	// When the order was issued (moved out of `estimate`).
	IssuedAt *time.Time `json:"issued_at"`
	// When the order was fulfilled and closed.
	CompletedAt *time.Time `json:"completed_at"`
	// When the first shipment against this order went out.
	FirstShipAt *time.Time `json:"first_ship_at"`
	// When this estimate expires, if an expiration was set.
	ExpiredAt *time.Time `json:"expired_at"`
	// Date promised to the customer for delivery, if one was committed.
	PromisedAt *time.Time `json:"promised_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleCustomerPurchaseOrderNumber = "PO-12345"
var sampleNote = "Rush order"

var SampleSalesOrderTotals = &SalesOrderTotals{
	Object:   constants.ObjectTypeSalesOrderTotals,
	Ordered:  "1234.560000000000000000000000000000",
	Packed:   "0.000000000000000000000000000000",
	Invoiced: "0.000000000000000000000000000000",
}

var SampleSalesOrder = &SalesOrder{
	ID:                          SampleSalesOrderID,
	Object:                      constants.ObjectTypeSalesOrder,
	Number:                      SampleSalesOrderNumber,
	CustomerPurchaseOrderNumber: &sampleCustomerPurchaseOrderNumber,
	Note:                        &sampleNote,
	Status:                      constants.SalesOrderStatusCodeEstimate,
	Priority:                    SamplePriorityCode,
	PaymentStatus:               constants.SalesOrderPaymentStatusUnpaid,
	AcknowledgmentStatus:        constants.AcknowledgmentStatusNotSent,
	Customer: &Customer{
		ID:     SampleCustomerID,
		Object: constants.ObjectTypeCustomer,
		Name:   SampleCustomerName,
		Number: SampleCustomerNumber,
	},
	BillToAddress: SampleAddress,
	ShipToAddress: SampleAddress,
	Freight:       SampleFreight,
	Lines:         NewList([]SalesOrderLine{*SampleSalesOrderLine}, PageInfo{}),
	LineCount:     1,
	Totals:        SampleSalesOrderTotals,
	Related:       &SalesOrderRelated{Object: constants.ObjectTypeSalesOrderRelated},
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SalesOrder) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSalesOrder)
}
