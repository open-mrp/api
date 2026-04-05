# Migration Verification: GET /v1/core/product-types/{id}

## Result: Parity Confirmed

No issues found. The Go implementation correctly matches the Dashboard behavior.

## What Was Compared

### Validation
- **Dashboard**: No validation beyond Prisma `findUnique` by ID
- **Go**: ID extracted from path parameter, passed directly to SQL query
- **Verdict**: Match — both simply look up by ID

### Permission Checks
- **Dashboard**: No active controller; ProductTypeSvc is empty (inherits BaseSvc with no methods). No permission checks enforced.
- **Go**: Checks `CheckIsInternalActor()` and `CheckHasPermission(PermissionDomainProductTypes, ActionRead)`
- **Verdict**: Go adds proper permission checks, which is an expected improvement over the legacy code that had no active endpoint

### DB Query
- **Dashboard**: `db.productType.findUnique({ where: { id }, select: { id, code, name, createdAt, updatedAt } })`
- **Go**: `SELECT id, name, code, created_at, updated_at FROM product_type WHERE id = ?`
- **Verdict**: Match — same table, same columns, same lookup by ID. No account_id filtering in either (product types are global lookup entities).

### Error Handling
- **Dashboard**: Returns null if not found (Prisma findUnique behavior)
- **Go**: `db.MapSQLError` converts `sql.ErrNoRows` to a not-found API error
- **Verdict**: Go returns a proper 404 error, which is correct REST behavior

### Response Shape
- **Dashboard**: `{ id, code, name, createdAt, updatedAt }`
- **Go**: `{ id, object, name, code, created_at, updated_at }`
- **Verdict**: Match — Go adds the standard `object` field per API resource conventions. Field names use snake_case per Go API conventions.

### Side Effects
- None in either implementation

### Idempotency
- GET endpoint — idempotent by design, no idempotency key needed

### Includes/Expands
- Neither implementation supports includes — product types are simple lookup entities with no sub-resources

## Notes
- The Dashboard never had an active controller for product types (no endpoint was registered), but the repository and domain model existed. The Go implementation is the canonical version.
- Product types are global (not account-scoped), which is correctly reflected in both the Dashboard Prisma query and the Go SQL query having no account_id filter.
