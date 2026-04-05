package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleInvoiceID = "iv_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleInvoiceLineID = "ivln_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleInvoiceAllocationID = "txal_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleShipmentID = "sh_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleTransactionID = "tx_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleOrderLineID = "soln_01jm4r6700f8nwq3v5hx2d9ktp"

// Shipment is a minimal shipment sub-resource.
type Shipment struct {
	// The unique identifier for the shipment.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipment"`
	// The shipment number.
	Number string `json:"number" validate:"required"`
}

var SampleShipment = &Shipment{
	ID:     SampleShipmentID,
	Object: constants.ObjectTypeShipment,
	Number: "SH-001",
}

// Transaction is a minimal transaction sub-resource.
type Transaction struct {
	// The unique identifier for the transaction.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction"`
}

var SampleTransaction = &Transaction{
	ID:     SampleTransactionID,
	Object: constants.ObjectTypeTransaction,
}

// SalesOrderLine is a minimal sales order line sub-resource.
type SalesOrderLine struct {
	// The unique identifier for the sales order line.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order"`
	// The item associated with this order line.
	Item *Item `json:"item"`
}

var SampleSalesOrderLine = &SalesOrderLine{
	ID:     SampleOrderLineID,
	Object: constants.ObjectTypeSalesOrder,
	Item:   SampleItem,
}

// InvoiceSummary represents a lightweight invoice for list views.
type InvoiceSummary struct {
	// The unique identifier for the invoice.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=invoice_summary"`
	// The invoice number.
	Number string `json:"number" validate:"required"`
	// A note attached to this invoice.
	Note *string `json:"note"`
	// The customer associated with this invoice.
	Customer *Customer `json:"customer" expandable:"true"`
	// The sales order associated with this invoice.
	Order *SalesOrder `json:"order" expandable:"true"`
	// The shipment associated with this invoice.
	Shipment *Shipment `json:"shipment" expandable:"true"`
	// The number of line items in this invoice.
	LineCount int32 `json:"line_count"`
	// The billing address for this invoice.
	BillingAddress *Address `json:"billing_address" expandable:"true"`
	// The priority code for the customer.
	PriorityCode constants.PriorityCode `json:"priority_code" validate:"required"`
	// The payment term for this invoice.
	PaymentTerm *PaymentTerm `json:"payment_term" expandable:"true"`
	// Whether the invoice has been paid in full.
	IsPaidInFull bool `json:"is_paid_in_full"`
	// Whether the invoice has been sent via EDI.
	IsEdiSent bool `json:"is_edi_sent"`
	// Whether the invoice has been sent.
	HasBeenSent bool `json:"has_been_sent"`
	// The total invoiced amount as a decimal string.
	TotalInvoiced string `json:"total_invoiced" validate:"required" format:"decimal"`
	// Whether the customer accepts invoice emails.
	AcceptsInvoiceEmails bool `json:"accepts_invoice_emails"`
	// Whether the customer is EDI enabled.
	CustomerIsEdiEnabled bool `json:"customer_is_edi_enabled"`
	// The timestamp when the invoice was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the invoice was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleInvoiceSummary = &InvoiceSummary{
	ID:     SampleInvoiceID,
	Object: constants.ObjectTypeInvoiceSummary,
	Number: "INV-001",
	Customer: &Customer{
		ID:     SampleCustomerID,
		Object: constants.ObjectTypeCustomer,
		Name:   SampleCustomerName,
		Number: SampleCustomerNumber,
	},
	Order: &SalesOrder{
		ID:     SampleSalesOrderID,
		Object: constants.ObjectTypeSalesOrder,
		Number: "PO-001",
	},
	LineCount:      3,
	BillingAddress: SampleAddress,
	PriorityCode:   constants.PriorityCodeNormal,
	PaymentTerm: &PaymentTerm{
		ID:     SamplePaymentTermID,
		Object: constants.ObjectTypePaymentTerm,
		Name:   SamplePaymentTermName,
	},
	TotalInvoiced:        "1234.560000000000000000000000000000",
	AcceptsInvoiceEmails: true,
	CreatedAt:            timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:            timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*InvoiceSummary) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleInvoiceSummary)
}

