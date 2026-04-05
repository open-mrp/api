# GET /v1/core/parts — Migration Verification

## Result: Issues found and fixed

## What was compared

- **Validation rules**: Request parameters (query, pagination, categoryIDs, attributeIDs, startDate, endDate)
- **Permission checks**: Actor type and permission domain/action
- **DB queries and logic**: Filters, joins, ordering, pagination
- **Error handling**: Error types and messages
- **Side effects**: None expected for GET (confirmed)
- **Response shape**: Field names, types, nested resources, expandables
- **Idempotency**: N/A for GET endpoints

## Issues found and fixed

### 1. Permission domain mismatch (fixed)

**Dashboard**: `checkHasPermission(this.identity, PermissionDomains.items, 'read')`
**Go (before fix)**: `identity.CheckHasPermission(types.PermissionDomainParts, types.ActionRead)` — domain `"parts"`
**Go (after fix)**: `identity.CheckHasPermission(types.PermissionDomainItems, types.ActionRead)` — domain `"items"`

The Dashboard uses the `items` permission domain for all part read operations. The Go product and material services also use `PermissionDomainItems` for reads, confirming this is the correct pattern.

**File**: `services/core-service/internal/service/part_service.go` line 86

### 2. Missing attribute loading in list results (fixed)

**Dashboard**: `PartAdapter.select` includes attributes in the query; attributes are always returned with each part in the list.
**Go (before fix)**: The `List` method in `part_repository.go` did not call `loadPartAttributes` for results. The `Get` method did.
**Go (after fix)**: Added `loadPartAttributes` calls to all three code paths in `List` (backward cursor, forward cursor, no cursor).

**File**: `services/core-service/internal/infrastructure/repository/part_repository.go` — 3 code paths updated

## Acceptable differences (not parity issues)

### Pagination model
- **Dashboard**: Offset-based (`take`/`skip`) with total `count`
- **Go**: Cursor-based with `page_info` (next/prev cursors, has_next/has_prev)

This is an intentional architectural improvement in the Go API. Cursor-based pagination is the standard pattern across all Go endpoints.

### Sort order
- **Dashboard**: Prisma `_relevance` on `sku` + `description` (fulltext relevance scoring)
- **Go**: `ORDER BY i.created_at DESC, i.id DESC`

The Go API uses deterministic created_at ordering consistent with cursor-based pagination. Relevance scoring is handled via LIKE filtering instead.

### Product type filter
- **Dashboard**: Includes `productCondition` filter (items with no product OR product type = "sale")
- **Go**: No such filter

This filter is effectively a no-op for parts since parts are queried via `FROM part p JOIN item i` which already constrains to part-type items. Parts don't have product records, so they always match `product: null`.

## Remaining concerns

- The `GetPart`, `CreatePart`, `UpdatePart`, and `DeletePart` methods also use `PermissionDomainParts` instead of `PermissionDomainItems`. These should be reviewed when those endpoints are verified. (The Dashboard uses `PermissionDomains.items` for all part CRUD operations.)
