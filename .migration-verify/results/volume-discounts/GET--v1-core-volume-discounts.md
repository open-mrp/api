# GET /v1/core/volume-discounts — Migration Verification

## Result: Issue found and fixed

## What was compared

- **Permission checks**: Dashboard checks `isAssignedActor`, then `discounts:read` for internal actors only. Go matches exactly.
- **Customer actor access**: Dashboard passes `customerID` for customer actors to filter visible discounts. Go sets `CustomerAccountID` from `identity.ActorAccountID()` — matches.
- **Customer filtering SQL**: Dashboard uses Prisma conditions for account group relationships (direct + price group) and "no groups = visible to all". Go SQL uses the same logic with `NOT EXISTS` / `EXISTS` subqueries on `account_group_quantity_discount`, `account_relation`, and `account_relation_price_group` — matches.
- **Search/query filtering**: See issue below.
- **Pagination**: Dashboard uses offset-based (take/skip). Go uses cursor-based pagination. This is an intentional migration-wide change — acceptable.
- **Response shape**: Both return volume discounts with tiers, customer_groups, product_lines, categories, attributes, and acceptable_units sub-resources. Go uses standard list envelope with page_info. Field names and types match.
- **Enrichment/batch loading**: Go batch-loads all sub-resources (tiers, customer groups, product lines, categories, attributes, units) for the list result — matches Dashboard's Prisma eager loading.
- **Side effects**: None for GET — both implementations are read-only.
- **Idempotency**: GET endpoint, not applicable.
- **Error handling**: Both handle missing identity, unauthorized actors appropriately.

## Issue found and fixed

**Search query filtering was incomplete.**

The Dashboard `VolumeDiscountAdapter.fetchInput()` searches across three fields with OR conditions:
1. Volume discount name (contains match)
2. Associated customer group (account group) names (contains match)
3. Associated product line names (contains match)

The Go SQL queries (`ListVolumeDiscountsForward`, `ListVolumeDiscountsBackward`, `ListVolumeDiscountsForCustomerForward`, `ListVolumeDiscountsForCustomerBackward`) only searched by volume discount name.

**Fix**: Added `EXISTS` subqueries to all four list queries to also search across `account_group.name` (via `account_group_quantity_discount` join) and `product_line.name` (via `_product_lines_quantity_discounts` join), matching Dashboard behavior. Regenerated sqlc.

## No remaining concerns
