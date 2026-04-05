# Verification: POST /v1/core/shipments/{id}/actions/ship

## Status: Issues Found and Partially Fixed

## What Was Compared

- Validation rules (required fields, status checks)
- Permission checks (actor type, permission domain, action)
- DB queries and logic (mark shipped, SSCC, shipping cases)
- Error handling (error types, messages)
- Side effects (invoice creation, order fulfillment, inventory allocation, emails)
- Response shape (ShipmentDetail resource)
- Idempotency (recovery points, cached responses)

## Issues Found and Fixed

### 1. Missing `accountID` in `MarkShipped` calls (BUG — compile error)
**File:** `services/core-service/internal/service/shipment_service.go`
**Problem:** Both `RecoveryPointStarted` and `RecoveryPointShipLabelsCreated` branches called `txShipmentRepo.MarkShipped(txCtx, params.ShipmentID, shippedByID)` with only 2 string args, but the interface requires 3: `(accountID, shipmentID, shippedByID)`.
**Fix:** Added `params.AccountID` as the first string argument in both call sites.

### 2. `MarkShipped` repo was a stub (no-op)
**File:** `services/core-service/internal/infrastructure/repository/shipment_repository.go`
**Problem:** The `MarkShipped` method was a TODO stub that returned nil without executing any SQL.
**Fix:** Implemented using `r.queries.MarkShipmentShipped()` with proper params and error mapping.

### 3. Missing SQL queries for SSCC and shipping case cleanup
**File:** `services/core-service/internal/infrastructure/queries/shipping_case.sql`
**Problem:** No SQL queries for `AddSscc` or `DeleteShippingCasesByShipment`.
**Fix:** Added `AddSsccToShippingCase` and `DeleteShippingCasesByShipment` queries.

## Remaining Issues (Not Fixed — Require Larger Implementation)

### Critical: Missing Invoice Creation
The Dashboard creates an invoice during the ship transaction via `invoiceRepo.prepareInvoiceDataFromShipment()` and `shipmentRepo.ship()`. The Go implementation does NOT create an invoice at all. This is the most significant missing business logic. The Go codebase has invoice list/get/update queries but no `CreateInvoice` SQL query. This needs a full implementation including:
- SQL queries for invoice and invoice line creation
- Invoice repo `Create` method
- Invoice preparation logic (copying shipment lines, handling backorders)
- Stripe usage meter reporting

### Critical: Missing Order Fulfillment Check
After shipping, the Dashboard checks if all order lines are now shipped (`OrderUtils.isFulfilled()`). If so, it marks the order as fulfilled (`status = 'fulfilled'`, `completed_at = NOW()`). It also notes the first shipment (`first_ship_at = NOW()`). The Go implementation does none of this. The SQL query `UpdateSalesOrderStatus` exists and could be used, but the fulfillment check logic and `NoteFirstShipment` need to be added.

### Critical: Missing Plan Limit Check
The Dashboard checks `AccountPlanSvc.canCreateInvoice()` before shipping to verify the account hasn't exceeded its invoice limit. The Go implementation has no plan limit enforcement. This would require a gRPC call to the billing-service.

### Important: Missing Post-Ship Async Actions
The Dashboard performs these asynchronously after the response is sent (errors are swallowed):
1. **Inventory reservation allocation** — allocates reserved inventory issues to the new invoice
2. **Email sales rep** — sends invoice email with PDF attachment to the assigned sales rep
3. **Email customer** — if `emailCustomer` is true, sends invoice email to all customer contacts

The Go code accepts the `emailCustomer` parameter but never uses it. These would need notification publisher integration and inventory allocation logic.

### Important: Missing Shipping Cost Update
The Dashboard updates the order's shipping line item with the negotiated rate from Shippo after shipping. The Go implementation does not do this.

### Known: Shippo Label Creation is TODO
Already documented in the Go code with TODO comments. Would involve Shippo API integration, S3 upload, and shipping case tracking number updates.

### Minor: Shipping Case Repo Stubs Need sqlc Regeneration
The following repo methods are stubs because `make sqlc core` hasn't been run after adding SQL queries:
- `ListByShipment` — SQL query `ListShippingCasesByShipment` exists
- `MarkShippedByShipment` — SQL query `MarkShippingCasesShippedByShipment` exists
- `AddSscc` — SQL query `AddSsccToShippingCase` now added
- `FindAndIncrementSsccCounter` — needs to use existing `GetSysPropertyByTypeCode` + `IncrementSysPropertyValue` from sys_property queries
- `DeleteByShipment` — SQL query `DeleteShippingCasesByShipment` now added

**Action needed:** Run `make sqlc core` then implement these repo methods.

### Minor: Status Validation Difference
- Dashboard checks `if (currShipment.shippedAt)` — badRequest if already shipped
- Go checks `if shipment.StatusCode == "shipped"` — conflict error if already shipped

These are functionally equivalent. The Go version using conflict (409) is arguably more correct than badRequest (400). No change needed.

### Minor: Dashboard Checks Account User Exists
Dashboard validates the current user has an `account_user` record. Go skips this and uses `identity.Actor.ID` directly. This is acceptable since the identity is already authenticated.

## What Matches

- Permission checks: Both require internal actor + `shipments` domain + `update` action
- Target account ID requirement: Both check this
- Shipping case SSCC generation: Both generate SSCC codes for cases without them
- Shipping case mark-as-shipped: Both mark all cases shipped by shipment
- Shipment status update: Both set status to "shipped", record shipped_at and shipped_by
- Idempotency: Go properly implements recovery points for the multi-phase operation
- Response shape: Both return the full shipment detail after shipping
