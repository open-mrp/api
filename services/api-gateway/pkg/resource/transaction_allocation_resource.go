package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAllocationEntryID = "txal_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleOpenCreditEntryID = "txn_01jm4r6700f8nwq3v5hx2d9ktp" // #nosec G101 -- sample ID, not a credential

// AllocationEntry represents a transaction allocation entry in list views.
type AllocationEntry struct {
	// The unique identifier for the allocation.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=allocation_entry"`
	// The allocated amount as a decimal string.
	Amount string `json:"amount" validate:"required"`
	// A human-readable formatted amount (e.g. "$500.00").
	DisplayAmount string `json:"display_amount" validate:"required"`
	// The customer associated with this allocation.
	Customer *AllocationCustomer `json:"customer" validate:"required"`
	// The transaction associated with this allocation.
	Transaction *AllocationTransaction `json:"transaction" validate:"required"`
	// The invoice associated with this allocation.
	Invoice *AllocationInvoice `json:"invoice" validate:"required"`
	// A note about this allocation.
	Note *string `json:"note"`
	// The timestamp when the allocation was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
}

// AllocationCustomer is a minimal customer sub-resource for allocation entries.
type AllocationCustomer struct {
	// The customer display name.
	Name string `json:"name" validate:"required"`
	// The customer number.
	Number *string `json:"number"`
}

// AllocationTransaction is a minimal transaction sub-resource for allocation entries.
type AllocationTransaction struct {
	// The unique identifier for the transaction.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction"`
	// The type of transaction (e.g. "payment", "credit").
	Type string `json:"type" validate:"required"`
	// The transaction method (e.g. "check", "wire").
	Method *string `json:"method"`
	// The adjustment type, if applicable.
	AdjustmentType *string `json:"adjustment_type"`
}

// AllocationInvoice is a minimal invoice sub-resource for allocation entries.
type AllocationInvoice struct {
	// The unique identifier for the invoice.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=invoice_summary"`
	// The invoice number.
	Number string `json:"number" validate:"required"`
}

var SampleAllocationEntry = &AllocationEntry{
	ID:            SampleAllocationEntryID,
	Object:        constants.ObjectTypeAllocationEntry,
	Amount:        "500.000000000000000000000000000000",
	DisplayAmount: "$500.00",
	Customer: &AllocationCustomer{
		Name: "Acme Corp",
	},
	Transaction: &AllocationTransaction{
		ID:     SampleTransactionID,
		Object: constants.ObjectTypeTransaction,
		Type:   "payment",
	},
	Invoice: &AllocationInvoice{
		ID:     SampleInvoiceID,
		Object: constants.ObjectTypeInvoiceSummary,
		Number: "INV-001",
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*AllocationEntry) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAllocationEntry)
}

// OpenCreditEntry represents an open (not fully allocated) credit transaction.
type OpenCreditEntry struct {
	// The unique identifier for the transaction.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=open_credit_entry"`
	// The transaction number.
	Number string `json:"number" validate:"required"`
	// The original transaction amount as a decimal string.
	OriginalAmount string `json:"original_amount" validate:"required"`
	// The total amount already allocated as a decimal string.
	AllocatedAmount string `json:"allocated_amount" validate:"required"`
	// The remaining unallocated amount as a decimal string.
	LeftoverAmount string `json:"leftover_amount" validate:"required"`
	// The customer associated with this transaction.
	Customer *AllocationCustomer `json:"customer" validate:"required"`
	// The type of transaction.
	TransactionType string `json:"transaction_type" validate:"required"`
	// The transaction method.
	TransactionMethod *string `json:"transaction_method"`
	// The adjustment type, if applicable.
	AdjustmentType *string `json:"adjustment_type"`
	// The responsible user's name.
	ResponsibleUserName *string `json:"responsible_user_name"`
	// A note about this transaction.
	Note *string `json:"note"`
	// The Stripe payment ID, if applicable.
	StripePaymentID *string `json:"stripe_payment_id"`
	// The allocations against invoices for this transaction.
	InvoiceAllocations []InvoiceAllocationEntry `json:"invoice_allocations"`
	// The timestamp when the transaction was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
}

// InvoiceAllocationEntry represents an allocation of a credit against an invoice.
type InvoiceAllocationEntry struct {
	// The invoice number.
	InvoiceNumber string `json:"invoice_number" validate:"required"`
	// The allocated amount as a decimal string.
	Amount string `json:"amount" validate:"required"`
}

var SampleOpenCreditEntry = &OpenCreditEntry{
	ID:              SampleOpenCreditEntryID,
	Object:          constants.ObjectTypeOpenCreditEntry,
	Number:          "TXN-001",
	OriginalAmount:  "1000.000000000000000000000000000000",
	AllocatedAmount: "500.000000000000000000000000000000",
	LeftoverAmount:  "500.000000000000000000000000000000",
	Customer: &AllocationCustomer{
		Name: "Acme Corp",
	},
	TransactionType: "payment",
	InvoiceAllocations: []InvoiceAllocationEntry{
		{InvoiceNumber: "INV-001", Amount: "500.000000000000000000000000000000"},
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*OpenCreditEntry) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOpenCreditEntry)
}
