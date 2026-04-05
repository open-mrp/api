# POST /v1/core/registration-flows — Migration Verification

## Result: PARITY CONFIRMED

No issues found. The Go implementation faithfully reproduces the Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **Permission check** | `account` domain / `update` action | `PermissionDomainAccount` / `ActionUpdate` | Yes |
| **Actor check** | `checkIsInternalActor` | `CheckIsInternalActor` | Yes |
| **Target account** | `identity.targetAccountID` | `identity.TargetAccountID` (checked via `CheckTargetAccountSet`) | Yes |
| **Required fields** | `name` required; option arrays required by schema | `name` required via `validate:"required"` tag | Yes |
| **ID generation** | Client-supplied `data.id` (TypeID `rf_*`) | Server-generated via `id.GenID(id.RegistrationFlowIDPrefix)` | OK (Go convention) |
| **Request shape** | `{name, customerGroupOptions: [{id,name}], paymentTermOptions: [{id,name}], shippingTermOptions: [{id,name}]}` | `{name, customer_group_ids: [string], payment_term_ids: [string], shipping_term_ids: [string]}` | OK (Go only needs IDs to create associations) |
| **DB: insert flow** | Prisma `create` with `id, name, accountID` | `InsertRegistrationFlow` SQL with `id, name, account_id, NOW(3), NOW(3)` | Yes |
| **DB: connect options** | `PrismaUtils.connect` for customerGroups, paymentTerms, shippingTerms | `writeOptions`: inserts into junction tables + sets `registration_flow_id` on account_group | Yes |
| **Response shape** | `{id, name, customerGroupOptions, paymentTermOptions, shippingTermOptions, createdAt, updatedAt}` | Same fields + `object` field (per Go API conventions) | Yes |
| **Option response shape** | `{id, name}` | `{id, object, name}` (adds `object` per conventions) | Yes |
| **Status code** | 201 Created | 201 Created | Yes |
| **Idempotency** | Not implemented in Dashboard | Full idempotency key support with recovery points | Correct (Go pattern for POST) |
| **Side effects** | None | None | Yes |
| **Error handling** | Prisma errors propagated | `db.MapSQLError` for SQL errors | Yes |

## Files Reviewed

### Dashboard
- `dashboard/apps/api/src/services/registration-flow.svc.ts` — Service layer (`create` method)
- `dashboard/apps/api/src/repositories/registration-flow.repo.ts` — Repository (`create` method)
- `dashboard/apps/api/src/controllers/registration-flow.ctrl.ts` — Controller (`createRegistrationFlow`)

### Go
- `services/api-gateway/endpoints/registration-flows/endpoint_create_registration_flow.go` — Endpoint definition
- `services/api-gateway/endpoints/registration-flows/service.go` — API gateway service (gRPC call)
- `services/api-gateway/endpoints/registration-flows/presenter.go` — Response presenter
- `services/api-gateway/pkg/resource/registration_flow_resource.go` — API resource definition
- `services/core-service/internal/infrastructure/grpc/registration_flow_handler.go` — gRPC handler
- `services/core-service/internal/service/registration_flow_service.go` — Domain service (`CreateRegistrationFlow`)
- `services/core-service/internal/infrastructure/repository/registration_flow_repository.go` — Repository (`Create`, `writeOptions`)
- `services/core-service/internal/infrastructure/queries/registration_flow.sql` — SQL queries
- `services/core-service/internal/domain/registration_flow_models.go` — Domain models

## Issues Found and Fixed

None — no changes were needed.

## Notes

- The Go API accepts option IDs as flat string arrays (`customer_group_ids`, `payment_term_ids`, `shipping_term_ids`) rather than objects with `{id, name}`. This is the correct Go API convention since names are resolved from the DB when building the response.
- The Go API adds `object` type fields to all resources, which is expected per API conventions.
- Idempotency support was correctly added in Go (required for POST endpoints per architecture patterns).
