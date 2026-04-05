package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

// Transaction represents a full transaction with all related data.
type Transaction struct {
	ID                    string
	Number                string `audit:"number"`
	AmountID              string
	AmountValue           string `audit:"amount_value"`
	AmountUnitID          string
	AmountUnitAbbr        string
	CustomerID            *string `audit:"customer_id"`
	CustomerName          *string
	CustomerNumber        *string
	ResponsibleUserID     *string `audit:"responsible_user_id"`
	ResponsibleUserName   *string
	Note                  *string `audit:"note"`
	TransactionTypeCode   string  `audit:"transaction_type_code"`
	TransactionTypeName   string
	TransactionTypeID     string
	TransactionMethodCode *string `audit:"transaction_method_code"`
	TransactionMethodName *string
	TransactionMethodID   *string
	AdjustmentTypeCode    *string `audit:"adjustment_type_code"`
	AdjustmentTypeName    *string
	AdjustmentTypeID      *string
	IsFullyAllocated      bool    `audit:"is_fully_allocated"`
	StripePaymentID       *string `audit:"stripe_payment_id"`
	AllocationCount       int32
	Allocations           []*TransactionAllocation
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// TransactionSummary represents a lightweight transaction for list views.
type TransactionSummary struct {
	ID                    string
	Number                string
	AmountID              string
	AmountValue           string
	AmountUnitID          string
	AmountUnitAbbr        string
	CustomerID            *string
	CustomerName          *string
	CustomerNumber        *string
	TransactionTypeCode   string
	TransactionTypeName   string
	TransactionTypeID     string
	TransactionMethodCode *string
	TransactionMethodName *string
	TransactionMethodID   *string
	AdjustmentTypeCode    *string
	AdjustmentTypeName    *string
	AdjustmentTypeID      *string
	IsFullyAllocated      bool
	AllocationCount       int32
	CreatedAt             time.Time
	UpdatedAt             time.Time
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
	StartDate           *time.Time
	EndDate             *time.Time
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
}

// UpdateTransactionParams holds parameters for updating a transaction.
type UpdateTransactionParams struct {
	AccountID              string
	TransactionID          string
	Number                 *string
	Note                   *string
	Amount                 *string
	TransactionMethodCode  *string
	AdjustmentTypeCode     *string
	ResponsibleUserID      *string
	ClearResponsibleUser   bool
	ClearTransactionMethod bool
	ClearAdjustmentType    bool
	IsFullyAllocated       *bool
}

// DeleteTransactionParams holds parameters for deleting a transaction.
type DeleteTransactionParams struct {
	AccountID     string
	TransactionID string
}

// ListAccountTransactionsParams holds parameters for listing transactions by customer account.
type ListAccountTransactionsParams struct {
	AccountID            string
	CustomerAccountID    string
	Cursor               *string
	Limit                int32
	Query                *string
	Status               *string
	Type                 *string
	IncludeChildAccounts bool
}

// ListAccountTransactionsResult holds the result of listing customer transactions.
type ListAccountTransactionsResult struct {
	Transactions []*Transaction
	PageInfo     pagination.PageInfo
}
