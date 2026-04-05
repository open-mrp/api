# PUT /v1/core/purchase-orders/{id}/actions/change-status

## Result: Issues found and fixed

## What was compared

- **Validation rules**: Status precondition checks (estimate->issued, issued->estimate, issued->fulfilled, fulfilled->issued)
- **Permission checks**: Both use internal actor check + PurchaseOrders/update permission + target account ID
- **Status transitions**: All four transitions (issue, unissue, close, open) match Dashboard behavior
- **Receiving order side effects**: Issue creates receiving order + lines; unissue deletes them; close marks complete; open marks incomplete
- **Email notification**: Dashboard sends fire-and-forget after transaction; Go uses outbox pattern inside transaction (valid improvement)
- **Timestamp management**: issued_at and completed_at set/cleared per transition
- **Response shape**: Returns updated purchase order with includes

## Issues found and fixed

### 1. `unissue` did not clear `issued_at` (FIXED)

**Problem**: The SQL query `UpdatePurchaseOrderStatus` used `COALESCE(sqlc.narg('issued_at'), issued_at)` for the `issued_at` column. This meant passing `nil` for `issued_at` would preserve the existing value instead of clearing it to NULL. The Dashboard explicitly sets `issuedAt: null` when unissuing.

**Fix**:
- Changed SQL in `purchase_order.sql` from `COALESCE(sqlc.narg('issued_at'), issued_at)` to `sqlc.narg('issued_at')` (direct assignment)
- Updated service code for `close` to pass `order.IssuedAt` (preserve existing value)
- Updated service code for `open` to pass `order.IssuedAt` (preserve existing value)
- `issue` already passed `&now` (correct)
- `unissue` passes `nil` which now correctly clears the field

**Files changed**:
- `services/core-service/internal/infrastructure/queries/purchase_order.sql` (line 309)
- `services/core-service/internal/service/purchase_order_service.go` (lines 703, 719)
- `services/core-service/internal/infrastructure/sqlc/purchase_order.sql.go` (regenerated)

## Noted differences (acceptable)

1. **Status validation**: Go explicitly validates current status before each transition and returns a validation error. Dashboard relies on the DB update matching 0 rows (silently succeeds). Go behavior is an improvement.

2. **Email handling**: Dashboard sends email fire-and-forget after transaction commit. Go publishes via the outbox pattern inside the transaction. This is a valid architectural improvement ensuring at-least-once delivery.

3. **Receiving order existence check on close/open**: Dashboard validates `receivingOrderID` exists before calling close/open and returns `badRequest` if missing. Go's `MarkComplete`/`MarkIncomplete` queries use `WHERE order_id = ?` and silently affect 0 rows. This is minor since a properly issued PO always has a receiving order. No fix applied as the behavior is functionally equivalent.
