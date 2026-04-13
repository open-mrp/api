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

// Transaction type resource.
type TransactionType struct {
	// Transaction type ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction_type"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Machine-readable code.
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

// Transaction method resource.
type TransactionMethod struct {
	// Transaction method ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction_method"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Machine-readable code.
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

// Full transaction resource.
type TransactionDetail struct {
	// Transaction ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction"`
	// Transaction number.
	Number string `json:"number" validate:"required"`
	// Transaction amount.
	Amount *Quantity `json:"amount" validate:"required"`
	// Associated customer.
	Customer *Customer `json:"customer" expandable:"true"`
	// Responsible user.
	ResponsibleUser *AccountUser `json:"responsible_user" expandable:"true"`
	// Note.
	Note *string `json:"note"`
	// Transaction type.
	TransactionType *TransactionType `json:"transaction_type" validate:"required"`
	// Transaction method.
	TransactionMethod *TransactionMethod `json:"transaction_method"`
	// Adjustment type.
	AdjustmentType *AdjustmentType `json:"adjustment_type"`
	// Whether fully allocated against invoices.
	IsFullyAllocated bool `json:"is_fully_allocated" validate:"required"`
	// Stripe payment ID.
	StripePaymentID *string `json:"stripe_payment_id"`
	// Number of allocations.
	AllocationCount int32 `json:"allocation_count" validate:"required"`
	// Allocations.
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

// Lightweight transaction for list views.
type TransactionSummary struct {
	// Transaction ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction_summary"`
	// Transaction number.
	Number string `json:"number" validate:"required"`
	// Transaction amount.
	Amount *Quantity `json:"amount" validate:"required"`
	// Associated customer.
	Customer *Customer `json:"customer" expandable:"true"`
	// Transaction type.
	TransactionType *TransactionType `json:"transaction_type" validate:"required"`
	// Transaction method.
	TransactionMethod *TransactionMethod `json:"transaction_method"`
	// Adjustment type.
	AdjustmentType *AdjustmentType `json:"adjustment_type"`
	// Whether fully allocated against invoices.
	IsFullyAllocated bool `json:"is_fully_allocated" validate:"required"`
	// Number of allocations.
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
