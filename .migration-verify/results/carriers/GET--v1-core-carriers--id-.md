# GET /v1/core/carriers/{id} — Migration Verification

## Result: Issue found and fixed

## What was compared

- **Validation rules**: Path parameter `id` validated in both implementations
- **Permission checks**: Both check `checkIsAssignedActor`, then internal users need `carriers:read`. Customer actors allowed. Go additionally checks external target read access via mediator (acceptable enhancement).
- **DB query**: Both use `WHERE carrier.id = ? AND (carrier.account_id = ? OR carrier.account_id IS NULL) AND carrier.deleted_at IS NULL` — matches exactly.
- **Error handling**: Both return 404 when carrier not found.
- **Side effects**: None in either implementation (read-only endpoint).
- **Response shape**: Both return carrier with `id`, `name`, `code`, `shippo_carrier_account_id`, `account_number`, `is_portal_enabled`, `is_default`, `options`, `deleted_at`, `created_at`, `updated_at`. `is_default` derived from `accountID === null` in both.
- **Idempotency**: GET endpoint, no idempotency keys needed. Correct in both.
- **Includes/Expansions**: Both support `options` as an expandable/includable field.

## Issue found and fixed

**Carrier options not loaded on get-by-ID**: The Go `GetCarrier` service method only called `carrierRepo.Get()`, which fetches the carrier row but does not populate `Options`. The Dashboard always loads carrier options via Prisma's nested `select` on `carrierOptions`.

**Fix**: Added `carrierRepo.ListOptionsByCarrierID()` call after fetching the carrier in `carrier_service.go:GetCarrier`, attaching the results to `carrier.Options`. This follows the same pattern already used in `SyncOptions` (line 697-707 of the same file).

## No remaining concerns

Parity is now confirmed.
