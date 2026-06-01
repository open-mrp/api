package apiresource

import (
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/pagination"
)

// PaginationRequest is the standard request type for paginated list endpoints.
// Embed this in a custom request struct if the endpoint needs additional query parameters.
type PaginationRequest struct {
	// Cursor token used to retrieve the next or previous page of results.
	Cursor *string `query:"cursor"`
	// Maximum number of results per page (default: 100, max: 1000).
	Limit int32 `query:"limit" default:"100" validate:"min=1,max=1000"`
	// Search query used to filter results.
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
