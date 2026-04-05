# GET /v1/core/inventory-change-logs/{id}

## Result: PARITY CONFIRMED

No issues found. The Go implementation is a faithful migration of the Dashboard endpoint.

## What was compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **Permission: actor type** | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| **Permission: domain/action** | `inventoryLogs` / `read` | `PermissionDomainInventoryLogs` / `ActionRead` | Yes |
| **Account isolation** | `accountID` filter in Prisma query | `account_id` filter in SQL WHERE clause | Yes |
| **404 handling** | `HttpError.notFound('Inventory change log not found.')` | `db.MapSQLError` converts no-rows to 404 | Yes |
| **Customer actor support** | Not supported (internal only) | Not supported (internal only) | Yes |
| **Idempotency** | N/A (GET) | N/A (GET) | Yes |

## Response shape comparison

| Field | Dashboard | Go |
|-------|-----------|-----|
| `id` | `id` | `id` |
| `object` | `object` | `object` (ObjectTypeInventoryChangeLog) |
| `actionType` / `action_type_code` | `actionType` (string) | `action_type_code` (string) |
| `quantity` | Sub-resource with `measure`, `unit` | Sub-resource with `value`, `display_value`, expandable `unit` |
| `item` | Sub-resource (id, sku, name, etc.) | Sub-resource (id, sku), expandable |
| `responsible_user` | Sub-resource or null | Sub-resource or null, expandable |
| `responsible_scanning_station` | Sub-resource or null | Sub-resource or null, expandable |
| `created_at` | timestamp | timestamp |
| `updated_at` | timestamp | timestamp |

## DB query comparison

- **Dashboard**: Prisma `findFirst` with `accountID` + `id` filter, includes related item, quantity (with unit), user, scanning station via Prisma select
- **Go**: SQL query with `JOIN item`, `JOIN quantity`, `JOIN unit`, `LEFT JOIN scanning_station`, `LEFT JOIN user`, filtered by `id` and `account_id`
- Both queries fetch the same data with equivalent joins

## Notes

- Field naming differences (`actionType` vs `action_type_code`, `measure` vs `value`) follow Go API conventions and are intentional
- Go API uses expandable sub-resources pattern (stubs by default, full data via include expansion) which is the standard Go API convention
- The Go API adds `display_value` on quantity which is a convenience field not in Dashboard — this is additive, not a breaking change
