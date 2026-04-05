# Verification: GET /v1/core/account-users

**Result: Parity confirmed with noted design differences**

No code changes were required. The core business logic matches the Dashboard implementation.

## What was compared

### Permission checks
- **Dashboard**: `checkIsAssignedActor` + `customers:read` for internal actors + `checkReadAccess` for cross-account
- **Go**: `CheckIsAssignedActor` + `PermissionDomainCustomers/ActionRead` for internal actors + `ReadAccess.CheckReadAccess` for external targets
- **Verdict**: Match

### Query parameters / filters
| Dashboard | Go | Match |
|---|---|---|
| `q` (query) | `q` (query) | Yes |
| `take` / `skip` (offset pagination) | `cursor` / `limit` (cursor pagination) | Intentional change |
| `roleType` | `role_type` | Yes |
| `includeRemoved` (default false) | `include_removed` (default false) | Yes |

### SQL queries
- Both join `account_user` → `user` → `role` (LEFT) → `department` (LEFT)
- Both use full-text search (`MATCH AGAINST`) on `user.name`
- Both use LIKE patterns on `username`, `email`, `role.name`, `department.name`
- Both filter by `account_id`, `status_code != 'removed'` (unless include_removed), `role_type_code`
- Both return a total count alongside results
- **Verdict**: Match

### Response shape
- Dashboard returns `{ items: AccountUser[], count: number }`
- Go returns `{ data: AccountUser[], page_info: PageInfo, total_count: number }`
- AccountUser fields match: `id`, `name`, `email`, `username`, `image_url`, `is_verified`, `status`, `role` (sub-resource), `department` (sub-resource), `last_used_at`, `created_at`, `updated_at`
- Go adds `object` field per API conventions
- **Verdict**: Match (structural differences follow Go API conventions)

### Error handling
- Both return authentication/authorization errors for missing identity or insufficient permissions
- Both return errors for missing target account ID
- **Verdict**: Match

### Side effects
- Both are read-only (no mutations)
- **Verdict**: Match

## Intentional design differences (not bugs)

1. **Pagination model**: Dashboard uses offset-based (`take`/`skip`), Go uses cursor-based (`cursor`/`limit`). This is a standard Go API convention change. The Go default limit is 100, max 1000.

2. **Ordering**: Dashboard orders by name relevance (full-text score descending) when a search query is provided. Go always orders by `created_at DESC, id DESC` for stable cursor pagination. This is a tradeoff required by cursor-based pagination.

## Noted gaps (not blocking)

1. **Photo URL resolution**: The Dashboard resolves presigned S3 URLs by constructing a key from `{accountID}/{userID}.png` and generating temporary signed URLs (1hr expiry). The Go API returns the raw `image_url` value from the `user` table. If the frontend relies on presigned URLs from the list endpoint, this may need a separate photo URL resolution mechanism.

2. **Notification preferences in list response**: The Dashboard conditionally includes notification preference flags (`receivesOrderAcknowledgements`, `receivesInvoiceNotifications`, `receivesPurchaseOrderSubmissionNotifications`) when listing users for an external target (cross-account). The Go `AccountUser` resource does not include these fields. The Go API manages notification preferences via the dedicated `PUT /v1/core/account-users/{id}/notification-preferences` endpoint instead. If the frontend needs these inline, this would require adding fields to the resource, proto, SQL, and presenter.
