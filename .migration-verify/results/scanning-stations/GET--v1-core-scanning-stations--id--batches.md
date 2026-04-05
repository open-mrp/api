# Migration Verification: GET /v1/core/scanning-stations/{id}/batches

## Result: PARITY CONFIRMED — No issues found

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Actor check | Internal actor only | Internal actor only | ✅ |
| Permission | batches:read | batches:read | ✅ |
| Account isolation | accountID from identity | account_id from identity | ✅ |
| Scanning station filter | scanningStationID from path | scanning_station_id from path | ✅ |
| Scanned-only filter | `scannedAt: { not: null }` | `scanned_at IS NOT NULL` | ✅ |
| Ordering | scannedAt DESC | scanned_at DESC, id DESC | ✅ (secondary sort for stable cursor pagination) |
| Pagination | Offset-based (take/skip) | Cursor-based (cursor/limit) | ✅ (Go API convention) |
| Response shape | `{ items, count }` | `{ data, page_info }` | ✅ (Go API convention) |
| Query param | Accepted but NOT used in DB query | Used for item SKU LIKE search | ✅ (enhancement) |
| Side effects | None | None | ✅ |
| Idempotency | N/A (GET) | N/A (GET) | ✅ |

## Intentional Architectural Differences

These differences align with Go API conventions and are expected:

1. **Pagination model**: Dashboard uses offset-based (take/skip), Go uses cursor-based pagination — standard for all Go API list endpoints.
2. **Secondary sort key**: Go adds `id DESC` as a tiebreaker for deterministic cursor pagination.
3. **Query parameter**: Dashboard accepts `query` in the DTO/service but the repository ignores it (destructured but not used in the Prisma where clause). Go implements it as a SKU search filter — this is an enhancement, not a regression.
4. **Response shape**: Go uses standard `List[Batch]` with `page_info` instead of `{ items, count }`.

## Files Reviewed

### Dashboard
- `dashboard/apps/api/src/controllers/scanning-station.ctrl.ts` — controller
- `dashboard/apps/api/src/services/batch.svc.ts` (lines 53-74) — service logic
- `dashboard/apps/api/src/repositories/batch.repo.ts` (lines 88-116) — repository/query

### Go
- `services/api-gateway/endpoints/batches/endpoint_list_by_scanning_station.go` — endpoint definition
- `services/core-service/internal/service/batch_service.go` — service logic
- `services/core-service/internal/infrastructure/grpc/grpc_batch_handler.go` — gRPC handler
- `services/core-service/internal/infrastructure/repository/batch_repository.go` — repository
- `services/core-service/internal/infrastructure/queries/batch.sql` — SQL queries

## Issues Found and Fixed

None — the Go implementation correctly preserves all Dashboard business logic.
