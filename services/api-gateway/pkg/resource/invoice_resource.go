package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleInvoiceID = "iv_018b5949ada8abca36358bbea9"
const SampleInvoiceLineID = "ivln_01999b9fa867e396ec797aab95"

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
	// Derived from the invoice's paid-in-full and overpaid flags rather than computed directly from its allocations.
	//
	// - `overpaid`: the applied allocations exceed the invoiced amount.
	// - `partially_paid`: reserved for a future signal and not currently emitted.
	PaymentStatus constants.InvoicePaymentStatus `json:"payment_status" validate:"required"`
	// Whether the invoice has been transmitted to the customer via EDI.
	IsEdiSent bool `json:"is_edi_sent"`
	// Whether the invoice has been sent to the customer.
	HasBeenSent bool `json:"has_been_sent"`
	// Total invoiced amount as a decimal string.
	TotalInvoiced string `json:"total_invoiced" validate:"required" format:"decimal"`
	// Whether the billed customer is configured to receive invoices by email.
	AcceptsInvoiceEmails bool `json:"accepts_invoice_emails"`
	// Whether the billed customer is configured to exchange documents via EDI.
	CustomerIsEdiEnabled bool `json:"customer_is_edi_enabled"`
	// Line items in this invoice.
	Lines *List[InvoiceLine] `json:"lines" expandable:"true"`
	// Transaction allocations applied against this invoice.
	//
	// The invoice's paid-in-full / overpaid state (and thus `payment_status`) is tracked separately and is not recomputed from these allocations.
	Allocations *List[InvoiceAllocation] `json:"allocations" expandable:"true"`
	// Timestamp when the invoice was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when the invoice was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
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
	TotalInvoiced:        "1234.560000000000000000000000000000",
	AcceptsInvoiceEmails: true,
	CustomerIsEdiEnabled: false,
	Lines:                NewList([]InvoiceLine{*SampleInvoiceLine}, PageInfo{}),
	Allocations:          NewList([]InvoiceAllocation{*SampleInvoiceAllocation}, PageInfo{}),
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
	// Quantity for this line.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// Unit price for this line.
	UnitPrice *Rate `json:"unit_price" validate:"required"`
	// Sales order line this invoice line bills against.
	OrderLine *SalesOrderLine `json:"order_line" expandable:"true"`
	// The item being invoiced, taken from the order line's item.
	//
	// Populated inline whenever invoice lines are included; it is not separately expandable.
	Item *Item `json:"item"`
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
// Allocations connect transactions (payments, rebates, adjustments, and credit memos) to the invoices they pay down. The invoice's paid-in-full / overpaid state (and thus `payment_status`) is tracked separately and is not recomputed from these allocations.
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
// Carries the fields needed to apply customer payments: the invoice total, paid-in-full state, and the allocations already applied.
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
	// Total invoiced amount as a decimal string.
	InvoiceTotal string `json:"invoice_total" validate:"required" format:"decimal"`
	// Whether the invoice has been paid in full.
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
	ID:         SampleInvoiceID,
	Object:     constants.ObjectTypeInvoiceForPayment,
	Number:     "INV-001",
	CustomerPO: &sampleInvoiceForPaymentCustomerPO,
	Customer: &Customer{
		ID:     SampleCustomerID,
		Object: constants.ObjectTypeCustomer,
		Name:   SampleCustomerName,
		Number: SampleCustomerNumber,
	},
	IsParentAccount: true,
	ParentAccount:   SampleAccount,
	IsPrepaid:       false,
	BillingAddress:  SampleAddress,
	InvoiceTotal:    "1234.560000000000000000000000000000",
	IsPaidInFull:    false,
	Allocations:     NewList([]InvoiceAllocation{*SampleInvoiceAllocation}, PageInfo{}),
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*InvoiceForPayment) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleInvoiceForPayment)
}
