# PATCH /v1/core/sales-orders/{id} — Migration Verification

## Result: Issues found and fixed

## What was compared

| Aspect | Dashboard (Express.js) | Go API | Match? |
|--------|----------------------|--------|--------|
| Permission checks (internal actor, salesOrders:update) | Yes | Yes | ✅ |
| Account scoping (owner_account_id) | Yes | Yes | ✅ |
| Idempotency keys | N/A (Express) | Yes | ✅ |
| Partial update via COALESCE | Prisma partial update | COALESCE in SQL | ✅ |
| Response shape (SalesOrderDetail) | Full order object | Full order object | ✅ |
| Include/expand support | N/A | Yes (customer, addresses, carrier, etc.) | ✅ |

## Issues found and fixed

### 1. Missing updatable fields (FIXED)
The Go endpoint was missing four fields that the Dashboard supports:
- **`number`** — order number (Dashboard also validates uniqueness)
- **`is_acknowledgment_sent`** — boolean flag for acknowledgment status
- **`promised_at`** — promised delivery date (nullable timestamp)
- **`customer_id`** — buyer account ID (customer association)

**Fix:** Added these fields across all layers:
- Proto: `UpdateSalesOrderRequest` (fields 27-30)
- API gateway endpoint request struct
- gRPC handler mapping
- Domain `UpdateSalesOrderParams`
- SQL `UpdateSalesOrder` query
- Repository mapping

### 2. Missing duplicate order number validation (FIXED)
Dashboard checks `isDuplicate` when the `number` field is updated and returns 409 Conflict if taken. The Go implementation had no such validation.

**Fix:** Added `IsDuplicateOrderNumber` check in the service layer when `params.Number` is set, using the already-existing repo method. Returns `ConflictErrorWithParam` matching Dashboard behavior.

### 3. Address fields accepted but never persisted (FIXED)
The Go endpoint accepted bill-to/ship-to address fields (name, street, city, etc.) but the service never used them to update address records. The SQL only updated `billing_address_id` and `shipping_address_id` foreign keys, which were never set.

**Fix:** Added address update logic in the service:
- If any bill-to fields are provided: updates the existing billing address record and its geolocation
- If any ship-to fields are provided: updates the existing shipping address record and its geolocation
- Uses the existing `AddressRepo.Update()` and `AddressRepo.UpdateGeolocation()` methods

## Remaining concerns

### Shipment cascading (NOT IMPLEMENTED)
Dashboard cascades ship-to address and carrier changes to linked shipments:
- `updateShipToAddress`: updates all shipments' ship-to when order ship-to changes
- `updateCarrier`: updates all shipments' carrier when order carrier changes
- Both recalculate estimated shipping charges afterward

The Go `ShipmentRepo` interface has no `UpdateShipToByOrder` or `UpdateCarrierByOrder` methods. Implementing this would require new SQL queries, repo interface methods, and repo implementations. This is noted as a gap for future work.

### Email contacts management (NOT IMPLEMENTED)
Dashboard supports updating `acknowledgementEmailContacts` and `invoiceEmailContacts` — it deletes existing contacts and recreates them. The Go API does not support these fields. This would require new SQL queries and potentially a new `OrderEmailContact` repo.

### Sales rep resolution difference (ACCEPTABLE)
Dashboard accepts `salesRep.userID` and resolves it to an `accountUserID` via DB lookup. Go accepts `sales_rep_id` directly (the account user ID). This is an acceptable API design difference — the Go API expects the caller to provide the correct ID.

## Files modified

- `proto/core_sales.proto` — added fields 27-30 to `UpdateSalesOrderRequest`
- `services/api-gateway/endpoints/sales-orders/endpoint_update_sales_order.go` — added 4 request fields
- `services/core-service/internal/infrastructure/grpc/grpc_sales_service_handler.go` — mapped new fields
- `services/core-service/internal/domain/sales_order_models.go` — added fields to `UpdateSalesOrderParams`
- `services/core-service/internal/infrastructure/queries/sales_order.sql` — added columns to UPDATE
- `services/core-service/internal/infrastructure/repository/sales_order_repo.go` — mapped new SQL params
- `services/core-service/internal/service/sales_order_service.go` — added duplicate check + address updates
- Generated files: `shared/proto/core/core_sales.pb.go`, `services/core-service/internal/infrastructure/sqlc/sales_order.sql.go`
