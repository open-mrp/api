# GET /v1/core/shipments — Migration Verification

**Status: Issues found and partially fixed**

## Files Compared

### Dashboard (Legacy)
- `dashboard/apps/api/src/services/shipment.svc.ts` — Service layer (permission checks, delegation)
- `dashboard/apps/api/src/repositories/shipment.repo.ts` — Repository (Prisma queries, pagination, ordering)
- `dashboard/packages/adapters/src/classes/shipments/Shipment.ts` — Adapter (fetchInput WHERE clause builder)
- `dashboard/packages/adapters/src/classes/shipments/ShipmentSummary.ts` — Summary adapter (select, map)
- `dashboard/packages/adapters/src/classes/orders/Order.ts` — Order adapter (fetchInput for order-level filters)

### Go (New)
- `services/api-gateway/endpoints/shipments/endpoint_list_shipments.go` — Endpoint definition
- `services/api-gateway/endpoints/shipments/service.go` — API gateway service
- `services/api-gateway/endpoints/shipments/presenter.go` — Response presenter
- `services/api-gateway/pkg/resource/shipment_resource.go` — API resource types
- `services/core-service/internal/service/shipment_service.go` — Core service layer
- `services/core-service/internal/infrastructure/grpc/grpc_shipping_handler.go` — gRPC handler
- `services/core-service/internal/infrastructure/queries/shipment.sql` — SQL queries
- `services/core-service/internal/infrastructure/repository/shipment_repository.go` — Repository (TODO stub)
- `services/core-service/internal/domain/shipment_models.go` — Domain models

## What Was Compared

- Permission checks (actor type, permission domain, action)
- Query parameters / filters
- DB query logic (joins, filters, search, ordering, pagination)
- Response shape (field names, types, nested resources)
- Error handling

## Issues Found and Fixed

### 1. Permission Check — Internal Actor Only (FIXED)
**Problem:** The Go service used `CheckIsAssignedActor()` which allows both internal and customer actors. The Dashboard uses `checkIsInternalActor()` which restricts to internal users only. Customer actors were inadvertently granted access to list shipments without any permission check.

**Fix:** Changed `shipment_service.go` ListShipments to use `CheckIsInternalActor()` and unconditional `CheckHasPermission()` (no longer wrapped in `if identity.IsInternalUser()`).

### 2. Search Query Missing Order-Level Fields (FIXED)
**Problem:** The Go SQL search only searched shipment-level fields (`s.number`, `s.note`, `s.bill_of_lading`, `s.master_tracking_number`). The Dashboard searches both shipment number AND order-level fields (order number, customer/buyer account name, customer PO number).

**Fix:** Added `so.number`, `ba.name`, and `so.customer_po_number` to the search clause in both `ListShipmentsForward` and `ListShipmentsBackward` SQL queries.

## Remaining Concerns (Not Fixed)

### 3. Response Shape Differences
The Go `ShipmentSummary` resource is restructured to follow Go API conventions (sub-resources with `object` types), which is an intentional improvement. However, several fields from the Dashboard summary are missing:

| Dashboard Field | Go Equivalent | Status |
|---|---|---|
| `shipTo` (address object) | — | **Missing** — Dashboard returns the shipping address on each summary |
| `priority` (from order) | — | **Missing** — Dashboard returns the order priority |
| `caseCount` (shipping cases count) | — | **Missing** — Dashboard returns number of shipping cases |
| `isReadyToShip` (computed) | — | **Missing** — Dashboard computes: `status == packed && all cases have weight > 0` |
| `orderID` (string) | `sales_order` (sub-resource) | Restructured (improvement) |
| `carrierName` (string) | `carrier` (sub-resource) | Restructured (improvement) |

These missing fields would require coordinated changes across SQL, proto, domain models, gRPC handlers, API resources, and presenters. They should be added before this endpoint is considered fully migrated, as the frontend likely depends on them.

### 4. Pagination Model Difference
The Dashboard uses offset pagination (`take`/`skip` with `count`), while the Go API uses cursor pagination. This is an intentional architectural decision in the Go API and not a bug, but consumers will need to adapt.

### 5. Ordering Difference
The Dashboard uses relevance-based ordering (Prisma full-text search on `number` field) when a search query is provided, falling back to `createdAt DESC`. The Go API always orders by `created_at DESC, id DESC`. This means search results may appear in a different order.

### 6. Repository Implementation is a Stub
The repository (`shipment_repository.go`) has TODO stubs and returns empty results. It needs to be wired up to the sqlc-generated queries after running `make sqlc core`.
