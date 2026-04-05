# POST /v1/core/materials — Migration Verification

## Result: Issues found and fixed

## What was compared

| Aspect | Parity? | Notes |
|--------|---------|-------|
| Validation (required fields) | Yes | Both require `sku` and `category_id`. Go also validates via struct tags. |
| Permission checks | Yes | Both check `checkIsInternalActor` and `materials:create` permission. |
| Target account header | Yes | Both require the target account ID. |
| SKU uniqueness check | Yes | Both check for duplicate SKU scoped to the account. |
| Item creation | Yes | Both create an item with type `material`, rates (unit_value, unit_cost, burn_rate) defaulted to 0, and category link. |
| Order point / lead time defaults | Yes | Both default to zero quantity with the category's base unit when not provided. |
| Rate creation | Yes | Both create 3 rates (unit_value, unit_cost, burn_rate) initialized to 0 with the category's base unit. |
| Idempotency | Yes | Go uses idempotency keys with recovery points. Dashboard does not have explicit idempotency. Go is stricter here. |
| Response shape | Yes | Both return the created material with nested item, order_point, and lead_time. |
| **Inventory side effects** | **Fixed** | Dashboard creates an inventory log and inventory change log (both with zero quantity) after material creation. Go was missing this. |

## Issues found and fixed

### 1. Missing inventory log and change log creation (FIXED)

**Dashboard behavior:** After creating the material, the Dashboard creates:
- An `inventory_log` entry with zero quantity (using the category's base unit)
- An `inventory_change_log` entry with zero quantity, action type `user_action`

These initialize the inventory tracking system for the newly created item.

**Go behavior (before fix):** The Go implementation did not create these records.

**Fix:** Added `CreateInventoryLog` and `CreateInventoryChangeLog` calls inside the transaction in `material_service.go`, using `decimal.Zero` for the measure and the category's `baseUnitID` for the unit, with action type `user_action`.

## Minor differences (acceptable)

1. **SKU conflict error message:** Dashboard says `"Item sku {sku} already exists."` (with the actual SKU value interpolated); Go says `"An item with this SKU already exists."` (generic). This is acceptable — the Go version uses `NewConflictErrorWithParam` which attaches the field name `sku` for client-side handling.

2. **Attributes filtering:** Dashboard filters `data.attributes` to only connect those with empty names. Go does not accept attributes on the create endpoint — attributes are managed separately. This is by design in the Go API.

3. **Transaction scope:** Dashboard runs inventory log creation in a `.finally()` block (outside the main transaction). Go runs it inside the same transaction. The Go approach is actually safer as it ensures atomicity.

## Files modified

- `services/core-service/internal/service/material_service.go` — Added inventory log and change log creation in `CreateMaterial`
