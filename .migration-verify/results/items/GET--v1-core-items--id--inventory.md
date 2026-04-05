# GET /v1/core/items/{id}/inventory

## Result: Issues found and fixed

## What was compared

- **Validation**: Request params (itemID from path) — matches
- **Permissions**: `checkIsInternalActor` + `checkHasPermission(items, read)` — matches
- **Account scoping**: Both use `identity.targetAccountID` — matches
- **DB queries**: Receipt and issue filtering logic — **had issues, fixed**
- **Error handling**: Item not found returns 404 via `sql.ErrNoRows` → `MapSQLError` — matches
- **Side effects**: None (GET endpoint) — matches
- **Idempotency**: Not applicable (GET) — correct
- **Response shape**: `{ object, on_hand, reserved, available_to_promise, short }` with `QuantityInfo` sub-objects — matches (snake_case per Go API conventions)

## Issues found and fixed

### 1. Cross-join bug in SQL query (critical)

The original SQL query LEFT JOINed both `inventory_receipt` and `inventory_issue` from a single params row, creating a cartesian product between receipts and issues. If an item had N receipts and M issues, each receipt quantity was counted M times and each issue quantity was counted N times, producing inflated sums.

**Fix**: Rewrote the query to use separate subqueries for receipts and issues, each pre-aggregated by `item_id`, then joined to the item row. This matches the dashboard's approach of querying receipts and issues independently.

### 2. AvailableToPromise never calculated (critical)

The repository returned an empty string for `AvailableToPromise` with a comment "Calculated at the service layer", but the service layer never performed this calculation.

Dashboard formula: `availableToPromise = onHand - reserved - short`

**Fix**: Added the ATP calculation directly in the SQL query as `CAST(COALESCE(on_hand, 0) - COALESCE(reserved, 0) - COALESCE(short, 0) AS DECIMAL(65,30))`, and the repository now reads it from the query result.

### 3. Unit IDs never populated (moderate)

All unit ID fields (`OnHandUnitID`, `ReservedUnitID`, `ATPUnitID`, `ShortUnitID`) were hardcoded to empty strings. The dashboard returns full unit information from the quantity's associated unit.

**Fix**: The SQL query now joins `item → rate (unit_value_id) → rate.denominator_unit_id` to get the item's base unit ID. All four quantity fields share the same unit (the item's base unit), matching the dashboard behavior where all quantities use the same unit type from the item's category unit group.

## No remaining concerns

All three issues have been fixed. The query logic now correctly matches the dashboard's `fetchCurrentInventory` behavior with separate receipt/issue aggregation, proper ATP calculation, and unit ID population.
