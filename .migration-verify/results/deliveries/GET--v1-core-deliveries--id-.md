# Verification: GET /v1/core/deliveries/{id}

**Status: PARITY CONFIRMED** — No issues found.

## What was compared

### Permission checks
- **Dashboard**: `checkIsInternalActor`, `checkHasPermission(deliveries, read)`
- **Go**: `CheckIsInternalActor`, `CheckHasPermission(PermissionDomainDeliveries, ActionRead)`, `CheckTargetAccountSet`
- Match confirmed.

### DB queries and logic
- **Dashboard**: Prisma `findFirst` with `WHERE { id, accountID: ownerAccountID }`, includes `purchaseOrder { id, number }` and nested `lines` with quantity, unitCost, item (via receivingOrderLine → orderLine), storageLocation, lot.
- **Go**: Two queries — `GetDelivery` (header with JOIN sales_order, filtered by `d.id` and `d.account_id`) + `ListDeliveryLines` (joins quantity, unit, rate, receiving_order_line → sales_order_line → item, storage_location, lot, ordered by `created_at ASC, id ASC`).
- Functionally equivalent. Both scope by account ID and delivery ID.

### Error handling
- **Dashboard**: Returns `HttpError.notFound('Delivery not found.')` when repo returns null.
- **Go**: `GetDelivery` is a `:one` sqlc query; `sql.ErrNoRows` is mapped by `db.MapSQLError` to a not-found error.
- Match confirmed.

### Response shape
Both return the same fields:
- `id`, `number`, `status` (from `delivery_status_code`/`acceptanceStatusCode`)
- `purchase_order`: `{ id, number }` (Go adds `object` field per API conventions)
- `lines[]`: `id`, `quantity` (id, value, unit), `unit_cost` (id, value, numerator_unit, denominator_unit), `item` (id, sku), `storage_location` (id, name), `lot` (id, lot_number), `accepted_at`, `rejected_at`, `created_at`, `updated_at`
- `accepted_at`, `rejected_at`, `created_at`, `updated_at`
- Go adds `object` fields throughout and wraps lines in a `List` type (both expected per Go API conventions).

### Minor behavioral difference (acceptable)
- **Dashboard** throws an error if `item` is null on a delivery line (`'Delivery line item is required'`).
- **Go** uses `LEFT JOIN item` and treats item as nullable, returning `null` for the item sub-resource if the item has been deleted.
- This is a defensive improvement in Go — not a parity issue.

### Side effects
None in either implementation.

### Idempotency
GET endpoint — idempotent by design, no idempotency key needed.

## Conclusion
No code changes required. The Go implementation faithfully reproduces the Dashboard business logic for this endpoint.
