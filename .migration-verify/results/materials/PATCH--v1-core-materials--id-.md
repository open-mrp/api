# PATCH /v1/core/materials/{id} — Migration Verification

## Result: Issue found and fixed

## What was compared

- **Validation rules**: Request fields (sku, description, notes, order_point, lead_time) are all optional `*string`/`*QuantityInputRequest`, matching the Dashboard's `Partial<Material>` semantics.
- **Permission checks**: Both check `isInternalActor` + `materials:update`. Go additionally checks `targetAccountSet`. Match.
- **Material existence check**: Dashboard finds by `itemID` only; Go scopes by `accountID + itemID` via `GetByItemID`. Functionally equivalent (IDs are globally unique), Go is slightly more secure.
- **SKU uniqueness**: Both check only when SKU is provided, excluding the current item. Match.
- **DB update logic**: Both update `sku` (via COALESCE/conditional), `description`, `notes` on the `item` table, then touch `material.updated_at`. Match.
- **Error handling**: Dashboard throws `HttpError.conflict` with SKU value in message; Go returns `ConflictErrorWithParam` with generic message. Acceptable difference.
- **Side effects**: Neither triggers side effects (no emails, webhooks, messages). Match.
- **Response shape**: Both return the full material with nested item, rates, quantities. Match.
- **Idempotency**: Go correctly implements idempotency keys for PATCH. Dashboard did not. Expected improvement.
- **Transaction handling**: Go wraps all mutations in a transaction. Dashboard does not use explicit transactions. Go is safer.

## Issue found and fixed

**`UpdateDescription` and `UpdateNotes` flags always set to `true`**

In `services/api-gateway/endpoints/materials/service.go`, the API gateway was hardcoding `UpdateDescription: true` and `UpdateNotes: true` when building the gRPC request. This meant that if a user sent a partial update like `{"sku": "NEW-SKU"}` (without description or notes fields), the Go code would set both description and notes to NULL via the SQL:

```sql
description = CASE WHEN true THEN NULL ELSE description END
```

The Dashboard (Prisma) correctly skips fields that are `undefined` in the update payload, preserving existing values.

**Fix**: Changed to `UpdateDescription: req.Description != nil` and `UpdateNotes: req.Notes != nil`, so these flags are only set when the caller explicitly provides a value.

## Additional notes

- The Go endpoint accepts `order_point` and `lead_time` for update, which the Dashboard did not support. This is additive functionality, not a parity gap.
- The Go error message for SKU conflicts is generic ("An item with this SKU already exists.") vs Dashboard which includes the SKU value. The Go approach is preferable (avoids leaking data).
