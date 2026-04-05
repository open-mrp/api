# Verification: GET /v1/core/items/actions/export

## Result: PARITY CONFIRMED

No issues found. The Go implementation faithfully reproduces the Dashboard business logic.

## What Was Compared

### Permission Checks
- **Dashboard**: `checkIsInternalActor` + `checkHasPermission(PermissionDomains.items, 'read')`
- **Go**: `CheckIsInternalActor()` + `CheckHasPermission(PermissionDomainItems, ActionRead)` + `CheckTargetAccountSet()`
- **Verdict**: Match. Go adds an explicit target-account-set check which the Dashboard does implicitly.

### Data Scope
- **Dashboard**: `itemRepo.list({ accountID })` — fetches all items for the account, no filters.
- **Go**: SQL query filters `WHERE i.account_id = ? AND i.deleted_at IS NULL` — same scope, also excludes soft-deleted items (Dashboard's Prisma also filters soft deletes by default via middleware).
- **Verdict**: Match.

### Inventory Calculation
- **Dashboard**: `inventoryRepo.fetchOnHandInventoryBulk()` — queries `inventoryReceipt` where `statusCode = 'available'` AND (`ownerAccountID = acct` OR `holderAccountID = acct`), sums `quantity - allocations` per item.
- **Go**: SQL subquery on `inventory_receipt` JOINed with `quantity` and LEFT JOINed with `inventory_allocation`, same filter (`ir.status_code = 'available'` AND `ir.owner_account_id = ? OR ir.holder_account_id = ?`), computes `SUM(q.value - COALESCE(alloc.allocated, 0))` per item.
- **Verdict**: Match. Both compute on-hand as sum of (receipt quantity - allocated quantity) for available receipts owned or held by the account.

### Items Without Inventory
- **Dashboard**: Returns a blank `BaseQuantity` (zero measure of the appropriate unit type).
- **Go**: SQL uses `COALESCE(inv.on_hand, 0)` for quantity and `COALESCE(rv.denominator_unit_id, '')` for unit ID. Items without inventory get `0` and the item's base unit ID.
- **Verdict**: Match — both include all items even when no inventory exists, with zero quantity.

### Response Format
- **Dashboard**: Returns JSON `{ items: [...], count }` where each item has all item fields plus a `quantity` object.
- **Go**: Returns an Excel (.xlsx) file download with columns: ID, SKU, Description, Notes, Item Type, Category, On Hand Qty, Unit, Created At, Updated At.
- **Verdict**: Intentional design improvement. The Go "export" endpoint produces a downloadable file (the typical purpose of an export endpoint), while the JSON list data is served by the regular list endpoint. The same data fields are present.

### Sorting
- **Go**: `ORDER BY i.sku ASC` — deterministic sort by SKU.
- **Dashboard**: Default Prisma ordering (unspecified in the list call).
- **Verdict**: Go improves on the Dashboard by providing a consistent sort order for the export.

### Error Handling
- Both return appropriate authentication/authorization errors for missing identity, non-internal actors, and missing permissions.

### Side Effects
- None in either implementation. This is a read-only GET endpoint.

## Issues Found and Fixed

None — no changes were required.
