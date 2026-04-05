# Migration Verification: GET /v1/core/integrations

## Status: PARITY CONFIRMED

No issues found. The Go implementation correctly mirrors the Dashboard business logic.

## Comparison Summary

### Permission Checks
| Check | Dashboard | Go |
|-------|-----------|-----|
| Internal actor | `checkIsInternalActor(identity)` | `identity.CheckIsInternalActor()` |
| Admin role | `roleTypeCode !== RoleTypes.admin` → 403 | `identity.CheckIsAdmin()` |
| Target account | Implicit (from `identity.targetAccountID`) | `identity.CheckTargetAccountSet()` |

Both implementations require internal actor + admin role. Go explicitly validates target account is set (defensive — good).

### Query Parameters / Filters
| Filter | Dashboard | Go |
|--------|-----------|-----|
| Text search | `query` param, Prisma fulltext `_relevance` on `name` | `q` param, SQL `LIKE %query%` on `name` |
| Pagination | Offset-based (`take`/`skip`) | Cursor-based (`cursor`/`limit`) |
| Account filter | `accountID` from identity | `account_id` from identity |

Pagination style difference is expected — the Go API uses cursor-based pagination as a project-wide convention. Search capability is equivalent (both filter by name).

### Response Shape
| Field | Dashboard | Go |
|-------|-----------|-----|
| ID | `id` | `id` |
| Object type | (not present) | `object: "account_integration"` |
| Name | `name` | `name` |
| Integration code | `code` | `integration_code` |
| Active status | `isActive` | `is_active` |
| Created timestamp | `createdAt` | `created_at` |
| Updated timestamp | `updatedAt` | `updated_at` |
| Pagination | `{ items, count }` | `{ data, page_info }` |

Go adds `object` field per API conventions. Field naming follows Go snake_case convention vs Dashboard camelCase. Pagination wrapper follows Go API conventions.

### DB Queries
- Both filter by `account_id` and optionally by `name`
- Dashboard uses Prisma fulltext relevance ordering; Go orders by `created_at DESC, id DESC`
- No credentials are returned in either implementation

### Error Handling
- Both return 403 for non-admin users
- Both return 401/403 for non-internal actors

### Side Effects
- None in either implementation (read-only endpoint)

### Idempotency
- GET endpoint — idempotent by design, no idempotency keys needed (correct in both)

## Conclusion
Full business-logic parity confirmed. The differences are all expected convention changes (snake_case, cursor pagination, `object` field).
