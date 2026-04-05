package tracing

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// WithTraceContext returns a derived slog.Logger that includes "trace_id" and
// "span_id" attributes from the active span in ctx. Every log entry produced by
// the returned logger will carry these fields, making it easy to correlate logs
// with traces in an observability backend.
//
// If the context has no recording span (e.g. tracing is disabled or there is no
// active span), the original logger is returned unchanged.
func WithTraceContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return logger
	}

	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		return logger
	}

	traceID := spanCtx.TraceID().String()
	spanID := spanCtx.SpanID().String()

	// Return a logger with trace context attributes
	return logger.With(
		slog.String("trace_id", traceID),
		slog.String("span_id", spanID),
	)
}

// LogWithTraceContext is a convenience wrapper that enriches logger with trace IDs
// from ctx (via [WithTraceContext]) and then emits a single log record at the given
// level. Use this for one-off log calls where creating a persistent traced logger
// via WithTraceContext would be wasteful.
func LogWithTraceContext(ctx context.Context, logger *slog.Logger, level slog.Level, msg string, args ...any) {
	logger = WithTraceContext(ctx, logger)
	logger.Log(ctx, level, msg, args...)
}
