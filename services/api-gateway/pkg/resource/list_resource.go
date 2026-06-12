package apiresource

import "github.com/augno/api/shared/constants"

// PageInfo contains URL-based pagination metadata.
type PageInfo struct {
	// Relative URL that fetches the next page of results.
	//
	// `null` when the last page has been reached.
	NextPageURL *string `json:"next_page_url"`
	// Relative URL that fetches the previous page of results.
	//
	// `null` while on the first page.
	PreviousPageURL *string `json:"previous_page_url"`
	// Whether more results exist after this page.
	HasNextPage bool `json:"has_next_page"`
	// Whether results exist before this page.
	HasPrevPage bool `json:"has_prev_page"`
}

// List represents a paginated list of resources.
type List[T any] struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// Pagination metadata.
	PageInfo PageInfo `json:"page_info"`
	// Resources in this page.
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
