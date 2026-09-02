---
name: canonical-logs
description: >-
  Canonical log line fields and the gRPC interceptor chain (CanonicalLog must be last).
  Use when touching logging, tracing, slog, request IDs, or shared/contracts interceptor
  order in a gRPC service.
---

# Canonical logs

Every gRPC unary call emits exactly one INFO `slog` record with `type=canonical-log-line`. Human spec: `docs/patterns/canonical-log-patterns.md`.

| Field | Notes |
|---|---|
| `type` | always `canonical-log-line` |
| `grpc_method` / `grpc_code` | full method, status name |
| `duration_ms` | fractional ms |
| `request_id` | from context if present |
| `auth_type` | `user` / `api_key` / `agent` / `unauthenticated` |
| `user_id` / `key_id` / `agent_id` | actor id for that auth_type |
| `target_account_id` / `account_mode` | if present (`prod`/`test`) |
| `trace_id` / `span_id` | if the span is recording |
| `error` | if the handler returned one |

## Interceptor order

`shared/contracts/grpc_server.go` (`GRPCServerConfig.WithDefaults`):

1. SpanRenamer
2. Recovery
3. Identity
4. IdempotencyKey
5. RequestID
6. ClientIP
7. **CanonicalLog — must be last**

It reads `request_id`, identity, and `account_mode` that upstream interceptors set. Last in the chain = innermost wrapper: run the handler, then log. Implementation: `shared/logging/canonical.go`.
