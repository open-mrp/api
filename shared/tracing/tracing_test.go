package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/augno/api/shared/contracts"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestRecordControllerErrorAPIErrorAddsApiErrorEvent(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	tracer := tp.Tracer("test")

	_, span := tracer.Start(context.Background(), "controller-span")
	apiErr := &contracts.APIError{
		Code:          contracts.ErrorCodeInvalidCredentials,
		Type:          contracts.ErrorTypeInvalidRequest,
		PublicMessage: "This refresh token has been revoked.",
	}

	RecordControllerError(span, apiErr)
	span.End()

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)

	recorded := spans[0]
	require.Equal(t, codes.Error, recorded.Status().Code)
	require.Equal(t, apiErr.PublicMessage, recorded.Status().Description)

	foundEvent := false
	for _, event := range recorded.Events() {
		if event.Name != "api.error" {
			continue
		}
		foundEvent = true
		attrMap := attrsToMap(event.Attributes)
		require.Equal(t, string(apiErr.Code), attrMap["error.code"])
		require.Equal(t, string(apiErr.Type), attrMap["error.type"])
		require.Equal(t, apiErr.PublicMessage, attrMap["error.public_message"])
	}
	require.True(t, foundEvent, "expected api.error event")
}

func TestRecordControllerErrorNilDoesNothing(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	tracer := tp.Tracer("test")

	_, span := tracer.Start(context.Background(), "no-error")
	RecordControllerError(span, nil)
	span.End()

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, codes.Unset, spans[0].Status().Code)
	require.Empty(t, spans[0].Events())
}

func TestRecordControllerErrorNonAPIErrorRecordsException(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	tracer := tp.Tracer("test")

	_, span := tracer.Start(context.Background(), "generic-error")
	err := errors.New("something bad happened")
	RecordControllerError(span, err)
	span.End()

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)

	recorded := spans[0]
	require.Equal(t, codes.Error, recorded.Status().Code)
	require.Equal(t, err.Error(), recorded.Status().Description)

	foundException := false
	for _, event := range recorded.Events() {
		if event.Name == "exception" {
			foundException = true
			break
		}
	}
	require.True(t, foundException, "expected exception event for generic error")
}

func attrsToMap(attrs []attribute.KeyValue) map[string]string {
	result := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		result[string(attr.Key)] = attr.Value.AsString()
	}
	return result
}
