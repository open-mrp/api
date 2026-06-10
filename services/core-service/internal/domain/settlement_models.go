package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
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
	InvoiceID         string
	InvoiceNumber     string `audit:"invoice_number"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
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

// CreateSettlementParams holds parameters for creating a settlement.
type CreateSettlementParams struct {
	AccountID         string
	ResponsibleUserID string
	Allocations       []CreateSettlementAllocationParams
}

// CreateSettlementAllocationParams holds parameters for a single allocation in a settlement.
type CreateSettlementAllocationParams struct {
	TransactionID string
	InvoiceID     string
	Amount        string
	Note          *string
}

// UpdateSettlementParams holds parameters for updating a settlement.
type UpdateSettlementParams struct {
	AccountID         string
	SettlementID      string
	Number            *string
	Note              *string
	ResponsibleUserID *string
}

// DeleteSettlementParams holds parameters for deleting a settlement.
type DeleteSettlementParams struct {
	AccountID    string
	SettlementID string
}
