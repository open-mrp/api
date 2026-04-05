# Verification: GET /v1/core/deliveries

**Status: PARITY CONFIRMED** — No code changes needed.

## What was compared

### Permission checks
- **Dashboard**: `checkIsInternalActor` + `checkHasPermission(deliveries, read)` + requires `targetAccountID`
- **Go**: `CheckIsInternalActor()` + `CheckHasPermission(PermissionDomainDeliveries, ActionRead)` + `CheckTargetAccountSet()`
- **Result**: Equivalent

### Query parameters / filters
| Filter | Dashboard | Go | Match? |
|---|---|---|---|
| `query` | Prisma fulltext relevance on `number` + PO number match | LIKE on `d.number` and `so.number` | Equivalent intent |
| `status` | Default `accepted`. Filters by `acceptedAt`/`rejectedAt` timestamps | Default `accepted`. Filters by `delivery_status_code` column | Equivalent (see note) |
| `itemIDs` | `lines.some.receivingOrderLine.orderLine.itemID IN` | EXISTS subquery on `delivery_line → receiving_order_line → sales_order_line.item_id IN` | Equivalent |
| `supplierIDs` | `purchaseOrder.sellerAccountID IN` | `so.seller_account_id IN` | Equivalent |
| `startDate` | `createdAt >= startDate` | `d.created_at >= start_date` | Equivalent |
| `endDate` | `createdAt <= endDate` | `d.created_at <= end_date` | Equivalent |
| Pagination | Offset-based (`take`/`skip`) with total `count` | Cursor-based with `PageInfo` | Intentional migration change |

### Status filter equivalence note
Dashboard filters: `accepted` → `acceptedAt IS NOT NULL`; `rejected` → `acceptedAt IS NULL AND rejectedAt IS NOT NULL`.
Go filters: `delivery_status_code = 'accepted'` or `= 'rejected'`.
These are equivalent because delivery creation sets `acceptedAt` when status is 'accepted' and leaves it null when 'rejected'.

### DB queries and logic
- Both join `delivery` with `sales_order` (purchase order) via `sales_order_id`
- Both count delivery lines for the summary view
- Go orders by `created_at DESC, id DESC`; Dashboard orders by relevance (fulltext) then `createdAt DESC` — acceptable difference
- Item filter uses same join path: `delivery_line → receiving_order_line → sales_order_line → item_id`

### Response shape
- **Dashboard**: `{ items: DeliverySummary[], count: number }`
- **Go**: Cursor-paginated `List[DeliverySummary]` with `PageInfo`
- Both `DeliverySummary` include: id, number, purchase_order (id + number), status, line_count, accepted_at, rejected_at, created_at, updated_at
- Go adds `object` field per API resource conventions

### Error handling
- Both return appropriate auth errors for missing identity, wrong actor type, missing permissions
- Go returns validation error for invalid cursor

### Side effects
- None — read-only endpoint

### Idempotency
- N/A — GET endpoint, inherently idempotent

## Intentional differences (not issues)
1. **Pagination model**: Dashboard uses offset (`take`/`skip` + `count`), Go uses cursor-based pagination — standard Go API migration pattern
2. **Search ordering**: Dashboard uses MySQL fulltext relevance ranking on `number`, Go uses `created_at DESC` — fulltext relevance is not critical business logic
3. **`object` field**: Added in Go per API resource conventions
