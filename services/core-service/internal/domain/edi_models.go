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
