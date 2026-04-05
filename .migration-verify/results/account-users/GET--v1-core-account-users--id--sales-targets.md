# Verification: GET /v1/core/account-users/{id}/sales-targets

**Status: Parity confirmed — no code changes needed**

## What was compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: actor type | Internal actor | Internal actor | Yes |
| Permission: domain/action | salesTargets / read | salesTargets / read | Yes |
| Account scoping | identity.targetAccountID | identity.TargetAccountID | Yes |
| DB filter: sales_rep_id | accountUser.id (looked up from userID) | Path param `{id}` (account-user ID directly) | Yes |
| DB filter: account_id | Path param accountID | identity.TargetAccountID | Yes |
| Pagination | take/skip | limit/offset | Yes (names differ, same capability) |
| Query search param | Accepted but unused in repo | Accepted in proto but unused in SQL | Yes |
| Response: list items | SalesTarget objects with amount/quantity | SalesTarget resources with nested Amount (Quantity) | Yes |
| Side effects | None | None | Yes |
| Idempotency | N/A (GET) | N/A (GET) | Yes |

## Minor differences (acceptable, no fix needed)

1. **Route structure**: Dashboard uses `/v1/identity/:accountID/users/:userID/targets` (takes user ID, looks up account_user). Go uses `/v1/core/account-users/{id}/sales-targets` (takes account-user ID directly). This is a deliberate route redesign for the Go API.

2. **Ordering**: Go adds `ORDER BY t.start_date DESC`. Dashboard Prisma query has no explicit ordering. The Go version is an improvement.

3. **Total count**: Dashboard returns `{ items: [...], count: N }`. Go's `List` resource uses cursor-based `PageInfo` without a total count field. The count is fetched from DB and passed through gRPC but dropped in the presenter. This is consistent with all other Go list endpoints.

4. **Account user existence validation**: Dashboard validates the account user exists first and returns 404 if not found. Go skips this check — a non-existent account-user ID simply yields an empty list. No security concern since the `account_id` filter prevents cross-account data leakage. Adding this check would require new repository methods across multiple layers for minimal benefit given the route redesign.

## Files reviewed

### Dashboard
- `dashboard/apps/api/src/controllers/account-user.ctrl.ts` — controller
- `dashboard/apps/api/src/services/account-user.svc.ts` — service (fetchSalesTargets)
- `dashboard/apps/api/src/repositories/sales-target.repo.ts` — repository (list)
- `dashboard/packages/dtos/src/sections/account-users.ts` — route/schema definition

### Go
- `services/api-gateway/endpoints/sales-targets/endpoint_list_sales_targets.go` — endpoint definition
- `services/api-gateway/endpoints/sales-targets/service.go` — API gateway service
- `services/api-gateway/endpoints/sales-targets/presenter.go` — presenter
- `services/api-gateway/pkg/resource/sales_target_resource.go` — API resource
- `services/core-service/internal/infrastructure/grpc/grpc_sales_target_handler.go` — gRPC handler
- `services/core-service/internal/service/sales_target_service.go` — domain service
- `services/core-service/internal/domain/sales_target_models.go` — domain models
- `services/core-service/internal/infrastructure/repository/sales_target_repository.go` — repository
- `services/core-service/internal/infrastructure/queries/sales_target.sql` — SQL queries
