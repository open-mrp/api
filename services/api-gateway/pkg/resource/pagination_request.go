package apiresource

import (
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/pagination"
)

// PaginationRequest is the standard request type for paginated list endpoints.
//
// Embed this in a custom request struct if the endpoint needs additional query parameters.
type PaginationRequest struct {
	// Opaque cursor token identifying where the page of results starts.
	//
	// Use the `cursor` value embedded in a previous response's `next_page_url` or `previous_page_url` to fetch the adjacent page. Omit to start from the first page.
	Cursor *string `query:"cursor"`
	// Maximum number of results to return in a single page.
	Limit int32 `query:"limit" default:"100" validate:"min=1,max=1000"`
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

var _ contracts.DocumentedType = (*PaginationRequest)(nil)

// SchemaExample documents standard list query parameters for OpenAPI.
func (*PaginationRequest) SchemaExample() any {
	q := "6061"
	return map[string]any{
		"cursor": pagination.EncodeDocumentationStringCursor(SampleAnalyticsPeriodStart, SampleItemID),
		"limit":  int64(100),
		"q":      q,
	}
}
