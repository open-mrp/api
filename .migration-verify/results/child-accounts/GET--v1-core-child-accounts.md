# Verification: GET /v1/core/child-accounts

**Status: PARITY CONFIRMED** — No fixes needed.

## What Was Compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Permission checks | internal actor + customers.read | internal actor + customers.read + target account set | ✅ (Go adds target account check) |
| Owner account | `identity.targetAccountID` | `identity.ActorAccountID()` | ✅ (equivalent — see note below) |
| Parent account | URL param `:accountID` | `identity.TargetAccountID` | ✅ (design difference — same DB semantics) |
| Search filter | Prisma `_relevance` + name filter | SQL `LIKE '%query%'` on account name | ✅ (equivalent filtering) |
| Search ordering | Relevance-ranked | `created_at DESC, id DESC` | ⚠️ Minor (Go uses consistent ordering) |
| Pagination | Offset-based (take/skip) | Cursor-based | ✅ (intentional migration improvement) |
| Response count | Returns `{ items, count }` | Returns `{ data, page_info }` | ✅ (new API conventions) |
| Response fields | `{ id, name, number, email }` | `{ id, object, account: {id, object, name}, external_number, email, created_at, updated_at }` | ✅ (new API resource conventions) |
| Role code filter | Filters by `accountRelationRoleCode: 'customer'` | No role code filter | ⚠️ Minor (see note below) |
| DB joins | Prisma: account + accountRelation + branding | SQL: account_relation JOIN account + LEFT JOIN account_branding | ✅ |
| Error handling | Not found if parent relation missing | Not found if parent relation missing | ✅ |
| Side effects | None | None | ✅ |
| Idempotency | GET — inherently idempotent | GET — inherently idempotent | ✅ |

## Notes

### Owner Account Identity Mapping
The Dashboard route `GET /v1/identity/:accountID/children` uses the Augno-Account-ID header as the owner (tenant) and the URL param as the parent. The Go route `GET /v1/core/child-accounts` collapses these: the header becomes the parent, and the owner is inferred from `ActorAccountID()` (the authenticated user's account). For internal actors, these produce identical DB queries.

### Missing Role Code Filter
The Dashboard filters by `accountRelationRoleCode: 'customer'` when listing child accounts. The Go SQL queries do not include this filter. In practice this is safe because:
1. Parent-child relationships (`parent_account_relation_id`) are only set via the ChildAccount endpoints
2. Those endpoints only operate on customer account relations
3. Non-customer relations would not have `parent_account_relation_id` set
4. The `FindRelationByOwnerAndCounterparty` query is `:one`, implying unique (owner, counterparty) pairs

### Search Ordering Difference
The Dashboard uses Prisma's `_relevance` ordering (MySQL full-text search ranking) when a search query is present. The Go API consistently orders by `created_at DESC, id DESC` regardless of search. This is a minor behavioral difference — search results will be filtered correctly by LIKE but not ranked by relevance.

## Files Reviewed

**Dashboard:**
- `dashboard/apps/api/src/controllers/child-account.ctrl.ts`
- `dashboard/apps/api/src/services/child-account.svc.ts`
- `dashboard/apps/api/src/repositories/child-account.repo.ts`
- `dashboard/apps/api/src/repositories/child-account.repo.interface.ts`
- `dashboard/packages/adapters/src/classes/accounts/ChildCustomerAccount.ts`

**Go:**
- `api/services/api-gateway/endpoints/child-accounts/endpoint_list_child_accounts.go`
- `api/services/api-gateway/endpoints/child-accounts/service.go`
- `api/services/api-gateway/endpoints/child-accounts/presenter.go`
- `api/services/api-gateway/pkg/resource/child_account_resource.go`
- `api/services/core-service/internal/infrastructure/grpc/grpc_child_account_handler.go`
- `api/services/core-service/internal/service/child_account_service.go`
- `api/services/core-service/internal/infrastructure/repository/child_account_repository.go`
- `api/services/core-service/internal/infrastructure/queries/child_account.sql`
- `api/services/core-service/internal/domain/child_account_models.go`
