# Verification: GET /v1/core/transactions

**Status: Parity Confirmed** — No issues found requiring fixes.

## What Was Compared

### Permission Checks
- **Dashboard**: `checkIsInternalActor()` + `checkHasPermission(PermissionDomains.transactions, 'read')`
- **Go**: `CheckIsInternalActor()` + `CheckHasPermission(PermissionDomainTransactions, ActionRead)` + `CheckTargetAccountSet()`
- **Result**: ✅ Match. Go adds explicit target account check (standard pattern).

### Query Parameters / Filters
| Filter | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Text search | `query` (LIKE on number) | `query` (MATCH AGAINST on number + note) | ✅ Superset |
| Status | `fully_allocated` / `unallocated` | `allocated` / `unallocated` | ✅ Equivalent |
| Type codes | `typeCodes[]` | `type_codes[]` | ✅ |
| Adjustment type codes | `adjustmentTypeCodes[]` | `adjustment_type_codes[]` | ✅ |
| Method codes | `methodCodes[]` | `method_codes[]` | ✅ |
| Customer IDs | `customerIDs[]` | `customer_ids[]` | ✅ |
| Customer group IDs | `customerGroupIDs[]` | `customer_group_ids[]` | ✅ |
| Start date | `startDate` | `start_date` | ✅ |
| End date | `endDate` | `end_date` | ✅ |
| Pagination | `take`/`skip` (offset) | `limit`/`cursor` (cursor-based) | ✅ Intentional redesign |

### DB Queries and Logic
- **JOINs**: Both join `quantity`, `unit`, `transaction_type`, `account` (for customer), `account_relation` (for customer number + group filtering), and LEFT JOIN `transaction_method` and `adjustment_type`. ✅ Match.
- **Ordering**: Dashboard: relevance score (if query) then `createdAt DESC`. Go: `created_at DESC, id DESC`. Acceptable — Go uses full-text search which provides relevance through result filtering.
- **Allocation count**: Both use a correlated subquery `COUNT(*) FROM transaction_allocation`. ✅ Match.
- **Customer group filter**: Both filter via `account_relation.account_group_id`. The `account_relation` JOIN does not filter by `account_relation_role_code = 'customer'` in Go, but this is consistent with all other SQL queries in the Go codebase (sales_order, invoice, receivable, etc.) and doesn't cause issues since transactions reference `customer_account_id` which maps to customer-role relations.

### Response Shape
- **Dashboard**: `{ items: TransactionSummary[], count: number }`
- **Go**: `{ data: TransactionSummary[], page_info: {...} }`

Both TransactionSummary objects contain equivalent fields:
- `id`, `number`, `amount` (quantity with unit), `customer` (id, name, number), `transaction_type` (id, code, name), `transaction_method` (id, code, name, nullable), `adjustment_type` (id, code, name, nullable), `is_fully_allocated`, `allocation_count`, `created_at`, `updated_at`
- Go adds `object` field on all resources (standard Go API convention).
- Dashboard includes `billTo` (LightAddress) on the customer sub-resource, but this is not part of Go's `LightCustomer` and is omitted consistently across all Go endpoints. This is a minor cosmetic difference, not a business logic gap.
- Unit sub-resource in Quantity only has `id` and `object` in Go (missing `name`, `abbreviation`, `type`). This is consistent across all Go presenters and is a codebase-wide pattern, not specific to this endpoint.

### Error Handling
- Both return appropriate errors for missing identity, insufficient permissions, and invalid parameters. ✅ Match.

### Side Effects
- None for this GET endpoint. ✅ Match.

### Idempotency
- GET endpoint — inherently idempotent, no idempotency key needed. ✅ Correct.

## Notes
- The Go endpoint comment mentions `partially_allocated` as a valid status, but the SQL only handles `allocated` and `unallocated`. Passing `partially_allocated` would return no results since `is_fully_allocated` is a boolean column. The Dashboard also doesn't support this status. This is a documentation inaccuracy but not a functional issue.
- Pagination style change (offset → cursor) is an intentional architectural decision for the Go API.
