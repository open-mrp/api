# PATCH /v1/core/registration-flows/{id}

## Result: Issues found and fixed

## What was compared

- **Validation rules**: Both implementations validate path ID and optional body fields (name, customer group IDs, payment term IDs, shipping term IDs)
- **Permission checks**: Both require internal actor + `account:update` permission + target account ID header — match confirmed
- **DB queries**: Name update uses COALESCE to preserve existing value when not provided — matches Prisma partial update semantics
- **Error handling**: 404 on missing flow (rows affected check), permission errors, identity errors — all match
- **Side effects**: Neither implementation has side effects (no emails, webhooks, etc.) — match confirmed
- **Response shape**: Both return full registration flow with nested customer group, payment term, and shipping term options — match confirmed
- **Idempotency**: Go implementation correctly uses idempotency keys with recovery points for the PATCH endpoint

## Issue found and fixed

**Bug: Options always cleared on partial update**

The Go repository's `Update` method unconditionally called `clearOptions()` + `writeOptions()`, which would delete ALL existing customer groups, payment terms, and shipping terms on every update — even when those arrays were not provided in the request body.

In the Dashboard (Prisma), `PrismaUtils.connect()` with `set: true` only replaces a relationship if the field was explicitly provided. Omitting a field leaves the existing associations untouched.

**Fix**: Added `has_customer_group_ids`, `has_payment_term_ids`, and `has_shipping_term_ids` boolean flags following the established volume discount pattern in the codebase. Options are now only cleared and rewritten when their corresponding `has_*` flag is true.

Files modified:
- `proto/core.proto` — Added `has_*` bool fields to `UpdateRegistrationFlowRequest` message
- `services/core-service/internal/domain/registration_flow_models.go` — Added `Has*` bools to `UpdateRegistrationFlowParams`
- `services/api-gateway/endpoints/registration-flows/endpoint_update_registration_flow.go` — Added `Has*` fields to request struct
- `services/api-gateway/endpoints/registration-flows/service.go` — Passes `Has*` flags to proto request
- `services/core-service/internal/infrastructure/grpc/registration_flow_handler.go` — Passes `Has*` flags to domain params
- `services/core-service/internal/infrastructure/repository/registration_flow_repository.go` — Conditionally clear/rewrite each option type only when its `Has*` flag is true
- Proto bindings regenerated via `make proto`

## No remaining concerns
