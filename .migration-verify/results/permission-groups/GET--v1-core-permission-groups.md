# Verification: GET /v1/core/permission-groups

## Status: PARITY CONFIRMED

No issues found. The Go implementation faithfully reproduces all Dashboard business logic.

## Comparison Summary

### Permission Checks
- **Dashboard:** `checkIsInternalActor` + `checkHasPermission(permissions, read)`
- **Go:** `identity.CheckIsInternalActor()` + `identity.CheckHasPermission(PermissionDomainPermissions, ActionRead)`
- **Verdict:** Match

### Query/Filter Parameters
- **Dashboard:** `query` (name search via LIKE), `take` (limit), `skip` (offset)
- **Go:** `query` (name search via LIKE), `limit`, `cursor` (cursor-based pagination)
- **Verdict:** Match — pagination style upgraded from offset to cursor-based, which is an intentional Go API convention. The `query` filter searches by `name` in both implementations using LIKE pattern matching.

### Database Queries
- **Dashboard:** Prisma `findMany` on `permissionGroup` with name filter, plus `count` query. Nested `permissions` selected inline.
- **Go:** SQL query on `permission_group` table with name LIKE filter and cursor-based pagination. Permissions batch-loaded separately via `ListPermissionsByGroupCodes` (by `permission_group_code IN (...)`), ordered by `permission.name ASC`.
- **Verdict:** Match — both return permission groups with their nested permissions. Go uses a more efficient batch-load pattern.

### Scoping
- **Dashboard:** Not account-scoped (global)
- **Go:** Not account-scoped (no `account_id` filter in queries, no `TargetAccountID` check in service)
- **Verdict:** Match

### Response Shape
- **Dashboard:** `{ items: PermissionGroup[], count: number }` where each group has `{ name, code, description, permissions[] }` and each permission has `{ code, name, description, groupID }`
- **Go:** Standard `List[PermissionGroup]` with `page_info` cursor pagination. Each group has `{ id, object, code, name, description, permissions, created_at, updated_at }` and each permission has `{ id, object, code, name, description, permission_group_code, created_at, updated_at }`
- **Verdict:** Match — Go adds `id`, `object`, and timestamp fields per API resource conventions. All Dashboard fields are present.

### Side Effects
- **Dashboard:** None
- **Go:** None
- **Verdict:** Match

### Idempotency
- GET endpoint — idempotent by design, no idempotency key needed.
- **Verdict:** Correct

## No Issues Found
The Go implementation is a correct migration of the Dashboard endpoint with expected improvements (cursor-based pagination, richer resource shape with `id`/`object`/timestamps).
