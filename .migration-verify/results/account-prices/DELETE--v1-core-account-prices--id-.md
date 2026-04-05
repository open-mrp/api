# Verification: DELETE /v1/core/account-prices/{id}

## Result: PARITY CONFIRMED — No issues found

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Actor check | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| Permission | `discounts:delete` | `PermissionDomainDiscounts, ActionDelete` | Yes |
| Account scoping | `ownerAccountID` from identity | `*identity.TargetAccountID` | Yes |
| Existence check | Explicit `checkExistence()` → 404 | Implicit via `GetRateIDByAccountPriceID` `:one` → `sql.ErrNoRows` → 404 | Functionally equivalent |
| Cascade: categories | `accountPriceCategory.deleteMany` | `DeleteAccountPriceCategoriesByPriceID` | Yes |
| Cascade: attributes | `accountPriceAttribute.deleteMany` | `DeleteAccountPriceAttributesByPriceID` | Yes |
| Cascade: rate | Not deleted | `DeleteRate` (additional cleanup) | Go does more — intentional improvement |
| Transaction | No (sequential Prisma calls) | Yes (`withTx` wrapper) | Go is better — atomicity guarantee |
| Response | 200 + deleted object body | 204 No Content | Go API convention |
| Idempotency | N/A (DELETE endpoint) | N/A (DELETE endpoint) | Correct — no idempotency keys needed |

## Files reviewed

**Dashboard:**
- `dashboard/apps/api/src/services/account-price.svc.ts` (lines 114-126)
- `dashboard/apps/api/src/repositories/account-price.repo.ts` (lines 182-233)

**Go:**
- `services/api-gateway/endpoints/account-prices/endpoint_delete_account_price.go`
- `services/core-service/internal/infrastructure/grpc/grpc_account_price_handler.go` (lines 175-186)
- `services/core-service/internal/service/account_price_service.go` (lines 302-337)
- `services/core-service/internal/infrastructure/repository/account_price_repository.go` (lines 385-426)
- `services/core-service/internal/infrastructure/queries/account_price.sql` (lines 155-188)

## Notes

- The Go implementation correctly preserves all Dashboard business logic (permission checks, account scoping, cascade deletes).
- The additional rate record deletion in Go is an improvement that prevents orphaned rate records.
- The transaction wrapper in Go provides atomicity that the Dashboard's sequential Prisma calls lack.
- The 204 No Content response is the standard convention for DELETE endpoints in the Go API.
- The 404 error message differs slightly ("Resource not found." vs "Account price not found.") but this is consistent with the Go API's generic `db.MapSQLError` pattern used across all services.
