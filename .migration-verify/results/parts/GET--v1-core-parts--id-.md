# Verification: GET /v1/core/parts/{id}

**Status:** Issues found and fixed

## What was compared

- Permission checks (actor type, permission domain, action)
- DB queries (joins, filters, soft delete check, tenant isolation)
- Attribute loading (fields selected, mapping)
- Response shape (field names, types, nested resources, expandables)
- Error handling (not found, validation)
- Idempotency (GET — naturally idempotent, no keys needed)
- Side effects (none in either implementation)

## Issues found and fixed

### 1. Permission domain mismatch (part_service.go)

**Dashboard:** `checkHasPermission(identity, PermissionDomains.items, 'read')`
**Go (before):** `identity.CheckHasPermission(types.PermissionDomainParts, types.ActionRead)`

The Dashboard uses the `items` permission domain for all part operations. The Go code was using `PermissionDomainParts` ("parts"), which is a different permission domain. Users with `items:read` but not `parts:read` would be blocked in Go but not in Dashboard.

**Fix:** Changed `PermissionDomainParts` to `PermissionDomainItems` in `GetPart`.

### 2. SQL query missing `order` column (part.sql)

**Dashboard:** Selects `order: true` for attributes via Prisma.
**Go (before):** `GetPartAttributes` query only selected `id, text, color_code, property_id`.

**Fix:** Added `a.\`order\`` to the `GetPartAttributes` SQL query and regenerated sqlc.

### 3. Repository not mapping `Order` field (part_repository.go)

**Dashboard:** Returns `order` on each attribute.
**Go (before):** `loadPartAttributes` did not set the `Order` field on `ItemAttribute`.

**Fix:** Added `Order: row.Order` to the attribute mapping in `loadPartAttributes`.

### 4. Presenter not passing `ColorCode` and `Order` (presenter.go)

**Dashboard:** Returns attributes with `color: { code: string }` and `order`.
**Go (before):** `lightAttributePresenter` only populated `ID`, `Object`, and `Text`.

The proto (`ItemAttributeInfo`) already carries `ColorCode` and `SortOrder`, and the gRPC handler (`itemAttributeToProto`) already populates them. The presenter was just not forwarding them to the API resource.

**Fix:** Added `ColorCode: a.GetColorCode()` and `Order: a.SortOrder` to `lightAttributePresenter`.

## Confirmed parity

- Actor type check: both require internal actor
- Tenant isolation: both filter by account_id
- Soft delete: both filter by `deleted_at IS NULL`
- Query structure: both join part → item → item_category → 3x rate
- Not found handling: Go returns 404 via `MapSQLError(sql.ErrNoRows)`, Dashboard throws `HttpError.notFound('Part not found.')`
- Response shape: Part with nested category, unit_value, unit_cost, burn_rate (expandable), attributes list, is_dirty, timestamps
- No side effects in either implementation
- GET is naturally idempotent — no idempotency key needed
