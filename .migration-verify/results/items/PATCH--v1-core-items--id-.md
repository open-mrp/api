# PATCH /v1/core/items/{id} — Migration Verification

## Result: Issue found and fixed

## What was compared

- **Validation rules**: Request accepts optional `sku`, `description`, `notes` fields — matches Dashboard's `ItemUtils.schema.partial()` body
- **Permission checks**: Internal actor + `items.update` permission — matches Dashboard's `checkIsInternalActor` + `checkHasPermission(PermissionDomains.items, 'update')`
- **DB queries and logic**: COALESCE-based update for non-null fields, explicit NULL setters for clearing nullable fields, SKU uniqueness check excluding current item — matches Dashboard behavior
- **Error handling**: Conflict error with message `"Item sku {sku} already exists."` on param `sku` — matches Dashboard's `HttpError.conflict`
- **Side effects**: None in either implementation — matches
- **Response shape**: Full Item resource with expandable category, unit_value, unit_cost, burn_rate, attributes — matches Dashboard's ItemAdapter.select/map
- **Idempotency**: Go implementation uses idempotency keys with recovery points for this PATCH endpoint — correctly added (Dashboard had none)

## Issue found and fixed

**Bug: `UpdateDescription` and `UpdateNotes` flags always set to `true`**

In `services/api-gateway/endpoints/items/service.go`, the `UpdateItem` method unconditionally set `UpdateDescription: true` and `UpdateNotes: true` when building the proto request. This caused any PATCH request that didn't include `description` or `notes` in the body to incorrectly clear those fields to NULL.

For example, sending `{"sku": "new-sku"}` would inadvertently set `description = NULL` and `notes = NULL`.

**Fix**: Changed to conditionally set these flags only when the respective field pointer is non-nil, matching the pattern used in the unit-groups service. Now:
- `{"sku": "new"}` → only SKU updated, description/notes unchanged
- `{"description": "new desc"}` → description updated
- `{"notes": "new notes"}` → notes updated

## Minor differences (acceptable)

1. **SKU trim in duplicate check**: Dashboard trims SKU (`value.trim()`) when checking for duplicates but not when saving. Go doesn't trim in either place. Go's behavior is more consistent.
2. **Deleted items in SKU check**: Dashboard's `isDuplicate` doesn't filter by `deletedAt`, meaning deleted items block SKU reuse. Go's `CheckItemSKUExists` includes `AND deleted_at IS NULL`, allowing SKU reuse after deletion. Go's behavior is an improvement.
