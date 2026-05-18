package appctx

import (
	"context"
	"net/url"
)

const requestURLKey contextKey = "request_url"

// WithRequestURL returns a child context carrying the current HTTP request URL.
// This is used by list presenters to build absolute pagination URLs.
func WithRequestURL(ctx context.Context, u *url.URL) context.Context {
	return context.WithValue(ctx, requestURLKey, u)
}

// GetRequestURL retrieves the request URL stored in the context.
func GetRequestURL(ctx context.Context) (*url.URL, bool) {
	u, ok := ctx.Value(requestURLKey).(*url.URL)
	return u, ok && u != nil
}
