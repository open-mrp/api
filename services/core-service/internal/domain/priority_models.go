package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/pagination"
)

type Priority struct {
	ID        string
	Name      string
	Code      constants.PriorityCode
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ListPrioritiesParams struct {
	Cursor *string
	Limit  int32
	Query  *string
}

type ListPrioritiesResult struct {
	Priorities []*Priority
	PageInfo   pagination.PageInfo
}
