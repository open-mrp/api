package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

// Transaction represents a full transaction with all related data.
type Transaction struct {
	ID                        string
	Number                    string `audit:"number"`
	AmountID                  string
	AmountValue               string `audit:"amount_value"`
	AmountUnitID              string
	AmountUnitAbbr            string
	CustomerID                *string `audit:"customer_id"`
	CustomerName              *string
	CustomerNumber            *string
	CustomerStatusCode        *string
	CustomerCommissionPolicy  *string
	CustomerCreatedAt         *time.Time
	CustomerUpdatedAt         *time.Time
	ResponsibleUserID         *string `audit:"responsible_user_id"`
	ResponsibleUserName       *string
	ResponsibleUserStatusCode *string
	ResponsibleUserCreatedAt  *time.Time
	ResponsibleUserUpdatedAt  *time.Time
	Note                      *string `audit:"note"`
	TransactionTypeCode       string  `audit:"transaction_type_code"`
	TransactionTypeName       string
	TransactionTypeID         string
	TransactionMethodCode     *string `audit:"transaction_method_code"`
	TransactionMethodName     *string
	TransactionMethodID       *string
	AdjustmentTypeCode        *string `audit:"adjustment_type_code"`
	AdjustmentTypeName        *string
	AdjustmentTypeID          *string
	IsFullyAllocated          bool    `audit:"is_fully_allocated"`
	StripePaymentID           *string `audit:"stripe_payment_id"`
	// FundsReceivedAt is when the money arrived; nil until it has. Only a transaction whose funds
	// have arrived can be settled or counts as an open credit.
	FundsReceivedAt *time.Time `audit:"funds_received_at"`
	AllocationCount int32
	Allocations     []*TransactionAllocation
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TransactionSummary represents a lightweight transaction for list views.
type TransactionSummary struct {
	ID                       string
	Number                   string
	AmountID                 string
	AmountValue              string
	AmountUnitID             string
	AmountUnitAbbr           string
	CustomerID               *string
	CustomerName             *string
	CustomerNumber           *string
	CustomerStatusCode       *string
	CustomerCommissionPolicy *string
	CustomerCreatedAt        *time.Time
	CustomerUpdatedAt        *time.Time
	TransactionTypeCode      string
	TransactionTypeName      string
	TransactionTypeID        string
	TransactionMethodCode    *string
	TransactionMethodName    *string
	TransactionMethodID      *string
	AdjustmentTypeCode       *string
	AdjustmentTypeName       *string
	AdjustmentTypeID         *string
	IsFullyAllocated         bool
	FundsReceivedAt          *time.Time
	AllocationCount          int32
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

// ListTransactionsParams holds parameters for listing transactions.
type ListTransactionsParams struct {
	AccountID           string
	Cursor              *string
	Limit               int32
	Query               *string
	Status              *string
	TypeCodes           []string
	AdjustmentTypeCodes []string
	MethodCodes         []string
	CustomerIDs         []string
	CustomerGroupIDs    []string
	// StartDate and EndDate bound when the funds were received, inclusive.
	StartDate *time.Time
	EndDate   *time.Time
}

// ListTransactionsResult holds the result of listing transactions.
type ListTransactionsResult struct {
	Transactions []*TransactionSummary
	PageInfo     pagination.PageInfo
}

// GetTransactionParams holds parameters for getting a single transaction.
type GetTransactionParams struct {
	AccountID     string
	TransactionID string
	Includes      []string
}

// CreateTransactionParams holds parameters for creating a transaction.
type CreateTransactionParams struct {
	AccountID             string
	CustomerID            string
	TransactionTypeCode   string
	Amount                string
	TransactionMethodCode *string
	AdjustmentTypeCode    *string
	ResponsibleUserID     *string
	Note                  *string
	StripePaymentID       *string
	// CreatedAt backdates the transaction; nil records it now.
	CreatedAt       *time.Time
	FundsReceivedAt *time.Time
}

// UpdateTransactionParams holds parameters for updating a transaction.
type UpdateTransactionParams struct {
	AccountID              string
	TransactionID          string
	Number                 *string
	Note                   *string
	ClearNote              bool
	Amount                 *string
	TransactionMethodCode  *string
	AdjustmentTypeCode     *string
	ResponsibleUserID      *string
	ClearResponsibleUser   bool
	ClearTransactionMethod bool
	ClearAdjustmentType    bool
	IsFullyAllocated       *bool
	CreatedAt              *time.Time
	FundsReceivedAt        *time.Time
	ClearFundsReceivedAt   bool
}

// DeleteTransactionParams holds parameters for deleting a transaction.
type DeleteTransactionParams struct {
	AccountID     string
	TransactionID string
}

// ListAccountTransactionsParams holds parameters for listing transactions by customer account.
type ListAccountTransactionsParams struct {
	AccountID         string
	CustomerAccountID string
	Cursor            *string
	Limit             int32
	Query             *string
	// Status "unallocated" returns the transactions still available to settle: not fully allocated
	// and with their funds received.
	Status *string
	Type   *string
	// WithAllocations loads each transaction's allocations.
	WithAllocations bool
}

// ListAccountTransactionsResult holds the result of listing customer transactions.
type ListAccountTransactionsResult struct {
	Transactions []*Transaction
	PageInfo     pagination.PageInfo
}
