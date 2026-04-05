# PATCH /v1/core/product-line-access/account-groups/{account_group_id}

## Result: Parity Confirmed

No issues found. The Go implementation faithfully reproduces all Dashboard business logic.

## What Was Compared

- **Permission checks**: Both require internal actor + `productLineAccess.update` permission + target account ID
- **Validation**: Both verify account group exists/belongs to account, existing mappings exist, and each product line exists/belongs to account
- **DB operations**: Both use delete-all-then-insert-new pattern for replacing product line associations, then re-fetch the updated record
- **Error handling**: Error types and messages are functionally equivalent
- **Response shape**: Go returns proper API resource with `object` fields, timestamps, and sub-resource structure per conventions
- **Side effects**: None in either implementation (no emails, webhooks, messages)
- **Idempotency**: Go properly uses idempotency keys for this PATCH endpoint (improvement over Dashboard)

## Acceptable Differences

1. **Missing mapping error type**: Dashboard returns 409 Conflict; Go returns 404 Not Found. Go's 404 is more semantically correct for "resource not found."
2. **Required field validation**: Dashboard checks `!data.productLines` in the repository (400 Bad Request). Go validates via `validate:"required"` on the request struct at the gateway layer — functionally equivalent, caught earlier in the request lifecycle.
3. **Idempotency**: Go adds idempotency key support per project conventions. Dashboard had none.

## Files Reviewed

### Dashboard
- `dashboard/apps/api/src/controllers/account-group-product-line-access.ctrl.ts`
- `dashboard/apps/api/src/services/account-group-product-line-access.svc.ts`
- `dashboard/apps/api/src/repositories/account-group-product-line-access.repo.ts`
- `dashboard/packages/dtos/src/sections/account-group-product-line-access.ts`

### Go
- `services/api-gateway/endpoints/account-group-product-line-access/endpoint_update_account_group_product_line_access.go`
- `services/api-gateway/endpoints/account-group-product-line-access/service.go`
- `services/api-gateway/endpoints/account-group-product-line-access/presenter.go`
- `services/api-gateway/pkg/resource/account_group_product_line_access_resource.go`
- `services/core-service/internal/service/account_group_product_line_access_service.go`
- `services/core-service/internal/infrastructure/repository/account_group_product_line_access_repository.go`
- `services/core-service/internal/infrastructure/grpc/grpc_account_group_product_line_access_handler.go`
