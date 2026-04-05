# Verification: GET /v1/core/items/{id}

## Result: Issues found and fixed

## Context

The Dashboard does not expose a dedicated `GET /items/:id` endpoint. The `ItemSvc.find` / `ItemRepo.find` method is used internally (e.g., by `updateInventory`). The Go endpoint is a new public API surface, so verification focused on ensuring the Go implementation correctly mirrors the Dashboard's internal `find` behavior and data shape.

## What was compared

| Aspect | Dashboard | Go | Parity? |
|--------|-----------|-----|---------|
| Permissions | `checkIsInternalActor` + `checkHasPermission(items, read)` | `CheckIsInternalActor()` + `CheckHasPermission(PermissionDomainItems, ActionRead)` + `CheckTargetAccountSet()` | Yes |
| Account isolation | `accountID` param on Prisma query | `account_id` in SQL WHERE clause | Yes |
| Soft delete filter | `deletedAt: null` | `deleted_at IS NULL` | Yes |
| Not-found error | Returns null, caller throws `HttpError.notFound` | `db.MapSQLError` maps `sql.ErrNoRows` to not-found | Yes |
| Item fields | id, sku, description, typeCode, notes, createdAt, updatedAt | id, sku, description, item_type_code, notes, is_dirty, created_at, updated_at | Yes |
| Category join | Prisma relation (`CategoryAdapter.select`) | `JOIN item_category` (id, name, type_code, unit_group_id) | Yes |
| Rate joins (x3) | Prisma relations (`RateAdapter.select`) for unitValue, unitCost, burnRate | `JOIN rate` (x3) with id, value, numerator/denominator unit IDs, timestamps | Yes |
| Attributes | Prisma relation (`AttributeAdapter.select`) | Separate `GetItemAttributes` query (id, text, color_code, property_id) | Yes |
| Expandable sub-resources | N/A (Dashboard returns full objects) | `category`, `unit_value`, `unit_cost`, `burn_rate` via IncludeConfig | Yes |
| Idempotency | N/A (GET) | N/A (GET) | Yes |
| Side effects | None | None | Yes |

## Issues found and fixed

### 1. `lightAttributePresenter` was dropping `ColorCode` from attributes

**File:** `services/api-gateway/endpoints/items/presenter.go`

The `lightAttributePresenter` function only populated `ID`, `Object`, and `Text` on the `Attribute` resource, but the `ColorCode` field is available in the proto (`ItemAttributeInfo.color_code`) and is marked `validate:"required"` on the `apiresource.Attribute` struct.

**Fix:** Added `ColorCode` mapping from the proto's optional `color_code` field, with nil-safe dereferencing.

## Notes

- The Go endpoint includes `is_dirty` which the Dashboard's `find` does not surface externally (but it's available in the DB). This is acceptable as new API surface.
- The Dashboard's `Item` type includes `startDate` and `endDate` which are always set to `null` by the adapter — these are not needed in the Go API.
- The Go SQL query uses INNER JOINs for category and rates, which is correct since these are required foreign keys in the schema.
