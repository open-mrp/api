package tracing

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace/noop"
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

func TestTraceCapturesStacktraceOnlyForServerErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		code           apierror.ErrorCode
		errType        apierror.ErrorType
		expectsCapture bool
	}{
		{
			name:           "internal error captures stack",
			code:           apierror.ErrorCodeInternalError,
			errType:        apierror.ErrorTypeAPI,
			expectsCapture: true,
		},
		{
			name:           "unmapped code defaults to 500 and captures stack",
			code:           apierror.ErrorCode("not_a_real_code"),
			errType:        apierror.ErrorTypeAPI,
			expectsCapture: true,
		},
		{
			name:           "validation failure skips stack",
			code:           apierror.ErrorCodeValidationFailed,
			errType:        apierror.ErrorTypeInvalidRequest,
			expectsCapture: false,
		},
		{
			name:           "not found skips stack",
			code:           apierror.ErrorCodeResourceNotFound,
			errType:        apierror.ErrorTypeInvalidRequest,
			expectsCapture: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			spanRecorder := tracetest.NewSpanRecorder()
			tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
			_, span := tracerProvider.Tracer("test").Start(context.Background(), "test-span")

			Trace(span, &apierror.APIError{Code: tt.code, Type: tt.errType, PublicMessage: "boom"})
			span.End()

			attrs := requireAPIErrorEventAttrs(t, spanRecorder)
			_, hasStack := attrs["error.stacktrace"]
			require.Equal(t, tt.expectsCapture, hasStack)
		})
	}
}

func TestTraceTruncatesOversizedStacktrace(t *testing.T) {
	t.Parallel()
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	_, span := tracerProvider.Tracer("test").Start(context.Background(), "deep-stack")

	// A stack deep enough to exceed maxStacktraceAttrLen, so the truncation branch actually runs.
	recurse(400, func() {
		Trace(span, &apierror.APIError{Code: apierror.ErrorCodeInternalError, Type: apierror.ErrorTypeAPI, PublicMessage: "boom"})
	})
	span.End()

	stack := requireAPIErrorEventAttrs(t, spanRecorder)["error.stacktrace"]
	require.Greater(t, len(stack), maxStacktraceAttrLen, "test must produce a stack past the truncation threshold")
	require.True(t, strings.HasSuffix(stack, "\n... truncated"))
	require.True(t, utf8.ValidString(stack), "invalid UTF-8 fails the whole export batch, not just this span")
}

func TestTraceNilSpanReturnsError(t *testing.T) {
	t.Parallel()
	apiErr := &apierror.APIError{Code: apierror.ErrorCodeInternalError, Type: apierror.ErrorTypeAPI, PublicMessage: "boom"}

	var returned *apierror.APIError
	require.NotPanics(t, func() { returned = Trace(nil, apiErr) })
	require.Same(t, apiErr, returned)
}

func TestTraceNonRecordingSpanReturnsError(t *testing.T) {
	t.Parallel()
	_, span := noop.NewTracerProvider().Tracer("test").Start(context.Background(), "non-recording")
	require.False(t, span.IsRecording())

	apiErr := &apierror.APIError{Code: apierror.ErrorCodeInternalError, Type: apierror.ErrorTypeAPI, PublicMessage: "boom"}
	require.Same(t, apiErr, Trace(span, apiErr))
}

func recurse(depth int, fn func()) {
	if depth <= 0 {
		fn()
		return
	}
	recurse(depth-1, fn)
}

func requireAPIErrorEventAttrs(t *testing.T, recorder *tracetest.SpanRecorder) map[string]string {
	t.Helper()
	spans := recorder.Ended()
	require.Len(t, spans, 1)
	for _, event := range spans[0].Events() {
		if event.Name == "api.error" {
			return attrsToMap(event.Attributes)
		}
	}
	t.Fatal("api.error event missing")
	return nil
}
