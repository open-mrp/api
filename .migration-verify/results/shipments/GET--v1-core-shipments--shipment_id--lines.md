# Verification: GET /v1/core/shipments/{shipment_id}/lines

## Status: Issues found and fixed

## Summary

The endpoint definition, gRPC handler, service layer, presenter, domain models, proto definitions, and SQL queries were all properly implemented. However, the **repository layer was entirely stubbed out** with TODO placeholders — none of the methods were wired to the generated sqlc code.

## What was compared

- **Permission checks**: Dashboard requires internal actor + `shipments.read`. Go implementation uses `CheckIsAssignedActor()` (allows both internal and customer actors) + `CheckHasPermission(shipments, read)` for internal users only. This is correct — it preserves customer portal access.
- **Account scoping**: Both implementations scope queries by account ID. Go service verifies the shipment belongs to the target account before listing lines.
- **DB queries**: SQL queries join `shipment_line` → `sales_order_line`, `quantity`, and `unit` tables. Cursor-based keyset pagination on `(created_at DESC, id DESC)`. Dashboard uses offset pagination; Go uses cursor pagination (an intentional improvement).
- **Response shape**: `ShipmentLine` resource includes `id`, `object`, `sales_order_line` (sub-resource with id/object/sku/description), `quantity` (sub-resource with id/object/value/unit), `created_at`, `updated_at`. Matches the Dashboard's data with proper sub-resource structure per conventions.
- **Validation**: Path parameter `shipment_id` is required. Pagination parameters (cursor, limit) handled by `PaginationRequest`.
- **Side effects**: None (read-only endpoint). Correct.
- **Idempotency**: GET endpoint, inherently idempotent. No idempotency key needed. Correct.
- **Error handling**: Proper not-found error when shipment doesn't exist in account.

## Issues found and fixed

### Repository not implemented (all methods were TODO stubs)

The entire `shipment_line_repository.go` file had TODO placeholders for every method:
- `List` — returned empty results
- `Get` — always returned not-found error
- `Create` — always returned "not yet implemented" error
- `Update` — always returned "not yet implemented" error
- `Delete` — no-op
- `IsInShipment` — always returned false
- `ListByShipment` — always returned nil
- `DeleteByShipment` — no-op

**Fix**: Implemented all repository methods using the generated sqlc code:
- `List`: Full cursor-based forward/backward pagination using `ListShipmentLinesForward`/`ListShipmentLinesBackward` queries with `pagination.BuildPageString`
- `Get`: Uses `GetShipmentLine` sqlc query with proper error mapping
- `Create`: Creates quantity first, then shipment line, then fetches the result
- `Update`: Updates quantity via `UpdateShipmentLine` query, checks rows affected
- `Delete`: Deletes quantity first (uses subquery on shipment_line), then deletes the line
- `IsInShipment`: Uses `IsShipmentLineInShipment` query
- `ListByShipment`: Uses `ListShipmentLinesByShipment` query (unpaginated, for internal use)
- `DeleteByShipment`: Deletes quantities first, then lines (for cascade delete)

## Remaining concerns

- The pre-existing build errors in `item_repository.go`, `sales_order_repo.go`, and `shipment_repository.go` are unrelated to this endpoint but may need to be addressed separately.
- The Dashboard does not have a standalone shipment lines listing endpoint — lines are returned as part of the shipment object. The Go API adds this as a dedicated paginated endpoint, which is an improvement for large shipments with many lines.
