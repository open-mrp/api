// Package grpclog provides structured logging for gRPC services. Its primary export is [CanonicalLogInterceptor], which emits a single "canonical log line" for every gRPC call - a structured slog record containing the method name, gRPC status code, response duration, caller identity, request ID, and trace context. These canonical lines serve as the authoritative audit trail for all service-to-service traffic and are designed to be queried in log aggregation backends (e.g. Datadog, Loki) for latency analysis, error investigation, and access auditing.
package grpclog

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/open-mrp/apikit/logging"
)

// CanonicalLogInterceptor returns a gRPC unary server interceptor that emits one structured log line per RPC at INFO level. The log message is the full gRPC method name (e.g. "/auth.v1.AuthService/LoginUser") and the record includes:
//
//   - type:              always "canonical-log-line" (for log-query filtering)
//   - grpc_method:       full gRPC method name
//   - grpc_code:         gRPC status code string (e.g. "OK", "NotFound")
//   - duration_ms:       handler execution time in fractional milliseconds
//   - request_id:        from [appctx.GetRequestID], if present in context
//   - auth_type:         identity type (user, api_key, agent, unauthenticated)
//   - user_id / key_id / agent_id: actor ID, depending on auth type
//   - target_account_id: account scope, if present
//   - account_mode:      production / test, if present
//   - trace_id, span_id: from the active OpenTelemetry span, if recording
//   - error:             error message string, if the handler returned an error
//
// This interceptor must be placed at the end of the interceptor chain so that upstream interceptors (identity extraction, request-ID propagation) have already populated the context values it reads.
func CanonicalLogInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		attrs := buildCanonicalAttrs(ctx, info.FullMethod, err, duration)

		logger.LogAttrs(ctx, slog.LevelInfo, info.FullMethod, attrs...)
		return resp, err
	}
}

// buildCanonicalAttrs assembles the slog attributes for a canonical log line. It always includes the base fields (type, grpc_method, grpc_code, duration_ms) and conditionally appends request_id, identity fields, trace IDs, and the error message based on what is available in the context and whether the handler errored. Duration is recorded as fractional milliseconds (microsecond precision) for consistency with trace-backend conventions.
func buildCanonicalAttrs(ctx context.Context, method string, err error, duration time.Duration) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("type", logging.TypeCanonicalLogLine),
		slog.String("grpc_method", method),
		slog.String("grpc_code", status.Code(err).String()),
		slog.Float64("duration_ms", float64(duration.Microseconds())/1000.0),
	}

	attrs = append(attrs, logging.RequestAttrs(ctx)...)

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	return attrs
}
