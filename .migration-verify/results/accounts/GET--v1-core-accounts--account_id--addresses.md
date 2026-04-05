# Verification: GET /v1/core/accounts/{account_id}/addresses

## Status: PARITY CONFIRMED

No code changes required.

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| **Permission: actor assigned** | `checkIsAssignedActor()` | `identity.CheckIsAssignedActor()` | Yes |
| **Permission: internal user** | `PermissionDomains.customers` / `read` (only when account differs) | `PermissionDomainAddresses` / `ActionRead` (always) | Intentional improvement — more granular domain, stricter check |
| **Read access check** | `accountRepo.checkReadAccess()` for all actors | `meds.ReadAccess.CheckReadAccess()` for external targets only | Equivalent — internal users gated by permission check |
| **Target account required** | Implicit via route param | `identity.CheckTargetAccountSet()` | Yes |
| **Search fields** | name, street_line_1, street_line_2, locality, state, postal_code, country | Same 7 fields | Yes |
| **Search mechanism** | MySQL full-text boolean search | LIKE `%query%` | Functional parity |
| **Account isolation** | `account_address.accountID` filter | `account_address.account_id` JOIN | Yes |
| **DB joins** | Prisma relation (address → geolocation) | `JOIN geolocation`, `JOIN account_address` | Yes |
| **Ordering** | Default Prisma ordering | `created_at DESC, id DESC` | Go has explicit ordering (improvement) |
| **Pagination** | Offset-based (take/skip/count) | Cursor-based keyset pagination | Intentional architectural change |
| **Response shape** | `{ items: Address[], count }` with flat fields | `{ object, page_info, data }` with nested geolocation sub-resource | Follows Go API conventions |
| **Idempotency** | N/A (GET) | N/A (GET) | Correct |
| **Side effects** | None | None | Yes |
| **Error handling** | HTTP errors via HttpError | API errors via apierror | Yes |

## Notes

- The permission domain change from `customers` to `addresses` is intentional — the Go API uses more granular permission domains.
- The Go implementation always checks `addresses.read` permission for internal users, while Dashboard only checks when accessing a different account. This is a stricter but cleaner authorization model.
- Pagination model change (offset → cursor-based) is an intentional Go API architectural decision, not a parity gap.
- Response shape differences (nested geolocation sub-resource, snake_case fields) follow Go API resource conventions documented in `docs/api-resource-conventions.md`.

## No issues found
