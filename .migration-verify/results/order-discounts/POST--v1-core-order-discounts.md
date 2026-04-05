# POST /v1/core/order-discounts — Verification Result

**Status: PARITY CONFIRMED** — No issues found.

## What was compared

### Permission checks
- **Dashboard**: `checkIsInternalActor` + `checkHasPermission(discounts, create)`
- **Go**: `CheckIsInternalActor` + `CheckHasPermission(PermissionDomainDiscounts, ActionCreate)` + `CheckTargetAccountSet`
- **Result**: Match. Go additionally validates target account is set (defensive check).

### Validation rules
- **Dashboard**: Zod schema validates `name`, `code`, `type` (required); `percentage`, `amount` (optional numbers)
- **Go**: `validate:"required"` on `Name`, `Code`, `DiscountType`; `Percentage` and `Amount` are `*string` (optional)
- **Result**: Match. Both enforce the same required/optional semantics.

### Code uniqueness check
- **Dashboard**: `findFirst` by code + accountID, throws `HttpError.conflict("An order discount with the code ${data.code} already exists.")`
- **Go**: `ExistsByCode` via `CountOrderDiscountsByCode` query, throws `NewConflictErrorWithParam("A discount with this code already exists.", "code")`
- **Result**: Match. Both check for duplicate code within the same account before insert.

### DB insert fields
- **Dashboard**: `id`, `name`, `code`, `percentage`, `amount`, `typeCode`, `accountID`
- **Go SQL**: `id`, `name`, `code`, `percentage`, `value`, `discount_type_code`, `account_id`, `created_at`, `updated_at`
- **Result**: Match. Go explicitly sets timestamps via `NOW()`.

### Response shape
- **Dashboard**: `{ id, name, code, amount, percentage, type, orderCount }`
- **Go**: `{ id, object, name, code, amount, percentage, discount_type, order_count, created_at, updated_at }`
- **Result**: Match. Go adds `object` field (expected per API conventions) and includes timestamps. Field naming follows Go API snake_case conventions.

### Idempotency
- Go properly implements idempotency keys with `WithIdempotencyTracking` in gRPC handler and recovery point pattern in service layer.
- Dashboard did not have idempotency for this endpoint.
- **Result**: Go correctly adds idempotency (required per conventions for POST endpoints).

### Side effects
- Neither implementation has side effects (no emails, webhooks, messages).
- **Result**: Match.

### Status code
- Both return **201 Created**.

## Issues found and fixed
None — full parity confirmed.
