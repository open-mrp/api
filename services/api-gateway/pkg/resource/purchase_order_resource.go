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

// Supplier represents a supplier sub-resource.
type Supplier struct {
	// The unique identifier for the supplier.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=supplier"`
	// The display name of the supplier.
	Name string `json:"name" validate:"required"`
	// The supplier number.
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

// EmailContact represents an email contact sub-resource.
type EmailContact struct {
	// The unique identifier for the email contact.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=email_contact"`
	// The account user associated with this contact.
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

// PurchaseOrderDetail represents a full purchase order resource.
type PurchaseOrderDetail struct {
	// The unique identifier for the purchase order.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=purchase_order"`
	// The purchase order number.
	Number string `json:"number" validate:"required"`
	// A note attached to this purchase order.
	Note *string `json:"note"`
	// Whether the acknowledgment has been sent.
	IsAcknowledgmentSent bool `json:"is_acknowledgment_sent"`
	// The supplier associated with this order.
	Supplier *Supplier `json:"supplier" expandable:"true"`
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
	// The receiving order associated with this purchase order.
	ReceivingOrder *ReceivingOrder `json:"receiving_order" expandable:"true"`
	// The order lines.
	Lines *List[PurchaseOrderLineDetail] `json:"lines" expandable:"true"`
	// The email contacts for this order.
	Contacts *List[EmailContact] `json:"contacts" expandable:"true"`
	// The timestamp when the order was issued.
	IssuedAt *time.Time `json:"issued_at"`
	// The timestamp when the order was completed.
	CompletedAt *time.Time `json:"completed_at"`
	// The timestamp when the order is scheduled/promised.
	ScheduledAt *time.Time `json:"scheduled_at"`
	// The timestamp when the order was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the order was last updated.
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

// PurchaseOrderSummary represents a lightweight purchase order for list views.
type PurchaseOrderSummary struct {
	// The unique identifier for the purchase order.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=purchase_order"`
	// The purchase order number.
	Number string `json:"number" validate:"required"`
	// The supplier associated with this order.
	Supplier *Supplier `json:"supplier" validate:"required"`
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
