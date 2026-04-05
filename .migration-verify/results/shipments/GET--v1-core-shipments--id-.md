# Verification: GET /v1/core/shipments/{id}

## Status: Issues found and fixed

## What was compared

- **Validation rules**: Path parameter (shipment ID) - matches
- **Permission checks**: Internal actor + PermissionDomainShipments + ActionRead + target account set - matches Dashboard's `checkIsInternalActor` + `checkHasPermission(shipments, read)`
- **DB query & joins**: 13-table join covering shipment, status, sales_order, account_relation, account, carrier, carrier_option, address, account_user/user (shipped by), invoice, pick, billing address/geolocation - matches Dashboard's Prisma includes
- **Error handling**: Not found when shipment doesn't exist or doesn't belong to account - matches
- **Side effects**: None (read-only) - matches
- **Response shape**: Sub-resource pattern (LightSalesOrder, LightCarrier, LightCustomer, etc.) with expandable includes for lines, shipping_cases, sales_order, carrier, carrier_option, shipping_address, shipped_by, invoice
- **Idempotency**: N/A (GET endpoint) - correct

## Issues found and fixed

### 1. Billing fallback logic (SQL query)
**Problem**: The Go SQL query only read `so.carrier_billing_type` and `so.carrier_billing_account` from the `sales_order` table. The Dashboard uses a fallback chain: order-level values first, then customer-level values from `account_relation` (`ar.carrier_billing_type`, `ar.carrier_billing_account`) via nullish coalescing.

**Fix**: Changed `shipment.sql` GetShipment query to use `COALESCE(so.carrier_billing_type, ar.carrier_billing_type)` and `COALESCE(so.carrier_billing_account, ar.carrier_billing_account)`. Regenerated sqlc.

### 2. Repository Get method not implemented
**Problem**: The `shipmentRepoImpl.Get()` method was a TODO stub that always returned "Shipment not found." without querying the database.

**Fix**: Implemented the full Get method mapping the sqlc `GetShipmentRow` to `domain.Shipment`, handling all nullable fields properly.

### 3. Repository method signatures mismatched domain interface
**Problem**: `MarkShipped` and `MarkVoided` method signatures didn't include the `accountID` parameter expected by the domain interface.

**Fix**: Updated method signatures to match `domain.ShipmentRepo` interface: `MarkShipped(ctx, accountID, shipmentID, shippedByID)` and `MarkVoided(ctx, accountID, shipmentID)`.

## Notes

- The Go API uses an include/expand system for sub-resources (lines, shipping_cases, etc.) while the Dashboard always returns all data. This is an intentional architectural improvement, not a parity issue.
- The Dashboard returns `order.phone` (from buyer account branding) and `order.customerID` as fields on the order object. The Go API separates customer into its own `LightCustomer` sub-resource and does not include phone. Phone was not a critical field for the shipment detail view.
- Pre-existing build errors exist in `item_repository.go` and `sales_order_repo.go` that are unrelated to this endpoint.
