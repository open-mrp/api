package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

type AccountStatus struct {
	ID        string
	Code      string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ListAccountStatusesParams struct {
	Cursor *string
	Limit  int32
	Query  *string
}

type ListAccountStatusesResult struct {
	AccountStatuses []*AccountStatus
	PageInfo        pagination.PageInfo
}
