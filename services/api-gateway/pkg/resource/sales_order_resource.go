package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleSalesOrderID = "or_9lqo07quiwyb"
const SampleSalesOrderNumber = "SO-001"

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

// Lightweight references to the records produced from a sales order.
//
// The group itself is returned only when at least one of its members is included.
type SalesOrderRelated struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_related"`
	// The pick created when the order was issued. `null` unless included.
	Pick *Record `json:"pick" expandable:"true"`
	// The production work started for the order. `null` unless included.
	Jobs *List[Record] `json:"jobs" expandable:"true"`
	// Shipments made against the order. `null` unless included.
	Shipments *List[Record] `json:"shipments" expandable:"true"`
	// Invoices raised against the order. `null` unless included.
	Invoices *List[Record] `json:"invoices" expandable:"true"`
}

// How a sales order ships: the carrier and service level, and who the carrier bills.
type SalesOrderFreight struct {
	// The carrier that ships the order. `null` unless included.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// The carrier's service level the order ships on. `null` unless included.
	ServiceLevel *ServiceLevel `json:"service_level" expandable:"true"`
	// Who the carrier bills for the order's freight. `null` when not set.
	CarrierBilling *CarrierBilling `json:"carrier_billing"`
}

// Who a carrier bills for freight.
type CarrierBilling struct {
	// `sender`: you pay the carrier. `third_party`: the carrier bills `account_number`.
	Type constants.CarrierBillingType `json:"type" validate:"required"`
	// The carrier account billed when `type` is `third_party`.
	AccountNumber *string `json:"account_number"`
}

// When a sales order is due to ship: what was promised or pinned, the resulting ship-by date, and which rule decided it.
type SalesOrderCommitment struct {
	// The date delivery was promised to the customer.
	//
	// The ship-by date is worked back from it through the carrier's transit and the customer's receiving days. At most one of `promised_on`, `lead_time_override` and `ship_by_override_on` is set.
	PromisedOn *timeutil.Date `json:"promised_on"`
	// Business days from the order being issued to it being due to ship, for this order alone in place of the customer's lead time.
	LeadTimeOverride *Quantity `json:"lead_time_override"`
	// The exact date the order is due to ship, bypassing transit and the customer's receiving days.
	//
	// Moved back to the nearest earlier day you ship on.
	ShipByOverrideOn *timeutil.Date `json:"ship_by_override_on"`
	// Whether the customer can accept the order late.
	//
	// `soft`: they can. `hard`: they can't. `null` when not set.
	Firmness *constants.CommitmentFirmness `json:"firmness"`
	// The date the order is due to ship. Set when the order is issued and cleared if it is unissued.
	//
	// Always a day you ship on. Moves while the order is open when what it was worked out from changes; a change to the customer's lead time or a calendar doesn't move a date already promised.
	ShipByOn *timeutil.Date `json:"ship_by_on"`
	// The lead time the ship-by date was promised on, in business days on your working calendar.
	//
	// The later of the customer's lead time and the slowest item's sales lead time, or the order's own `lead_time_override`. `null` when a date was promised or pinned instead, or before the order is issued.
	PromisedLeadTime *Quantity `json:"promised_lead_time"`
	// Where `promised_lead_time` came from.
	//
	// `customer`: the customer's lead time (its own, its parent customer's, its group's or your account default). `item`: an item's sales lead time, longer than the customer's. `override`: the order's `lead_time_override`.
	PromisedLeadTimeSource *constants.PromisedLeadTimeSource `json:"promised_lead_time_source"`
	// Which rule produced the ship-by date, at the level of the customer chain.
	LeadTimeSource *constants.LeadTimeSource `json:"lead_time_source" visibility:"internal"`
	// Business days the carrier needs for the order's lane, subtracted from `promised_on`.
	TransitDays *int32 `json:"transit_days" visibility:"internal"`
	// Where `transit_days` came from.
	TransitSource *constants.TransitSource `json:"transit_source" visibility:"internal"`
	// Days the receiving and shipping calendars pulled the ship-by date back beyond transit.
	CalendarAdjustmentDays *int32 `json:"calendar_adjustment_days" visibility:"internal"`
	// The ship-by date at your pickup cutoff, when your shipping calendar sets one.
	ShipByCutoffAt *time.Time `json:"ship_by_cutoff_at" visibility:"internal"`
}

