# Verification: GET /v1/core/purchase-orders/statuses

## Result: PARITY CONFIRMED

No issues found. No changes needed.

## What Was Compared

### Dashboard (Express.js)
- **Controller:** `purchase-order.ctrl.ts` — `getPurchaseOrderStatuses`
- **Service:** `order-status.svc.ts` — delegates to repository after `checkIsAssignedActor` permission check
- **Repository:** `order-status.repo.ts` — queries `orderStatus` table with Prisma, supports `query` (text search on name), `take`/`skip` (offset pagination)
- **Response shape:** `{ items: [{ code, name, createdAt, updatedAt }], count }`

### Go API
- **Endpoint:** `endpoint_list_purchase_order_statuses.go` — `GET /v1/core/purchase-orders/statuses`
- **Gateway service:** `service.go` — `ListPurchaseOrderStatuses` calls gRPC `ListSalesOrderStatuses`
- **gRPC handler:** `grpc_sales_service_handler.go` — `ListSalesOrderStatuses`
- **Domain service:** `sales_order_status_service.go` — checks `CheckIsAuthenticated()`, delegates to repo
- **Repository:** `sales_order_status_repository.go` — cursor-based pagination with search
- **SQL:** `sales_order_status.sql` — queries `sales_order_status` table with optional `name LIKE` search
- **Resource:** `sales_order_status_resource.go` — `{ id, object, code, name, created_at, updated_at }`

## Comparison Details

| Aspect | Dashboard | Go | Match? |
|---|---|---|---|
| DB table | `orderStatus` | `sales_order_status` | Same table |
| Search | `query` param, fulltext on name | `query` param, LIKE on name | Equivalent |
| Pagination | Offset (`take`/`skip`) | Cursor-based (`cursor`/`limit`) | Architectural choice |
| Auth | `checkIsAssignedActor` (internal/customer/supplier) | `CheckIsAuthenticated()` | Equivalent — all assigned actors are authenticated |
| Permission domain | None (just actor type check) | None | Match |
| Account filter | None (global data) | None (global data) | Match |
| Response fields | code, name, createdAt, updatedAt | id, object, code, name, created_at, updated_at | Go adds id/object per conventions |
| Idempotency | N/A (GET) | N/A (GET) | Match |
| Side effects | None | None | Match |

## Notes

- The Go API uses cursor-based pagination instead of offset-based — this is the standard pagination pattern for all Go API list endpoints, not a discrepancy.
- The Go API adds `id` and `object` fields per API resource conventions — expected enhancement.
- Both implementations query the same underlying database table with equivalent search/filter capabilities.
