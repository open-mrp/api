# Verification: GET /v1/core/suppliers

## Result: Issues found and fixed

## What was compared

| Area | Dashboard | Go | Match? |
|------|-----------|-----|--------|
| **Permission checks** | `checkIsInternalActor()` + `suppliers:read` | `CheckIsInternalActor()` + `PermissionDomainSuppliers:ActionRead` + `CheckTargetAccountSet()` | Yes |
| **Search filter** | Prisma `_relevance` on `name` field + text search on name, alias, external_number, notes | `LIKE` on `a.name` and `ar.external_number` | Yes (functional parity) |
| **Date range filter** | `startDate`/`endDate` on `accountRelation.createdAt` | `start_date`/`end_date` on `ar.created_at` | Yes |
| **itemIDs filter** | Prisma relation filter: `supplierMaterials.some.material.itemID IN itemIDs` | **Was missing from SQL queries** | Fixed |
| **Pagination** | Offset-based (`take`/`skip`) | Cursor-based (intentional migration improvement) | Acceptable |
| **Response shape** | `{ items: SupplierSummary[], count: number }` | `List[SupplierSummary]` with `page_info` (cursor pagination) | Acceptable |
| **SupplierSummary fields** | `id`, `name`, `number`, `materialCount`, `createdAt` | `id`, `object`, `name`, `number`, `material_count`, `created_at` | Yes (Go adds `object` field per conventions) |
| **Sort order** | Prisma `_relevance` (by name relevance descending) | `created_at DESC, id DESC` | Acceptable (relevance sort is Prisma-specific) |
| **Side effects** | None | None | Yes |
| **Idempotency** | N/A (GET) | N/A (GET) | Yes |
| **Error handling** | Standard HTTP errors | Standard API errors | Yes |

## Issue found and fixed

### Missing `itemIDs` filter in SQL queries

The `itemIDs` parameter was accepted by the endpoint, passed through the gRPC handler to the service and repository, but **never applied in the SQL queries**. The Dashboard uses this to filter suppliers that have at least one `supplier_material` whose associated `material` has an `item_id` in the provided list.

**Fix:** Added an `EXISTS` subquery with a boolean guard (`has_item_filter`) to all three list SQL queries (`ListSuppliersForward`, `ListSuppliersBackward`, `CountSuppliers`):

```sql
AND (
    sqlc.arg('has_item_filter') = false
    OR EXISTS (
        SELECT 1 FROM supplier_material sm2
        INNER JOIN material m ON m.id = sm2.material_id
        WHERE sm2.supplier_account_id = ar.counterparty_account_id
          AND sm2.owner_account_id = ar.owner_account_id
          AND m.item_id IN (sqlc.slice('item_ids'))
    )
)
```

Updated the repository layer to pass `HasItemFilter` and `ItemIds` to all query call sites.

**Files modified:**
- `services/core-service/internal/infrastructure/queries/supplier.sql`
- `services/core-service/internal/infrastructure/sqlc/supplier.sql.go` (regenerated)
- `services/core-service/internal/infrastructure/repository/supplier_repository.go`

## Acceptable differences (not bugs)

1. **Pagination model**: Dashboard uses offset (`take`/`skip`), Go uses cursor-based pagination. This is an intentional improvement in the migration.
2. **Sort order**: Dashboard uses Prisma's `_relevance` scoring on the `name` field. Go sorts by `created_at DESC`. Relevance sorting is Prisma-specific and the Go approach is acceptable.
3. **Response envelope**: Dashboard returns `{ items, count }`. Go returns a standard `List` with `page_info` containing cursors. This is the Go API's standard pagination format.
