package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleInvoiceID = "iv_m982ezb0fgp7"
const SampleInvoiceLineID = "ivln_q6k84g39xlnk"

// Same allocation row as SampleAllocationEntryID (invoice example embeds that entry).
const SampleInvoiceAllocationID = SampleAllocationEntryID

// An invoice billing a customer for goods shipped against a sales order.
type Invoice struct {
	// Invoice ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=invoice"`
	// Invoice number.
	Number string `json:"number" validate:"required"`
	// Note attached to the invoice.
	Note *string `json:"note"`
	// Customer associated with this invoice.
	Customer *Customer `json:"customer" expandable:"true"`
	// Sales order this invoice bills against.
	Order *SalesOrder `json:"order" expandable:"true"`
	// Shipment whose shipped goods this invoice bills for.
	Shipment *Shipment `json:"shipment" expandable:"true"`
	// Number of line items on the invoice.
	LineCount int32 `json:"line_count"`
	// Address the invoice is billed to.
	BillingAddress *Address `json:"billing_address" expandable:"true"`
	// Priority level carried onto the invoice from the order it bills.
	PriorityCode constants.PriorityCode `json:"priority" validate:"required"`
	// Payment term governing when the invoice is due.
	PaymentTerm *PaymentTerm `json:"payment_term" expandable:"true"`
	// Payment status of the invoice.
	//
	// Reported from the invoice's stored paid-in-full and overpaid flags, so marking an invoice paid through Update Invoice changes this value even when no payment has been allocated.
	//
	// - `unpaid`: the invoice is not marked paid in full, which includes invoices carrying partial payments.
	// - `paid`: the invoice is marked paid in full.
	// - `overpaid`: the payments applied to the invoice exceed the invoiced amount.
	// - `partially_paid`: not currently returned; an invoice carrying a partial payment reports `unpaid`.
	PaymentStatus constants.InvoicePaymentStatus `json:"payment_status" validate:"required"`
	// Whether the invoice has been transmitted to the customer via EDI.
	//
	// Nothing in the platform sets this flag; it is recorded through Update Invoice once the invoice has been transmitted elsewhere.
	IsEdiSent bool `json:"is_edi_sent"`
	// Whether the invoice has been sent to the customer.
	//
	// Set automatically when the invoice is emailed through Email Record, and can also be set directly through Update Invoice.
	HasBeenSent bool `json:"has_been_sent"`
	// Total amount billed by this invoice.
	//
	// The sum across the invoice's lines of the billed quantity multiplied by the unit price on the sales order line.
	TotalInvoiced string `json:"total_invoiced" validate:"required" format:"decimal"`
	// Whether the sales order behind this invoice has at least one contact set to receive invoice emails.
	//
	// These contacts are the recipients used by Email Record. When no contact is configured, emailing the invoice marks it sent without delivering anything.
	AcceptsInvoiceEmails bool `json:"accepts_invoice_emails"`
	// Whether the billed customer is configured to exchange documents via EDI.
	CustomerIsEdiEnabled bool `json:"customer_is_edi_enabled"`
	// Line items in this invoice.
	Lines *List[InvoiceLine] `json:"lines" expandable:"true"`
	// Transaction allocations applied against this invoice.
	//
	// These are the payments and credits recorded against the invoice; recording a settlement refreshes `payment_status` from them, while Update Invoice can set that status directly.
	Allocations *List[InvoiceAllocation] `json:"allocations" expandable:"true"`
	// Records this invoice bills against — its order and shipment.
	Related *InvoiceRelated `json:"related"`
	// Timestamp when the invoice was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when the invoice was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Groups the records an invoice bills against: the order it belongs to and the shipment that
// raised it. Returned only when at least one member has been expanded.
type InvoiceRelated struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=invoice_related"`
	// The sales order this invoice bills against.
	SalesOrder *Record `json:"sales_order" expandable:"true"`
	// The shipment whose shipping raised this invoice.
	Shipment *Record `json:"shipment" expandable:"true"`
}

var sampleInvoiceNote = "Net 30 terms; contact accounts payable for remittance details."

var SampleInvoice = &Invoice{
	ID:       SampleInvoiceID,
	Object:   constants.ObjectTypeInvoice,
	Number:   "INV-001",
	Note:     &sampleInvoiceNote,
	Customer: SampleCustomer,
	Order:    SampleSalesOrder,
	Shipment: &Shipment{
		ID:        SampleShipmentID,
		Object:    constants.ObjectTypeShipment,
		Number:    SampleShipmentNumber,
		Status:    constants.ShipmentStatusShipped,
		Priority:  SamplePriorityCode,
		CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
		UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
	},
	LineCount:            3,
	BillingAddress:       SampleAddress,
	PriorityCode:         constants.PriorityCodeNormal,
	PaymentTerm:          SamplePaymentTerm,
	PaymentStatus:        constants.InvoicePaymentStatusUnpaid,
	IsEdiSent:            false,
	HasBeenSent:          true,
	TotalInvoiced:        "1234.56",
	AcceptsInvoiceEmails: true,
	CustomerIsEdiEnabled: false,
	Lines:                NewList([]InvoiceLine{*SampleInvoiceLine}, PageInfo{}),
	Allocations:          NewList([]InvoiceAllocation{*SampleInvoiceAllocation}, PageInfo{}),
	Related:              &InvoiceRelated{Object: constants.ObjectTypeInvoiceRelated},
	CreatedAt:            timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:            timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Invoice) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleInvoice)
}

