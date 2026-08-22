package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleSalesOrderID = "or_9lqo07quiwyb"
const SampleSalesOrderNumber = "SO-001"

// Sales order type sub-resource.
//
// NOTE: currently unreferenced — sales orders and purchase orders both expose status, type, and priority as plain codes.
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
// NOTE: currently unreferenced (see the SalesOrderType note).
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

// The monetary amount that has reached one fulfillment stage, together with how far that stage has progressed.
type SalesOrderStageTotal struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_stage_total"`
	// Amount that has reached this stage, as a decimal string (unit price times the quantity at this stage).
	Amount string `json:"amount" validate:"required" format:"decimal"`
	// Progress through this stage, as a fraction between 0 and 1.
	//
	// Calculated as the quantity that has reached this stage divided by the quantity ordered, so `1` means the whole order has cleared the stage and `0` means nothing has reached it yet.
	Completion float64 `json:"completion"`
}

// Derived monetary totals for a sales order or one of its lines.
//
// Fulfillment runs ordered -> picked -> packed -> invoiced, and each downstream stage reports both the money that has reached it and its progress against the ordered baseline.
type SalesOrderTotals struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_totals"`
	// Total ordered amount as a decimal string (unit price x quantity ordered).
	//
	// This is the baseline the stage completions are measured against.
	Ordered string `json:"ordered" validate:"required" format:"decimal"`
	// Picked amount and completion.
	Picked SalesOrderStageTotal `json:"picked"`
	// Packed amount and completion.
	Packed SalesOrderStageTotal `json:"packed"`
	// Invoiced amount and completion.
	Invoiced SalesOrderStageTotal `json:"invoiced"`
}

// A sales order's email recipients, grouped by the notification they receive.
type OrderContact struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=order_contact"`
	// Email addresses that receive invoices for this order.
	Invoice []string `json:"invoice"`
	// Email addresses that receive order acknowledgements for this order.
	Acknowledgement []string `json:"acknowledgement"`
}

// The fulfillment records produced from a sales order.
//
// The group itself is returned only when at least one of its members has been expanded.
type SalesOrderRelated struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_related"`
	// The pick created when the order was issued.
	Pick *Record `json:"pick" expandable:"true"`
	// The production run created for this order, if one was started.
	ProductionRun *Record `json:"production_run" expandable:"true"`
	// Shipments made against this order.
	Shipments *List[Record] `json:"shipments" expandable:"true"`
	// Invoices raised against this order.
	Invoices *List[Record] `json:"invoices" expandable:"true"`
}

