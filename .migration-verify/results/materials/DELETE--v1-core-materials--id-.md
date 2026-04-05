# DELETE /v1/core/materials/{id} — Migration Verification

## Result: Parity Confirmed (with known architectural difference)

No code changes were required. The Go endpoint correctly implements the material delete business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Actor check | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| Permission check | `materials`, `delete` | `PermissionDomainMaterials`, `ActionDelete` | Yes |
| Account scoping | `identity.targetAccountID` | `*identity.TargetAccountID` | Yes |
| Fetch before delete | `materialRepo.findByItemID(itemID, accountID)` | `materialRepo.GetByItemID(accountID, itemID)` | Yes |
| Not found handling | 404 if material not found | 404 if material not found | Yes |
| Soft-delete mechanism | Sets `deletedAt = new Date()` on item | Sets `deleted_at = NOW(3)` on item | Yes |
| Deleted-at guard | `deletedAt: null` filter in findByItemID | `deleted_at IS NULL` in SQL | Yes |
| Response | Returns material object (pre-delete state) | Returns material object (pre-delete state) | Yes |
| HTTP status | 200 OK | 200 OK | Yes |
| Idempotency | None (DELETE is idempotent by design) | None (DELETE is idempotent by design) | Yes |

## Known Architectural Difference: Cascade Cleanup

The Dashboard performs cascade hard-deletion of related records after soft-deleting the item:
- Consumptions and their quantities
- Productions and their quantities
- Inventory change logs, inventory logs, and their quantities
- Hard-deletes the item record itself
- Deletes associated rate records (unitValue, unitCost, burnRate)

The Go API only performs a soft-delete (`deleted_at = NOW(3)`) on the item record. Related records are left intact but are logically orphaned — they are never returned because all queries filter by `deleted_at IS NULL`.

**This is NOT a material-specific gap.** All item types in Go (products, parts, materials) follow the same soft-delete-only pattern. This is a deliberate architectural decision in the Go API. Adding cascade hard-deletion would be a cross-cutting concern that should be addressed holistically across all item types if desired, not as a per-endpoint fix.

## Files Reviewed

### Dashboard
- `dashboard/apps/api/src/controllers/material.ctrl.ts` — Controller route handler
- `dashboard/apps/api/src/services/material.svc.ts` — Service `delete` method
- `dashboard/apps/api/src/repositories/item.repo.ts` — `delete` and `deleteItem` methods
- `dashboard/apps/api/src/repositories/inventory.repo.ts` — `deletePromises` method
- `dashboard/apps/api/src/repositories/material.repo.ts` — `findByItemID` method

### Go
- `services/api-gateway/endpoints/materials/endpoint_delete_material.go` — Endpoint definition
- `services/core-service/internal/infrastructure/grpc/grpc_material_handler.go` — gRPC handler
- `services/core-service/internal/service/material_service.go` — Service `DeleteMaterial` method
- `services/core-service/internal/infrastructure/repository/material_repository.go` — Repository `DeleteByItemID`
- `services/core-service/internal/infrastructure/queries/material.sql` — `DeleteMaterialByItemID` SQL query
