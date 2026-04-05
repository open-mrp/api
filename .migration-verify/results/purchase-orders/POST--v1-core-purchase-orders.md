# POST /v1/core/purchase-orders — Verification Result

**Status: Issues found and fixed**

## What was compared

- **Permission checks**: Both check internal actor + `purchaseOrders` / `create` permission + target account set. **Parity confirmed.**
- **Idempotency**: Go correctly uses idempotency keys with recovery points. Dashboard does not use idempotency keys. **Go is an improvement.**
- **Address creation**: Dashboard upserts addresses and links to account. Go creates new addresses with geolocation and account_address records. **Functionally equivalent.**
- **Order number generation**: Both get-and-increment from `sys_property`. Go also checks for duplicate order numbers. **Parity confirmed (Go is stricter).**
- **Order creation**: Both create with status "estimate", correct buyer/seller assignment, all fields (note, carrier, priority, terms, promised_at). **Parity confirmed.**
- **Line creation**: Both create lines with quantity, unit price, and optional unit cost. Go additionally supports unit cost which Dashboard does not explicitly surface. **Parity confirmed.**
- **Email contacts**: Both create `orderEmailContact` records with `purchaseOrderSubmission` notification type. **Parity confirmed.**
- **Supplier material link**: Both call `ensureSupplierMaterialLink` for each line with an item_id. **Issues found and fixed (see below).**
- **Response shape**: Go returns 201 Created (vs Dashboard 200 OK) with `PurchaseOrderDetail` resource including expandable sub-resources. **Acceptable divergence (201 is more correct for POST).**
- **Error handling**: Both return appropriate errors for missing permissions, missing account, conflicts. **Parity confirmed.**
- **Side effects**: Dashboard creates a change log entry via `LoggingRepo.createChangeLog`. Go does not have a general change log repository. This is a known gap across the entire migration, not specific to purchase orders. **Noted as remaining concern.**

## Issues found and fixed

### 1. SupplierId proto mapping was wrong (grpc_purchase_service_handler.go)

**Bug**: `SupplierId` was mapped to `o.BuyerAccountID` (the owner account) instead of `o.SellerAccountID` (the actual supplier). This would cause the supplier to be reported as the owner account in all API responses.

**Fix**: Changed `SupplierId: o.BuyerAccountID` to `SupplierId: o.SellerAccountID` in `purchaseOrderToProto()`.

### 2. Missing SupplierPartNumber in ensureSupplierMaterialLink (purchase_order_service.go)

**Bug**: The Dashboard sets `supplierPartNumber: itemSku` when creating a supplier-material link, but the Go code passed an empty string (zero value). This means newly created supplier-material links would be missing their part number.

**Fix**: Added `itemSKU string` parameter to `ensureSupplierMaterialLink()` and passed `SupplierPartNumber: itemSKU` in the `CreateSupplierMaterialParams`. Updated all 3 call sites (create PO, create PO line, update PO line).

## Remaining concerns

1. **Change log creation**: Dashboard creates a `change_log` record on PO creation (action: createRecord, model: purchaseOrder). The Go codebase does not have a general ChangeLogRepo. This is a systemic gap across the migration — no migrated endpoint creates change log entries. Should be addressed as a cross-cutting concern.
2. **Address handling difference**: Dashboard upserts (create-or-update) addresses while Go always creates new ones. This is functionally equivalent for creation but means Go will create more address records over time if the same address data is resubmitted. This is an acceptable trade-off given the idempotency key protection.
