package appctx

import "context"

const propagatedClientIPKey contextKey = "propagated_client_ip"

// WithPropagatedClientIP attaches the original HTTP client IP when it was
// forwarded from the API gateway via gRPC metadata.
func WithPropagatedClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, propagatedClientIPKey, ip)
}

// GetPropagatedClientIP returns the client IP from [WithPropagatedClientIP] when
// present and non-empty.
func GetPropagatedClientIP(ctx context.Context) (string, bool) {
	s, ok := ctx.Value(propagatedClientIPKey).(string)
	return s, ok && s != ""
}
