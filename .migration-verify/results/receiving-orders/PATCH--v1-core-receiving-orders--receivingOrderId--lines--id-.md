# PATCH /v1/core/receiving-orders/{receivingOrderId}/lines/{id}

**Status: Issues found and fixed**

## What was compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission domain | `receivingOrders` | `PermissionDomainReceivingOrders` | Yes |
| Permission action | `update` | `ActionUpdate` | Yes |
| Actor type check | `checkIsInternalActor` | `CheckIsInternalActor` | Yes |
| Target account required | Yes (via `targetAccountID`) | Yes (`CheckTargetAccountSet`) | Yes |
| Line-in-order validation | `checkIsInReceivingOrder` | `IsLineInReceivingOrder` | Yes |
| Account ownership | WHERE clause in update | Pre-check via `IsInAccount` | Yes (equivalent) |
| Updatable fields | `quantity` (value only) | `quantity_value` | Yes |
| Transaction wrapping | Prisma `$transaction` | `withTx` | Yes |
| Idempotency | Not in Dashboard | Yes (idempotency keys) | Yes (Go correctly adds per conventions) |
| Response shape | `ReceivingOrderLine` with rejected quantity | `ReceivingOrderLine` resource | Fixed (see below) |
| Side effects | None | None | Yes |

## Issues found and fixed

### 1. Missing `rejected_quantity_value` in `GetReceivingOrderLine` SQL query

**Problem:** The `GetReceivingOrderLine` SQL query (used to fetch the line after update) did not include the `rejected_quantity_value` subquery. The `ListReceivingOrderLinesByOrderID` query included it, but the single-line query did not. This meant the PATCH response always returned `null` for `rejected_quantity`, even if rejected delivery lines existed.

**Fix:**
- Added the rejected quantity subquery to `GetReceivingOrderLine` in `receiving_order.sql` (matching the pattern in `ListReceivingOrderLinesByOrderID`)
- Ran `make sqlc core` to regenerate the sqlc code
- Updated `mapGetReceivingOrderLineRow` in `receiving_order_repo.go` to populate `RejectedQuantityValue` (matching the pattern in `mapReceivingOrderLineRow`)

## Remaining concerns

None. The Go implementation correctly mirrors the Dashboard business logic with the addition of idempotency keys (required by Go API conventions for PATCH endpoints).
