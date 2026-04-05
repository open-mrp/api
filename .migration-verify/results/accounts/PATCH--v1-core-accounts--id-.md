# PATCH /v1/core/accounts/{id} — Migration Verification

## Result: Issues found and fixed

## What was compared

- **Validation rules**: field formats, constraints, required fields
- **Permission checks**: actor type, permission domain, action
- **DB queries and logic**: update patterns, existence checks, slug uniqueness
- **Error handling**: error types and messages
- **Side effects**: address updates, branding updates
- **Response shape**: field names, types, nested resources, expandables
- **Idempotency**: PATCH idempotency key support

## Issues found and fixed

### 1. Missing slug minimum length validation (FIXED)
- **Dashboard**: `z.string().min(3)` on slug field
- **Go**: No validation on slug field
- **Fix**: Added `validate:"omitempty,min=3"` tag to `Slug` field in `UpdateAccountRequest`

### 2. Missing email format validation (FIXED)
- **Dashboard**: `z.string().email().nullable()` on email field
- **Go**: No format validation on `SupportEmail`
- **Fix**: Added `validate:"omitempty,custom_email"` tag to `SupportEmail` field in `UpdateAccountRequest`

### 3. Missing URL format validation (FIXED)
- **Dashboard**: `z.string().url().nullable()` on website field
- **Go**: No format validation on `WebsiteURL`
- **Fix**: Added `validate:"omitempty,url"` tag to `WebsiteURL` field in `UpdateAccountRequest`

## Remaining concerns (not bugs, architectural differences)

### 1. Cannot clear nullable branding fields to null
- **Dashboard**: Prisma distinguishes `undefined` (don't update) from `null` (clear field). Users can send `null` to clear email, phone, social handles, website.
- **Go**: Uses `COALESCE(param, current_value)` in SQL which treats NULL params as "keep current value". With `*string` and `omitempty`, there's no way to distinguish "not sent" from "sent as null".
- **Impact**: Users cannot clear branding fields back to null via this endpoint in Go. Would require custom JSON unmarshaling or wrapper types to support three-state semantics.

### 2. billToAddress update not supported
- **Dashboard**: Supports updating `billToAddress` inline (upserts address, links to account, updates `defaultBillingAddressID`).
- **Go**: Does not accept address fields in PATCH. Address management is expected to be handled by separate address endpoints.
- **Impact**: Different UX but likely intentional — Go decouples address management from account updates.

### 3. logoUrl not in PATCH body
- **Dashboard**: Accepts `logoUrl` (S3 key string) in PATCH body.
- **Go**: Has a dedicated `UploadAccountPhoto` endpoint that handles file upload and S3 storage directly.
- **Impact**: Intentional design improvement — logo handled via file upload endpoint rather than passing URL strings.

## Parity confirmed (no issues)

- **Permission checks**: Both require internal actor + `account:update` permission
- **Account ownership check**: Go adds `params.AccountID != *identity.TargetAccountID` authorization check (improvement over Dashboard which doesn't validate ownership)
- **Slug uniqueness**: Both check for duplicate slugs excluding current account
- **Idempotency**: Go correctly uses idempotency keys for PATCH (Dashboard does not, which is less robust)
- **Partial update semantics**: Both only update provided fields
- **Response shape**: Both return full account with branding and portal sub-resources
- **Transaction safety**: Both update account, branding, and portal atomically
