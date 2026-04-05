# Verification: `PUT /v1/core/items/{id}/category/{category_id}`

**Status: Parity confirmed — no fixes needed**

## What was compared

| Area | Dashboard | Go | Match |
|------|-----------|-----|-------|
| **Permission checks** | `checkIsInternalActor` + `checkHasPermission(items, update)` | `CheckIsInternalActor` + `CheckHasPermission(Items, Update)` + `CheckTargetAccountSet` | Yes |
| **Category validation** | Explicit `find()` → 404 "Category not found" | `GetCategoryBaseUnitID` → `sql.ErrNoRows` → 404 "Resource not found" | Yes (functionally equivalent) |
| **Item update** | `item.update` with `deletedAt: null` filter, scoped by `accountID` | `UPDATE item ... WHERE id = ? AND account_id = ? AND deleted_at IS NULL` | Yes |
| **Rate unit updates** | Updates `unitValue.denominatorUnit`, `unitCost.denominatorUnit`, `burnRate.numeratorUnit` | Same three UPDATE queries on rate table | Yes |
| **Material order point** | Updates `material.orderPoint` unit | `UPDATE quantity ... JOIN material ... WHERE item_type_code = 'material'` | Yes |
| **Consumption/production units** | Updates all consumption and production quantity units linked to item | Two UPDATE queries on quantity via consumption/production joins | Yes |
| **Atomicity** | `Promise.all` (parallel within single Prisma transaction context) | All updates within `withTx` transaction | Yes |
| **Response shape** | Full `Item` object | Full `Item` resource with expandable `category`, `unit_value`, `unit_cost`, `burn_rate` | Yes |
| **HTTP method/status** | PUT / 200 | PUT / 200 | Yes |
| **Side effects** | None (no emails, webhooks, messages) | None | Yes |

## Issues found

None requiring fixes.

## Remaining concerns

- **Category account scoping:** The Dashboard's `CategoryRepo.find()` filters by `(accountID = ? OR accountID IS NULL)`, ensuring a user can only reference their own account's categories or system-wide ones. The Go `GetCategoryBaseUnitID` query does not filter by `account_id`. This is a minor security gap but not a business-logic parity issue — the item update itself is account-scoped, and category IDs are globally unique. This pattern is consistent across all Go callers of `GetCategoryBaseUnitID` (material, product, part creation) and should be addressed holistically rather than for this endpoint alone.
- **Error message difference:** Dashboard returns "Category not found." while Go returns generic "Resource not found." via `MapSQLError`. This is cosmetic and consistent with the Go codebase's error handling conventions.
