package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

type DCLocation struct {
	ID             string
	Location       string `audit:"location"`
	AccountID      string `audit:"account_id"`
	CustomerName   string `audit:"customer_name"`
	OwnerAccountID string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ListDCLocationsParams struct {
	OwnerAccountID string
	Cursor         *string
	Limit          int32
	Query          *string
}

type ListDCLocationsResult struct {
	DCLocations []*DCLocation
	PageInfo    pagination.PageInfo
}

type GetDCLocationParams struct {
	OwnerAccountID string
	DCLocationID   string
}

type CreateDCLocationParams struct {
	OwnerAccountID string
	AccountID      string
	Location       string
}

type UpdateDCLocationParams struct {
	OwnerAccountID string
	DCLocationID   string
	AccountID      *string
	Location       *string
}

type DeleteDCLocationParams struct {
	OwnerAccountID string
	DCLocationID   string
}

type EDIRun struct {
	ID           string
	CompletedAt  time.Time
	HasSucceeded bool
	AccountID    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ListEDIRunsParams struct {
	AccountID    string
	Cursor       *string
	Limit        int32
	HasSucceeded *bool
	Query        *string
}

type ListEDIRunsResult struct {
	EDIRuns  []*EDIRun
	PageInfo pagination.PageInfo
}

// The document an outbound EDI transmission carries; the dashboard API's transmitter sends it.
const (
	// EdiDocumentTypeInvoice is the X12 810 invoice.
	EdiDocumentTypeInvoice = "810"
	// EdiSubjectTypeInvoice names an invoice as the record a transmission is about.
	EdiSubjectTypeInvoice = "invoice"
)

// Records an outbound document owed to a trading partner, due for transmission now.
type EnqueueEdiTransmissionParams struct {
	ID                    string
	AccountID             string
	DocumentType          string
	SubjectType           string
	SubjectID             string
	CounterpartyAccountID string
}