// A sales order's email recipients' addresses, grouped by the document they receive.
type OrderContact struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=order_contact"`
	// Addresses that receive the order's invoices.
	Invoice []string `json:"invoice"`
	// Addresses that receive the order's acknowledgment.
	Acknowledgement []string `json:"acknowledgement"`
}

// An order a customer placed with you, from draft through fulfillment.
type SalesOrder struct {
	// Sales order ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order"`
	// Order number, unique within your account, such as `SO-001`. Assigned when the order is created.
	Number string `json:"number" validate:"required"`
	// Where the order is in its lifecycle.
	//
	// `draft`: not yet committed. `issued`: being fulfilled. `fulfilled`: every line shipped or closed (set automatically). `canceled`: stopped before anything shipped. Issue, unissue and cancel are the only actions; `fulfilled` follows the shipments.
	Status constants.SalesOrderStatus `json:"status" validate:"required"`
	// The order's currency. Every amount on the order is in it.
	Currency constants.Currency `json:"currency" validate:"required"`
	// The customer's own purchase order number. Unique among the customer's orders.
	CustomerPurchaseOrderNumber *string `json:"customer_purchase_order_number"`
	// Your team's instructions for the order, such as how to pack it.
	//
	// Copied from the customer's when the order was created without its own. Always `null` for customer and supplier portal users.
	Instructions *string `json:"instructions" sensitive:"internal"`
	// How much of the order has been paid.
	//
	// Derived from invoices, payments and settlements; read-only.
	PaymentStatus constants.SalesOrderPaymentStatus `json:"payment_status" validate:"required"`
	// Whether an order acknowledgment was sent.
	//
	// Becomes `sent` when the order is issued with `notify_customer` and has acknowledgment recipients. Set it yourself when you acknowledged the order outside OpenMRP.
	AcknowledgmentStatus constants.AcknowledgmentStatus `json:"acknowledgment_status" validate:"required"`
	// The customer the order is for. `null` unless included.
	Customer *Customer `json:"customer" expandable:"true"`
	// The site the order ships from. `null` when the order uses your default site, and unless included.
	Site *Location `json:"site" expandable:"true"`
	// Your account user credited with the order. `null` unless included.
	//
	// Chosen when the order is created without one: the customer's default sales rep, else the sales territory covering the ship-to address.
	SalesRep *AccountUser `json:"sales_rep" expandable:"true"`
	// Who created the order, and their relation (internal/customer/system). `null` unless included.
	CreatedBy *CreatedBy `json:"created_by" expandable:"true"`
	// The address the order is billed to, as it was when set on the order.
	//
	// `address` is the saved address it came from, if any. Later changes to that address don't affect this order.
	BillTo *AddressSnapshot `json:"bill_to"`
	// The address the order ships to, as it was when set on the order.
	//
	// `address` is the saved address it came from, if any. Later changes to that address don't affect this order.
	ShipTo *AddressSnapshot `json:"ship_to"`
	// How the order ships. The freight charge itself is a line on the order.
	Freight *SalesOrderFreight `json:"freight" validate:"required"`
	// The order's payment term. `null` unless included.
	PaymentTerm *PaymentTerm `json:"payment_term" expandable:"true"`
	// The order's shipping term. `null` unless included.
	ShippingTerm *ShippingTerm `json:"shipping_term" expandable:"true"`
	// The order discount applied to the order. `null` unless included.
	OrderDiscount *OrderDiscount `json:"order_discount" expandable:"true"`
	// Who receives the order's documents by email: each recipient is one of the customer's users and the documents they get.
	//
	// Copied from the customer's `notification_recipients` when the order was created without its own. `null` unless included.
	EmailRecipients *List[EmailRecipient] `json:"email_recipients" expandable:"true"`
	// When the order is due to ship: what was promised or pinned, the resulting ship-by date, and which rule decided it.
	Commitment *SalesOrderCommitment `json:"commitment" validate:"required"`
	// The order's lines. `null` unless included.
	Lines *List[SalesOrderLine] `json:"lines" expandable:"true"`
	// Number of lines on the order.
	LineCount int32 `json:"line_count"`
	// Derived monetary totals and per-stage fulfillment progress. `null` unless included.
	Totals *SalesOrderTotals `json:"totals" expandable:"true"`
	// The saved address the order bills to, as it reads now. `null` unless included.
	BillToAddress *Address `json:"bill_to_address" expandable:"true" visibility:"internal"`
	// The saved address the order ships to, as it reads now. `null` unless included.
	ShipToAddress *Address `json:"ship_to_address" expandable:"true" visibility:"internal"`
	// The order's email recipients' addresses, by document. `null` unless included.
	Contacts *OrderContact `json:"contacts" expandable:"true" visibility:"internal"`
	// Lightweight references to the records produced from this order: its pick, production work, shipments and invoices.
	//
	// Retrieve a record by its ID for full details.
	Related *SalesOrderRelated `json:"related"`
	// When the order was issued.
	IssuedAt *time.Time `json:"issued_at"`
	// When the order became fulfilled or was canceled.
	FulfilledAt *time.Time `json:"fulfilled_at"`
	// When the first shipment against the order went out.
	FirstShippedAt *time.Time `json:"first_shipped_at"`
	// The date a draft quoted to the customer expires, if one was set.
	ExpiresOn *timeutil.Date `json:"expires_on"`
	// Key-value pairs you attach to the order for your own references, such as its ID in another system.
	//
	// Your own: the other party to the order keeps its own metadata and never sees yours.
	Metadata map[string]string `json:"metadata"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleCustomerPurchaseOrderNumber = "PO-12345"
var sampleSalesOrderInstructions = "Pack in double-wall cartons."

var SampleSalesOrderTotals = &SalesOrderTotals{
	Object:   constants.ObjectTypeSalesOrderTotals,
	Ordered:  "1234.56",
	Picked:   SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: "617.280000000000000000000000000000", Completion: 0.5},
	Packed:   SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: "308.640000000000000000000000000000", Completion: 0.25},
	Invoiced: SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: "0.000000000000000000000000000000", Completion: 0},
}

var (
	sampleShipByOn          = timeutil.NewDate(2026, time.August, 20)
	samplePromisedLeadTime  = &Quantity{Value: "12", Unit: SampleDayUnit}
	samplePromisedLTSource  = constants.PromisedLeadTimeSourceCustomer
	sampleCommitmentFirmnes = constants.CommitmentFirmnessSoft
)

// SampleSalesOrderCommitment is an order promised on its customer's lead time.
var SampleSalesOrderCommitment = &SalesOrderCommitment{
	Firmness:               &sampleCommitmentFirmnes,
	ShipByOn:               &sampleShipByOn,
	PromisedLeadTime:       samplePromisedLeadTime,
	PromisedLeadTimeSource: &samplePromisedLTSource,
}

var SampleSalesOrder = &SalesOrder{
	ID:                          SampleSalesOrderID,
	Object:                      constants.ObjectTypeSalesOrder,
	Number:                      SampleSalesOrderNumber,
	Status:                      constants.SalesOrderStatusIssued,
	Currency:                    constants.CurrencyUSD,
	CustomerPurchaseOrderNumber: &sampleCustomerPurchaseOrderNumber,
	Instructions:                &sampleSalesOrderInstructions,
	PaymentStatus:               constants.SalesOrderPaymentStatusUnpaid,
	AcknowledgmentStatus:        constants.AcknowledgmentStatusNotSent,
	Customer:                    SampleCustomer,
	SalesRep:                    SampleAccountUser,
	CreatedBy:                   SampleCreatedBy,
	BillTo:                      SampleAddressSnapshot,
	ShipTo:                      SampleAddressSnapshot,
	Freight: &SalesOrderFreight{
		Carrier:        SampleCarrier,
		ServiceLevel:   SampleServiceLevel,
		CarrierBilling: &CarrierBilling{Type: constants.CarrierBillingTypeSender},
	},
	PaymentTerm:     SamplePaymentTerm,
	ShippingTerm:    SampleShippingTerm,
	OrderDiscount:   SampleOrderDiscount,
	EmailRecipients: NewList([]EmailRecipient{*SampleEmailRecipient}, PageInfo{}),
	Commitment:      SampleSalesOrderCommitment,
	Lines:           NewList([]SalesOrderLine{*SampleSalesOrderLine}, PageInfo{}),
	LineCount:       1,
	Totals:          SampleSalesOrderTotals,
	Related:         &SalesOrderRelated{Object: constants.ObjectTypeSalesOrderRelated},
	IssuedAt:        new(timeutil.TimestampToTime(sampleCreatedAtTimestamp)),
	Metadata:        map[string]string{"edi_po": "81078093"},
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SalesOrder) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSalesOrder)
}
