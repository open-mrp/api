# Verification: GET /v1/core/shipping-cases/{id}

## Result: PARITY CONFIRMED

No issues found. The Go implementation correctly mirrors the Dashboard business logic.

## What Was Compared

### Permission Checks
- **Dashboard:** `checkIsInternalActor` + `checkHasPermission(identity, PermissionDomains.shipments, 'read')`
- **Go:** `CheckIsInternalActor` + `CheckHasPermission(identity, PermissionDomainShipments, ActionRead)`
- Match: YES

### Account Scoping
- **Dashboard:** Prisma `findUnique` with composite key `{ accountID: ownerAccountID, id }`
- **Go:** SQL `WHERE sc.id = ? AND sc.account_id = ?`
- Match: YES

### DB Query / Joins
- **Dashboard:** Joins freight_amount (Quantity + Unit), freight_weight (Quantity + Unit), carrier
- **Go:** Joins quantity (freight_amount) + unit, quantity (freight_weight) + unit, carrier
- Match: YES

### Error Handling
- **Dashboard:** 404 "Shipping case not found" if not found
- **Go:** `ResourceNotFoundError` via `db.MapSQLError()` on no rows
- Match: YES

### Response Shape
- Both return: id, number, sscc, tracking_number, shipped_at, freight_amount, freight_weight, carrier, created_at, updated_at
- Go adds `object` field on all resources (required by API conventions)
- Go adds `shipment` as expandable sub-resource (enhancement)
- Go uses `LightCarrier` (id, object, name) instead of full carrier object (follows sub-resource conventions)
- Go's Quantity has `value` + `display_value` instead of Dashboard's `measure` (richer format)

### Intentionally Omitted from Go
- `shipping_label_url`: Go has a separate dedicated endpoint for fetching label URLs (evidenced by `ShippingCaseLabelURL` resource)
- `shippo_transaction_id`: Internal implementation detail of the Shippo integration, not appropriate for public API

### Side Effects
- None in either implementation (read-only endpoint)

### Idempotency
- GET endpoint, inherently idempotent — no idempotency key needed (correct in both)

## Conclusion

The Go implementation faithfully reproduces all Dashboard business logic for this endpoint. Response shape differences are intentional design improvements following Go API conventions (object fields, sub-resources, expandable relations).
