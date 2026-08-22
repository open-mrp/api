package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

type CustomerProductLineAccess struct {
	CustomerID     string
	CustomerName   string            `audit:"customer_name"`
	CustomerNumber string            `audit:"customer_number"`
	ProductLines   []ProductLineInfo `audit:"product_lines"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ListCustomerProductLineAccessParams struct {
	AccountID string
	Query     *string
	Cursor    *string
	Limit     int32
}

type ListCustomerProductLineAccessResult struct {
	Items    []*CustomerProductLineAccess
	PageInfo pagination.PageInfo
}

type CreateCustomerProductLineAccessParams struct {
	AccountID      string
	CustomerID     string
	ProductLineIDs []string
}

type UpdateCustomerProductLineAccessParams struct {
	AccountID      string
	CustomerID     string
	ProductLineIDs []string
}
