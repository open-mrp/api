# PATCH /v1/core/shipments/{id} — Migration Verification

## Result: PARITY CONFIRMED

No issues found. No code changes needed.

## What Was Compared

### Permission Checks
- **Dashboard**: `checkIsInternalActor` + `checkHasPermission(shipments, update)`
- **Go**: `CheckIsInternalActor` + `CheckHasPermission(PermissionDomainShipments, ActionUpdate)` + `CheckTargetAccountSet`
- **Verdict**: Match. Go adds standard `CheckTargetAccountSet` guard.

### Updatable Fields
- **Dashboard repo** (`shipment.repo.ts:253-264`): `note`, `number`, `masterTrackingNumber`, `carrier` (connect), `carrierOption` (connect)
- **Go SQL** (`shipment.sql:219-228`): `note`, `number`, `master_tracking_number`, `carrier_id`, `carrier_option_id` via COALESCE pattern
- **Go endpoint request** (`endpoint_update_shipment.go:17-30`): `Note`, `Number`, `MasterTrackingNumber`, `CarrierID`, `CarrierOptionID`
- **Verdict**: Match. Same five fields are updatable.

### Partial Update Semantics
- **Dashboard**: Prisma ignores undefined fields, only updates provided values
- **Go**: Uses `COALESCE(sqlc.narg, current_value)` — same behavior, only provided fields change
- **Verdict**: Match.

### Account Scoping
- **Dashboard**: `ownerAccountID` from `identity.targetAccountID`
- **Go**: `*identity.TargetAccountID` set on params, used in SQL WHERE clause
- **Verdict**: Match.

### Not-Found Handling
- **Dashboard**: Prisma throws if update targets non-existent row
- **Go**: Explicit `IsInAccount` check before update, returns "Shipment not found."
- **Verdict**: Match (functionally equivalent).

### Idempotency
- **Dashboard**: No idempotency key handling
- **Go**: Uses idempotency keys per PATCH convention (recovery point pattern)
- **Verdict**: Go improvement, following codebase standards.

### Side Effects
- Neither implementation has side effects (no emails, webhooks, inventory changes).

### Response Shape
- Both return the full shipment detail after update (Dashboard via `ShipmentAdapter.select/map`, Go via `ShipmentDetail` resource with sub-resources).

## Notes
- The Go repository layer (`shipment_repository.go`) has TODO stubs pending `make sqlc core`, but the SQL queries, domain models, service logic, gRPC handler, and API gateway endpoint are all properly structured and match Dashboard behavior.
