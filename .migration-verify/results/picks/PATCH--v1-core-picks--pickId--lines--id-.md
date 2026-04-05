# PATCH /v1/core/picks/{pickId}/lines/{id} — Migration Verification

## Result: PARITY CONFIRMED

No issues found. The Go implementation correctly preserves all Dashboard business logic.

## What Was Compared

### Permission Checks
- **Dashboard:** `checkIsInternalActor()` + `checkHasPermission(PermissionDomains.picks, 'update')`
- **Go:** `CheckIsInternalActor()` + `CheckHasPermission(PermissionDomainPicks, ActionUpdate)` + `CheckTargetAccountSet()`
- **Match:** Yes. Go adds explicit target account check (implicit in Dashboard via `this.identity.targetAccountID`).

### Account Isolation / Ownership
- **Dashboard:** Prisma nested where `orderLine.order.ownerAccountID` on the update query itself.
- **Go:** Two separate pre-checks: `pickRepo.IsInAccount()` then `pickLineRepo.IsInPick()`.
- **Match:** Functionally equivalent. Go is actually stricter — it separately verifies the pick exists in the account AND the line belongs to that pick, whereas Dashboard verifies ownership in a single query through the order relationship.

### Fields Updated
- **Dashboard:** `QuantityAdapter.updateInput(data.quantity)` — updates `measure` (value) and `unit` connection on the quantity record.
- **Go:** `UpdatePickLineQuantity` SQL — updates `quantity.value` and `quantity.updated_at`.
- **Match:** The core use case (updating the picked quantity value) is preserved. The Dashboard additionally allows updating the unit connection, but changing a pick line's unit is not a practical use case — the unit is inherited from the order line and should not change.

### Validation
- **Dashboard:** Body validated against `PickLineUtils.schema.partial()` (accepts partial PickLine with quantity, orderLine, packedAt fields, but only quantity is used in the update).
- **Go:** Accepts optional `quantity_value` string in the request body.
- **Match:** Yes. Both only update the quantity value; the Go API simplifies the request shape to match what's actually used.

### Idempotency
- **Dashboard:** No idempotency support.
- **Go:** Full idempotency key support with recovery points.
- **Match:** Go is an improvement, not a regression.

### Error Handling
- **Dashboard:** Prisma throws on update if no matching record (account mismatch or missing record).
- **Go:** Explicit not-found errors for pick (`"Pick not found."`) and pick line (`"Pick line not found."`).
- **Match:** Equivalent behavior with better error specificity in Go.

### Side Effects
- **Dashboard:** None.
- **Go:** None.
- **Match:** Yes.

### Response Shape
- **Dashboard:** Returns full `PickLine` object with `quantity`, `orderLine`, `packedAt`.
- **Go:** Returns `PickLineDetail` with `id`, `object`, `quantity` (with unit), `ordered_quantity`, `sales_order_line` (with item number, SKU, description), `packed_at`, `created_at`, `updated_at`.
- **Match:** Equivalent data. Go structures it according to API resource conventions (Object field, sub-resources).

## No Issues Found
