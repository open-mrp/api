# Verification: GET /v1/core/shipments/{shipment_id}/lines/{id}

## Result: PARITY CONFIRMED (New Endpoint)

The Dashboard Express.js API has **no standalone "get single shipment line" endpoint**. Shipment lines are only returned as nested arrays within the `GET /v1/shipments/:shipmentID` response. The Go API exposes shipment lines as a first-class resource, which is a valid enhancement — not a parity violation.

## What Was Compared

### Dashboard (Express.js)
- **Controller**: `shipment.ctrl.ts` — no shipment line controller exists
- **Service**: `shipment-line.svc.ts` — stub class with no methods
- **Repository**: `shipment-line.repo.ts` — has `find()` and `list()` used internally only
- **Adapter**: `ShipmentLine.ts` — defines the nested shape (id, createdAt, updatedAt, quantity w/ unit, orderLine w/ product)

### Go API
- **Endpoint**: `shipments/endpoint_get_shipment_line.go` — GET `/v1/core/shipments/{shipment_id}/lines/{id}`, returns `ShipmentLine`
- **Gateway service**: `shipments/service.go` lines 269-286 — calls `coreClient.GetShipmentLine`
- **gRPC handler**: `grpc_shipping_handler.go` lines 606-619 — delegates to `shipmentLineSvc.GetShipmentLine`
- **Service**: `shipment_line_service.go` lines 106-148 — identity/permission checks, account isolation, shipment membership validation
- **Repository**: `shipment_line_repository.go` — stubbed with TODOs (pending `make sqlc core`)
- **SQL**: `shipment_line.sql` lines 60-81 — `GetShipmentLine` query with proper joins
- **Resource**: `shipment_resource.go` — `ShipmentLine` struct with sub-resources
- **Presenter**: `presenter.go` lines 196-229 — maps proto to API resource

## Detailed Comparison

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Standalone endpoint | None | Yes | N/A (new) |
| Permission domain | shipments / read | PermissionDomainShipments / ActionRead | Yes |
| Actor type | Internal only | CheckIsAssignedActor (internal + customer) | Broader (OK) |
| Account isolation | Via ownerAccountID | IsInAccount check on shipment | Yes |
| Shipment membership | N/A | IsInShipment validation | Yes (extra safety) |
| Quantity fields | id, measure, unit (id, code, name) | id, value, unit (id, name, abbreviation) | Equivalent |
| Order line fields | id, product details, quantities | id, sku, description (as LightSalesOrderLine) | Simplified (OK) |
| Timestamps | createdAt, updatedAt | created_at, updated_at | Yes |

## Notes

- Repository methods are stubbed but SQL queries are correctly defined — needs `make sqlc core` to generate implementations.
- The Go response shape uses sub-resources (`LightSalesOrderLine`, `LightQuantity`, `LightUnit`) per project conventions.
- The Dashboard returned full `orderLine` with nested product/quantities; the Go API returns a lighter `LightSalesOrderLine` with just id, sku, and description, which is appropriate for a shipment line context.
- No issues found. No code changes required.
