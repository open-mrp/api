package apiresource

import "github.com/augno/api/shared/constants"

// PageInfo contains cursor-based pagination metadata.
type PageInfo struct {
	// Cursor to fetch the next page of results, null if no more pages.
	NextCursor *string `json:"next_cursor"`
	// Cursor to fetch the previous page of results, null if on the first page.
	PrevCursor *string `json:"prev_cursor"`
	// Whether there are more results after this page.
	HasNextPage bool `json:"has_next_page"`
	// Whether there are results before this page.
	HasPrevPage bool `json:"has_prev_page"`
}

// List represents a paginated list of resources.
type List[T any] struct {
	// The object type.
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// Pagination metadata.
	PageInfo PageInfo `json:"page_info"`
	// The array of resources in this page.
	Data []T `json:"data" validate:"required"`
}

// NewList creates a new List of resources.
func NewList[T any](data []T, pageInfo PageInfo) *List[T] {
	if data == nil {
		data = []T{}
	}
	return &List[T]{
		Object:   constants.ObjectTypeList,
		PageInfo: pageInfo,
		Data:     data,
	}
}
