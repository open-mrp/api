package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePurchaseOrderDetailID = "po_01jm4r6700f8nwq3v5hx2d9ktp"
const SamplePurchaseOrderNumber = "PO-001"
const SampleSupplierID = "ac_02kn5s7811g9qwce7cizr4e0mq"
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
	// Account user.
	AccountUser *AccountUser `json:"account_user" validate:"required"`
}

var SampleEmailContact = &EmailContact{
	ID:     "ec_01jm4r6700f8nwq3v5hx2d9ktp",
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
type PurchaseOrderDetail struct {
	// Purchase order ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=purchase_order"`
	// Purchase order number.
	Number string `json:"number" validate:"required"`
	// Order note.
	Note *string `json:"note"`
	// Whether the acknowledgment has been sent.
	IsAcknowledgmentSent bool `json:"is_acknowledgment_sent"`
	// Supplier.
	Supplier *Supplier `json:"supplier" expandable:"true"`
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
	// Receiving order.
	ReceivingOrder *ReceivingOrder `json:"receiving_order" expandable:"true"`
	// Order lines.
	Lines *List[PurchaseOrderLineDetail] `json:"lines" expandable:"true"`
	// Email contacts.
	Contacts *List[EmailContact] `json:"contacts" expandable:"true"`
	// Issued timestamp.
	IssuedAt *time.Time `json:"issued_at"`
	// Completed timestamp.
	CompletedAt *time.Time `json:"completed_at"`
	// Scheduled/promised timestamp.
	ScheduledAt *time.Time `json:"scheduled_at"`
	// Created timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var samplePurchaseOrderNote = "Please expedite"

var SamplePurchaseOrderDetail = &PurchaseOrderDetail{
	ID:                   SamplePurchaseOrderDetailID,
	Object:               constants.ObjectTypePurchaseOrder,
	Number:               SamplePurchaseOrderNumber,
	Note:                 &samplePurchaseOrderNote,
	IsAcknowledgmentSent: false,
	Supplier:             SampleSupplier,
	BillToAddress:        SampleAddress,
	ShipToAddress:        SampleAddress,
	Status:               SampleSalesOrderStatusDetail,
	Type:                 SampleSalesOrderType,
	Priority:             SamplePriority,
	Lines:                NewList([]PurchaseOrderLineDetail{*SamplePurchaseOrderLineDetail}, PageInfo{}),
	CreatedAt:            timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:            timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PurchaseOrderDetail) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePurchaseOrderDetail)
}

// Lightweight purchase order for list views.
type PurchaseOrderSummary struct {
	// Purchase order ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=purchase_order"`
	// Purchase order number.
	Number string `json:"number" validate:"required"`
	// Supplier.
	Supplier *Supplier `json:"supplier" validate:"required"`
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
	// Created timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SamplePurchaseOrderSummary = &PurchaseOrderSummary{
	ID:     SamplePurchaseOrderDetailID,
	Object: constants.ObjectTypePurchaseOrder,
	Number: SamplePurchaseOrderNumber,
	Supplier: &Supplier{
		ID:     SampleSupplierID,
		Object: constants.ObjectTypeSupplier,
		Name:   SampleSupplierName,
		Number: SampleSupplierNumber,
	},
	Status:    SampleSalesOrderStatusDetail,
	Type:      SampleSalesOrderType,
	Priority:  SamplePriority,
	LineCount: 3,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PurchaseOrderSummary) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePurchaseOrderSummary)
}
