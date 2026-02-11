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

// LoggerHandler is a slog.Handler decorator that automatically injects "trace_id"
// and "span_id" attributes from the context into every log record before delegating
// to the wrapped handler. Install it at application startup so all structured logs
// are automatically correlated with the active trace without callers needing to
// remember to use [WithTraceContext]:
//
//	handler := tracing.NewLoggerHandler(slog.NewTextHandler(os.Stdout, nil))
//	slog.SetDefault(slog.New(handler))
type LoggerHandler struct {
	handler slog.Handler
}

// NewLoggerHandler creates a LoggerHandler that decorates the given handler with
// automatic trace context injection.
func NewLoggerHandler(handler slog.Handler) *LoggerHandler {
	return &LoggerHandler{handler: handler}
}

// Enabled delegates to the wrapped handler to determine if the given log level is
// active. Trace context injection only happens in Handle, so Enabled has no overhead.
func (h *LoggerHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle enriches the log record with "trace_id" and "span_id" from the active span
// in ctx (if present and recording), then delegates to the wrapped handler. When no
// valid span exists the record is passed through unmodified.
func (h *LoggerHandler) Handle(ctx context.Context, record slog.Record) error {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		spanCtx := span.SpanContext()
		if spanCtx.IsValid() {
			record.AddAttrs(
				slog.String("trace_id", spanCtx.TraceID().String()),
				slog.String("span_id", spanCtx.SpanID().String()),
			)
		}
	}
	return h.handler.Handle(ctx, record)
}

// WithAttrs returns a new LoggerHandler wrapping a copy of the inner handler that
// has the given attributes pre-applied. Preserves the trace-injection behavior.
func (h *LoggerHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LoggerHandler{handler: h.handler.WithAttrs(attrs)}
}

// WithGroup returns a new LoggerHandler wrapping a copy of the inner handler scoped
// to the given group. Trace IDs are injected at the record's top level, outside
// the group, so they remain consistently accessible.
func (h *LoggerHandler) WithGroup(name string) slog.Handler {
	return &LoggerHandler{handler: h.handler.WithGroup(name)}
}
