# POST /v1/core/receiving-orders/{id}/actions/stock — Verification Result

**Status: Issues found and fixed**

## What was compared
- Permission checks (actor type, permission domain, action)
- Validation rules and unstocked line filtering
- DB queries and transaction logic
- Side effects (delivery creation, inventory receipts, change logs, PO fulfillment, open issue allocation)
- Idempotency key handling
- Response shape

## Issues found and fixed

### 1. Stocking data completely ignored (CRITICAL)
The Go service received `StockingData` (line items with allocations, lot numbers, rejected quantities) via `params.Data` but never used it. The entire purpose of the stocking data — creating deliveries, inventory receipts, and allocating inventory — was missing.

**Fixed by:** Adding full business logic in the service layer with helper methods `createDeliveryRecords`, `createInventoryChangeLogs`, and `allocateOpenIssues`.

### 2. Missing: Create delivery records with delivery lines
Dashboard creates a `delivery` record with `delivery_line` entries for each accepted and rejected allocation, including lot tracking and storage location assignment.

**Fixed by:** Added SQL queries (`InsertDelivery`, `InsertDeliveryLine`, `CountDeliveriesByPurchaseOrder`) and repo methods. The service now creates delivery records matching Dashboard behavior: accepted allocations get `accepted_at` timestamps, rejected quantities get `rejected_at` timestamps, delivery status is "accepted" if any allocations exist.

### 3. Missing: Create inventory receipts
Dashboard creates `inventory_receipt` records for each accepted allocation with `storage_location_id`, `lot_id`, and `order_id` (linking back to the purchase order).

**Fixed by:** Added `InsertInventoryReceiptForDelivery` SQL query that includes all three optional columns. The existing `InsertInventoryReceipt` only supported `batch_id`.

### 4. Missing: Upsert lot records
Dashboard upserts lot records by `(account_id, item_id, lot_number)` composite key when a lot number is provided.

**Fixed by:** Added `UpsertLot` (INSERT IGNORE) and `GetLotByKey` SQL queries, plus repo method that handles the upsert-then-fetch pattern.

### 5. Missing: Create inventory change logs
Dashboard creates `inventory_change_log` entries with `action_type = system_action` for each accepted allocation.

**Fixed by:** Service now calls `InventoryMutationRepo.CreateInventoryChangeLog` for each accepted allocation with the responsible user from the identity context.

### 6. Missing: Mark purchase order as fulfilled
Dashboard marks the purchase order as fulfilled (`status = 'fulfilled'`, `completed_at = NOW()`) when the receiving order becomes complete (all lines stocked).

**Fixed by:** Added `MarkPurchaseOrderFulfilled` SQL query and repo method. Service checks the `isComplete` return value from `MarkCompleteIfAllStocked` and calls this when true.

### 7. Missing: Allocate open inventory issues (FIFO)
Dashboard calls `allocateOpenIssues` for each unique item that received inventory, performing FIFO allocation of open issues against available receipts.

**Fixed by:** Added `AllocateOpenIssuesForItem` public method to `InventoryReservationRepo` that reuses the existing private `allocateOpenIssue` FIFO logic. Added `FindOpenIssuesForItem` and `GetAllocationSumForIssue` SQL queries.

### 8. Missing: `completed_at IS NULL` filter on unstocked lines query
Dashboard filters unstocked lines to only find them on incomplete receiving orders (`completedAt: null`). The Go query was missing this filter, allowing stocking on already-completed receiving orders.

**Fixed by:** Added `AND ro.completed_at IS NULL` to the `FindUnstockedLineIDs` SQL query.

### 9. Missing: Guard for empty unstocked lines
Dashboard only proceeds with the transaction if `unstockedLines.length > 0`. Go processed unconditionally.

**Fixed by:** Wrapped the transaction body with `if len(unstockedLines) > 0`.

## Infrastructure added

### New ID prefixes
- `VocDelivery = "dv"`, `VocLot = "lt"`
- `DeliveryIDPrefix`, `DeliveryLineIDPrefix`, `LotIDPrefix`

### New SQL queries (in `receiving_order.sql`)
- `GetReceivingOrderLineUnitPrice` — fetches unit price info for delivery/receipt creation
- `GetPurchaseOrderIDForReceivingOrder` — gets PO ID from receiving order
- `CountDeliveriesByPurchaseOrder` — for delivery number generation
- `UpsertLot` / `GetLotByKey` — lot upsert
- `InsertDelivery` / `InsertDeliveryLine` — delivery creation
- `InsertInventoryReceiptForDelivery` — receipt with storage/lot/order support
- `FindOpenIssuesForItem` / `GetAllocationSumForIssue` — open issue allocation
- `MarkPurchaseOrderFulfilled` — PO status change

### New repo interface methods
- `DeliveryRepo.CountByPurchaseOrder`, `CreateDelivery`, `CreateDeliveryLine`
- `ReceivingOrderRepo.GetLineUnitPrices`, `GetPurchaseOrderID`, `UpsertLot`, `InsertInventoryReceiptForDelivery`, `MarkPurchaseOrderFulfilled`, `FindOpenIssuesForItem`, `GetAllocationSumForIssue`
- `InventoryMutationRepo.CreateQuantityForInventory`, `CreateRateForInventory`
- `InventoryReservationRepo.AllocateOpenIssuesForItem`

## Remaining concerns
- Delivery number generation: Dashboard uses `deliveryCount >= 1` to append `-N` suffix. Go implementation matches this logic.
- The `action_type` code `system_action` is used for inventory change logs — this should match the seed data value in the database.
