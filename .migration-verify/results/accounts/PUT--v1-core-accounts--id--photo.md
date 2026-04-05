# PUT /v1/core/accounts/{id}/photo — Verification Result

## Status: Issues found and fixed

## What was compared

- **Validation rules**: Account ID from path, content-type header defaulting
- **Permission checks**: Internal actor check, account update permission
- **Authorization**: Target account must match path account ID
- **S3 upload logic**: Key generation, bucket selection, content type, server-side encryption
- **DB update**: Branding logo URL persistence
- **Error handling**: Error types and messages
- **Response shape**: `{ success: boolean }`
- **Idempotency**: PUT endpoint — idempotent by design, no idempotency key needed

## Issues found and fixed

### 1. Missing S3 server-side encryption (fixed)

**Dashboard**: Uploads with `serverSideEncryption: 'AES256'` explicitly set on the PutObject call.
**Go (before fix)**: The shared `s3.Client.Upload` method did not set `ServerSideEncryption`.
**Fix**: Added `ServerSideEncryption: s3types.ServerSideEncryptionAes256` to the `PutObjectInput` in `shared/cloud/s3/s3.go`.

## Minor differences (no fix needed)

### Account existence check

**Dashboard**: Repository calls `checkExistence({ id: accountID })` before uploading to S3.
**Go**: Does not explicitly check account existence before uploading.
**Why no fix**: The Go service already validates that `accountID == *identity.TargetAccountID`, and the identity is derived from an authenticated session — the account must exist for the user to have a valid identity. The `UpdateBrandingLogoURL` SQL UPDATE targets `owner_account_id`, so a non-existent account would simply be a no-op. This path is effectively unreachable.

### Default content type source

**Dashboard**: Controller defaults to `'image/png'`; repo interface has a separate default of `'image/jpeg'` (never reached because the controller always passes a value).
**Go**: Service defaults to `"image/png"` when empty — matches the Dashboard controller behavior.

## Parity confirmed for

- Permission model: internal actor + account update permission ✓
- Authorization: target account must match path account ID ✓
- S3 key format: `{accountID}/logo.png` ✓
- Content-type default: `image/png` ✓
- Response shape: `{ success: true }` ✓
- HTTP method: PUT (idempotent by design) ✓
- No side effects (no emails, webhooks, or messages) ✓
