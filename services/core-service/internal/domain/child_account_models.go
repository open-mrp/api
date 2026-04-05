package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

type ChildAccount struct {
	RelationID     string
	AccountID      string
	AccountName    string  `audit:"account_name"`
	ExternalNumber string  `audit:"external_number"`
	Email          *string `audit:"email"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ListChildAccountsParams struct {
	OwnerAccountID  string
	ParentAccountID string
	Cursor          *string
	Limit           int32
	Query           *string
}

type ListChildAccountsResult struct {
	Items    []*ChildAccount
	PageInfo pagination.PageInfo
}
