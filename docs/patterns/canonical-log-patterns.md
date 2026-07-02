# Canonical Log Patterns

This document describes the canonical log line system used by all gRPC services.

## Overview

Every gRPC unary call emits exactly one structured `slog` record at INFO level — the "canonical log line." These lines serve as the authoritative audit trail for all service-to-service traffic and are designed to be queried in log aggregation backends (Datadog, Loki, etc.).

## Log Fields

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"canonical-log-line"` (for filtering) |
| `grpc_method` | string | Full method name, e.g. `/auth.v1.AuthService/LoginUser` |
| `grpc_code` | string | gRPC status code, e.g. `"OK"`, `"NotFound"` |
| `duration_ms` | float64 | Handler execution time in fractional milliseconds |
| `request_id` | string | From context, if present |
| `auth_type` | string | `"user"`, `"api_key"`, `"agent"`, or `"unauthenticated"` |
| `user_id` | string | Actor ID when auth_type is `"user"` |
| `key_id` | string | Actor ID when auth_type is `"api_key"` |
| `agent_id` | string | Actor ID when auth_type is `"agent"` |
| `target_account_id` | string | Account scope, if present |
| `account_mode` | string | `"prod"` or `"test"`, if present |
| `trace_id` | string | OpenTelemetry trace ID, if span is recording |
| `span_id` | string | OpenTelemetry span ID, if span is recording |
| `error` | string | Error message, if handler returned an error |

## Default Interceptor Chain

The interceptor chain is defined in `shared/contracts/grpc_server.go` (`GRPCServerConfig.WithDefaults`):

```
1. SpanRenamer       — renames OTel spans to gRPC method name
2. Recovery          — recovers from panics
3. Identity          — extracts identity from gRPC metadata into context
4. IdempotencyKey    — extracts idempotency key from metadata into context
5. RequestID         — extracts or generates request ID into context
6. ClientIP          — extracts the client IP from metadata into context
7. CanonicalLog      — emits the canonical log line (MUST BE LAST)
```

### Why CanonicalLog Must Be Last

The canonical log interceptor reads context values (`request_id`, `identity`, `account_mode`) that are set by upstream interceptors. If it runs before them, those fields will be missing from the log line.

The interceptor wraps the handler call, so "last in the chain" means it is the innermost wrapper — it runs the handler and then logs the result with all context values already populated.

## Reference Files

- `shared/logging/canonical.go` — interceptor implementation and attribute builder
- `shared/contracts/grpc_server.go` — default interceptor chain configuration
- `shared/appctx/` — context value getters used by the canonical logger
