# GET /v1/core/materials/{id} — Migration Verification

## Result: Issue found and fixed

## What was compared

- **Validation**: Both implementations validate the path parameter `id` (item ID). Parity confirmed.
- **Permission checks**: Both check `checkIsInternalActor` + `checkHasPermission(items, read)` + target account set. Parity confirmed.
- **DB query**: Both filter by `item_id`, `account_id`, and `deleted_at IS NULL`. Both JOIN material → item → item_category → quantities (order_point, lead_time) → units → rates (unit_value, unit_cost, burn_rate). Parity confirmed.
- **Error handling**: Dashboard returns 404 "Material not found." when not found; Go returns 404 "Resource not found." via shared `MapSQLError`. Minor message difference is acceptable (shared utility). Parity confirmed.
- **Side effects**: None in either implementation. Parity confirmed.
- **Response shape**: Both return material with nested item (including category, rates, quantities). Parity confirmed.
- **Idempotency**: N/A — GET endpoint, no idempotency keys needed. Parity confirmed.

## Issue found and fixed

**Missing item attributes**: The Dashboard `ItemAdapter.select` always includes `attributes` when fetching a material's item. The Go `GetByItemID` in `material_repository.go` did NOT load item attributes, meaning the `attributes` field on the nested item would always be null.

**Fix**: Added a `loadItemAttributes` helper to `materialRepoImpl` (mirroring the existing pattern in `itemRepoImpl`) and called it after fetching the material in `GetByItemID`. This queries `_item_attributes` for the item and populates `material.Item.Attributes`.

File changed: `services/core-service/internal/infrastructure/repository/material_repository.go`

## No remaining concerns
