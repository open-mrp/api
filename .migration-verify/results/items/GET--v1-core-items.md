# Verification: GET /v1/core/items

## Result: Issues found and fixed

## What was compared
- Permission checks (actor type, permission domain, action)
- Validation rules (query params, filter formats)
- DB queries and logic (filters, joins, ordering, pagination)
- Search behavior (exact match vs partial match)
- Response shape (field names, types, nested resources, expandables)
- Side effects (none expected for GET)

## Issues found and fixed

### 1. Missing product type filter
**Dashboard behavior:** Filters items where either `product IS NULL` (no product for the item) or `product.productTypeCode = 'sale'`. This excludes items with non-sale product types.
**Go behavior (before fix):** No product type filter — all items regardless of product type were returned.
**Fix:** Added product type filter to both `ListItemsForward` and `ListItemsBackward` SQL queries:
```sql
AND (
    NOT EXISTS (SELECT 1 FROM product p WHERE p.item_id = i.id)
    OR EXISTS (SELECT 1 FROM product p WHERE p.item_id = i.id AND p.product_type_code = 'sale')
)
```

### 2. Missing `onlyInitialSubassemblies` filter
**Dashboard behavior:** When `onlyInitialSubassemblies=true`, filters to items that have at least one production whose production step has no parent steps (i.e., root/initial steps in the production chain).
**Go behavior (before fix):** The `only_initial_subassemblies` parameter was accepted in the endpoint and passed through gRPC, but was never used in the SQL queries.
**Fix:** Added the filter to both SQL queries and passed the parameter from the repository to sqlc:
```sql
AND (
    sqlc.arg('only_initial_subassemblies') = false
    OR EXISTS (
        SELECT 1 FROM production prd
        WHERE prd.item_id = i.id
        AND prd.production_step_id IS NOT NULL
        AND NOT EXISTS (
            SELECT 1 FROM _parent_child_production_steps pcps
            WHERE pcps.B = prd.production_step_id
        )
    )
)
```

### 3. Exact match search also searches description
**Dashboard behavior:** When `isExactMatch=true`, SKU is matched exactly (`equals: query`) but description is still searched with full-text/partial match. Both conditions are OR'd.
**Go behavior (before fix):** When `is_exact_match=true`, only exact SKU match was performed. Description was not searched at all.
**Fix:** Updated the SQL search clause so that when `is_exact_match=true`, it matches exact SKU OR partial description (LIKE). Also simplified `buildItemSearchParams` to always produce the LIKE-wrapped query since the SQL handles the exact match logic separately via the `search_exact` parameter.

## Files modified
- `services/core-service/internal/infrastructure/queries/item.sql` — Added product type filter, onlyInitialSubassemblies filter, and fixed exact match search in both Forward and Backward queries
- `services/core-service/internal/infrastructure/repository/item_repository.go` — Simplified `buildItemSearchParams`, added `OnlyInitialSubassemblies` to all sqlc param structs
- `services/core-service/internal/infrastructure/sqlc/item.sql.go` — Regenerated via `make sqlc core`

## Accepted differences (not parity gaps)
- **Pagination model:** Dashboard uses offset-based (`take`/`skip`), Go uses cursor-based. This is an intentional migration improvement.
- **Response shape:** Dashboard returns `{ items, count }`, Go returns `{ data, page_info }` per new API conventions.
- **Search implementation:** Dashboard uses Prisma full-text search with relevance ordering; Go uses SQL LIKE. Both find the same items; relevance ordering is not applicable with cursor-based pagination's stable sort.
