# DELETE /v1/core/shipments/{id} — Verification Result

**Status:** Issues found and fixed

## What was compared

- Permission checks (actor type, permission domain, action)
- Business logic (pick unpacking before deletion)
- DB queries and transaction logic
- Cascading delete order (shipping cases -> shipment lines -> shipment)
- Error handling
- Response shape

## Issues found and fixed

### 1. Wrong permission action (fixed)

**Dashboard:** `checkHasPermission(this.identity, PermissionDomains.shipments, 'update')`
**Go (before):** `identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionDelete)`
**Fix:** Changed to `types.ActionUpdate` to match Dashboard behavior.

**File:** `services/core-service/internal/service/shipment_service.go`

### 2. Missing pick unpack logic (fixed)

**Dashboard:** Before deleting a shipment, calls `pickRepo.unpackByShipment()` which:
1. For each shipment line, finds the corresponding pick line by order line ID and clears `packed_at` (sets to NULL)
2. Finds the pick associated with the shipment's order and clears `finished_at` (marks pick as unpacked)

**Go (before):** Did not perform any pick unpack logic — went straight to deleting shipping cases, shipment lines, and shipment.

**Fix:** Added unpack logic to the transaction before the delete cascade:
- Added `PickLineRepo.UnpackByShipment(ctx, shipmentID)` — clears `packed_at` on all pick lines whose `sales_order_line_id` matches a line in the shipment
- Added `PickRepo.FindIDByShipmentOrder(ctx, accountID, shipmentID)` — finds the pick for the shipment's sales order
- Calls `PickRepo.ClearFinishedAt()` on the found pick to mark it unpacked

**Files modified:**
- `services/core-service/internal/domain/repositories.go` — added `FindIDByShipmentOrder` to `PickRepo`, `UnpackByShipment` to `PickLineRepo`
- `services/core-service/internal/infrastructure/queries/pick.sql` — added `UnpackPickLinesByShipment` and `FindPickIDByShipmentOrder` SQL queries
- `services/core-service/internal/infrastructure/repository/pick_repository.go` — added stub for `FindIDByShipmentOrder`
- `services/core-service/internal/infrastructure/repository/pick_line_repository.go` — added stub for `UnpackByShipment`
- `services/core-service/internal/service/shipment_service.go` — added unpack logic before delete cascade

## Acceptable differences

- **Response shape:** Dashboard returns HTTP 200 with the deleted shipment object. Go returns HTTP 204 No Content. This is an intentional REST convention improvement and is acceptable.

## Remaining concerns

- Repository implementations for `UnpackByShipment` and `FindIDByShipmentOrder` are stubs (TODO). They need `make sqlc core` to be run to generate the sqlc code, then the stubs should be filled in with actual query calls.
- Mocks will need regeneration via `make mocks core` after sqlc generation.
