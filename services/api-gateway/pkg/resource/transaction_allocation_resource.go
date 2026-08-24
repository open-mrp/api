package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleAllocationEntryID = "txal_2o8lu50zvphn"
const SampleOpenCreditEntryID = "txn_wq90iimtw6ct" // #nosec G101 -- sample ID, not a credential

// An application of part of a transaction's amount against a specific invoice, as returned in list views.
type AllocationEntry struct {
	// Allocation ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=allocation_entry"`
	// The part of the transaction's amount applied to this invoice, as a decimal string in US dollars.
	Amount string `json:"amount" validate:"required"`
	// Human-readable formatted amount (e.g. "$500.00").
	DisplayAmount string `json:"display_amount" validate:"required"`
	// The customer whose transaction was applied.
	Customer *AllocationCustomer `json:"customer" validate:"required"`
	// The transaction the money came from.
	Transaction *AllocationTransaction `json:"transaction" validate:"required"`
	// The invoice the money was applied to.
	Invoice *AllocationInvoice `json:"invoice" validate:"required"`
	// Free-form note carried by the underlying transaction, not a note specific to this allocation.
	Note *string `json:"note"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
}

// Minimal customer reference carried by allocation entries and open-credit entries.
//
// Open-credit entries identify the customer by `id`; allocation entries carry only the customer's name and number.
type AllocationCustomer struct {
	// Customer account ID.
	ID *string `json:"id"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=allocation_customer"`
	// Customer display name.
	Name string `json:"name" validate:"required"`
	// The customer number for this customer, matching the `number` on your customer record for it.
	Number *string `json:"number"`
}

// Minimal transaction sub-resource for allocation entries.
type AllocationTransaction struct {
	// Transaction ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction"`
	// Type code of the transaction the money came from.
	Type constants.TransactionType `json:"type" validate:"required"`
	// Payment method code.
	//
	// Typically set only when `type` is `payment`.
	Method *constants.TransactionMethod `json:"method"`
	// Adjustment category code (e.g. `discount`, `write_off`).
	//
	// Typically set only when `type` is `adjustment`.
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
	Note:      new("Applied to the oldest open invoice first."),
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*AllocationEntry) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAllocationEntry)
}

// A transaction that still has credit available to apply to invoices.
//
// Whether a transaction counts as an open credit is driven by its `is_fully_allocated` flag rather than by a recomputed balance, so a transaction keeps appearing here until that flag is set — even if its allocations already cover the full amount.
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
	// Credit still available to apply, as a decimal string (`original_amount` minus `allocated_amount`).
	LeftoverAmount string `json:"leftover_amount" validate:"required"`
	// The customer this credit belongs to.
	Customer *AllocationCustomer `json:"customer" validate:"required"`
	// Display name of the transaction's type, such as "Payment" or "Credit Memo".
	TransactionType string `json:"transaction_type" validate:"required"`
	// Display name of the payment method, such as "Check" or "Credit Card".
	//
	// Typically set only on payment transactions.
	TransactionMethod *string `json:"transaction_method"`
	// Display name of the adjustment category, such as "Discount" or "Write Off".
	//
	// Typically set only on adjustment transactions.
	AdjustmentType *string `json:"adjustment_type"`
	// Username of the account user recorded as responsible for the transaction.
	ResponsibleUserName *string `json:"responsible_user_name"`
	// Free-form note attached to the transaction.
	Note *string `json:"note"`
	// Identifier of the Stripe payment that produced this transaction.
	StripePaymentID *string `json:"stripe_payment_id"`
	// The invoices this transaction has already been applied to, and how much went to each.
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
	TransactionType:     "payment",
	TransactionMethod:   new("check"),
	ResponsibleUserName: new(SampleUserName),
	Note:                new("Customer check deposited; partially applied."),
	InvoiceAllocations: NewList([]InvoiceAllocationEntry{
		{Object: constants.ObjectTypeInvoiceAllocationEntry, InvoiceNumber: "INV-001", Amount: "500.000000000000000000000000000000"},
	}, PageInfo{}),
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*OpenCreditEntry) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOpenCreditEntry)
}
