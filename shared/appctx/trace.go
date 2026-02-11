package appctx

import "context"

// TraceMetadata carries trace/span IDs through the HTTP request lifecycle.
type TraceMetadata struct {
	TraceID string
	SpanID  string
}

const traceMetadataKey contextKey = "trace_metadata"

// noTraceKey is the context key that signals tracing should be suppressed.
var noTraceKey = noTraceKeyType{}

// WithTraceMetadata returns a child context carrying the given trace metadata.
func WithTraceMetadata(ctx context.Context, meta TraceMetadata) context.Context {
	return context.WithValue(ctx, traceMetadataKey, meta)
}

// GetTraceMetadata retrieves the trace metadata from the context.
func GetTraceMetadata(ctx context.Context) (TraceMetadata, bool) {
	meta, ok := ctx.Value(traceMetadataKey).(TraceMetadata)
	return meta, ok
}

// WithNoTrace returns a derived context that suppresses span creation for the
// current call tree.
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
