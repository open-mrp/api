# GET /v1/core/departments — Verification Result

## Status: Issues found and fixed

## What was compared

- **Permission checks**: Both check internal actor + departments read permission. Go also checks target account is set. Parity confirmed.
- **Search/filter**: Dashboard filters by `accountID` and `name LIKE query`. Go does the same via `deptBuildSearchParams` and SQL `LIKE`. Parity confirmed.
- **Pagination**: Dashboard uses offset-based (take/skip + count). Go uses cursor-based pagination. This is an intentional migration-wide change, not a parity gap.
- **Response shape**: Dashboard returns `{ items: Department[], count: number }`. Go returns `{ data: Department[], page_info: {...} }`. Expected difference due to cursor-based pagination.
- **Sub-resources**: Dashboard includes `scanningStations`, `machines`, and `location` for each department in the list. Go was only including `location` (via SQL JOIN) but NOT scanning stations or machines.
- **Error handling**: Both return appropriate errors for auth/permission failures. Parity confirmed.
- **Side effects**: None for either. Parity confirmed.

## Issues found and fixed

### Missing sub-resources in list response

The Go repository's `List` method did not call `attachSubResources` for each department, while the dashboard's Prisma select always includes `scanningStations`, `machines`, and `location` relations.

**Fix**: Added `attachSubResources` calls after `pagination.BuildPageString` in all three code paths of the `List` method (no cursor, forward cursor, backward cursor) in `department_repository.go`.

## No remaining concerns
