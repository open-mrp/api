package apicontext

import "context"

type TraceMetadata struct {
	TraceID string
	SpanID  string
}

const traceMetadataKey contextKey = "trace_metadata"

func WithTraceMetadata(ctx context.Context, meta TraceMetadata) context.Context {
	return context.WithValue(ctx, traceMetadataKey, meta)
}

func GetTraceMetadata(ctx context.Context) (TraceMetadata, bool) {
	meta, ok := ctx.Value(traceMetadataKey).(TraceMetadata)
	return meta, ok
}
