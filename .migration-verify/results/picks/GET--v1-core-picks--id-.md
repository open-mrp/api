# Migration Verification: GET /v1/core/picks/{id}

## Status: Issues found and fixed

## What was compared

- **Validation rules**: Path parameter (pick ID) — both implementations equivalent
- **Permission checks**: Actor type restrictions and permission domain/action
- **DB queries**: Joins, filters, account scoping
- **Error handling**: 404 on not found, account isolation
- **Response shape**: Field names, types, nested sub-resources
- **Side effects**: None (read-only endpoint)
- **Idempotency**: N/A (GET endpoint)

## Issues found and fixed

### 1. Permission check allowed non-internal actors (security fix)

**Dashboard**: `checkIsInternalActor` — only internal users can access picks.

**Go (before fix)**: `CheckIsAssignedActor` — allowed internal, customer, AND supplier actors. The permission check was also conditional (`if identity.IsInternalUser()`), meaning customer/supplier actors bypassed permission checks entirely.

**Fix**: Changed `CheckIsAssignedActor()` to `CheckIsInternalActor()` and made the permission check unconditional, matching the Dashboard behavior.

**File**: `services/core-service/internal/service/pick_svc.go` (lines 106-113)

### 2. Departments not always returned

**Dashboard**: Departments are always included in the pick response (fetched via Prisma `select` with the pick query).

**Go (before fix)**: Departments were only returned when explicitly requested via the `include` query parameter. Additionally, `"departments"` was not even listed in the endpoint's `IncludeConfig` (only `"lines"` was), so there was no way to request them.

**Fix**: Moved department fetching out of the include loop so departments are always fetched and returned, matching Dashboard behavior.

**File**: `services/core-service/internal/service/pick_svc.go` (lines 127-131)

## Remaining notes

- The Dashboard returns a deeply nested order object with full details (lines, shipments, invoices, addresses, etc.). The Go API returns simplified sub-resources (`sales_order` with just ID/object, `customer` with ID/object/name/number, `priority` with code/object/name). This is consistent with the Go API's sub-resource convention and is an intentional design difference, not a parity issue.
- The `PickSalesOrder` sub-resource does not include the sales order `number` field, even though it is fetched by the SQL query and available in the domain model. This is a minor omission but not a strict parity issue since the Dashboard returns the full order object.
- No unit tests exist for `GetPick` in the Go codebase.
