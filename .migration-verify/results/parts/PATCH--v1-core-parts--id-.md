# PATCH /v1/core/parts/{id} — Migration Verification

## Result: Issues found and fixed

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Permission check (actor) | `checkIsInternalActor` | `CheckIsInternalActor` | Yes |
| Permission check (domain) | `PermissionDomains.items` / `update` | `PermissionDomainParts` / `ActionUpdate` | **No — fixed** |
| Existence check | `part.findUnique({ where: { itemID } })` → 404 | `partRepo.Get(ctx, {AccountID, ItemID})` → 404 | Yes |
| SKU uniqueness | `itemRepo.isDuplicate()` excluding current ID | `partRepo.ExistsBySKU()` excluding current item | Yes |
| Fields updated | `sku`, `description`, `notes` via Prisma | `sku`, `description`, `notes` via sqlc | Yes |
| Null-clearing (description/notes) | Prisma sets field to null when value is null | Go uses `UpdateDescription`/`UpdateNotes` flags + `SetItemDescription`/`SetItemNotes` queries | Yes |
| Touch part updated_at | `part.update({ updatedAt: new Date() })` | `partRepo.TouchUpdatedAt(ctx, itemID)` — `NOW(3)` | Yes |
| Transaction | Prisma optional `context` | `withTx` wrapping all mutations | Yes |
| Idempotency | N/A (dashboard doesn't use idempotency keys) | Idempotency key with recovery points | Yes (Go improvement) |
| Response shape | Returns full Part via `PartAdapter.map` | Returns full Part via `partRepo.Get` after update | Yes |
| HTTP status | 200 OK | 200 OK | Yes |
| Side effects | None | None | Yes |

## Issues found and fixed

### 1. Permission domain mismatch (FIXED)

**File:** `services/core-service/internal/service/part_service.go:278`

The Dashboard uses `PermissionDomains.items` (value `"items"`) for the update permission check. The Go code was using `PermissionDomainParts` (value `"parts"`), which is a different permission domain. This would cause users with `items:update` permission to be rejected by the Go API.

**Fix:** Changed `PermissionDomainParts` → `PermissionDomainItems` for the update action.

## Remaining concerns

1. **Create endpoint also uses `PermissionDomainParts`** (line 136) instead of `PermissionDomainItems`. This is the same type of mismatch but is outside the scope of this endpoint's verification.
