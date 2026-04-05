# Migration Verification: PATCH /v1/core/product-types/{id}

## Result: No Dashboard Counterpart — Go Implementation Reviewed

The Dashboard API has **no update product type endpoint**. The `ProductTypeSvc` is empty (constructor only), `ProductTypeRepo` only has read operations (`list`, `find`, `doesExist`), and no controller or route registration exists for product type mutations. Product types were read-only in the legacy API.

This Go endpoint is **new functionality**, not a migration. No parity verification is applicable.

## Go Implementation Review

The Go implementation was reviewed for correctness against project patterns:

### What Was Compared
- **Endpoint definition** (`endpoint_update_product_type.go`): PATCH `/v1/core/product-types/{id}`, accepts optional `name` and `code`
- **gRPC handler** (`grpc_handler.go`): Proper nil check, idempotency tracking, param mapping
- **Service layer** (`product_type_service.go:189-263`): Identity checks, permission checks, idempotency with recovery points, uniqueness validation, transactional update
- **Repository layer** (`product_type_repository.go:151-173`): COALESCE-based partial update, duplicate key mapping, rows-affected check, re-fetch after update
- **SQL queries** (`product_type.sql`): `UPDATE ... SET name = COALESCE(?, name), code = COALESCE(?, code)` with `updated_at = NOW(3)`

### Verification Details
- **Validation**: Name and code uniqueness checked (excluding current ID) before update
- **Permissions**: Internal actor required + `PermissionDomainProductTypes` / `ActionUpdate`
- **Idempotency**: Full idempotency key support with cached responses and recovery points
- **Error handling**: 404 (not found), 409 (duplicate name/code), auth errors
- **Response shape**: Returns full `ProductType` resource (`id`, `object`, `name`, `code`, `created_at`, `updated_at`)
- **Side effects**: None (no messages, webhooks, or cascading changes)

### Issues Found
None. The implementation follows all project patterns correctly.

### Remaining Concerns
None. This is new functionality not present in the Dashboard API.
