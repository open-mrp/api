# Verification: GET /v1/core/suppliers/{supplier_id}/materials/{id}

## Result: Parity Confirmed

No issues found. The Go implementation correctly matches the Dashboard business logic.

## What Was Compared

### Permission Checks
- **Dashboard:** `checkIsInternalActor` + `checkHasPermission(suppliers, read)`
- **Go:** `CheckIsInternalActor()` + `CheckHasPermission(PermissionDomainSuppliers, ActionRead)` + `CheckTargetAccountSet()`
- **Verdict:** Match. Go additionally validates target account is set (consistent Go pattern).

### Request Parameters
- **Dashboard:** Extracts `supplierID` and `itemID` from URL params
- **Go:** Extracts `supplier_id` (path) and `id` (path, mapped to ItemID)
- **Verdict:** Match. Same two path parameters used.

### DB Query & Logic
- **Dashboard (Prisma):** `findFirst` on `supplierMaterial` where `supplierAccountID = supplierID`, `material.itemID = itemID`, `ownerAccountID = ownerAccountID`
- **Go (SQL):** Joins `supplier_material → material → item → item_category`, plus left joins for quantities and joins for rates. Filters by `supplier_account_id`, `item_id` (via material join), `owner_account_id`, and `deleted_at IS NULL` on item.
- **Verdict:** Match. Go adds `deleted_at IS NULL` filter on items, which is a safe guard against returning supplier materials for soft-deleted items. The Go query also fetches more joined data (rates, quantities, category) to support the expandable material/item sub-resources.

### Error Handling
- **Dashboard:** Returns 404 with "Supplier material not found." if repo returns null
- **Go:** sqlc `:one` query returns `sql.ErrNoRows` → `MapSQLError` returns 404 with "Resource not found."
- **Verdict:** Match. Error message differs slightly ("Supplier material not found." vs "Resource not found.") but this is the standard Go pattern across all endpoints.

### Response Shape
- **Dashboard:** `{ id, supplierPartNumber, supplierDescription, isActive, supplier: { id, name }, item: { itemID, materialID, sku, description }, createdAt, updatedAt }`
- **Go:** `{ id, object, material (expandable), supplier_part_number, supplier_description, is_active, created_at, updated_at }` with expandable `material` → `{ id, object, item (expandable), order_point, lead_time, created_at, updated_at }`
- **Verdict:** Match (with expected shape differences). Go uses the new API resource conventions: `object` field, snake_case, expandable sub-resources for `material` and `material.item` instead of inlined flat fields.

### Side Effects
- None in either implementation (read-only GET endpoint).

### Idempotency
- GET endpoint — idempotent by nature, no idempotency key needed. Neither implementation uses one.

## Notes
- No code changes were required.
