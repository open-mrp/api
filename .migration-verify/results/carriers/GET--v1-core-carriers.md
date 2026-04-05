# GET /v1/core/carriers — Verification Result

**Status: PARITY CONFIRMED** — No issues found.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **Permission: actor type** | `checkIsAssignedActor` (internal + customer) | `CheckIsAssignedActor()` | Yes |
| **Permission: domain/action** | `carriers` / `read` (internal only) | `carriers` / `read` (internal only) | Yes |
| **Permission: customer access** | Customers can read without permission check | Same — only internal actors need explicit permission | Yes |
| **Target account required** | Implicit via `identity.targetAccountID` | Explicit `CheckTargetAccountSet()` | Yes (Go is stricter) |
| **External target read access** | Not checked | `ReadAccess.CheckReadAccess()` for external targets | Yes (Go enhancement) |
| **Search filter** | `name LIKE %query%` | `carrier.name LIKE %query%` | Yes |
| **Account filter** | `accountID = target OR accountID IS NULL` | `account_id = ? OR account_id IS NULL` | Yes |
| **Soft-delete filter** | `deletedAt: null` | `deleted_at IS NULL` | Yes |
| **Default carrier logic** | `isDefault = (accountID === null)` | `IsDefault = (AccountID == nil)` | Yes |
| **Response fields** | id, name, code, shippoCarrierAccountId, accountNumber, isPortalEnabled, isDefault, options, deletedAt, createdAt, updatedAt | Same set of fields | Yes |
| **Side effects** | None | None | Yes |
| **Idempotency** | N/A (GET) | N/A (GET) | Yes |

## Intentional Differences (by design)

1. **Pagination model**: Dashboard uses offset-based (`take`/`skip` + `count`). Go uses cursor-based (`cursor`/`limit` + `page_info`). This is the standard Go API pagination pattern.

2. **Options expandability**: Dashboard always includes `options` (carrier options nested array) in the list response. Go makes `options` expandable via the `include=options` query parameter, returning `null` by default. This follows the Go API's include/expand pattern documented in `api-resource-conventions.md`.

## No Fixes Required

The Go implementation faithfully reproduces all Dashboard business logic for this endpoint. The differences noted above are intentional architectural patterns of the Go API, not bugs.
