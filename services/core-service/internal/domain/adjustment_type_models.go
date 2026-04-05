package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

type AdjustmentType struct {
	ID        string
	Name      string
	Code      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ListAdjustmentTypesParams struct {
	Cursor *string
	Limit  int32
	Query  *string
}

type ListAdjustmentTypesResult struct {
	AdjustmentTypes []*AdjustmentType
	PageInfo        pagination.PageInfo
}
