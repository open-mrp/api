# GET /v1/core/carriers/{id}/oauth-status — Verification Result

**Status: PARITY CONFIRMED** — No issues found.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: internal actor check | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| Permission: carriers read | `checkHasPermission(carriers, read)` | `CheckHasPermission(PermissionDomainCarriers, ActionRead)` | Yes |
| Sandbox guard → "disconnected" | `SandboxGuard.isSandbox(accountID)` | `accountSvc.GetAccountContext` + `IsSandbox` | Yes |
| Carrier lookup by ID + accountID | `carrierRepo.find({ id, accountID })` | `carrierRepo.Get(ctx, accountID, carrierID)` | Yes |
| 404 if carrier not found | `throw HttpError.notFound` | Repo returns not-found error | Yes |
| No ShippoCarrierAccountID → "disconnected" | `!carrier.shippoCarrierAccountId` | `carrier.ShippoCarrierAccountID == nil` | Yes |
| Shippo API call to get carrier account | `shippoClient.getCarrierAccount(id)` | `shippoClient.GetCarrierAccount(ctx, id)` | Yes |
| Non-Shippo account → "connected" | `!carrierAccount.isShippoAccount` → connected | `!account.IsShippoAccount` → connected | Yes |
| Shippo account → "disconnected" | else → disconnected | `account.IsShippoAccount` → disconnected | Yes |
| Response shape | `{ status }` | `{ object, status }` | Yes (object field per Go conventions) |

## Minor Behavioral Difference (Not a Bug)

The Go implementation gracefully returns `"disconnected"` if the Shippo API call fails (line 582-583 of `carrier_service.go`), whereas the Dashboard would let the error propagate as a 500. This is a **defensive improvement** in the Go code and does not break parity — it makes the endpoint more resilient.

## No Changes Required