// An order placed by a customer, tracked from estimate through fulfillment.
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
	// Free-form note about the order.
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
	// Payment state of the order, derived from settlement allocations, invoices, and Stripe payments.
	PaymentStatus constants.SalesOrderPaymentStatus `json:"payment_status" validate:"required"`
	// Stripe payment intent IDs recorded against this order.
	PaymentIntentIDs []string `json:"payment_intent_ids"`
	// Whether an order acknowledgment has been sent to the customer.
	//
	// Becomes `sent` when the order is issued with customer notification requested and the order has acknowledgement contacts to send to. It can also be set directly when an acknowledgement was sent outside OpenMRP.
	AcknowledgmentStatus constants.AcknowledgmentStatus `json:"acknowledgment_status" validate:"required"`
	// The customer this order is for.
	Customer *Customer `json:"customer" expandable:"true"`
	// The sales representative credited with the order.
	//
	// Chosen automatically at creation when none is supplied, from the customer's default rep or the sales territory covering the ship-to address.
	SalesRep *Actor `json:"sales_rep" expandable:"true"`
	// Who created this order, and their relation (internal/customer/system).
	CreatedBy *CreatedBy `json:"created_by" expandable:"true"`
	// Address the order is billed to.
	BillToAddress *Address `json:"bill_to_address" expandable:"true"`
	// Address the order ships to.
	ShipToAddress *Address `json:"ship_to_address" expandable:"true"`
	// Carrier selection and freight billing for this order.
	//
	// The freight charge itself is carried as a line on the order, not on this object.
	Freight *Freight `json:"freight" expandable:"true"`
	// Payment terms agreed for this order.
	PaymentTerm *PaymentTerm `json:"payment_term" expandable:"true"`
	// Shipping terms agreed for this order.
	ShippingTerm *ShippingTerm `json:"shipping_term" expandable:"true"`
	// Order-level discount applied to this order.
	//
	// The discount is charged through a negative-priced line on the order, so it is already reflected in the order totals.
	OrderDiscount *OrderDiscount `json:"order_discount" expandable:"true"`
	// The order's lines, including the automatically generated freight and discount lines.
	Lines *List[SalesOrderLine] `json:"lines" expandable:"true"`
	// Number of lines on this order.
	LineCount int32 `json:"line_count"`
	// Derived monetary totals and per-stage fulfillment progress.
	Totals *SalesOrderTotals `json:"totals" expandable:"true"`
	// Fulfillment records produced from this order.
	Related *SalesOrderRelated `json:"related"`
	// Email recipients grouped by notification purpose.
	Contacts *OrderContact `json:"contacts" expandable:"true"`
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
	// Days between issue and the ship-by date, set on this order alone in place of the customer's standing lead time.
	LeadTimeOverrideDays *int32 `json:"lead_time_override_days"`
	// The ship date pinned on this order, bypassing transit and the customer's receiving days.
	ShipByOverrideDate *time.Time `json:"ship_by_override_date"`
	// Date this order is contractually due to ship.
	//
	// Stamped when the order is issued. With a promised delivery date, this is that date less the carrier's transit for the order's lane and less any day the customer cannot receive on — the day the order has to leave to arrive when promised. Otherwise it comes from a lead time, whether this order's own or the one on the customer, its parent account, its account group, or the account.
	//
	// Always a day the plant actually ships on, whichever rule produced it.
	//
	// It is not recomputed afterwards, so neither renegotiating a customer's lead time, nor a later carrier estimate, nor a holiday added to a calendar moves commitments already made. Cleared if the order is unissued.
	ShipByDate *time.Time `json:"ship_by_date"`
	// The ship-by date at the plant's pickup cutoff — the moment freight has to be tendered by, not just the day.
	//
	// Only set when the account's shipping calendar carries a cutoff time.
	ShipByCutoffAt *time.Time `json:"ship_by_cutoff_at"`
	// Days the customer's receiving calendar and the plant's shipping calendar pulled the ship-by date back, beyond what carrier transit accounted for.
	//
	// Zero means every date along the way already fell on an open day. This is what explains a ship-by date that is earlier than transit alone would suggest.
	CalendarAdjustmentDays *int32 `json:"calendar_adjustment_days"`
	// Calendar days between issue and the ship-by date.
	LeadTimeDays *int32 `json:"lead_time_days"`
	// Which rule produced the ship-by date.
	LeadTimeSource *constants.LeadTimeSource `json:"lead_time_source"`
	// Business days the carrier needs to cover this order's lane, subtracted from the promised delivery date to reach the ship-by date.
	//
	// Only set when a delivery date was promised and the lane could be priced. Without it the ship-by date falls back to the promised date itself.
	TransitDays *int32 `json:"transit_days"`
	// Where the transit estimate came from.
	TransitSource *constants.TransitSource `json:"transit_source"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleCustomerPurchaseOrderNumber = "PO-12345"
var sampleNote = "Rush order"

var SampleSalesOrderTotals = &SalesOrderTotals{
	Object:   constants.ObjectTypeSalesOrderTotals,
	Ordered:  "1234.56",
	Picked:   SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: "617.280000000000000000000000000000", Completion: 0.5},
	Packed:   SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: "308.640000000000000000000000000000", Completion: 0.25},
	Invoiced: SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: "0.000000000000000000000000000000", Completion: 0},
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
	PaymentIntentIDs:            []string{},
	AcknowledgmentStatus:        constants.AcknowledgmentStatusNotSent,
	Customer:                    SampleCustomer,
	SalesRep:                    SampleActor,
	CreatedBy:                   SampleCreatedBy,
	BillToAddress:               SampleAddress,
	ShipToAddress:               SampleAddress,
	Freight:                     SampleFreight,
	PaymentTerm:                 SamplePaymentTerm,
	ShippingTerm:                SampleShippingTerm,
	OrderDiscount:               SampleOrderDiscount,
	Lines:                       NewList([]SalesOrderLine{*SampleSalesOrderLine}, PageInfo{}),
	LineCount:                   1,
	Totals:                      SampleSalesOrderTotals,
	Related:                     &SalesOrderRelated{Object: constants.ObjectTypeSalesOrderRelated},
	Contacts: &OrderContact{
		Object:          constants.ObjectTypeOrderContact,
		Invoice:         []string{"ap@acme.example.com"},
		Acknowledgement: []string{"purchasing@acme.example.com"},
	},
	PromisedAt: timeutil.TimestampToTimePtr(sampleExpiresAtTimestamp),
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SalesOrder) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSalesOrder)
}
