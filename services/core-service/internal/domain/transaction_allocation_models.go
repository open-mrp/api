package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

// AllocationEntry represents a lightweight transaction allocation for list views.
type AllocationEntry struct {
	ID                string
	AmountValue       string
	AmountUnitAbbr    string
	CustomerName      string
	CustomerNumber    *string
	TransactionID     string
	TransactionType   string
	TransactionMethod *string
	AdjustmentType    *string
	InvoiceID         string
	InvoiceNumber     string
	Note              *string
	CreatedAt         time.Time
}

// OpenCreditEntry represents an open (not fully allocated) credit transaction.
type OpenCreditEntry struct {
	ID                  string
	Number              string
	OriginalAmount      string
	AllocatedAmount     string
	LeftoverAmount      string
	CustomerName        string
	CustomerNumber      *string
	TransactionType     string
	TransactionMethod   *string
	AdjustmentType      *string
	ResponsibleUserName *string
	Note                *string
	StripePaymentID     *string
	InvoiceAllocations  []InvoiceAllocationEntry
	CreatedAt           time.Time
}

// InvoiceAllocationEntry represents an allocation against an invoice for the open credits view.
type InvoiceAllocationEntry struct {
	InvoiceNumber string
	Amount        string
}

// ListAllocationEntriesParams holds parameters for listing allocation entries.
type ListAllocationEntriesParams struct {
	AccountID       string
	Cursor          *string
	Limit           int32
	Query           *string
	TransactionType *string
	StartDate       *time.Time
	EndDate         *time.Time
}

// ListAllocationEntriesResult holds the result of listing allocation entries.
type ListAllocationEntriesResult struct {
	Entries  []*AllocationEntry
	PageInfo pagination.PageInfo
}

// UpdateTransactionAllocationParams holds parameters for updating a transaction allocation.
type UpdateTransactionAllocationParams struct {
	AccountID    string
	AllocationID string
	Amount       *string
}

// DeleteTransactionAllocationParams holds parameters for deleting a transaction allocation.
type DeleteTransactionAllocationParams struct {
	AccountID    string
	AllocationID string
}

// ListOpenCreditsParams holds parameters for listing open credits.
type ListOpenCreditsParams struct {
	AccountID   string
	StartDate   *time.Time
	EndDate     *time.Time
	CustomerIDs []string
	SearchQuery *string
	Cursor      *string
	Limit       int32
}

// ListOpenCreditsResult holds a page of open credit entries.
type ListOpenCreditsResult struct {
	Entries  []*OpenCreditEntry
	PageInfo pagination.PageInfo
}
