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

// Invoice resource.
type Invoice struct {
	// Invoice ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=invoice"`
	// Invoice number.
	Number string `json:"number" validate:"required"`
	// Note attached to the invoice.
	Note *string `json:"note"`
	// Customer associated with this invoice. Expandable via include[]=customer.
	Customer *Customer `json:"customer" expandable:"true"`
	// Sales order associated with this invoice. Expandable via include[]=order.
	Order *SalesOrder `json:"order" expandable:"true"`
	// Shipment associated with this invoice. Expandable via include[]=shipment.
	Shipment *Shipment `json:"shipment" expandable:"true"`
	// Number of line items.
	LineCount int32 `json:"line_count"`
	// Billing address. Expandable via include[]=billing_address.
	BillingAddress *Address `json:"billing_address" expandable:"true"`
	// Customer priority code.
	PriorityCode constants.PriorityCode `json:"priority" validate:"required"`
	// Payment term. Expandable via include[]=payment_term.
	PaymentTerm *PaymentTerm `json:"payment_term" expandable:"true"`
	// Payment status of the invoice.
	PaymentStatus constants.InvoicePaymentStatus `json:"payment_status" validate:"required"`
	// Whether the invoice has been sent via EDI.
	IsEdiSent bool `json:"is_edi_sent"`
	// Whether the invoice has been sent.
	HasBeenSent bool `json:"has_been_sent"`
	// Total invoiced amount as a decimal string.
	TotalInvoiced string `json:"total_invoiced" validate:"required" format:"decimal"`
	// Whether the customer accepts invoice emails.
	AcceptsInvoiceEmails bool `json:"accepts_invoice_emails"`
	// Whether the customer is EDI enabled.
	CustomerIsEdiEnabled bool `json:"customer_is_edi_enabled"`
	// Line items in this invoice. Expandable via include[]=lines.
	Lines *List[InvoiceLine] `json:"lines" expandable:"true"`
	// Allocations against this invoice. Expandable via include[]=allocations.
	Allocations *List[InvoiceAllocation] `json:"allocations" expandable:"true"`
	// Timestamp when the invoice was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when the invoice was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleInvoice = &Invoice{
	ID:                   SampleInvoiceID,
	Object:               constants.ObjectTypeInvoice,
	Number:               "INV-001",
	LineCount:            3,
	BillingAddress:       SampleAddress,
	PriorityCode:         constants.PriorityCodeNormal,
	PaymentStatus:        constants.InvoicePaymentStatusUnpaid,
	TotalInvoiced:        "1234.560000000000000000000000000000",
	AcceptsInvoiceEmails: true,
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
	// Sales order line associated with this invoice line. Expandable via include[]=lines.order_line.
	OrderLine *SalesOrderLine `json:"order_line" expandable:"true"`
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
		Unit: &Unit{
			ID:           SampleUnitID,
			Object:       constants.ObjectTypeUnit,
			Name:         SampleUnitName,
			Abbreviation: SampleUnitAbbreviation,
			Type:         constants.UnitTypeMass,
		},
	},
	UnitPrice: SampleRate,
	OrderLine: SampleSalesOrderLine,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*InvoiceLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleInvoiceLine)
}

// Transaction allocation against an invoice.
type InvoiceAllocation struct {
	// Allocation ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=invoice_allocation"`
	// Transaction associated with this allocation. Expandable via include[]=allocations.transaction.
	Transaction *TransactionDetail `json:"transaction" expandable:"true"`
	// Allocated amount.
	Amount *Quantity `json:"amount" validate:"required"`
	// Note about this allocation.
	Note *string `json:"note"`
	// Timestamp when the allocation was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when the allocation was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleInvoiceAllocation = &InvoiceAllocation{
	ID:          SampleInvoiceAllocationID,
	Object:      constants.ObjectTypeInvoiceAllocation,
	Transaction: SampleTransactionDetail,
	Amount: &Quantity{
		ID:           SampleQuantityID,
		Object:       constants.ObjectTypeQuantity,
		Value:        "500.000000000000000000000000000000",
		DisplayValue: "$500.00",
		Unit: &Unit{
			ID:           SampleUnitID,
			Object:       constants.ObjectTypeUnit,
			Name:         "US Dollar",
			Abbreviation: "$",
			Type:         constants.UnitTypeCurrency,
		},
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*InvoiceAllocation) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleInvoiceAllocation)
}

// Invoice in the customer payment context.
type InvoiceForPayment struct {
	// Invoice ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=invoice_for_payment"`
	// Invoice number.
	Number string `json:"number" validate:"required"`
	// Customer's purchase order number.
	CustomerPO *string `json:"customer_po"`
	// Customer associated with this invoice.
	Customer *Customer `json:"customer" expandable:"true"`
	// Whether the customer is a parent account.
	IsParentAccount bool `json:"is_parent_account"`
	// Parent account if this is a child account.
	ParentAccount *Account `json:"parent_account" expandable:"true"`
	// Whether the order was prepaid.
	IsPrepaid bool `json:"is_prepaid"`
	// Billing address.
	BillingAddress *Address `json:"billing_address" expandable:"true"`
	// Total invoiced amount as a decimal string.
	InvoiceTotal string `json:"invoice_total" validate:"required" format:"decimal"`
	// Whether the invoice has been paid in full.
	IsPaidInFull bool `json:"is_paid_in_full"`
	// Allocations against this invoice.
	Allocations *List[InvoiceAllocation] `json:"allocations" expandable:"true"`
	// Timestamp when the invoice was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when the invoice was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleInvoiceForPayment = &InvoiceForPayment{
	ID:     SampleInvoiceID,
	Object: constants.ObjectTypeInvoiceForPayment,
	Number: "INV-001",
	Customer: &Customer{
		ID:     SampleCustomerID,
		Object: constants.ObjectTypeCustomer,
		Name:   SampleCustomerName,
		Number: SampleCustomerNumber,
	},
	InvoiceTotal: "1234.560000000000000000000000000000",
	Allocations:  NewList([]InvoiceAllocation{*SampleInvoiceAllocation}, PageInfo{}),
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:    timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*InvoiceForPayment) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleInvoiceForPayment)
}
