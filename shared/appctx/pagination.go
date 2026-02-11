package appctx

import "context"

const paginationKey contextKey = "pagination"

// PaginationParams contains the parsed pagination information from the request.
type PaginationParams struct {
	Cursor *string `json:"cursor"`
	Limit  int32   `json:"limit" default:"10"`
	Query  *string `json:"query"`
}

// WithPagination returns a child context carrying pagination parameters.
func WithPagination(ctx context.Context, p *PaginationParams) context.Context {
	return context.WithValue(ctx, paginationKey, p)
}

// GetPagination retrieves the pagination parameters from the context.
func GetPagination(ctx context.Context) (*PaginationParams, bool) {
	p, ok := ctx.Value(paginationKey).(*PaginationParams)
	return p, ok && p != nil
}
