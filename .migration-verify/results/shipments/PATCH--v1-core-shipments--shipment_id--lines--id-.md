# PATCH /v1/core/shipments/{shipment_id}/lines/{id}

## Result: Issues found and fixed

## Summary

This endpoint does **not** exist in the Dashboard Express.js API. The Dashboard has no exposed route, controller, service method, or DTO for updating shipment lines. The repository layer (`shipment-line.repo.ts`) has a low-level `update()` method that only updates the `orderLine` relationship field, but it is never surfaced as an API endpoint.

The Go implementation is a **new endpoint** that allows updating the quantity (value and unit) on a shipment line. Since there is no Dashboard counterpart, parity verification focused on ensuring the Go implementation follows project conventions and has no bugs.

## What was compared

- **Dashboard**: Service (`shipment-line.svc.ts` — empty), Repository (`shipment-line.repo.ts`), Controller (`shipment.ctrl.ts` — no shipment line update), Routes (`index.ts` — no route registered), DTOs (`shipments.ts` — no schema defined)
- **Go**: API gateway endpoint, presenter, gRPC handler, core-service service layer, repository, SQL queries, domain models, proto definition

## Issues found and fixed

1. **Missing account ownership validation (security fix)**: The `UpdateShipmentLine` service method did not verify that the shipment belongs to the caller's account before proceeding. Other methods like `CreateShipmentLine` and `ListShipmentLines` correctly call `shipmentRepo.IsInAccount()`, but `UpdateShipmentLine` skipped this check entirely — it only validated `IsInShipment`. This could allow a user to update a shipment line on a shipment belonging to a different account if they knew the IDs. Fixed by adding the `IsInAccount` check inside the transaction, matching the pattern used by `CreateShipmentLine`.

## No remaining concerns

- Idempotency: Correctly uses idempotency keys with recovery points
- Authorization: Internal actor + `PermissionDomainShipments` / `ActionUpdate`
- SQL: Uses COALESCE for partial updates on the quantity table
- Response shape: Returns full `ShipmentLine` resource with nested `LightQuantity` sub-resource
- gRPC handler: Properly maps optional fields and uses `WithIdempotencyTracking`
