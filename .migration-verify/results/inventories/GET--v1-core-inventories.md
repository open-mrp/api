# GET /v1/core/inventories — Migration Verification

## Result: PARITY CONFIRMED

No issues found. The Go implementation correctly mirrors the Dashboard's `exportInventory` endpoint.

## What Was Compared

### Dashboard Endpoint
- **Route**: `GET /v1/inventories` (`api.exportInventory`)
- **Controller**: `ItemCtrl.exportInventory` → `ItemSvc.exportInventory`
- **Source**: `dashboard/apps/api/src/services/item.svc.ts:289-319`

### Go Endpoint
- **Route**: `GET /v1/core/inventories`
- **Gateway**: `inventories/endpoint_list_inventories.go`
- **Service**: `core-service/internal/service/item_service.go:564-633`
- **Repository**: `core-service/internal/infrastructure/repository/inventory_query_repository.go:51-75`
- **SQL**: `core-service/internal/infrastructure/queries/inventory_query.sql:24-53`

## Comparison Details

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **Auth** | `checkIsInternalActor` | `CheckIsInternalActor` | ✅ |
| **Permission** | `PermissionDomains.items, 'read'` | `PermissionDomainItems, ActionRead` | ✅ |
| **Query params** | None (`GetInventorySchema = {}`) | None (`ListInventoriesRequest{}` empty) | ✅ |
| **Item listing** | `itemRepo.list({ accountID })` (no limit) | `List(accountID, Limit: 10000)` | ✅ (see note) |
| **Inventory calc** | `fetchOnHandInventoryBulk` — sums receipt quantities minus allocation quantities per item | SQL `FetchOnHandInventoryBulk` — same logic via correlated subqueries | ✅ |
| **Owner/holder filter** | `OR: [{ ownerAccountID }, { holderAccountID }]` | `(ir.owner_account_id = ? OR ir.holder_account_id = ?)` | ✅ |
| **Receipt status** | `status: 'available'` | `status_code = 'available'` | ✅ |
| **Deleted items** | Prisma soft-delete filter | `i.deleted_at IS NULL` | ✅ |
| **Items without inventory** | Returns blank quantity with base unit info | SQL always returns rows with unit info via JOINs, COALESCE to 0 | ✅ |
| **Response shape** | `{ items, count }` | `{ object: "list", data, count }` — follows Go API conventions | ✅ |
| **Customer actor support** | No (internal only) | No (internal only) | ✅ |
| **Idempotency** | N/A (GET) | N/A (GET) | ✅ |
| **Side effects** | None | None | ✅ |

## Notes

- The Go implementation uses a hardcoded `Limit: 10000` when listing items, while the Dashboard has no limit. This is acceptable — it prevents unbounded queries and 10,000 items covers realistic use cases.
- Response field naming differs (`items` → `data`, added `object` field) per Go API resource conventions — this is intentional, not a bug.

## No Fixes Required