// Line item in an invoice.
type InvoiceLine struct {
	// Invoice line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=invoice_line"`
	// Quantity billed on this line.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// Price per unit billed on this line, carried over from the sales order line.
	UnitPrice *Rate `json:"unit_price" validate:"required"`
	// Sales order line this invoice line bills against.
	//
	// Expand with `lines.order_line` for the sold `product_sku` and the ordered quantities, and with `lines.order_line.product` for the product itself. To show only the SKU, read `item` below instead — it needs no expansion.
	OrderLine *SalesOrderLine `json:"order_line" expandable:"true"`
	// What this line bills, as recorded on the originating sales order line.
	Item *Item `json:"item" expandable:"true"`
	// Timestamp when the line was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when the line was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleInvoiceLine = &InvoiceLine{
	ID:     SampleInvoiceLineID,
	Object: constants.ObjectTypeInvoiceLine,
	Quantity: &Quantity{
		ID:           SampleQuantityID,
		Object:       constants.ObjectTypeQuantity,
		Value:        "100.000000000000000000000000000000",
		DisplayValue: "100 lb",
		Unit:         newSampleUnit(SampleUnitName, SampleUnitAbbreviation, constants.UnitTypeMass),
	},
	UnitPrice: SampleRate,
	OrderLine: SampleSalesOrderLine,
	Item:      SampleItem,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*InvoiceLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleInvoiceLine)
}

// A portion of a transaction applied against an invoice.
//
// Allocations connect transactions (payments, rebates, adjustments, and credit memos) to the invoices they pay down. Recording a settlement refreshes the invoice's paid-in-full and overpaid state — and so its `payment_status` — from every allocation against it, but that state can also be set directly through Update Invoice.
type InvoiceAllocation struct {
	// Allocation ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=invoice_allocation"`
	// Transaction whose amount is being applied to the invoice.
	Transaction *TransactionDetail `json:"transaction" expandable:"true"`
	// Portion of the transaction applied to the invoice by this allocation.
	Amount *Quantity `json:"amount" validate:"required"`
	// Note about this allocation.
	Note *string `json:"note"`
	// Timestamp when the allocation was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when the allocation was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleInvoiceAllocationNote = "Partial payment applied from customer check #4021."

var SampleInvoiceAllocation = &InvoiceAllocation{
	ID:          SampleInvoiceAllocationID,
	Object:      constants.ObjectTypeInvoiceAllocation,
	Transaction: SampleTransactionDetail,
	Note:        &sampleInvoiceAllocationNote,
	Amount: &Quantity{
		ID:           SampleQuantityID,
		Object:       constants.ObjectTypeQuantity,
		Value:        "500.000000000000000000000000000000",
		DisplayValue: "$500.00",
		Unit:         newSampleUnit("US Dollar", "$", constants.UnitTypeCurrency),
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*InvoiceAllocation) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleInvoiceAllocation)
}

// A payment-oriented view of an invoice, as returned by List Customer Invoices.
//
// Carries the fields needed to apply a customer payment: the invoice total, the allocations already applied, and the billing relationship of the customer being charged. Only invoices that still owe a balance are represented.
type InvoiceForPayment struct {
	// Invoice ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=invoice_for_payment"`
	// Invoice number.
	Number string `json:"number" validate:"required"`
	// Purchase order number the customer supplied for the underlying order.
	CustomerPO *string `json:"customer_po"`
	// Customer associated with this invoice.
	Customer *Customer `json:"customer" expandable:"true"`
	// Whether the billed customer is a child of a parent account.
	//
	// When `true`, `parent_account` identifies that parent.
	IsParentAccount bool `json:"is_parent_account"`
	// The customer's parent account, when the billed customer is a child account.
	ParentAccount *Account `json:"parent_account" expandable:"true"`
	// Whether the billed customer's payment term is prepaid.
	IsPrepaid bool `json:"is_prepaid"`
	// Address the invoice is billed to.
	BillingAddress *Address `json:"billing_address" expandable:"true"`
	// Total amount billed by this invoice.
	InvoiceTotal string `json:"invoice_total" validate:"required" format:"decimal"`
	// Whether the invoice has been paid in full.
	//
	// Always `false` here, because only invoices that still owe a balance are listed.
	IsPaidInFull bool `json:"is_paid_in_full"`
	// Transaction allocations already applied against this invoice.
	Allocations *List[InvoiceAllocation] `json:"allocations" expandable:"true"`
	// Timestamp when the invoice was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when the invoice was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleInvoiceForPaymentCustomerPO = "PO-88231"

var SampleInvoiceForPayment = &InvoiceForPayment{
	ID:              SampleInvoiceID,
	Object:          constants.ObjectTypeInvoiceForPayment,
	Number:          "INV-001",
	CustomerPO:      &sampleInvoiceForPaymentCustomerPO,
	Customer:        SampleCustomer,
	IsParentAccount: true,
	ParentAccount:   SampleAccount,
	IsPrepaid:       false,
	BillingAddress:  SampleAddress,
	InvoiceTotal:    "1234.56",
	IsPaidInFull:    false,
	Allocations:     NewList([]InvoiceAllocation{*SampleInvoiceAllocation}, PageInfo{}),
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*InvoiceForPayment) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleInvoiceForPayment)
}
