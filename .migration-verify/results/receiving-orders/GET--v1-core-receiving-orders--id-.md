# Verification: GET /v1/core/receiving-orders/{id}

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Actor type validation and permission domain/action
- **DB queries**: Joins, filters, tenant isolation, data selected
- **Response shape**: Fields, nested resources, nullable fields
- **Error handling**: Not-found behavior
- **Side effects**: None expected (read-only endpoint)

## Issues found and fixed

### 1. Permission check was too permissive

**Dashboard**: `checkIsInternalActor` + `checkHasPermission(receivingOrders, read)` — only internal users allowed.

**Go (before fix)**: `CheckIsAssignedActor()` + conditional permission check only if internal user — allowed customer/supplier actors without permission checks.

**Fix**: Changed to `CheckIsInternalActor()` + unconditional `CheckHasPermission(ReceivingOrders, Read)` in `services/core-service/internal/service/receiving_order_svc.go`.

### 2. Rejected quantity not populated on receiving order lines

**Dashboard**: Computes `rejectedQuantity` by joining `delivery_line` rows where `rejected_at IS NOT NULL` and summing their quantities.

**Go (before fix)**: The `ListReceivingOrderLinesByOrderID` SQL query did not join on `delivery_line`, so `RejectedQuantityValue` was never populated despite the domain model, proto, and presenter all supporting it.

**Fix**: Added a correlated subquery to the SQL query in `services/core-service/internal/infrastructure/queries/receiving_order.sql` to compute `SUM(rq.value)` from `delivery_line` joined with `quantity` where `rejected_at IS NOT NULL`. Updated `mapReceivingOrderLineRow` in the repository to extract the value from the `interface{}` column. Regenerated sqlc.

## Remaining notes

- The repo `Get()` method fetches lines internally, and then the service calls `ListLines()` separately, causing a duplicate query. This is an efficiency issue, not a parity issue.
- Dashboard error message says "Delivery not found." — the Go code relies on `db.MapSQLError` for the not-found case, which returns a generic not-found error. This is acceptable.
- Tenant isolation is equivalent: Dashboard uses `order.ownerAccountID`, Go uses `ro.account_id` — both scope to the target account.
- Response shape matches: nested `purchase_order`, `supplier`, `lines` with `quantity`, `rejected_quantity`, `order_line` sub-resources.
