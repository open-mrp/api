# POST /v1/core/carriers — Migration Verification

## Status: Issues found and fixed

## What was compared

- **Validation rules**: Request fields, required/optional, defaults
- **Permission checks**: Actor type, permission domain, action
- **Business logic**: Shippo integration flow (FedEx OAuth vs UPS/USPS account number), sandbox handling, service level sync
- **DB operations**: Carrier creation, carrier option nested creation, name uniqueness check
- **Error handling**: Error propagation from Shippo API calls
- **Response shape**: Field names, types, nested carrier options
- **Idempotency**: POST uses idempotency keys with recovery points

## Issues found and fixed

### 1. Service level fetch error handling (carrier_service.go ~line 284)
- **Dashboard**: Errors from `getCarrierServiceLevels()` propagate up — carrier creation fails if service levels can't be fetched.
- **Go (before fix)**: Errors were silenced (`levels = nil`) — carrier would be created without options even if the Shippo API failed.
- **Fix**: Changed to propagate the error with `return nil, tracing.Trace(span, apiErr)`.

### 2. CarrierOption `isDefault` for Shippo-created options (carrier_service.go ~line 303)
- **Dashboard**: `CarrierOptionUtils.create()` defaults `isDefault` to `false` for auto-created Shippo options.
- **Go (before fix)**: Set `IsDefault: true` for Shippo-created options.
- **Fix**: Changed to `IsDefault: false` to match Dashboard behavior.

### 3. Unused ID generation in service level loop (carrier_service.go ~line 290)
- **Go (before fix)**: Generated an `optionID` via `id.GenID()` inside the loop building `shippoOptions`, assigned it to `_` with comment "used below", but never used it. A separate `id.GenID()` call at line 319 inside the transaction generates the actual option ID.
- **Fix**: Removed the unused `id.GenID()` call from the loop.

## Noted differences (not fixed — acceptable)

### `isPortalEnabled` during create
- **Dashboard**: The Prisma `create` call in `carrier.repo.ts` does NOT include `isPortalEnabled` in the insert data, so it always defaults to the DB default (`false`) regardless of user input.
- **Go**: Passes `is_portal_enabled` through from the request to the DB.
- **Assessment**: The Go behavior is more correct — the Dashboard appears to have a bug where `isPortalEnabled` is accepted in the request schema but silently ignored during creation. No change needed.

### Name uniqueness check
- **Dashboard**: No explicit uniqueness check before insert (relies on DB constraints or allows duplicates).
- **Go**: Explicitly checks `ExistsByName()` before insert, returning a conflict error.
- **Assessment**: Additional safety in Go. Not a regression.

## Parity confirmed for

- Permission checks: Both require internal actor + `carriers:create` permission
- Shippo integration flow: FedEx OAuth vs UPS/USPS account number logic matches
- Sandbox handling: Both skip Shippo API calls for sandbox accounts, set `shippoCarrierAccountId` to null and options to empty
- Account number validation: Both require account number for non-FedEx Shippo carriers
- Response shape: Go adds `object` field per API conventions; all other fields present and correctly mapped
- Idempotency: Go properly uses idempotency keys with recovery points for the POST endpoint
- Atomic creation: Both create carrier and options atomically (Dashboard via Prisma nested writes, Go via DB transaction)
