# Migration Verification: GET /v1/core/settlements

## Status: CONFIRMED PARITY

No code changes required.

## What Was Compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Permission: isInternalActor | Yes | Yes | Yes |
| Permission: settlements.read | Yes | Yes | Yes |
| Requires target account ID | Yes | Yes | Yes |
| Filter: query (search) | number field only | number + note (fulltext) | Superset |
| Filter: transactionIDs | Yes (allocation join) | Yes (allocation join) | Yes |
| Filter: invoiceIDs | Yes (allocation join) | Yes (allocation join) | Yes |
| Filter: startDate/endDate | createdAt gte/lte | createdAt gte/lte | Yes |
| Pagination | Offset-based (take/skip) | Cursor-based | By design |
| Ordering | relevance desc, createdAt desc | createdAt DESC, id DESC | Acceptable |
| Response: allocationCount | Computed in-memory | SQL COUNT aggregate | Yes |
| Response: totalPayments | Computed in-memory by type | SQL SUM by type code | Yes |
| Response: totalRebates | Computed in-memory by type | SQL SUM by type code | Yes |
| Response: totalAdjustments | Computed in-memory by type | SQL SUM by type code | Yes |
| Response: totalCredits | Computed in-memory by type | SQL SUM by type code | Yes |
| Response: invoiceNumbers | Computed in-memory | SQL GROUP_CONCAT | Yes |
| Response: customerNames | Computed in-memory (`customers`) | SQL GROUP_CONCAT (`customer_names`) | Yes (name differs, route differs) |
| Side effects | None | None | Yes |
| Error handling | Standard HTTP errors | Standard API errors | Yes |

## Notes

1. **Search scope expanded**: Go fulltext search includes both `number` and `note` fields, while Dashboard only searches `number`. This is a strict superset — all Dashboard results are returned plus additional matches on note. Not a regression.

2. **Pagination model change**: Dashboard uses offset-based (take/skip + count), Go uses cursor-based pagination. This is an intentional architectural decision for the Go API and is consistent across all migrated endpoints.

3. **Relevance ordering**: Dashboard uses Prisma `_relevance` on the number field to rank search results. Go uses fulltext MATCH for filtering but orders by `created_at DESC, id DESC` regardless. When no search query is active, both effectively order by createdAt desc.

4. **Field naming**: Dashboard returns `customers` array, Go returns `customer_names`. Since routes differ (`/v1/settlements` vs `/v1/core/settlements`), exact field name parity is not required.

5. **Aggregation approach**: Dashboard fetches allocations and computes totals in-memory via the SettlementSummaryAdapter map function. Go computes aggregates directly in SQL with SUM/CASE/GROUP_CONCAT, which is more efficient.

## Files Reviewed

### Dashboard
- `dashboard/apps/api/src/controllers/settlement.ctrl.ts` — Controller
- `dashboard/apps/api/src/services/settlement.svc.ts` — Service
- `dashboard/apps/api/src/repositories/settlement.repo.ts` — Repository
- `dashboard/packages/dtos/src/sections/settlements.ts` — Request/response schemas
- `dashboard/packages/adapters/src/classes/payments/Settlement.ts` — Adapter (fetchInput, select)
- `dashboard/packages/adapters/src/classes/payments/SettlementSummary.ts` — Summary adapter
- `dashboard/packages/objects/src/classes/payments/SettlementSummary.ts` — Summary object

### Go
- `api/services/api-gateway/endpoints/settlements/endpoint_list_settlements.go` — Endpoint definition
- `api/services/api-gateway/endpoints/settlements/service.go` — Gateway service
- `api/services/api-gateway/endpoints/settlements/presenter.go` — Response presenter
- `api/services/api-gateway/pkg/resource/settlement_resource.go` — API resource types
- `api/services/core-service/internal/infrastructure/grpc/grpc_settlement_handler.go` — gRPC handler
- `api/services/core-service/internal/service/settlement_service.go` — Domain service
- `api/services/core-service/internal/infrastructure/repository/settlement_repository.go` — Repository
- `api/services/core-service/internal/infrastructure/queries/settlement.sql` — SQL queries
- `api/services/core-service/internal/domain/settlement_models.go` — Domain models
