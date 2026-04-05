# Verification: GET /v1/core/order-discounts

**Status: Issues found and fixed**

## What was compared

- **Permission checks**: Actor type validation and permission domain/action
- **DB queries**: Filters, search, ordering, pagination
- **Response shape**: Field names, types, nested resources
- **Side effects**: None expected (read-only endpoint)
- **Error handling**: Error types and messages

## Issues found and fixed

### 1. Permission check: customer actors blocked (FIXED)

**Dashboard behavior**: Uses `checkIsAssignedActor` which allows both internal and customer actors. Only internal actors are checked for `discounts:read` permission — customer actors skip the permission check.

**Go behavior (before fix)**: Used `CheckIsInternalActor()` which rejected customer actors entirely.

**Fix**: Changed to `CheckIsAssignedActor()` and made the `discounts:read` permission check conditional on `identity.IsInternalUser()`, matching the Dashboard logic.

File: `services/core-service/internal/service/order_discount_service.go`, lines 83-90.

## Noted differences (acceptable)

### Search query scope

- **Dashboard**: Searches only by `name` field (`OR: [{ name: { contains: query } }]`)
- **Go**: Searches by both `name` and `code` (`od.name LIKE ... OR od.code LIKE ...`)

This is strictly additive (returns a superset of matches) and doesn't break existing behavior. Searching by discount code is a reasonable improvement.

### Pagination model

- **Dashboard**: Offset-based pagination (`take`/`skip`)
- **Go**: Cursor-based pagination (standard for the new API)

This is an intentional migration improvement consistent with all other Go API endpoints.

## Confirmed parity

- Account scoping: Both filter by `identity.targetAccountID`
- Response fields: `id`, `name`, `code`, `percentage`, `amount`, `discount_type`, `order_count`, `created_at`, `updated_at` — all present in Go resource
- Order count subquery: Both count related sales orders via subquery
- No side effects in either implementation (read-only)
