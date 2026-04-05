# GET /v1/core/receiving-orders — Migration Verification

## Result: Issues found and fixed

## What was compared
- Permission checks (actor type, permission domain, action)
- Query parameters and filters (status, query, item_ids, supplier_ids, start_date, end_date)
- Default status filter behavior
- DB query logic (joins, filters, ordering, pagination)
- Search behavior (receiving order number + purchase order number)
- Response shape (summary fields: id, number, purchase_order, supplier, line_count, completion_percentage, completed_at, timestamps)
- Error handling

## Issues found and fixed

### 1. Default status filter (fixed)
- **Dashboard**: Defaults to `"open"` when no status parameter is provided (only returns uncompleted receiving orders)
- **Go (before fix)**: No default — returned ALL orders when status was omitted
- **Fix**: Added default status of `"open"` in `ListReceivingOrders` service method when `params.Status` is nil

### 2. Permission check too permissive (fixed)
- **Dashboard**: Uses `checkIsInternalActor()` — only internal users can access
- **Go (before fix)**: Used `CheckIsAssignedActor()` — allowed customer actors through without permission checks
- **Fix**: Changed to `CheckIsInternalActor()` and unconditional `CheckHasPermission()` (removed the `if identity.IsInternalUser()` guard)

## Confirmed parity
- Query/search: Both search on receiving order number AND purchase order number via LIKE
- Filters: All filters match (status open/completed/all, item_ids, supplier_ids, date range)
- Sorting: Both sort by created_at DESC (Go adds id DESC as tiebreaker)
- Response fields: Summary includes id, number, purchase_order sub-resource, supplier sub-resource, line_count, completion_percentage, completed_at, created_at, updated_at — all match Dashboard
- Tenant isolation: Both filter by account_id from identity

## Acceptable differences
- Pagination: Dashboard uses offset-based (take/skip), Go uses cursor-based — expected for migration
- Response wrapper: Dashboard returns `{ items, count }`, Go returns paginated list with cursor info — expected
- Search: Dashboard uses Prisma full-text relevance sorting, Go uses LIKE with consistent ordering — functionally equivalent
