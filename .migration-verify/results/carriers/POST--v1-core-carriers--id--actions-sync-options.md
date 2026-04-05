# Verification: POST /v1/core/carriers/{id}/actions/sync-options

## Status: Issues found and fixed

## What was compared

- **Permission checks**: Both require internal actor + `carriers:update` permission. Go also checks `TargetAccountSet`. ✅ Parity
- **Sandbox guard**: Dashboard throws `HttpError.badRequest`; Go returns `ValidationError`. ✅ Equivalent
- **Carrier lookup**: Both fetch carrier by ID + accountID, return not-found if missing. ✅ Parity
- **Shippo account check**: Both require `shippoCarrierAccountId` to be set. ✅ Parity
- **Shippo client construction**: Dashboard uses `AccountIntegrationRepo.buildShippoClient`; Go uses `getShippoClient` (same flow: fetch encrypted creds, decrypt, build client). ✅ Parity
- **Service level fetch**: Both call `getCarrierServiceLevels` with the Shippo carrier account ID. ✅ Parity
- **Stale option removal**: Both delete options whose `serviceLevelToken` is no longer in Shippo's response. ✅ Parity
- **New option creation**: Both add options for service levels not yet present. ✅ Parity
- **Response**: Both re-fetch and return the updated carrier with options. ✅ Parity
- **HTTP status**: Both return 200 OK. ✅ Parity
- **Idempotency**: gRPC handler uses `WithIdempotencyTracking`. The operation is inherently idempotent (sync to current state). ✅ Acceptable
- **Transaction**: Go wraps deletes and creates in a single transaction (improvement over Dashboard which does them sequentially). ✅ Acceptable improvement

## Issues found and fixed

1. **`IsDefault` field on newly synced options**: Go `SyncOptions` was creating new carrier options with `IsDefault: true`, but the Dashboard's `CarrierOptionUtils.create` defaults `isDefault` to `false`. The Go `CreateCarrier` method also correctly uses `IsDefault: false` for Shippo-synced options. Fixed `SyncOptions` to use `IsDefault: false`.

## No remaining concerns
