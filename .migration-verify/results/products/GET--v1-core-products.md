# Verification: GET /v1/core/products

**Status: Issues found and fixed**

## What was compared

- Permission checks (actor type, permission domain, action)
- Query parameters and filter logic
- DB queries (joins, conditions, ordering, pagination)
- Customer product-line access filtering (3 paths)
- Response shape and nested resources
- Error handling

## Issues found and fixed

### 1. Missing `owner_account_id` check in customer filter subqueries (security fix)

The Dashboard Prisma queries include `ownerAccountID: accountID` in all three customer-access subqueries, ensuring that account relations are scoped to the current owner account. The Go SQL was missing this check, which could return product lines accessible via account relations owned by other accounts.

**Fix:** Added `AND ar.owner_account_id = i.account_id` to all three customer filter subqueries in both `ListProductsFullForward` and `ListProductsFullBackward`.

### 2. Product line + customer filter logic: AND vs OR (behavioral fix)

The Dashboard combines `productLineIDs` and `customerIDs` filters with OR logic inside the product line relation filter. When both are provided, a product matches if its product line is in the direct list OR accessible to the specified customers.

The Go SQL had these as separate AND conditions, meaning both had to be satisfied simultaneously (stricter behavior).

**Fix:** Combined the `include_product_line_filter` and `include_customer_filter` blocks into a single OR expression:
- Neither filter active -> pass through
- Product line filter active -> match by direct product line IDs
- Customer filter active -> match by customer-accessible product lines
- Both active -> match if EITHER condition is satisfied (OR logic)

## Confirmed parity (no issues)

- **Permission checks**: Both check `isAssignedActor`, then customer actors get `customerIDs` overridden to their own account ID and `isPortalReady` forced to `true`. Internal actors require `items.read` permission. Invalid actor types rejected.
- **Query parameters**: Same set of filters supported (query, customerIDs, productLineIDs, categoryIDs, attributeIDs, startDate, endDate, isPortalReady). Go uses cursor pagination instead of offset - expected for the migration.
- **Search logic**: Both search `sku` and `description` with LIKE/contains matching. Dashboard additionally sorts by full-text relevance when a query is present; Go sorts by `created_at DESC` always. This is an acceptable difference for cursor-based pagination.
- **Category and attribute filters**: Both use the same logic (IN clause for categories, EXISTS subquery on `_item_attributes` for attributes).
- **Date filters**: Both filter on `item.created_at` with `>=` / `<=`.
- **Product type filter**: Both hardcode `product_type_code = 'sale'`.
- **Soft delete**: Both check `deleted_at IS NULL`.
- **Response shape**: Go returns Product resource with expandable sub-resources (item, productType, productLine) matching the Dashboard's nested structure.

## Remaining concerns

- **Search ordering**: Dashboard uses Prisma full-text relevance scoring when a search query is present. Go always orders by `created_at DESC`. This means search result ordering may differ, but the result set is the same. This is acceptable given cursor-based pagination.
- The `isExactMatch` parameter from Dashboard is not exposed in the Go API endpoint. This was only used for internal service-to-service calls (not the public API), so this is acceptable.
