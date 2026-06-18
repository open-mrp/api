package appctx

import "context"

// noTraceKey is the context key that signals tracing should be suppressed.
var noTraceKey = noTraceKeyType{}

// WithNoTrace returns a derived context that suppresses span creation for the current call tree.
func WithNoTrace(ctx context.Context) context.Context {
	return context.WithValue(ctx, noTraceKey, true)
}

// ShouldTrace returns true if tracing has NOT been suppressed on ctx via WithNoTrace.
func ShouldTrace(ctx context.Context) bool {
	if v, ok := ctx.Value(noTraceKey).(bool); ok && v {
		return false
	}
	return true
}
