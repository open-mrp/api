package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

type AccountGroupProductLineAccess struct {
	AccountGroupID   string
	AccountGroupName string            `audit:"account_group_name"`
	ProductLines     []ProductLineInfo `audit:"product_lines"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ProductLineInfo struct {
	ID   string
	Name string
}

type ListAccountGroupProductLineAccessParams struct {
	AccountID string
	Query     *string
	Cursor    *string
	Limit     int32
}

type ListAccountGroupProductLineAccessResult struct {
	Items    []*AccountGroupProductLineAccess
	PageInfo pagination.PageInfo
}

type CreateAccountGroupProductLineAccessParams struct {
	AccountID      string
	AccountGroupID string
	ProductLineIDs []string
}

type UpdateAccountGroupProductLineAccessParams struct {
	AccountID      string
	AccountGroupID string
	ProductLineIDs []string
}
