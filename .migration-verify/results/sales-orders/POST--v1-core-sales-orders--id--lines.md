# Verification: POST /v1/core/sales-orders/{id}/lines

## Result: Issues found and fixed

## What was compared

- Permission checks (actor type, permission domain, action)
- Price rounding (unit price and unit cost)
- DB queries and logic (line creation, line item numbering, quantity/rate record creation)
- Side effects (pick line creation for remaining quantity)
- Error handling
- Response shape and field mapping
- Idempotency (POST uses idempotency keys)
- Validation (order existence check)

## Issues found and fixed

### 1. Permission check used wrong action (fixed)

**Dashboard:** `checkHasPermission(identity, PermissionDomains.salesOrders, 'update')`
**Go (before):** `CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionCreate)`

The Dashboard checks for `update` permission on sales orders when creating a line (since adding a line is modifying an existing order). Fixed Go to use `ActionUpdate`.

### 2. Missing price rounding to nearest cent (fixed)

**Dashboard:** Rounds both `unitPrice` and `unitCost` (if present) to nearest cent via `RateUtils.roundToNearestCent()` which does `Math.round(value * 100) / 100`.
**Go (before):** No rounding was performed.

Added `roundToNearestCent()` helper function and applied it to `UnitPriceValue` and `UnitCostValue` before persistence.

### 3. Missing pick line creation side effect (fixed)

**Dashboard:** After creating the order line, checks if the sales order has an associated pick. If it does, calls `PickLineRepo.createPickLineForRemainingQuantity()` which:
1. Calculates remaining quantity to be picked (ordered - already picked)
2. Skips if remaining <= 0
3. Skips if an unpacked pick line already exists for the order line
4. Otherwise creates a new pick line with a quantity matching the remaining amount

**Go (before):** No pick line creation after creating an order line.

Added `createPickLineForRemainingQuantity()` function that uses existing Go infrastructure (`PickLineRepo.CalculateRemainingForOrderLine`, `HasUnpackedPickLineForOrderLine`, `CreateForRemaining`) and `SalesOrderLineRepo.CreateQuantity` to replicate this behavior within the same transaction.

## Confirmed parity

- Internal actor check: Both require internal actor
- Target account ID: Both require it
- Line item number auto-generation: Both use `MAX(line_item_number) + 1`
- Quantity, rate, and order line record creation: Matching
- Order existence validation: Both validate the order exists
- Idempotency: Go correctly uses idempotency keys for this POST endpoint
- Response shape: Fields match (line item number, product SKU, description, quantity ordered/picked/packed/invoiced, unit price, unit cost, EDI line item ID, timestamps)
- Optional fields: Unit cost, product description, EDI line item ID, item ID all handled as optional

## Remaining concerns

None. All identified discrepancies have been fixed.
