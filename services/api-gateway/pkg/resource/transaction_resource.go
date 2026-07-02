package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleTransactionDetailID = "tx_01fc4d4f2b2ee1fa6b6d87257a"
const SampleTransactionMethodID = "txmd_011b68c574f7c84504fc256ca7"
const SampleTransactionTypeID = "txtp_01552974c3952ed8178ad671b8"

// The category of a financial transaction, such as a payment or credit memo.
type TransactionType struct {
	// Transaction type ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction_type"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Machine-readable code identifying the kind of transaction.
	//
	// - `payment`: money received from the customer.
	// - `credit_memo`: a credit issued to the customer.
	// - `adjustment`: a manual correction (see the transaction's `adjustment_type`).
	// - `rebate`: a rebate granted to the customer.
	Code constants.TransactionType `json:"code" validate:"required"`
}

var SampleTransactionType = &TransactionType{
	ID:     SampleTransactionTypeID,
	Object: constants.ObjectTypeTransactionType,
	Name:   "Payment",
	Code:   constants.TransactionTypePayment,
}

func (*TransactionType) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleTransactionType)
}

// The payment method used to make a transaction, such as cash or check.
type TransactionMethod struct {
	// Transaction method ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction_method"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Machine-readable code identifying how the transaction was made.
	Code constants.TransactionMethod `json:"code" validate:"required"`
}

var SampleTransactionMethod = &TransactionMethod{
	ID:     SampleTransactionMethodID,
	Object: constants.ObjectTypeTransactionMethod,
	Name:   "Credit Card",
	Code:   constants.TransactionMethodCreditCard,
}

func (*TransactionMethod) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleTransactionMethod)
}

// A financial transaction recorded against a customer, such as a payment, credit memo, adjustment, or rebate.
type TransactionDetail struct {
	// Transaction ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction"`
	// Human-readable transaction number.
	//
	// Generated automatically as a per-account sequence when the transaction is created. It can be changed later, but must remain unique within the account.
	Number string `json:"number" validate:"required"`
	// The transaction amount, in US dollars.
	Amount *Quantity `json:"amount" validate:"required"`
	// The customer the transaction was recorded against.
	Customer *Customer `json:"customer" expandable:"true"`
	// The account user responsible for the transaction.
	//
	// When none is specified at creation, the account user making the request is recorded as responsible.
	ResponsibleUser *AccountUser `json:"responsible_user" expandable:"true"`
	// Free-form note attached to the transaction.
	Note *string `json:"note"`
	// The transaction's type (payment, credit memo, adjustment, or rebate).
	TransactionType *TransactionType `json:"transaction_type" validate:"required"`
	// Payment method used.
	//
	// Typically present only on payment transactions and null for credit memos, adjustments, and rebates.
	TransactionMethod *TransactionMethod `json:"transaction_method"`
	// Adjustment category.
	//
	// Typically populated for `adjustment` transactions; null for other types.
	AdjustmentType *AdjustmentType `json:"adjustment_type"`
	// Whether the full transaction amount has been allocated against invoices.
	//
	// When `false`, some of the amount remains as an open (unapplied) balance and the transaction appears in the open credits list. This flag is set explicitly (see Update Transaction); it is not recomputed automatically when allocations change.
	IsFullyAllocated bool `json:"is_fully_allocated" validate:"required"`
	// Stripe payment ID.
	//
	// Set only for transactions collected through Stripe; null for transactions recorded outside Stripe.
	StripePaymentID *string `json:"stripe_payment_id"`
	// Number of allocations against invoices for this transaction.
	AllocationCount int32 `json:"allocation_count" validate:"required"`
	// Allocations of this transaction against invoices.
	Allocations *List[TransactionAllocation] `json:"allocations" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleTransactionNote = "Payment received"

var SampleTransactionDetail = &TransactionDetail{
	ID:     SampleTransactionDetailID,
	Object: constants.ObjectTypeTransaction,
	Number: "1",
	Amount: &Quantity{
		ID:           SampleQuantityID,
		Object:       constants.ObjectTypeQuantity,
		Value:        "500.000000000000000000000000000000",
		DisplayValue: "$500.00",
		Unit:         newSampleUnit("US Dollar", "$", constants.UnitTypeCurrency),
	},
	Customer: nil,
	ResponsibleUser: &AccountUser{
		ID:     SampleAccountUserID,
		Object: constants.ObjectTypeAccountUser,
	},
	Note:              &sampleTransactionNote,
	TransactionType:   SampleTransactionType,
	TransactionMethod: SampleTransactionMethod,
	IsFullyAllocated:  false,
	StripePaymentID:   new("pi_3PqR8s2eZvKYlo2C0AbCdEfG"),
	AllocationCount:   1,
	Allocations: NewList([]TransactionAllocation{{
		ID:     SampleAllocationEntryID,
		Object: constants.ObjectTypeTransactionAllocation,
		Amount: &Quantity{
			ID:           SampleQuantityID,
			Object:       constants.ObjectTypeQuantity,
			Value:        "500.000000000000000000000000000000",
			DisplayValue: "$500.00",
			Unit:         newSampleUnit("US Dollar", "$", constants.UnitTypeCurrency),
		},
		Invoice: &AllocationInvoice{
			ID:     SampleInvoiceID,
			Object: constants.ObjectTypeInvoiceSummary,
			Number: "INV-001",
		},
		CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
		UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
	}}, PageInfo{}),
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*TransactionDetail) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleTransactionDetail)
}

// Lightweight transaction for list views.
type TransactionSummary struct {
	// Transaction ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction_summary"`
	// Human-readable transaction number, unique within the account.
	Number string `json:"number" validate:"required"`
	// The transaction amount, in US dollars.
	Amount *Quantity `json:"amount" validate:"required"`
	// The customer the transaction was recorded against.
	Customer *Customer `json:"customer" expandable:"true"`
	// The transaction's type (payment, credit memo, adjustment, or rebate).
	TransactionType *TransactionType `json:"transaction_type" validate:"required"`
	// Payment method used.
	//
	// Typically present only on payment transactions and null for credit memos, adjustments, and rebates.
	TransactionMethod *TransactionMethod `json:"transaction_method"`
	// Adjustment category.
	//
	// Typically populated for `adjustment` transactions; null for other types.
	AdjustmentType *AdjustmentType `json:"adjustment_type"`
	// Whether the full transaction amount has been allocated against invoices.
	//
	// When `false`, some of the amount remains as an open (unapplied) balance and the transaction appears in the open credits list. This flag is set explicitly (see Update Transaction); it is not recomputed automatically when allocations change.
	IsFullyAllocated bool `json:"is_fully_allocated" validate:"required"`
	// Number of allocations against invoices for this transaction.
	AllocationCount int32 `json:"allocation_count" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleTransactionSummary = &TransactionSummary{
	ID:     SampleTransactionDetailID,
	Object: constants.ObjectTypeTransactionSummary,
	Number: "1",
	Amount: &Quantity{
		ID:           SampleQuantityID,
		Object:       constants.ObjectTypeQuantity,
		Value:        "500.000000000000000000000000000000",
		DisplayValue: "$500.00",
		Unit:         newSampleUnit("US Dollar", "$", constants.UnitTypeCurrency),
	},
	Customer:          nil,
	TransactionType:   SampleTransactionType,
	TransactionMethod: SampleTransactionMethod,
	IsFullyAllocated:  false,
	AllocationCount:   1,
	CreatedAt:         timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:         timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*TransactionSummary) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleTransactionSummary)
}
