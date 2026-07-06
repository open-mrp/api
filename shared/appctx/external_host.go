package appctx

import (
	"context"
)

const externalHostKey contextKey = "external_host"

// WithExternalHost returns a child context carrying the host the browser addressed (from X-Forwarded-Host when the request arrived through a trusted front proxy such as the frontend's custom-domain API rewrite, otherwise the Host header). Auth cookie scoping keys off this value.
func WithExternalHost(ctx context.Context, host string) context.Context {
	return context.WithValue(ctx, externalHostKey, host)
}

// GetExternalHost retrieves the external request host stored in the context.
func GetExternalHost(ctx context.Context) (string, bool) {
	host, ok := ctx.Value(externalHostKey).(string)
	return host, ok && host != ""
}
