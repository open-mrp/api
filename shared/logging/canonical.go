// Package logging provides structured logging utilities for gRPC services. Its
// primary export is [CanonicalLogInterceptor], which emits a single "canonical log
// line" for every gRPC call - a structured slog record containing the method name,
// gRPC status code, response duration, caller identity, request ID, and trace
// context. These canonical lines serve as the authoritative audit trail for all
// service-to-service traffic and are designed to be queried in log aggregation
// backends (e.g. Datadog, Loki) for latency analysis, error investigation, and
// access auditing.
package logging

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/appctx"
)

// CanonicalLogInterceptor returns a gRPC unary server interceptor that emits one
// structured log line per RPC at INFO level. The log message is the full gRPC method
// name (e.g. "/auth.v1.AuthService/LoginUser") and the record includes:
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
// This interceptor must be placed at the end of the interceptor chain so that
// upstream interceptors (identity extraction, request-ID propagation) have already
// populated the context values it reads.
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

// buildCanonicalAttrs assembles the slog attributes for a canonical log line. It
// always includes the base fields (type, grpc_method, grpc_code, duration_ms) and
// conditionally appends request_id, identity fields, trace IDs, and the error
// message based on what is available in the context and whether the handler errored.
// Duration is recorded as fractional milliseconds (microsecond precision) for
// consistency with trace-backend conventions.
func buildCanonicalAttrs(ctx context.Context, method string, err error, duration time.Duration) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("type", "canonical-log-line"),
		slog.String("grpc_method", method),
		slog.String("grpc_code", status.Code(err).String()),
		slog.Float64("duration_ms", float64(duration.Microseconds())/1000.0),
	}

	if requestID, ok := appctx.GetRequestID(ctx); ok {
		attrs = append(attrs, slog.String("request_id", requestID))
	}

	if identity, ok := appctx.GetIdentityFromContext(ctx); ok && identity != nil {
		attrs = append(attrs, extractIdentityAttrs(identity)...)
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		spanCtx := span.SpanContext()
		if spanCtx.IsValid() {
			attrs = append(attrs,
				slog.String("trace_id", spanCtx.TraceID().String()),
				slog.String("span_id", spanCtx.SpanID().String()),
			)
		}
	}

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	return attrs
}

// extractIdentityAttrs converts the caller's Identity into slog attributes for the
// canonical log line. The attributes vary by identity type:
//
//   - User identity:    auth_type="user",    user_id="usr_..."
//   - API key identity: auth_type="api_key", key_id="apke_..."
//   - Agent identity:   auth_type="agent",   agent_id=<agent definition ID>
//   - Unauthenticated:  auth_type="unauthenticated" (no actor ID)
//
// Optional fields (target_account_id, account_mode) are included only when non-empty
// to keep log lines compact for unauthenticated or account-less calls.
func extractIdentityAttrs(identity *types.Identity) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("auth_type", string(identity.Type)),
	}

	if identity.Actor != nil {
		switch identity.Type {
		case types.IdentityActorTypeUser:
			attrs = append(attrs, slog.String("user_id", identity.Actor.ID))
		case types.IdentityActorTypeAPIKey:
			attrs = append(attrs, slog.String("key_id", identity.Actor.ID))
		case types.IdentityActorTypeAgent:
			attrs = append(attrs, slog.String("agent_id", identity.Actor.ID))
		}
	}

	if identity.Target != nil && identity.Target.AccountID != "" {
		attrs = append(attrs, slog.String("target_account_id", identity.Target.AccountID))
	}

	if identity.AccountMode != "" {
		attrs = append(attrs, slog.String("account_mode", string(identity.AccountMode)))
	}

	return attrs
}
