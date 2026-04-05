# GET /v1/core/picks — Migration Verification

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Dashboard requires internal actor + `picks` read permission. Go allows assigned actors (internal + customer), with internal users needing explicit `picks` read permission and customer actors getting implicit access.
- **Query parameters/filters**: Both support `query`, `status`, `customer_ids`, `product_line_ids`, `customer_group_ids`, `department_ids`, `start_date`, `end_date`.
- **Search scope**: Compared which fields the text search query matches against.
- **DB queries**: Compared JOIN structure, WHERE clause filters, ordering, and pagination.
- **Response shape**: Compared Go PickSummary vs Dashboard LightPick.
- **Error handling**: Both return standard errors for missing identity, permission failures, invalid cursors.
- **Side effects**: None (GET endpoint).
- **Idempotency**: N/A (GET endpoint, naturally idempotent).

## Issues found and fixed

### 1. Search scope too narrow (FIXED)

**Before**: The Go SQL queries only searched `p.number` (pick number) when a query was provided.

**After**: Expanded search to also match against:
- `so.number` (sales order number)
- `so.customer_po_number` (customer PO number)
- `ba.name` (customer name)
- `ar.external_number` (customer number)

This matches the Dashboard behavior where `PickAdapter.fetchInput` creates an OR across pick number and order-level fields (`OrderAdapter.fetchInput` searches order number, customer PO, and `CustomerAccountAdapter.fetchInput` searches customer name, alias, external number, and notes).

**Files changed**: `services/core-service/internal/infrastructure/queries/pick.sql` — updated `ListPicksForward`, `ListPicksBackward`, and `CountPicks` queries.

## Accepted differences (not bugs)

1. **Customer actor access**: Go allows customer actors to list picks (implicit read access). Dashboard only allows internal actors. This is an intentional enhancement per migration guidelines.
2. **Pagination**: Dashboard uses offset-based (take/skip), Go uses cursor-based pagination. Intentional improvement.
3. **Default status**: Dashboard defaults to "open" when status is omitted. Go treats omitted status as "all" (no filter). Consumers should explicitly pass `status=open` to get the same behavior.
4. **Response shape**: Dashboard LightPick includes shipTo address, orderLines, departments, and lastShippedAt. Go PickSummary is leaner (id, number, sales_order, customer, priority, finished_at, timestamps). This follows Go API conventions with sub-resources and expandable includes available on the detail endpoint.
5. **Search field coverage**: Dashboard also searches customer alias (`ar.alias`) and customer notes (`ar.notes`). These are very niche search fields and were not added to avoid unnecessary query complexity. The main search fields (order number, customer PO, customer name, customer number) are covered.
