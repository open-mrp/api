package tracing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

func TestGRPCSpanNameFormatter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		method   string
		expected string
	}{
		{
			name:     "full method with package",
			method:   "/auth.AuthService/Login",
			expected: "grpc.login",
		},
		{
			name:     "multi word method",
			method:   "/auth.AuthService/RevokeRefreshToken",
			expected: "grpc.revoke_refresh_token",
		},
		{
			name:     "method only",
			method:   "/Login",
			expected: "grpc.login",
		},
		{
			name:     "empty method",
			method:   "/",
			expected: "grpc.unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := grpcSpanNameFormatter(tt.method); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestRenamingClientHandlerSetsGrpcPrefix(t *testing.T) {
	// Not parallel: mutates global otel tracer provider.
	origTP := otel.GetTracerProvider()
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	handler := newRenamingClientHandler()
	ctx := context.Background()

	ctx = handler.TagRPC(ctx, &stats.RPCTagInfo{FullMethodName: "/auth.AuthService/Login"})
	handler.HandleRPC(ctx, &stats.End{BeginTime: time.Now(), EndTime: time.Now()})

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, "grpc.login", spans[0].Name())
}

func TestIsHealthCheckMethod(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		method   string
		expected bool
	}{
		{
			name:     "standard health check method",
			method:   "/grpc.health.v1.Health/Check",
			expected: true,
		},
		{
			name:     "health check with Health/Check pattern",
			method:   "/some.package.Health/Check",
			expected: true,
		},
		{
			name:     "non-health check method",
			method:   "/auth.AuthService/Login",
			expected: false,
		},
		{
			name:     "method with Health but not Check",
			method:   "/auth.HealthService/GetStatus",
			expected: false,
		},
		{
			name:     "method with Check but not Health",
			method:   "/auth.AuthService/CheckToken",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHealthCheckMethod(tt.method); got != tt.expected {
				t.Fatalf("expected %v, got %v for method %q", tt.expected, got, tt.method)
			}
		})
	}
}

func TestRenamingClientHandlerSkipsHealthCheck(t *testing.T) {
	// Not parallel: mutates global otel tracer provider.
	origTP := otel.GetTracerProvider()
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	handler := newRenamingClientHandler()
	ctx := context.Background()

	// Health check should not create a span
	ctx = handler.TagRPC(ctx, &stats.RPCTagInfo{FullMethodName: "/grpc.health.v1.Health/Check"})
	handler.HandleRPC(ctx, &stats.End{BeginTime: time.Now(), EndTime: time.Now()})

	spans := spanRecorder.Ended()
	require.Len(t, spans, 0, "health check should not create telemetry spans")
}

func TestRenamingClientHandlerStillTracesNonHealthCheckMethods(t *testing.T) {
	// Not parallel: mutates global otel tracer provider.
	origTP := otel.GetTracerProvider()
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	handler := newRenamingClientHandler()
	ctx := context.Background()

	// Non-health check method should still create a span
	ctx = handler.TagRPC(ctx, &stats.RPCTagInfo{FullMethodName: "/auth.AuthService/Login"})
	handler.HandleRPC(ctx, &stats.End{BeginTime: time.Now(), EndTime: time.Now()})

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1, "non-health check methods should still create telemetry spans")
	require.Equal(t, "grpc.login", spans[0].Name())
}

func TestToSnakeLower(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "pascal case",
			input:    "LoginUser",
			expected: "login_user",
		},
		{
			name:     "camel case",
			input:    "loginUser",
			expected: "login_user",
		},
		{
			name:     "consecutive capitals stay joined",
			input:    "GetHTTPStatus",
			expected: "get_httpstatus",
		},
		{
			name:     "already snake case",
			input:    "already_snake",
			expected: "already_snake",
		},
		{
			name:     "underscore before capital adds no separator",
			input:    "get_Status",
			expected: "get_status",
		},
		{
			name:     "single letter",
			input:    "A",
			expected: "a",
		},
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := toSnakeLower(tt.input); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestUnarySpanRenamer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		fullMethod   string
		expectedName string
	}{
		{
			name:         "renames ambient span",
			fullMethod:   "/auth.AuthService/RevokeRefreshToken",
			expectedName: "grpc.revoke_refresh_token",
		},
		{
			name:         "health check is left untouched",
			fullMethod:   "/grpc.health.v1.Health/Check",
			expectedName: "original",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			spanRecorder := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
			ctx, span := tp.Tracer("test").Start(context.Background(), "original")

			handlerCalled := false
			resp, err := UnarySpanRenamer()(ctx, "req", &grpc.UnaryServerInfo{FullMethod: tt.fullMethod}, func(context.Context, any) (any, error) {
				handlerCalled = true
				return "resp", nil
			})
			span.End()

			require.NoError(t, err)
			require.Equal(t, "resp", resp)
			require.True(t, handlerCalled)

			spans := spanRecorder.Ended()
			require.Len(t, spans, 1)
			require.Equal(t, tt.expectedName, spans[0].Name())
		})
	}
}

func TestUnarySpanRenamerWithoutSpanIsNoop(t *testing.T) {
	t.Parallel()
	handlerErr := errors.New("handler failed")

	var resp any
	var err error
	require.NotPanics(t, func() {
		resp, err = UnarySpanRenamer()(context.Background(), "req", &grpc.UnaryServerInfo{FullMethod: "/auth.AuthService/Login"}, func(context.Context, any) (any, error) {
			return nil, handlerErr
		})
	}, "an RPC with no ambient span must not crash the server")

	require.Nil(t, resp)
	require.ErrorIs(t, err, handlerErr)
}
