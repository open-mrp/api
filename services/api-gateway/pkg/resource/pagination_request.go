package apiresource

// PaginationRequest is the standard request type for paginated list endpoints.
// Embed this in a custom request struct if the endpoint needs additional query parameters.
type PaginationRequest struct {
	// Cursor for fetching the next page, from a previous response's next_cursor field.
	Cursor *string `query:"cursor"`
	// Maximum number of results to return per page (default: 100, max: 1000).
	Limit int32 `query:"limit" default:"100" validate:"min=1,max=1000"`
	// Optional search query to filter results.
	Query *string `query:"q"`
}