// Invoice represents a full invoice with expandable lines and allocations.
type Invoice struct {
	// The unique identifier for the invoice.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=invoice"`
	// The invoice number.
	Number string `json:"number" validate:"required"`
	// A note attached to this invoice.
	Note *string `json:"note"`
	// The sales order associated with this invoice.
	Order *SalesOrder `json:"order" expandable:"true"`
	// The billing address for this invoice.
	BillingAddress *Address `json:"billing_address" expandable:"true"`
	// The shipment associated with this invoice.
	Shipment *Shipment `json:"shipment" expandable:"true"`
	// Whether the invoice has been paid in full.
	IsPaidInFull bool `json:"is_paid_in_full"`
	// Whether the invoice has been overpaid.
	IsOverPaid bool `json:"is_over_paid"`
	// Whether the invoice has been sent via EDI.
	IsEdiSent bool `json:"is_edi_sent"`
	// Whether the invoice has been sent.
	HasBeenSent bool `json:"has_been_sent"`
	// Whether the customer accepts invoice emails.
	AcceptsInvoiceEmails bool `json:"accepts_invoice_emails"`
	// The line items in this invoice.
	Lines *List[InvoiceLine] `json:"lines" expandable:"true"`
	// The allocations against this invoice.
	Allocations *List[InvoiceAllocation] `json:"allocations" expandable:"true"`
	// The timestamp when the invoice was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the invoice was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleInvoice = &Invoice{
	ID:     SampleInvoiceID,
	Object: constants.ObjectTypeInvoice,
	Number: "INV-001",
	Order: &SalesOrder{
		ID:     SampleSalesOrderID,
		Object: constants.ObjectTypeSalesOrder,
		Number: "PO-001",
	},
	BillingAddress: SampleAddress,
	Lines:          NewList([]InvoiceLine{*SampleInvoiceLine}, PageInfo{}),
	Allocations:    NewList([]InvoiceAllocation{*SampleInvoiceAllocation}, PageInfo{}),
	CreatedAt:      timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:      timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Invoice) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleInvoice)
}

// InvoiceLine represents a line item in an invoice.
type InvoiceLine struct {
	// The unique identifier for the invoice line.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=invoice_line"`
	// The quantity for this line.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// The unit price for this line.
	UnitPrice *Rate `json:"unit_price" validate:"required"`
	// The sales order line associated with this invoice line.
	OrderLine *SalesOrderLine `json:"order_line" validate:"required"`
	// The timestamp when the line was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the line was last updated.
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

// InvoiceAllocation represents a transaction allocation against an invoice.
type InvoiceAllocation struct {
	// The unique identifier for the allocation.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=invoice_allocation"`
	// The transaction associated with this allocation.
	Transaction *Transaction `json:"transaction" validate:"required"`
	// The allocated amount.
	Amount *Quantity `json:"amount" validate:"required"`
	// A note about this allocation.
	Note *string `json:"note"`
	// The timestamp when the allocation was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the allocation was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleInvoiceAllocation = &InvoiceAllocation{
	ID:          SampleInvoiceAllocationID,
	Object:      constants.ObjectTypeInvoiceAllocation,
	Transaction: SampleTransaction,
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

// InvoiceForPayment represents an invoice in the customer payment context.
type InvoiceForPayment struct {
	// The unique identifier for the invoice.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=invoice_for_payment"`
	// The invoice number.
	Number string `json:"number" validate:"required"`
	// The customer's purchase order number.
	CustomerPO *string `json:"customer_po"`
	// The customer associated with this invoice.
	Customer *Customer `json:"customer" expandable:"true"`
	// Whether the customer is a parent account.
	IsParentAccount bool `json:"is_parent_account"`
	// The parent account if this is a child account.
	ParentAccount *Account `json:"parent_account" expandable:"true"`
	// Whether the order was prepaid.
	IsPrepaid bool `json:"is_prepaid"`
	// The billing address for this invoice.
	BillingAddress *Address `json:"billing_address" expandable:"true"`
	// The total invoiced amount as a decimal string.
	InvoiceTotal string `json:"invoice_total" validate:"required" format:"decimal"`
	// Whether the invoice has been paid in full.
	IsPaidInFull bool `json:"is_paid_in_full"`
	// The allocations against this invoice.
	Allocations *List[InvoiceAllocation] `json:"allocations" expandable:"true"`
	// The timestamp when the invoice was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the invoice was last updated.
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
