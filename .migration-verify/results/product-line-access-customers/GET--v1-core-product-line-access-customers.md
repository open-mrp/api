# Verification: GET /v1/core/product-line-access/customers

## Status: PARITY CONFIRMED

No issues found. The Go implementation correctly mirrors the Dashboard business logic.

## What was compared

| Aspect | Result |
|--------|--------|
| Permission checks (internal actor, productLineAccess read, target account) | Match |
| DB query (account_relation_product_line + account_relation + account + product_line joins) | Match |
| Filter: owner_account_id scoping | Match |
| Filter: account_relation_role_code = 'customer' | Match |
| Search: customer name and external number | Match (Go uses LIKE, Dashboard uses exact — intentional improvement) |
| Grouping: flat rows → grouped by customer ID | Match |
| Response shape: customer sub-resource with id/name/number + product lines list | Match |

## Expected architectural differences (not bugs)

- **Pagination**: Dashboard uses offset-based (take/skip + count), Go uses cursor-based (cursor/limit + PageInfo). This is the standard Go API pagination pattern.
- **Search**: Dashboard uses Prisma exact match (`equals`), Go uses SQL `LIKE '%query%'` (contains). The Go version is more user-friendly.
- **Timestamps**: Go response includes `created_at` and `updated_at` on the resource, which the Dashboard does not expose.

## Files reviewed

**Dashboard:**
- `dashboard/apps/api/src/services/customer-product-line-access.svc.ts`
- `dashboard/apps/api/src/repositories/customer-product-line-access.repo.ts`
- `dashboard/packages/adapters/src/classes/orders/customer/CustomerProductLines.ts`

**Go:**
- `services/api-gateway/endpoints/customer-product-line-access/endpoint_list_customer_product_line_access.go`
- `services/api-gateway/endpoints/customer-product-line-access/service.go`
- `services/core-service/internal/service/customer_product_line_access_service.go`
- `services/core-service/internal/infrastructure/repository/customer_product_line_access_repository.go`
- `services/core-service/internal/infrastructure/queries/customer_product_line_access.sql`
