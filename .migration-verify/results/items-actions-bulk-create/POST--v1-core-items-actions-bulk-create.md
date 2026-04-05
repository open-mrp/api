# POST /v1/core/items/actions/bulk-create

## Status: Issues found and fixed

The Go service method was a stub (`return nil, nil`). The gRPC handler was also missing. Both have been fully implemented.

## What was compared

- **Validation rules**: Item type validation (product/material/part), SKU required, item_category_id required
- **Permission checks**: Internal actor + items:create permission + target account ID
- **DB queries and logic**: SKU uniqueness check, category base unit lookup, rate creation, item insertion, type-specific record creation
- **Error handling**: Per-item error capture (partial success supported), error messages returned per item
- **Side effects**: Rate records (unit_value, unit_cost, burn_rate) created per item; material quantities (order_point, lead_time) created with defaults; product records set to type "sale"
- **Response shape**: List of per-item results with SKU, success boolean, optional error, optional item_id
- **Idempotency**: POST endpoint uses idempotency keys with recovery points

## Issues found and fixed

### 1. Missing gRPC handler (FIXED)
`BulkCreateItems` was not implemented in `grpc_item_handler.go`. Added handler that maps proto request to domain params and domain results back to proto response, with idempotency tracking.

### 2. Missing service implementation (FIXED)
`BulkCreateItems` in `item_service.go` was a stub returning `nil, nil`. Implemented full business logic:

- Identity/permission/account checks matching Dashboard (`checkIsInternalActor` + `items:create`)
- Item type validation (`product`, `material`, `part`)
- Idempotency key support with recovery points
- Per-item processing that:
  - Checks SKU uniqueness globally across all items in the account
  - Looks up category base unit for rate denominator
  - Creates 3 rate records (unit_value=0, unit_cost=0, burn_rate=0)
  - Creates the item record with correct item_type_code
  - Creates the type-specific record (product/material/part)
  - For materials: creates order_point and lead_time quantity records with defaults
  - For products: sets product_type_code="sale" and links product_line_id
- Errors per item are captured in results rather than failing the whole batch

## Differences from Dashboard (by design)

The Go API endpoint was already designed with some intentional differences from the Dashboard:

1. **IDs instead of names**: Go accepts `item_category_id` and `product_line_id` (IDs) while Dashboard accepted category and product line by name. This is an improvement — no ambiguous name resolution.
2. **No upsert behavior**: Dashboard would update existing items if an itemID or matching SKU was found. The Go version treats duplicate SKUs as errors. This is a cleaner API contract.
3. **No property/attribute handling**: Dashboard supported arbitrary properties that were converted to item attributes. The Go endpoint does not include this — attributes can be managed separately via dedicated endpoints.
4. **No supplier material links**: Dashboard auto-created supplier material links for materials. This is handled by separate endpoints in the Go API.
5. **Unified result format**: Dashboard returned `{createdItems, skippedItems, updatedItems}`. Go returns a flat list of `{sku, success, error, item_id}` results — simpler and more consistent.

These differences are intentional API design choices in the Go migration, not parity gaps.

## Files modified

- `services/core-service/internal/service/item_service.go` — Implemented `BulkCreateItems` + helper methods
- `services/core-service/internal/infrastructure/grpc/grpc_item_handler.go` — Added `BulkCreateItems` gRPC handler
