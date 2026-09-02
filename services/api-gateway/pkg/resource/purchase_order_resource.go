package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SamplePurchaseOrderID = "po_3ov2ym1pca8m"
const SampleEmailContactID = "ec_dmyas2bqcm95"
const SamplePurchaseOrderNumber = "PO-001"
const SampleSupplierID = "ac_gwy8tfbc074f"
const SampleSupplierName = "Acme Supplies Inc."
const SampleSupplierNumber = "SUP-001"

// A contact that receives the purchase order email when an order is issued with the `send_email` option.
type EmailContact struct {
	// Email contact ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=email_contact"`
	// Account user whose email address receives order communications.
	AccountUser *AccountUser `json:"account_user" validate:"required"`
}

var SampleEmailContact = &EmailContact{
	ID:          SampleEmailContactID,
	Object:      constants.ObjectTypeEmailContact,
	AccountUser: SampleAccountUser,
}

func (*EmailContact) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleEmailContact)
}

// An order placed with a supplier to purchase materials or products.
//
// The list endpoint returns this same resource as the retrieve endpoint, except that list rows never carry the note or the scheduled date and can only expand the supplier and the lines.
type PurchaseOrder struct {
	// Purchase order ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=purchase_order"`
	// Human-readable identifier for the order.
	//
	// Assigned automatically from a per-account sequence at creation; can be changed via update but must stay unique within the account.
	Number string `json:"number" validate:"required"`
	// Free-form note recorded on the order.
	Note *string `json:"note"`
	// Lifecycle status of the order.
	//
	// - `estimate`: a draft that has not yet been issued to the supplier.
	// - `issued`: the order has been issued to the supplier and is open for receiving.
	// - `fulfilled`: the order is complete and closed.
	Status constants.SalesOrderStatusCode `json:"status" validate:"required"`
	// Priority level for fulfilling the order, relative to other open orders.
	Priority constants.PriorityCode `json:"priority" validate:"required"`
	// Whether the order acknowledgment email has been sent to the supplier.
	//
	// Advances to `sent` when the order is issued with the `send_email` option; otherwise stays `not_sent`.
	AcknowledgmentStatus constants.AcknowledgmentStatus `json:"acknowledgment_status" validate:"required"`
	// Supplier the order is placed with.
	Supplier *Supplier `json:"supplier" expandable:"true"`
	// Address the supplier bills this order to.
	BillToAddress *Address `json:"bill_to_address" expandable:"true"`
	// Address the supplier ships the ordered goods to.
	ShipToAddress *Address `json:"ship_to_address" expandable:"true"`
	// Carrier selection and freight billing for this order.
	Freight *Freight `json:"freight" expandable:"true"`
	// Payment terms agreed with the supplier.
	PaymentTerm *PaymentTerm `json:"payment_term" expandable:"true"`
	// Shipping terms for the order.
	ShippingTerm *ShippingTerm `json:"shipping_term" expandable:"true"`
	// The records produced from this purchase order.
	Related *PurchaseOrderRelated `json:"related"`
	// Line items on the order.
	Lines *List[PurchaseOrderLine] `json:"lines" expandable:"true"`
	// Total number of lines on the order.
	LineCount int32 `json:"line_count"`
	// Contacts that receive the purchase order email when the order is issued with the `send_email` option.
	Contacts *List[EmailContact] `json:"contacts" expandable:"true"`
	// When the order was issued to the supplier.
	//
	// Cleared again if the order is unissued back to `estimate`.
	IssuedAt *time.Time `json:"issued_at"`
	// When the order was closed as fulfilled.
	//
	// Cleared again if the order is re-opened.
	CompletedAt *time.Time `json:"completed_at"`
	// Date the supplier promised delivery for.
	//
	// Set through the `promised_at` field on create and update.
	ScheduledAt *time.Time `json:"scheduled_at"`
	// Created timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var samplePurchaseOrderNote = "Please expedite"

// PurchaseOrderRelated names the records produced from a purchase order.
type PurchaseOrderRelated struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=purchase_order_related"`
	// Receiving order used to receive inventory against this purchase order.
	//
	// Created automatically, with a line per order line, when the order is issued, and deleted again if the order is unissued.
	ReceivingOrder *Record `json:"receiving_order" expandable:"true"`
	// The deliveries booked against this order, oldest first.
	Deliveries *List[Record] `json:"deliveries" expandable:"true"`
}

var SamplePurchaseOrderRelated = &PurchaseOrderRelated{
	Object: constants.ObjectTypePurchaseOrderRelated,
}

func (*PurchaseOrderRelated) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePurchaseOrderRelated)
}

var SamplePurchaseOrder = &PurchaseOrder{
	ID:                   SamplePurchaseOrderID,
	Object:               constants.ObjectTypePurchaseOrder,
	Number:               SamplePurchaseOrderNumber,
	Note:                 &samplePurchaseOrderNote,
	Status:               constants.SalesOrderStatusCodeEstimate,
	Priority:             SamplePriorityCode,
	AcknowledgmentStatus: constants.AcknowledgmentStatusNotSent,
	Supplier:             SampleSupplier,
	BillToAddress:        SampleAddress,
	ShipToAddress:        SampleAddress,
	Freight:              SampleFreight,
	PaymentTerm:          SamplePaymentTerm,
	ShippingTerm:         SampleShippingTerm,
	Lines:                NewList([]PurchaseOrderLine{*SamplePurchaseOrderLine}, PageInfo{}),
	LineCount:            1,
	Contacts:             NewList([]EmailContact{*SampleEmailContact}, PageInfo{}),
	ScheduledAt:          timeutil.TimestampToTimePtr(sampleExpiresAtTimestamp),
	CreatedAt:            timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:            timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PurchaseOrder) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePurchaseOrder)
}
