# PATCH /v1/core/suppliers/{supplier_id}/materials/{id}

## Result: Issue found and fixed

## What was compared

- **Validation**: Both use optional fields for `supplier_part_number`, `supplier_description`, and `is_active`
- **Permissions**: Both check `internal actor` + `suppliers:update` permission + target account header
- **DB logic**: Both look up the existing supplier material by (owner_account_id, supplier_account_id, item_id), then apply a partial update using COALESCE for optional fields
- **Error handling**: Both return 404 if the supplier material is not found
- **Response shape**: Both return the full updated supplier material with nested material/item data
- **Idempotency**: Go correctly uses idempotency keys with recovery points (PATCH endpoint requirement)
- **Side effects**: Neither endpoint has side effects beyond the DB update
- **Status code**: Both return HTTP 200 OK

## Issue found and fixed

**`UpdateDescription` hardcoded to `true`** in `services/api-gateway/endpoints/supplier-materials/service.go` (line 107).

The Go API gateway was always setting `UpdateDescription: true` when building the proto request. This means any PATCH request that omits `supplier_description` would clear it to NULL via the SQL:

```sql
supplier_description = CASE WHEN sqlc.arg('update_description') THEN sqlc.narg('supplier_description') ELSE supplier_description END
```

In the Dashboard, Prisma treats an `undefined` field as "don't update," preserving the existing value. The fix changes `UpdateDescription` from `true` to `req.SupplierDescription != nil`, so the description is only updated when the user explicitly includes it in the request body.

## Remaining concerns

None. The Go implementation now matches Dashboard behavior for all three updatable fields.
