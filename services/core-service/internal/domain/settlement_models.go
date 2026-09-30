package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

// Settlement represents a full settlement with expandable allocations.
type Settlement struct {
	ID                  string
	Number              string  `audit:"number"`
	Note                *string `audit:"note"`
	ResponsibleUserID   *string
	ResponsibleUserName *string `audit:"responsible_user_name"`
	Allocations         []*TransactionAllocation
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// SettlementSummary represents a lightweight settlement for list views.
type SettlementSummary struct {
	ID               string
	Number           string
	AllocationCount  int32
	TotalPayments    *string
	TotalRebates     *string
	TotalAdjustments *string
	TotalCredits     *string
	InvoiceNumbers   []string
	CustomerNames    []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// TransactionAllocation represents an allocation of a transaction against an invoice within a settlement.
type TransactionAllocation struct {
	ID                string
	AmountID          string
	AmountValue       string `audit:"amount_value"`
	AmountUnitID      string
	AmountUnitAbbr    string  `audit:"amount_unit_abbr"`
	Note              *string `audit:"note"`
	TransactionID     string
	TransactionNumber string `audit:"transaction_number"`
	TransactionType   string `audit:"transaction_type"`
	// The drawn-on transaction's method, adjustment type, customer and creation time, which a
	// transaction's own allocation list shows without loading the transaction again.
	TransactionMethodCode     *string
	TransactionAdjustmentType *string
	TransactionCustomerID     string
	TransactionCreatedAt      time.Time
	InvoiceID                 string
	InvoiceNumber             string `audit:"invoice_number"`
	// The settlement that recorded the allocation; nil for one made outside a settlement.
	SettlementID     *string
	SettlementNumber *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ListSettlementsParams holds parameters for listing settlements.
type ListSettlementsParams struct {
	AccountID      string
	Cursor         *string
	Limit          int32
	Query          *string
	TransactionIDs []string
	InvoiceIDs     []string
	StartDate      *time.Time
	EndDate        *time.Time
}

// ListSettlementsResult holds the result of listing settlements.
type ListSettlementsResult struct {
	Settlements []*SettlementSummary
	PageInfo    pagination.PageInfo
}

// GetSettlementParams holds parameters for getting a single settlement.
type GetSettlementParams struct {
	AccountID    string
	SettlementID string
	Includes     []string
}

// RecomputePaymentFlagsEvent is the outbox command payload asking a consumer to re-derive payment flags.
// It carries only identities: the flags are recomputed from current allocations, so repeated commands
// coalesce and a redelivery computes the same values.
type RecomputePaymentFlagsEvent struct {
	AccountID      string   `json:"account_id"`
	TransactionIDs []string `json:"transaction_ids"`
	InvoiceIDs     []string `json:"invoice_ids"`
}

// CreateSettlementParams holds parameters for creating a settlement.
type CreateSettlementParams struct {
	AccountID         string
	ResponsibleUserID string
	Allocations       []CreateSettlementAllocationParams
	// NewTransactions are recorded with the settlement: adjustments and credits entered while
	// settling, which exist only as the allocations drawn from them.
	NewTransactions []NewSettlementTransactionParams
}

// NewSettlementTransactionParams describes a transaction created by the settlement that uses it.
// Its amount is the sum of the allocations naming its key, and it is dated, and its funds counted as
// received, at the first of those allocations.
type NewSettlementTransactionParams struct {
	// Key names the transaction within the request; allocations draw on it by this key.
	Key                   string
	TransactionTypeCode   string
	TransactionMethodCode *string
	AdjustmentTypeCode    *string
	CustomerID            string
}

// CreateSettlementAllocationParams holds parameters for a single allocation in a settlement.
type CreateSettlementAllocationParams struct {
	// TransactionID names an existing transaction; TransactionKey one of the settlement's new
	// transactions. Exactly one is set.
	TransactionID  string
	TransactionKey string
	InvoiceID      string
	Amount         string
	Note           *string
	// CreatedAt dates the allocation (the day the payment was applied); nil dates it now.
	CreatedAt *time.Time
}

// UpdateSettlementParams holds parameters for updating a settlement.
type UpdateSettlementParams struct {
	AccountID         string
	SettlementID      string
	Number            *string
	Note              *string
	ClearNote         bool
	ResponsibleUserID *string
}

// DeleteSettlementParams holds parameters for deleting a settlement.
type DeleteSettlementParams struct {
	AccountID    string
	SettlementID string
}
