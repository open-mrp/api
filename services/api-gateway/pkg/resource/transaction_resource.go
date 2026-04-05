package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleTransactionDetailID = "tx_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleTransactionMethodID = "txmd_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleTransactionTypeID = "txtp_01jm4r6700f8nwq3v5hx2d9ktp"

// TransactionType represents a type of transaction.
type TransactionType struct {
	// The unique identifier for the transaction type.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction_type"`
	// The display name of the transaction type.
	Name string `json:"name" validate:"required"`
	// The machine-readable code for the transaction type.
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

// TransactionMethod represents a method used for a transaction.
type TransactionMethod struct {
	// The unique identifier for the transaction method.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction_method"`
	// The display name of the transaction method.
	Name string `json:"name" validate:"required"`
	// The machine-readable code for the transaction method.
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

// TransactionDetail represents a full transaction API resource.
type TransactionDetail struct {
	// The unique identifier for the transaction.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction"`
	// The transaction number.
	Number string `json:"number" validate:"required"`
	// The transaction amount.
	Amount *Quantity `json:"amount" validate:"required"`
	// The customer associated with this transaction.
	Customer *Customer `json:"customer" expandable:"true"`
	// The user responsible for this transaction.
	ResponsibleUser *AccountUser `json:"responsible_user" expandable:"true"`
	// A note attached to this transaction.
	Note *string `json:"note"`
	// The type of this transaction.
	TransactionType *TransactionType `json:"transaction_type" validate:"required"`
	// The method used for this transaction.
	TransactionMethod *TransactionMethod `json:"transaction_method"`
	// The adjustment type, if this is an adjustment transaction.
	AdjustmentType *AdjustmentType `json:"adjustment_type"`
	// Whether this transaction is fully allocated against invoices.
	IsFullyAllocated bool `json:"is_fully_allocated" validate:"required"`
	// The Stripe payment ID associated with this transaction.
	StripePaymentID *string `json:"stripe_payment_id"`
	// The number of allocations for this transaction.
	AllocationCount int32 `json:"allocation_count" validate:"required"`
	// The allocations for this transaction.
	Allocations *List[TransactionAllocation] `json:"allocations" expandable:"true"`
	// The timestamp when the transaction was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the transaction was last updated.
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
		Unit: &Unit{
			ID:           SampleUnitID,
			Object:       constants.ObjectTypeUnit,
			Name:         "US Dollar",
			Abbreviation: "$",
			Type:         constants.UnitTypeCurrency,
		},
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
	AllocationCount:   1,
	CreatedAt:         timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:         timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*TransactionDetail) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleTransactionDetail)
}

// TransactionSummary represents a lightweight transaction for list views.
type TransactionSummary struct {
	// The unique identifier for the transaction.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction_summary"`
	// The transaction number.
	Number string `json:"number" validate:"required"`
	// The transaction amount.
	Amount *Quantity `json:"amount" validate:"required"`
	// The customer associated with this transaction.
	Customer *Customer `json:"customer" expandable:"true"`
	// The type of this transaction.
	TransactionType *TransactionType `json:"transaction_type" validate:"required"`
	// The method used for this transaction.
	TransactionMethod *TransactionMethod `json:"transaction_method"`
	// The adjustment type, if this is an adjustment transaction.
	AdjustmentType *AdjustmentType `json:"adjustment_type"`
	// Whether this transaction is fully allocated against invoices.
	IsFullyAllocated bool `json:"is_fully_allocated" validate:"required"`
	// The number of allocations for this transaction.
	AllocationCount int32 `json:"allocation_count" validate:"required"`
	// The timestamp when the transaction was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the transaction was last updated.
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
		Unit: &Unit{
			ID:           SampleUnitID,
			Object:       constants.ObjectTypeUnit,
			Name:         "US Dollar",
			Abbreviation: "$",
			Type:         constants.UnitTypeCurrency,
		},
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
