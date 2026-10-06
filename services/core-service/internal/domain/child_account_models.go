package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

type ChildAccount struct {
	RelationID     string
	AccountID      string
	AccountName    string  `audit:"account_name"`
	ExternalNumber string  `audit:"external_number"`
	Email          *string `audit:"email"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	// AccountCreatedAt and AccountUpdatedAt are the child account's own timestamps; CreatedAt and UpdatedAt are the relation's.
	AccountCreatedAt time.Time
	AccountUpdatedAt time.Time
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
