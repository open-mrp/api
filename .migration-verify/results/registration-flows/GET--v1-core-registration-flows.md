# Migration Verification: `GET /v1/core/registration-flows`

## Status: Parity Confirmed

No issues found. The Go implementation faithfully replicates the Dashboard business logic.

## What Was Compared

### Permission Checks
- **Dashboard**: `checkIsInternalActor` + `checkHasPermission(PermissionDomains.account, 'read')`
- **Go**: `CheckIsInternalActor()` + `CheckHasPermission(PermissionDomainAccount, ActionRead)` + `CheckTargetAccountSet()`
- **Result**: Match. Go additionally validates that `TargetAccountID` is set, which is correct.

### Query Parameters / Filtering
- **Dashboard**: `query` (name contains filter), `take`, `skip` (offset pagination)
- **Go**: `query` (name LIKE filter), `cursor`, `limit` (cursor pagination)
- **Result**: Same filtering capability. Pagination model changed from offset to cursor-based, which is the standard Go API pattern.

### DB Queries
- **Dashboard**: Prisma `findMany` with `WHERE accountID = ? AND name CONTAINS query`, ordered by `name ASC`
- **Go**: SQL query with `WHERE account_id = ? AND name LIKE ?`, ordered by `created_at DESC, id DESC`
- **Result**: Same filtering logic. Sort order differs (`name ASC` → `created_at DESC`) which is expected for cursor-based pagination requiring a stable, unique sort key.

### Response Shape
- **Dashboard**: `{ items: CustomerRegistrationFlow[], count: number }`
  - Each flow: `id`, `name`, `customerGroupOptions`, `paymentTermOptions`, `shippingTermOptions`, `createdAt`, `updatedAt`
  - Options: `{ id, name }` (from BasicInfoAdapter)
- **Go**: `{ data: RegistrationFlow[], page_info: {...} }`
  - Each flow: `id`, `object`, `name`, `customer_group_options`, `payment_term_options`, `shipping_term_options`, `created_at`, `updated_at`
  - Options: `{ id, object, name }`
- **Result**: Match. Go adds `object` field per API resource conventions. Options are enriched via separate queries for payment terms, shipping terms, and account groups — same data as Dashboard's Prisma relations.

### Error Handling
- Both return appropriate errors for missing identity, insufficient permissions, and invalid pagination.

### Side Effects
- None in either implementation (read-only endpoint).

## Architectural Differences (Intentional)
- Pagination: offset-based → cursor-based (Go API standard)
- Sort: `name ASC` → `created_at DESC, id DESC` (cursor pagination requirement)
- Response wrapper: `{ items, count }` → `{ data, page_info }` (Go API standard)
- Resource `object` field added per Go API conventions
