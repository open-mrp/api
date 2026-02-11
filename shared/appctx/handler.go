package appctx

import (
	"context"
)

const handlerKey contextKey = "handler"

// WithHandler returns a child context carrying the given handler name.
func WithHandler(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, handlerKey, method)
}

// GetHandler retrieves the handler name from the context.
func GetHandler(ctx context.Context) (string, bool) {
	s, ok := ctx.Value(handlerKey).(string)
	return s, ok && s != ""
}
