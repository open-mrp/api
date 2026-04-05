# Verification: PUT /v1/core/receiving-orders/{receivingOrderId}/lines/{id}/actions/receive

## Status: Issues found and fixed

## What was compared

- **Permission checks**: Both require internal actor + `receivingOrders/update` permission ✅
- **Account scoping**: Dashboard uses `targetAccountID` via `ownerAccountID`; Go checks `IsInAccount` then uses `TargetAccountID` ✅
- **Validation**: Both verify line belongs to receiving order. Go additionally verifies receiving order is in account (stricter, acceptable) ✅
- **Quantity calculation logic**: Both calculate `ordered - received` for the sales/purchase order line ✅
- **Update logic**: Both update line quantity only if remaining > 0 ✅
- **Response**: Both return the receiving order line with 200 OK ✅
- **Error handling**: Both return not-found errors for missing resources ✅
- **Idempotency**: PUT endpoint, idempotent by design (no idempotency keys needed) ✅
- **Side effects**: Neither triggers inventory changes, emails, or other side effects ✅
- **Transaction safety**: Dashboard uses `prisma.$transaction()`; Go uses a single atomic UPDATE statement ✅

## Issue found and fixed

### `CalculateQuantityYetToBeReceived` SQL query had incorrect `stocked_at` filter

**File**: `services/core-service/internal/infrastructure/queries/receiving_order.sql` (line 339)

**Problem**: The Go SQL query included `AND all_rol.stocked_at IS NOT NULL` when joining receiving order lines to sum the received total. This meant only previously stocked/received lines were counted toward the received quantity. The Dashboard sums ALL receiving order lines for the sales order line, regardless of `stocked_at` status.

**Impact**: The Go could calculate a larger remaining quantity than the Dashboard in cases where there are unstocked receiving order lines with non-zero quantities, leading to over-receiving.

**Fix**: Removed the `AND all_rol.stocked_at IS NOT NULL` condition from the LEFT JOIN:
```sql
-- Before
LEFT JOIN receiving_order_line all_rol ON all_rol.sales_order_line_id = sol.id AND all_rol.stocked_at IS NOT NULL

-- After
LEFT JOIN receiving_order_line all_rol ON all_rol.sales_order_line_id = sol.id
```

Regenerated sqlc code with `make sqlc core`.

## Remaining concerns

- None. The endpoint now matches Dashboard behavior.
