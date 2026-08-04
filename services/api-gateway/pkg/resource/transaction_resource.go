package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleTransactionDetailID = "tx_hvh9thtzaezn"
const SampleTransactionMethodID = "txmd_hvm86tao3zbx"
const SampleTransactionTypeID = "txtp_vnml00fjmorb"

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
	// How the money moved, such as a check or an ACH transfer.
	//
	// Typically set only on payment transactions.
	TransactionMethod *TransactionMethod `json:"transaction_method"`
	// The kind of correction this transaction represents, such as a discount or a write-off.
	//
	// Typically set only on `adjustment` transactions.
	AdjustmentType *AdjustmentType `json:"adjustment_type"`
	// Whether the full transaction amount has been applied to invoices.
	//
	// Recording a settlement that uses this transaction sets the flag to `true`, and deleting that settlement resets it to `false`. Editing or deleting an individual allocation does not recompute it, so it can also be set directly with Update Transaction.
	//
	// While it is `false`, the transaction is treated as an open credit and is returned by List Open Credits.
	IsFullyAllocated bool `json:"is_fully_allocated" validate:"required"`
	// Identifier of the Stripe payment that produced this transaction.
	StripePaymentID *string `json:"stripe_payment_id"`
	// Number of allocations against invoices for this transaction.
	AllocationCount int32 `json:"allocation_count" validate:"required"`
	// The portions of this transaction that have been applied to individual invoices.
	//
	// Allocations are created by recording a settlement; there is no endpoint that creates one directly.
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
	// How the money moved, such as a check or an ACH transfer.
	//
	// Typically set only on payment transactions.
	TransactionMethod *TransactionMethod `json:"transaction_method"`
	// The kind of correction this transaction represents, such as a discount or a write-off.
	//
	// Typically set only on `adjustment` transactions.
	AdjustmentType *AdjustmentType `json:"adjustment_type"`
	// Whether the full transaction amount has been applied to invoices.
	//
	// Recording a settlement that uses this transaction sets the flag to `true`, and deleting that settlement resets it to `false`. Editing or deleting an individual allocation does not recompute it, so it can also be set directly with Update Transaction.
	//
	// While it is `false`, the transaction is treated as an open credit and is returned by List Open Credits.
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
