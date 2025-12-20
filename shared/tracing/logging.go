package tracing

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// WithTraceContext returns a new slog.Logger that includes trace_id and span_id
// from the active span in the context for all log entries.
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

// LogWithTraceContext is a convenience function that logs with trace context.
// It extracts trace_id and span_id from the context and includes them in the log entry.
func LogWithTraceContext(ctx context.Context, logger *slog.Logger, level slog.Level, msg string, args ...any) {
	logger = WithTraceContext(ctx, logger)
	logger.Log(ctx, level, msg, args...)
}

// LoggerHandler wraps a slog.Handler to automatically include trace context.
type LoggerHandler struct {
	handler slog.Handler
}

// NewLoggerHandler creates a new LoggerHandler that wraps the given handler.
func NewLoggerHandler(handler slog.Handler) *LoggerHandler {
	return &LoggerHandler{handler: handler}
}

// Enabled reports whether the handler handles records at the given level.
func (h *LoggerHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle handles the record by adding trace context if available.
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

// WithAttrs returns a new handler with the given attributes.
func (h *LoggerHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LoggerHandler{handler: h.handler.WithAttrs(attrs)}
}

// WithGroup returns a new handler with the given group.
func (h *LoggerHandler) WithGroup(name string) slog.Handler {
	return &LoggerHandler{handler: h.handler.WithGroup(name)}
}
