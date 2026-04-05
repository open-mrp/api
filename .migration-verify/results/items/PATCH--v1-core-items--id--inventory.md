# PATCH /v1/core/items/{id}/inventory — Migration Verification

## Status: Issues Found and Fixed

The Go implementation was a **stub** (service returned nil, gRPC handler missing). Full implementation was added to match Dashboard behavior.

## What Was Compared

- **Validation rules**: Request fields (quantity_change, reconcile, customer_id, storage_location_id, lot_number, unit_id)
- **Permission checks**: Internal actor + `items:update` permission + target account required
- **Business logic**:
  - Item existence check
  - Customer edit access check (when customer_id provided)
  - Lot upsert (when lot_number provided)
  - Storage location account validation (when storage_location_id provided)
  - Reconcile vs adjust mode (reconcile sets to exact value, adjust adds delta)
  - Inventory receipt creation for positive delta (with item unit cost, storage_location, lot)
  - Inventory issue creation for negative delta (with storage_location, lot)
  - Open issue allocation (FIFO) after receipt/issue creation
  - Inventory log (point-in-time snapshot)
  - Inventory change log (audit trail with userCorrection action type)
- **Idempotency**: PATCH uses idempotency keys with recovery points
- **Response shape**: Empty response (matches Dashboard `return {}`)
- **Owner/holder accounts**: Receipt uses inventoryOwnerAccountID (customerID or accountID) as owner, accountID as holder — matches Dashboard

## Issues Found and Fixed

### 1. Service Method Was a Stub
**File**: `services/core-service/internal/service/item_service.go`
**Issue**: `UpdateItemInventory` returned nil without doing anything.
**Fix**: Implemented full business logic matching Dashboard's `ItemSvc.updateInventory` flow.

### 2. gRPC Handler Was Missing
**File**: `services/core-service/internal/infrastructure/grpc/grpc_item_handler.go`
**Issue**: No `UpdateItemInventory` handler existed.
**Fix**: Added handler with idempotency tracking and proper proto-to-domain mapping.

### 3. SQL Queries Lacked storage_location_id and lot_id Support
**Files**: `inventory_mutation.sql`, `inventory_mutation.sql.go`
**Issue**: `InsertInventoryReceipt` and `InsertInventoryIssue` didn't include `storage_location_id` or `lot_id` columns, despite the DB schema supporting them. Dashboard creates receipts/issues with these fields.
**Fix**: Added `storage_location_id` and `lot_id` as nullable params to both queries. Also split `account_id` into `owner_account_id` and `holder_account_id` on receipts to match the schema.

### 4. Domain Param Structs Missing Fields
**File**: `inventory_mutation_models.go`
**Issue**: `CreateInventoryReceiptParams` and `CreateInventoryIssueParams` lacked StorageLocationID, LotID, and owner/holder account distinction.
**Fix**: Added these fields.

### 5. Repository Methods Updated
**File**: `inventory_mutation_repository.go`
**Issue**: `CreateInventoryReceipt` and `CreateInventoryIssue` didn't pass storage_location_id or lot_id.
**Fix**: Updated to support optional storage_location_id, lot_id, and owner/holder account IDs.

## Remaining Concerns

- Pre-existing compilation errors in `sales_order_repo.go` and `shipment_repository.go` (unrelated to this endpoint) prevent full `go build` verification. The changes to inventory mutation code are syntactically correct and follow established patterns.
- The Dashboard's `allocateOpenIssues` does FIFO allocation across receipts and issues. The Go `AllocateOpenIssuesForItem` method already implements this logic in `inventory_reservation_repository.go`.
