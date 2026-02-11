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

// WithTracingInterceptors returns the gRPC server options needed to instrument
// inbound RPCs with OpenTelemetry tracing. It installs an otelgrpc stats handler
// that creates a server span for every non-health-check RPC. Combine the returned
// options with any other server options when calling grpc.NewServer.
func WithTracingInterceptors() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.StatsHandler(newServerHandler()),
	}
}

// DialOptionsWithTracing returns the gRPC dial options needed to instrument
// outbound RPCs with OpenTelemetry tracing. It installs a renaming client stats
// handler that creates a client span for every non-health-check RPC and immediately
// renames it to the "grpc.<snake_case_method>" convention.
func DialOptionsWithTracing() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithStatsHandler(newRenamingClientHandler()),
	}
}

// newServerHandler creates an otelgrpc server stats handler that creates a span for
// each inbound RPC. Health-check RPCs (grpc.health.v1.Health/Check) are filtered
// out to avoid noisy traces from Kubernetes probes.
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

// grpcSpanNameFormatter converts a gRPC full method name (e.g.
// "/auth.v1.AuthService/LoginUser") into a concise span name following the
// "grpc.<snake_case_method>" convention (e.g. "grpc.login_user"). It strips the
// leading slash and package/service prefix, then converts the remaining PascalCase
// method name to snake_case via [toSnakeLower].
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

// isHealthCheckMethod returns true if fullMethodName matches the standard gRPC
// health checking protocol ("/grpc.health.v1.Health/Check") or any variant
// containing "Health/Check". Used by both server and client handlers to suppress
// tracing for Kubernetes liveness/readiness probes.
func isHealthCheckMethod(fullMethodName string) bool {
	return strings.Contains(fullMethodName, "/grpc.health.v1.Health/Check") ||
		strings.Contains(fullMethodName, "Health/Check")
}

// toSnakeLower converts a PascalCase or camelCase string to snake_case. It inserts
// an underscore before each uppercase letter that follows a non-uppercase,
// non-underscore character (e.g. "LoginUser" → "login_user", "GetHTTPStatus" →
// "get_httpstatus"). Used by grpcSpanNameFormatter to normalize method names.
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

// UnarySpanRenamer returns a gRPC unary server interceptor that renames the span
// created by the otelgrpc stats handler to the "grpc.<snake_case_method>" format.
// The stats handler creates spans with the raw "/package.Service/Method" name;
// this interceptor normalizes them so the trace backend shows concise, consistent
// names (e.g. "grpc.login_user" instead of "/auth.v1.AuthService/LoginUser").
// Health-check RPCs are skipped since they are already filtered out at the stats
// handler level.
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

// renamingClientHandler wraps otelgrpc's client stats handler to rename outbound
// RPC spans immediately after creation (in TagRPC) so they follow the
// "grpc.<snake_case_method>" naming convention in the trace backend. All other
// stats.Handler methods delegate directly to the wrapped base handler.
type renamingClientHandler struct {
	base stats.Handler
}

// newRenamingClientHandler creates a client stats handler that wraps otelgrpc's
// default client handler with span renaming. Health-check RPCs are filtered out.
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

// TagRPC delegates to the base handler (which creates the client span), then
// renames the span to the "grpc.<snake_case_method>" convention.
func (h *renamingClientHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	ctx = h.base.TagRPC(ctx, info)
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		span.SetName(grpcSpanNameFormatter(info.FullMethodName))
	}
	return ctx
}

// HandleRPC delegates RPC stats events (message sent/received, etc.) to the base handler.
func (h *renamingClientHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	h.base.HandleRPC(ctx, rs)
}

// TagConn delegates connection tagging to the base handler.
func (h *renamingClientHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return h.base.TagConn(ctx, info)
}

// HandleConn delegates connection stats events to the base handler.
func (h *renamingClientHandler) HandleConn(ctx context.Context, cs stats.ConnStats) {
	h.base.HandleConn(ctx, cs)
}
