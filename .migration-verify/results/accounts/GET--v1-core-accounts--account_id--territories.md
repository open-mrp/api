# Verification: GET /v1/core/accounts/{account_id}/territories

**Status: Issues found and fixed**

## What was compared

- **Permission checks**: Both check `checkIsInternalActor` + `territories.read` permission + target account set. ✅ Parity confirmed.
- **Search/filter logic**: Both search by state, sales rep name/email, product line name, and zipcode containment. Go additionally searches user email (superset). ✅
- **Sort order**: Dashboard: `createdAt desc`. Go: `created_at DESC, id DESC` (adds ID tiebreaker for stable cursor pagination). ✅ Acceptable.
- **Pagination**: Dashboard uses offset-based (take/skip). Go uses cursor-based pagination. ✅ Intentional Go API convention.
- **Response shape**: Dashboard returns `{items, count}`. Go returns standard list with `page_info`. ✅ Go API convention.
- **Error handling**: Both return not-found for missing territories, validation errors for invalid params. ✅
- **Side effects**: None in either implementation. ✅
- **Idempotency**: GET endpoint, not applicable. ✅

## Issue found and fixed

**Zipcode search logic discrepancy**

The Go SQL required both `start_zipcode IS NOT NULL AND end_zipcode IS NOT NULL` for zipcode containment matching. The Dashboard handles territories where `start_zipcode` is set but `end_zipcode` is null — these match when the query exactly equals `start_zipcode`.

Dashboard logic (TerritoryAdapter.parseZipcodeQuery):
```
AND [
  OR [startZipcode IS NULL, startZipcode <= zipcodeNumber],
  OR [(endZipcode IS NULL AND startZipcode = zipcodeNumber), endZipcode >= zipcodeNumber]
]
```

**Files modified:**
- `services/core-service/internal/infrastructure/queries/territory.sql` — Updated both Forward and Backward queries to match Dashboard's zipcode containment logic
- `services/core-service/internal/infrastructure/sqlc/territory.sql.go` — Updated generated code to match (added `ZipcodeQuery_4` param)
- `services/core-service/internal/infrastructure/repository/territory_repository.go` — Pass the additional `ZipcodeQuery_4` param

## Remaining notes

- Run `make sqlc core` to regenerate sqlc code and confirm it matches the manual edits.
