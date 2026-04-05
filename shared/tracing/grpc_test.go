package tracing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
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
