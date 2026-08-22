package apiresource

import "github.com/open-mrp/api/shared/constants"

// PageInfo describes where the current page sits within a paginated result set and how to move to the adjacent pages.
//
// Page a list by following the URLs below rather than assembling cursors yourself. For a top-level list endpoint the URL repeats the original request's query string with only the cursor swapped, so following it preserves the same filters, search term, and page size.
type PageInfo struct {
	// Relative URL that fetches the next page of results.
	NextPageURL *string `json:"next_page_url"`
	// Relative URL that fetches the previous page of results.
	PreviousPageURL *string `json:"previous_page_url"`
	// Whether more results exist after this page.
	HasNextPage bool `json:"has_next_page"`
	// Whether results exist before this page.
	HasPrevPage bool `json:"has_prev_page"`
}

// A single page of resources, together with the metadata needed to page through the rest of the result set.
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
