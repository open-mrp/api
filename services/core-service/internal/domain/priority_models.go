package domain

import (
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/pagination"
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
