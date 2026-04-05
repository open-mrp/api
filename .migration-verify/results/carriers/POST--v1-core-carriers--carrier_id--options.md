# POST /v1/core/carriers/{carrier_id}/options — Migration Verification

## Result: PARITY CONFIRMED

No issues found. The Go implementation matches the Dashboard Express.js behavior.

## What Was Compared

### Validation Rules
- **Dashboard**: Requires `name` and `code`. Validates carrier exists via `CarrierRepo.checkExistence()`. Checks code uniqueness within carrier via `findFirst({ code, carrierID })`.
- **Go**: Requires `name` and `code` (via `validate:"required"` tags). Validates carrier exists via `CarrierRepo.Get()`. Checks code uniqueness via `ExistsByCodeInCarrier()` with matching SQL (`WHERE code = ? AND carrier_id = ?`).
- **Verdict**: Match.

### Permission Checks
- **Dashboard**: `checkIsInternalActor()`, `checkHasPermission(PermissionDomains.carriers, 'create')`.
- **Go**: `CheckIsInternalActor()`, `CheckHasPermission(types.PermissionDomainCarriers, types.ActionCreate)`, `CheckTargetAccountSet()`.
- **Verdict**: Match. Go additionally validates target account is set (standard pattern).

### DB Queries
- **Dashboard**: `INSERT INTO carrierOption (id, name, code, serviceLevelToken, carrierID, accountID)`.
- **Go**: `INSERT INTO carrier_option (id, name, code, service_level_token, is_portal_enabled, is_default, carrier_id, account_id, created_at, updated_at)`.
- **Verdict**: Match. Go explicitly inserts `is_portal_enabled`, `is_default`, and timestamps (Dashboard relies on DB defaults for these).

### Error Handling
- **Dashboard**: Conflict error if code already exists in carrier. Not found if carrier doesn't exist.
- **Go**: `apierror.NewConflictErrorWithParam("A carrier option with this code already exists in this carrier.", "code")`. Carrier not found returns 404 from `CarrierRepo.Get()`.
- **Verdict**: Match.

### Side Effects
- **Dashboard**: None beyond DB insert.
- **Go**: None beyond DB insert.
- **Verdict**: Match.

### Response Shape
- **Dashboard**: `{ id, name, code, serviceLevelToken, isPortalEnabled, isDefault, createdAt, updatedAt }` (camelCase).
- **Go**: `{ id, object, name, code, service_level_token, is_portal_enabled, is_default, created_at, updated_at }` (snake_case, with `object` field per API conventions).
- **Verdict**: Match. The `object` field addition and snake_case naming are expected Go API conventions.

### Idempotency
- **Go**: Uses idempotency keys with recovery points (correct for POST endpoint).
- **Verdict**: Properly implemented.

### Account Scoping
- Both filter by `account_id` from identity context.
- Both support `account_id IS NULL` for default/system options in Get queries.
- **Verdict**: Match.

## Issues Found
None.

## Remaining Concerns
None.
