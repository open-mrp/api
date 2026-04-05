# Verification: GET /v1/core/suppliers/{supplier_id}/materials

## Result: Parity Confirmed

No issues found. The Go implementation is a faithful migration of the Dashboard endpoint.

## What Was Compared

| Area | Dashboard | Go | Status |
|------|-----------|-----|--------|
| **Permission checks** | `checkIsInternalActor`, `suppliers:read` | `CheckIsInternalActor()`, `CheckHasPermission(suppliers, read)`, `CheckTargetAccountSet()` | Match |
| **Account scoping** | `ownerAccountID` from identity | `OwnerAccountID` from `identity.TargetAccountID` | Match |
| **Supplier filter** | `supplierAccountID: supplierID` | `supplier_account_id = ?` | Match |
| **Search fields** | `supplierPartNumber`, `supplierDescription`, `material.item.sku`, `material.item.description` | Same 4 fields via SQL LIKE | Match |
| **isActive filter** | Repo supports it, but controller does NOT expose it | Not exposed in list endpoint | Match |
| **Soft delete filter** | Prisma handles transparently | Explicit `i.deleted_at IS NULL` | Match |
| **Pagination** | Offset-based (`take`/`skip`) | Cursor-based (`cursor`/`limit`) | Intentional architectural change |
| **Ordering** | `_relevance` (full-text search) | `created_at DESC, id DESC` | Intentional (cursor pagination requires deterministic order) |
| **Response shape** | `{ items, count }` | Standard list with `page_info` | Follows Go API conventions |
| **Side effects** | None | None | Match |
| **Idempotency** | N/A (GET) | N/A (GET) | Match |
| **Expandable fields** | N/A (dashboard returns full objects) | `material`, `material.item` via include system | Go API enhancement |

## Issues Found and Fixed

None.

## Remaining Concerns

None.
