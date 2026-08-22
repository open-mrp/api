package tracing

import (
	"context"
	"testing"

	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTraceAddsAPIErrorEventWithoutException(t *testing.T) {
	t.Parallel()
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	tracer := tracerProvider.Tracer("test")

	_, span := tracer.Start(context.Background(), "test-span")
	apiErr := &apierror.APIError{
		Code:            apierror.ErrorCodeInvalidCredentials,
		Type:            apierror.ErrorTypeInvalidRequest,
		PublicMessage:   "Invalid credentials.",
		InternalMessage: "invalid login attempt",
	}

	returnedErr := Trace(span, apiErr)
	require.Same(t, apiErr, returnedErr)

	span.End()

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)

	recordedSpan := spans[0]
	hasAPIErrorEvent := false
	for _, event := range recordedSpan.Events() {
		if event.Name == "api.error" {
			hasAPIErrorEvent = true
		}
		if event.Name == "exception" {
			t.Fatalf("unexpected exception event recorded: %+v", event)
		}
	}

	require.True(t, hasAPIErrorEvent, "api.error event missing")
	require.Equal(t, codes.Error, recordedSpan.Status().Code)
}

func TestTraceSkipsNilError(t *testing.T) {
	t.Parallel()
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	tracer := tracerProvider.Tracer("test")

	_, span := tracer.Start(context.Background(), "nil-error")
	returnedErr := Trace(span, nil)
	require.Nil(t, returnedErr)
	span.End()

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)

	recordedSpan := spans[0]
	require.Empty(t, recordedSpan.Events(), "no events should be recorded")
	require.Equal(t, codes.Unset, recordedSpan.Status().Code)
}
