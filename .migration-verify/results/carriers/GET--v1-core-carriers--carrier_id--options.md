# Verification: GET /v1/core/carriers/{carrier_id}/options

**Status: PARITY CONFIRMED** — No issues found.

## What was compared

### Authorization
- **Dashboard:** `checkIsAssignedActor` (allows internal + customer actors), then `checkHasPermission(carriers, read)` for internal actors only.
- **Go:** `CheckIsAssignedActor()`, `CheckHasPermission(PermissionDomainCarriers, ActionRead)` for internal users, `CheckTargetAccountSet()`, and `ReadAccess.CheckReadAccess()` for external targets.
- **Result:** Matches. Go adds standard external-target read access check which is an expected enhancement.

### DB Queries / Filtering
- **Dashboard:** Prisma `findMany` filtering by `accountID`, `carrierID`, optional `query` (fulltext relevance search on `name`). Includes `account_id IS NULL` records.
- **Go:** SQL filtering by `carrier_id`, `account_id` (OR `account_id IS NULL`), optional `search_query` (LIKE on `name`). Cursor-based pagination on `created_at DESC, id DESC`.
- **Result:** Matches. Both include system-wide options (NULL account_id) alongside account-specific ones. Pagination style differs (offset vs cursor) as expected per Go API conventions.

### Response Shape
- **Dashboard:** `{ items: CarrierOption[], count: number }`
- **Go:** Standard `List[CarrierOption]` with `data` array and `page_info` (cursor-based).
- **Result:** Expected difference — Go uses standard cursor-based list resource format.

### Resource Fields
- **Dashboard:** id, name, code, serviceLevelToken, isPortalEnabled, isDefault, carrierId, accountId, createdAt, updatedAt
- **Go:** id, object, name, code, service_level_token, is_portal_enabled, is_default, created_at, updated_at
- **Result:** All fields present. Go adds `object` field per convention. `carrier_id`/`account_id` not exposed in resource (carrier is parent in URL path), which is correct.

### Error Handling
- Both handle missing identity, permission errors, and standard list error paths.
- **Result:** Matches.

### Side Effects
- None on either side (read-only endpoint).

### Idempotency
- GET endpoint — inherently idempotent, no idempotency key needed. Both implementations are correct.

## Files Reviewed

**Dashboard:**
- `dashboard/apps/api/src/services/carrier-option.svc.ts` — `list()` method
- `dashboard/apps/api/src/repositories/carrier-option.repo.ts` — `list()` method

**Go:**
- `services/api-gateway/endpoints/carrier-options/endpoint_list_carrier_options.go`
- `services/api-gateway/endpoints/carrier-options/service.go` — `ListCarrierOptions()`
- `services/api-gateway/endpoints/carrier-options/presenter.go`
- `services/api-gateway/pkg/resource/carrier_option_resource.go`
- `services/core-service/internal/service/carrier_option_service.go` — `ListCarrierOptions()`
- `services/core-service/internal/infrastructure/queries/carrier_option.sql` — `ListCarrierOptionsForward`, `ListCarrierOptionsBackward`
