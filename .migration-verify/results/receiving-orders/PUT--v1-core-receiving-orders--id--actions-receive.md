# Verification: PUT /v1/core/receiving-orders/{id}/actions/receive

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Both check internal actor + update permission on ReceivingOrders. Go also checks target account is set (extra safety). Match.
- **Validation**: Both extract the receiving order ID from the path. Match.
- **Business logic flow**: Both find unstocked lines (enforceNonZero: false), compute remaining quantities (ordered - received), and update unstocked lines. Match.
- **Error handling**: Both return not-found if the receiving order doesn't exist (Go via sqlc `:one` → `sql.ErrNoRows`). Match.
- **Response shape**: Both return the full receiving order with all lines. Match.
- **Side effects**: Neither has side effects (no emails, webhooks, messages). Match.
- **Idempotency**: PUT endpoint, no idempotency keys needed. Match.

## Issues found and fixed

### 1. Missing guard for empty unstocked lines (FIXED)

**Dashboard**: Checks `if (unstockedLines.length > 0)` before entering the transaction and calling `bulkReceiveRemainingQuantities`.

**Go (before fix)**: Always entered the transaction and called `BulkReceiveRemainingQuantities` even when `orderLineIDs` was empty. This could cause a SQL error with an empty `IN()` clause in `GetOrderedQuantityForLine`.

**Fix**: Moved `FindUnstockedLineIDs` outside the transaction and wrapped the transaction in `if len(unstockedLines) > 0` guard, matching the Dashboard behavior.

## Noted differences (intentionally kept)

### completedAt filter in FindUnstockedLineIDs

Dashboard's Prisma query includes `completedAt: null` on the receiving order when finding unstocked lines. The Go SQL does not have this filter. This is redundant because a completed receiving order has all lines stocked (`stocked_at IS NOT NULL`), so the `stocked_at IS NULL` filter already returns 0 rows for completed orders. No change needed.

### Stocked-only vs all-lines in quantity calculation

Dashboard's `bulkReceiveRemainingQuantities` sums ALL receiving order lines (stocked + unstocked) when computing total received. Go's `GetOrderedQuantityForLine` only sums STOCKED lines. The Go approach is more correct: it makes the PUT endpoint truly idempotent (calling it twice produces the same result). The Dashboard version is not idempotent (second call would incorrectly set quantities to 0). Since PUT endpoints must be idempotent per codebase conventions, the Go implementation is an intentional improvement. On first call (normal usage), both produce identical results since unstocked lines start with quantity 0.
