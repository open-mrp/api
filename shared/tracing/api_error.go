package tracing

import (
	"net/http"
	"runtime/debug"

	apierror "github.com/open-mrp/api/shared/errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// maxStacktraceAttrLen caps the length of the stacktrace string stored as a span attribute. Traces with very deep stacks are truncated to avoid inflating span payloads beyond what trace backends can efficiently store and display.
const maxStacktraceAttrLen = 8192

// Trace records a structured "api.error" event on the given span and sets the span status to Error. It returns the same *APIError so callers can use it inline:
//
//	return tracing.Trace(span, apierror.NewBadRequest("invalid input"))
//
// The recorded event carries attributes for every populated APIError field: error.code, error.type, error.public_message, error.internal_message, error.param, error.doc_url, error.internal_error. For 5xx errors a goroutine stacktrace is captured (truncated to [maxStacktraceAttrLen]) to aid post-mortem debugging.
//
// A nil error or nil span is handled gracefully (returns nil / no-ops respectively).
func Trace(span trace.Span, err *apierror.APIError) *apierror.APIError {
	if err == nil {
		return nil
	}

	attrs := []attribute.KeyValue{
		attribute.String("error.code", string(err.Code)),
		attribute.String("error.type", string(err.Type)),
		attribute.String("error.public_message", err.PublicMessage),
		attribute.String("error.internal_message", err.InternalMessage),
	}

	if shouldCaptureStacktrace(err) {
		stack := captureStack()
		if len(stack) > maxStacktraceAttrLen {
			stack = stack[:maxStacktraceAttrLen] + "\n... truncated"
		}
		attrs = append(attrs, attribute.String("error.stacktrace", stack))
	}

	if err.Param != "" {
		attrs = append(attrs, attribute.String("error.param", err.Param))
	}
	if err.DocURL != "" {
		attrs = append(attrs, attribute.String("error.doc_url", err.DocURL))
	}
	if err.Internal != nil {
		attrs = append(attrs, attribute.String("error.internal_error", err.Internal.Error()))
	}

	if span != nil {
		span.AddEvent("api.error", trace.WithAttributes(attrs...))
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// captureStack returns the current goroutine's stack trace as a string. It wraps debug.Stack in a deferred recover to guarantee it never panics — if the stack capture itself fails, it returns a placeholder string rather than crashing.
func captureStack() string {
	var stack string
	func() {
		defer func() {
			if recover() != nil && stack == "" {
				stack = "(stack capture panicked)"
			}
		}()
		stack = string(debug.Stack())
	}()
	return stack
}

// shouldCaptureStacktrace returns true when the error maps to an HTTP status code >= 500 (Internal Server Error). Stacktraces are expensive and only valuable for unexpected server-side failures; client errors (4xx) are expected conditions that don't warrant a stack dump. Panic-recovery errors are naturally included because they are translated to 500 status codes.
func shouldCaptureStacktrace(apiErr *apierror.APIError) bool {
	if apiErr == nil {
		return false
	}

	// Panics are translated into internal server errors, so they naturally fall under this condition.
	return apierror.GetHTTPStatusCode(apiErr.Code) >= http.StatusInternalServerError
}
