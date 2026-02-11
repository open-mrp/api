package appctx

import (
	"context"
	"net/http"
)

const httpResponseMetadataKey contextKey = "http_response_metadata"

// HTTPResponseMetadata carries response-scoped data (cookies, replay flag)
// through the request lifecycle via a shared pointer in context.
type HTTPResponseMetadata struct {
	Cookies  []*http.Cookie
	Replayed bool
}

// WithHTTPResponseMetadata returns a child context carrying a fresh metadata
// pointer. The pointer is shared so downstream code can mutate it.
func WithHTTPResponseMetadata(ctx context.Context) (context.Context, *HTTPResponseMetadata) {
	meta := &HTTPResponseMetadata{}
	return context.WithValue(ctx, httpResponseMetadataKey, meta), meta
}

// GetHTTPResponseMetadata retrieves the HTTP response metadata from the context.
func GetHTTPResponseMetadata(ctx context.Context) (*HTTPResponseMetadata, bool) {
	meta, ok := ctx.Value(httpResponseMetadataKey).(*HTTPResponseMetadata)
	return meta, ok && meta != nil
}

// AddCookies appends cookies to the HTTP response metadata in context.
func AddCookies(ctx context.Context, cookies []*http.Cookie) {
	if meta, ok := GetHTTPResponseMetadata(ctx); ok {
		meta.Cookies = append(meta.Cookies, cookies...)
	}
}

// SetHTTPReplayed sets the Replayed flag on the HTTP response metadata in context.
func SetHTTPReplayed(ctx context.Context, replayed bool) {
	if meta, ok := GetHTTPResponseMetadata(ctx); ok {
		meta.Replayed = replayed
	}
}
