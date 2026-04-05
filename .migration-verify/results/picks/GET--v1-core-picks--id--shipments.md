# GET /v1/core/picks/{id}/shipments

## Status: Issues found and fixed

## What was compared

- **Permission checks**: Both require internal actor + `picks.read` permission + target account header
- **Request parameters**: Dashboard supports `query` (search), `take` (limit), `skip` (offset); Go only had `pickID`
- **SQL queries**: Dashboard filters through `shipment → order → pick` with account scoping on shipment; Go used subquery on pick's `sales_order_id` with account scoping only on pick
- **Response shape**: Dashboard returns `{ items: string[], count: number }`; Go returned `{ shipment_numbers: string[] }`
- **Ordering**: Both order by `created_at ASC`
- **Side effects**: None (GET endpoint)
- **Idempotency**: N/A (GET endpoint)

## Issues found and fixed

### 1. Missing query/search parameter
**Dashboard** supports an optional `query` parameter that filters shipment numbers using search (full-text in boolean mode). **Go** had no search support.

**Fix**: Added `query` parameter to the request struct, proto message, and SQL query. The Go implementation uses `LIKE %query%` matching (consistent with how other Go list endpoints handle search in this codebase).

### 2. Missing pagination (limit/offset)
**Dashboard** supports `take` and `skip` for pagination. **Go** returned all results with no pagination.

**Fix**: Added `limit` and `offset` parameters across all layers (endpoint request, proto, gRPC handler, service, repository, SQL). Defaults to limit=100 if not provided.

### 3. Missing count in response
**Dashboard** returns a `count` field alongside the items. **Go** only returned the shipment numbers array.

**Fix**: Added `count` field to the response resource and proto. Added a `CountPickShipmentNumbers` SQL query to get the total count. The response now includes both `shipment_numbers` and `count`.

### 4. Missing account_id filter on shipment table
**Dashboard** filters shipments by `accountId = ownerAccountID`. **Go** only checked account on the pick subquery, not on the shipment itself.

**Fix**: Added `s.account_id = sqlc.arg('account_id')` to the shipment WHERE clause for defense-in-depth.

## Files modified

- `proto/core_picking.proto` — Added query, limit, offset to request; count to response
- `services/core-service/internal/infrastructure/queries/pick.sql` — Updated SQL with search, pagination, count query
- `services/core-service/internal/domain/pick_models.go` — Added `GetPickShipmentsParams` and `PickShipmentsResult`
- `services/core-service/internal/domain/repositories.go` — Updated `PickRepo.GetShipmentNumbers` signature
- `services/core-service/internal/domain/services.go` — Updated `PickSvc.GetPickShipments` signature
- `services/core-service/internal/infrastructure/repository/pick_repository.go` — Updated implementation
- `services/core-service/internal/service/pick_svc.go` — Updated implementation
- `services/core-service/internal/infrastructure/grpc/grpc_picking_handler.go` — Pass new params
- `services/api-gateway/endpoints/picks/endpoint_get_pick_shipments.go` — Added query params
- `services/api-gateway/endpoints/picks/service.go` — Pass new params to proto
- `services/api-gateway/pkg/resource/pick_resource.go` — Added Count field

## Remaining notes

- The response field name differs: Dashboard uses `items`/`count`, Go uses `shipment_numbers`/`count`. This is an intentional Go API convention (named fields rather than generic `items`).
- Dashboard uses MySQL boolean mode full-text search; Go uses `LIKE %query%`. Both achieve substring matching on shipment numbers. The Go approach is consistent with other list endpoints in the codebase.
- All tests pass after changes.
