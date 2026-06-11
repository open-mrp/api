package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePurchaseOrderID = "po_0169aa3a722b081b117ac0e44f"
const SampleEmailContactID = "ec_01758010d10f5629ce3880a4ab"
const SamplePurchaseOrderNumber = "PO-001"
const SampleSupplierID = "ac_0177902104bccac5fbb173cd96"
const SampleSupplierName = "Acme Supplies Inc."
const SampleSupplierNumber = "SUP-001"

// Supplier sub-resource.
type Supplier struct {
	// Supplier ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=supplier"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Supplier number.
	Number string `json:"number" validate:"required"`
}

var SampleSupplier = &Supplier{
	ID:     SampleSupplierID,
	Object: constants.ObjectTypeSupplier,
	Name:   SampleSupplierName,
	Number: SampleSupplierNumber,
}

func (*Supplier) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSupplier)
}

// Email contact sub-resource.
type EmailContact struct {
	// Email contact ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=email_contact"`
	// Account user whose email address receives order communications.
	AccountUser *AccountUser `json:"account_user" validate:"required"`
}

var SampleEmailContact = &EmailContact{
	ID:     SampleEmailContactID,
	Object: constants.ObjectTypeEmailContact,
	AccountUser: &AccountUser{
		ID:     SampleAccountUserID,
		Object: constants.ObjectTypeAccountUser,
	},
}

func (*EmailContact) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleEmailContact)
}

// Full purchase order resource.
type PurchaseOrder struct {
	// Purchase order ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=purchase_order"`
	// Purchase order number.
	Number string `json:"number" validate:"required"`
	// Order note.
	Note *string `json:"note"`
	// Lifecycle status of the order.
	//
	// - `estimate`: a draft that has not yet been issued to the supplier.
	// - `issued`: the order has been issued to the supplier and is open for receiving.
	// - `fulfilled`: the order is complete and closed.
	Status constants.SalesOrderStatusCode `json:"status" validate:"required"`
	// Priority level for fulfilling the order.
	//
	// - `low`
	// - `normal`
	// - `high`
	Priority constants.PriorityCode `json:"priority" validate:"required"`
	// Whether an acknowledgment has been sent to the supplier.
	//
	// - `not_sent`: no acknowledgment has been sent.
	// - `sent`: the acknowledgment has been sent.
	AcknowledgmentStatus constants.AcknowledgmentStatus `json:"acknowledgment_status" validate:"required"`
	// Supplier.
	Supplier *Supplier `json:"supplier" expandable:"true"`
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
	// Receiving order.
	ReceivingOrder *ReceivingOrder `json:"receiving_order" expandable:"true"`
	// Order lines.
	Lines *List[PurchaseOrderLine] `json:"lines" expandable:"true"`
	// Total number of lines on the order, independent of whether `lines` is expanded.
	LineCount int32 `json:"line_count"`
	// Supplier-side contacts that order communications are sent to.
	Contacts *List[EmailContact] `json:"contacts" expandable:"true"`
	// When the order was issued to the supplier.
	//
	// Null until status reaches `issued`.
	IssuedAt *time.Time `json:"issued_at"`
	// When the order was completed.
	//
	// Null until status reaches `fulfilled`.
	CompletedAt *time.Time `json:"completed_at"`
	// Promised or scheduled date for the order, if one has been set.
	ScheduledAt *time.Time `json:"scheduled_at"`
	// Created timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var samplePurchaseOrderNote = "Please expedite"

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
	Lines:                NewList([]PurchaseOrderLine{*SamplePurchaseOrderLine}, PageInfo{}),
	LineCount:            1,
	CreatedAt:            timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:            timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PurchaseOrder) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePurchaseOrder)
}
