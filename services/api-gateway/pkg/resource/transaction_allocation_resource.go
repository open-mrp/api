package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAllocationEntryID = "txal_016cc92c2d9c0b12801e3160e0"
const SampleOpenCreditEntryID = "txn_0102a8419c19035a1062bfd5b1" // #nosec G101 -- sample ID, not a credential

// An application of part of a transaction's amount against a specific invoice, as returned in list views.
type AllocationEntry struct {
	// Allocation ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=allocation_entry"`
	// Allocated amount as a decimal string.
	Amount string `json:"amount" validate:"required"`
	// Human-readable formatted amount (e.g. "$500.00").
	DisplayAmount string `json:"display_amount" validate:"required"`
	// Customer associated with this allocation.
	Customer *AllocationCustomer `json:"customer" validate:"required"`
	// Transaction associated with this allocation.
	Transaction *AllocationTransaction `json:"transaction" validate:"required"`
	// Invoice associated with this allocation.
	Invoice *AllocationInvoice `json:"invoice" validate:"required"`
	// Note about this allocation.
	Note *string `json:"note"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
}

// Minimal customer sub-resource for allocation entries and open-credit entries.
//
// It carries its own allocation_customer discriminator (not customer) because the customer id is not always present (allocation list entries omit it), so it is not a guaranteed-resolvable customer reference.
type AllocationCustomer struct {
	// Customer account ID.
	ID *string `json:"id"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=allocation_customer"`
	// Customer display name.
	Name string `json:"name" validate:"required"`
	// Customer number.
	Number *string `json:"number"`
}

// Minimal transaction sub-resource for allocation entries.
type AllocationTransaction struct {
	// Transaction ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction"`
	// Transaction type code: one of `payment`, `credit_memo`, `adjustment`, or `rebate`.
	Type string `json:"type" validate:"required"`
	// Payment method code (e.g. `check`, `ach`).
	//
	// Typically present only on payment transactions and null for credit memos, adjustments, and rebates.
	Method *string `json:"method"`
	// Adjustment category code.
	//
	// Typically populated when `type` is `adjustment`; null for other types.
	AdjustmentType *string `json:"adjustment_type"`
}

// Minimal invoice sub-resource for allocation entries.
type AllocationInvoice struct {
	// Invoice ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=invoice_summary"`
	// Invoice number.
	Number string `json:"number" validate:"required"`
}

var SampleAllocationEntry = &AllocationEntry{
	ID:            SampleAllocationEntryID,
	Object:        constants.ObjectTypeAllocationEntry,
	Amount:        "500.000000000000000000000000000000",
	DisplayAmount: "$500.00",
	Customer: &AllocationCustomer{
		Object: constants.ObjectTypeAllocationCustomer,
		Name:   SampleCustomerName,
	},
	Transaction: &AllocationTransaction{
		ID:     SampleTransactionDetailID,
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

// Open (not fully allocated) credit transaction.
type OpenCreditEntry struct {
	// Transaction ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=open_credit_entry"`
	// Transaction number.
	Number string `json:"number" validate:"required"`
	// Original transaction amount as a decimal string.
	OriginalAmount string `json:"original_amount" validate:"required"`
	// Total amount already allocated against invoices as a decimal string.
	AllocatedAmount string `json:"allocated_amount" validate:"required"`
	// Remaining unallocated credit as a decimal string (`original_amount` minus `allocated_amount`).
	//
	// This is the balance still available to apply.
	LeftoverAmount string `json:"leftover_amount" validate:"required"`
	// Customer associated with this transaction.
	Customer *AllocationCustomer `json:"customer" validate:"required"`
	// Type of the underlying transaction.
	//
	// Corresponds to one of the standard transaction types: payment, credit memo, adjustment, or rebate.
	TransactionType string `json:"transaction_type" validate:"required"`
	// Payment method of the underlying transaction.
	//
	// Typically present only on payment transactions and null for credit memos, adjustments, and rebates.
	TransactionMethod *string `json:"transaction_method"`
	// Adjustment category of the underlying transaction.
	//
	// Typically populated for adjustment transactions; null for other types.
	AdjustmentType *string `json:"adjustment_type"`
	// Responsible user's name.
	ResponsibleUserName *string `json:"responsible_user_name"`
	// Note about this transaction.
	Note *string `json:"note"`
	// Stripe payment ID, if applicable.
	StripePaymentID *string `json:"stripe_payment_id"`
	// Allocations against invoices for this transaction.
	InvoiceAllocations *List[InvoiceAllocationEntry] `json:"invoice_allocations"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
}

// Allocation of a credit against an invoice.
type InvoiceAllocationEntry struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=invoice_allocation_entry"`
	// Invoice number.
	InvoiceNumber string `json:"invoice_number" validate:"required"`
	// Allocated amount as a decimal string.
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
		ID:     new(SampleCustomerID),
		Object: constants.ObjectTypeAllocationCustomer,
		Name:   SampleCustomerName,
	},
	TransactionType: "payment",
	InvoiceAllocations: NewList([]InvoiceAllocationEntry{
		{Object: constants.ObjectTypeInvoiceAllocationEntry, InvoiceNumber: "INV-001", Amount: "500.000000000000000000000000000000"},
	}, PageInfo{}),
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*OpenCreditEntry) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOpenCreditEntry)
}
