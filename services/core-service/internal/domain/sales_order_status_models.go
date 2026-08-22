package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

type SalesOrderStatus struct {
	ID        string
	Code      string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ListSalesOrderStatusesParams struct {
	Cursor *string
	Limit  int32
	Query  *string
}

type ListSalesOrderStatusesResult struct {
	SalesOrderStatuses []*SalesOrderStatus
	PageInfo           pagination.PageInfo
}
