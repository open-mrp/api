package tracing

import (
	"net/http"
	"runtime/debug"

	"github.com/augno/api/shared/contracts"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Trace records the error and then returns it
func Trace(span trace.Span, err *contracts.APIError) *contracts.APIError {
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
		attrs = append(attrs, attribute.String("error.stacktrace", string(debug.Stack())))
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

	span.AddEvent("api.error", trace.WithAttributes(attrs...))
	span.SetStatus(codes.Error, err.Error())

	return err
}

func shouldCaptureStacktrace(apiErr *contracts.APIError) bool {
	if apiErr == nil {
		return false
	}

	// Panics are translated into internal server errors, so they naturally
	// fall under this condition.
	return contracts.GetHTTPStatusCode(apiErr.Code) >= http.StatusInternalServerError
}
