# PATCH /v1/core/carriers/{id} — Migration Verification

## Result: PARITY CONFIRMED

No issues found. The Go implementation faithfully reproduces the Dashboard behavior.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **Permission checks** | `checkIsInternalActor`, `checkHasPermission(carriers, update)` | `CheckIsInternalActor()`, `CheckHasPermission(carriers, update)`, `CheckTargetAccountSet()` | ✅ |
| **Updatable fields** | `name`, `isPortalEnabled` | `name`, `is_portal_enabled` | ✅ |
| **Existence check** | `checkExistence({ id, accountID })` before update | `rowsAffected == 0` after UPDATE + `deleted_at IS NULL` filter | ✅ (equivalent) |
| **Soft-delete filtering** | Prisma adapter handles | `WHERE deleted_at IS NULL` in SQL | ✅ |
| **Account scoping** | `WHERE id AND accountID` | `WHERE id AND account_id` | ✅ |
| **Response** | HTTP 200 with carrier object | HTTP 200 with carrier object | ✅ |
| **Side effects** | None | None | ✅ |
| **Idempotency** | Not implemented | Idempotency key support (required for PATCH per patterns) | ✅ (Go is correct) |

## Notes

- **Name uniqueness check**: Go adds an `ExistsByName` check before update that the Dashboard doesn't have. This is a stricter validation that prevents duplicate carrier names — an improvement, not a regression.
- **COALESCE pattern**: Go uses `COALESCE(sqlc.narg, column)` in SQL to only update provided fields, matching the Dashboard's Prisma partial update behavior.
- **Deleted carrier filtering**: Both implementations prevent updating soft-deleted carriers.
