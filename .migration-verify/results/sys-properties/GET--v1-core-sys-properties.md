# Migration Verification: GET /v1/core/sys-properties

## Status: CONFIRMED PARITY

## What Was Compared

| Aspect | Result |
|--------|--------|
| Permission checks (internal actor, systemProperties/read) | Match |
| Account scoping via target account ID | Match |
| Search/filter by type name (LIKE query) | Match |
| DB query: joins sys_property_type to get type info | Match |
| Response includes type sub-resource (id, name, code) | Match (Go is richer with object field) |
| No side effects (read-only endpoint) | Match |
| No idempotency needed (GET) | Match |

## Expected Differences (by design)

- **Pagination**: Dashboard uses offset-based (take/skip + count), Go uses cursor-based pagination (standard Go API pattern)
- **Ordering**: Dashboard orders by `type.name ASC`, Go orders by `created_at DESC, id DESC` (required by cursor-based pagination design). With max ~8 sys properties per account, ordering difference is not functionally significant.
- **Response shape**: Dashboard returns `{ items, count }`, Go returns standard list resource with page info. This is the expected Go API convention.
- **Type field**: Dashboard returns type as a string enum code. Go returns type as a sub-resource `{ id, object, name, code }` following the API resource conventions.

## Issues Found

None. The Go implementation correctly preserves all business logic from the Dashboard.
