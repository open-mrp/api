package tracing

import (
	"context"
	"strings"
	"unicode"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

func WithTracingInterceptors() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.StatsHandler(newServerHandler()),
	}
}

func DialOptionsWithTracing() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithStatsHandler(newRenamingClientHandler()),
	}
}

func newServerHandler() stats.Handler {
	opts := []otelgrpc.Option{
		otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
		otelgrpc.WithFilter(func(info *stats.RPCTagInfo) bool {
			// Skip telemetry for health check calls
			return !isHealthCheckMethod(info.FullMethodName)
		}),
	}
	return otelgrpc.NewServerHandler(opts...)
}

func grpcSpanNameFormatter(fullMethodName string) string {
	method := strings.TrimPrefix(fullMethodName, "/")
	if idx := strings.LastIndex(method, "/"); idx >= 0 && idx < len(method)-1 {
		method = method[idx+1:]
	}
	if method == "" {
		return "grpc.unknown"
	}
	return "grpc." + toSnakeLower(method)
}

// isHealthCheckMethod returns true if the method is a gRPC health check call.
func isHealthCheckMethod(fullMethodName string) bool {
	return strings.Contains(fullMethodName, "/grpc.health.v1.Health/Check") ||
		strings.Contains(fullMethodName, "Health/Check")
}

func toSnakeLower(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := rune(s[i-1])
				if prev != '_' && !unicode.IsUpper(prev) {
					b.WriteByte('_')
				}
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// UnarySpanRenamer renames the server span created by the otelgrpc stats handler
// to follow the grpc.<method> format (e.g. grpc.login).
func UnarySpanRenamer() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Skip renaming for health check calls (they shouldn't have spans anyway due to filter)
		if !isHealthCheckMethod(info.FullMethod) {
			if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
				span.SetName(grpcSpanNameFormatter(info.FullMethod))
			}
		}
		return handler(ctx, req)
	}
}

// renamingClientHandler wraps otelgrpc's client stats handler to rename spans
// immediately after they are created so recorded spans follow the grpc.<method> format.
type renamingClientHandler struct {
	base stats.Handler
}

func newRenamingClientHandler() stats.Handler {
	opts := []otelgrpc.Option{
		otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
		otelgrpc.WithFilter(func(info *stats.RPCTagInfo) bool {
			// Skip telemetry for health check calls
			return !isHealthCheckMethod(info.FullMethodName)
		}),
	}
	return &renamingClientHandler{base: otelgrpc.NewClientHandler(opts...)}
}

func (h *renamingClientHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	ctx = h.base.TagRPC(ctx, info)
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		span.SetName(grpcSpanNameFormatter(info.FullMethodName))
	}
	return ctx
}

func (h *renamingClientHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	h.base.HandleRPC(ctx, rs)
}

func (h *renamingClientHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return h.base.TagConn(ctx, info)
}

func (h *renamingClientHandler) HandleConn(ctx context.Context, cs stats.ConnStats) {
	h.base.HandleConn(ctx, cs)
}
