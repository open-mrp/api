# POST /v1/core/parts — Migration Parity Verification

## Result: Issue found and fixed

## What was compared

| Aspect | Dashboard | Go | Parity |
|--------|-----------|-----|--------|
| **Permission: actor type** | `checkIsInternalActor` | `CheckIsInternalActor` | Match |
| **Permission: domain** | `PermissionDomains.items` ("items") | `PermissionDomainParts` ("parts") | Intentional split — see note |
| **Permission: action** | `'create'` | `ActionCreate` | Match |
| **Account scoping** | `identity.targetAccountID` | `identity.TargetAccountID` | Match |
| **SKU uniqueness check** | `isDuplicate()` on item table within account | `CheckPartSKUExists` on item table within account, excluding soft-deleted | Match |
| **SKU conflict error** | 409: "Item sku {sku} already exists." | 409: "An item with this SKU already exists." (param: "sku") | Acceptable — Go uses parameterized errors |
| **Rate initialization** | Rates from request body (defaults to measure=0) | Hardcoded "0" with category base unit | Intentional simplification — Go API only accepts sku/description/category_id |
| **Item insert** | Prisma `item.create` with all fields | `InsertItemForPart` SQL with matching fields | Match |
| **Part insert** | Prisma `part.create` | `InsertPart` SQL | Match |
| **Inventory log creation** | `inventoryRepo.createLog()` with blank quantity | Was **MISSING** — now added | **Fixed** |
| **Inventory change log creation** | `inventoryRepo.createChangeLog()` with `ActionTypes.userAction` | Was **MISSING** — now added | **Fixed** |
| **Idempotency** | Not present in dashboard | Full idempotency key support with recovery points | Go improvement |
| **Response shape** | Full Part with nested Category, Rates, Attributes | Full Part with nested Category, Rates, Attributes via presenter | Match |
| **Attributes in response** | Loaded via Prisma select | Loaded via `GetPartAttributes` query | Match |

## Issue found and fixed

**Missing inventory log and change log initialization** — The dashboard creates an inventory log and an inventory change log (both with zero quantity) immediately after creating a part. The Go part service was missing this. The Go material service already had this pattern implemented correctly.

**Fix applied** in `services/core-service/internal/service/part_service.go`:
- Added `CreateInventoryLog` call with zero measure and the category's base unit ID
- Added `CreateInventoryChangeLog` call with zero measure, base unit ID, and `"user_action"` action type
- Both calls happen inside the transaction, matching the material service pattern

## Notes

- **Permission domain split**: The dashboard uses a single `"items"` permission domain for all item types. The Go API intentionally splits this into `"parts"`, `"materials"`, `"products"` etc. This is a deliberate design improvement, not a parity gap.
- **Rate unit initialization**: The Go API initializes all rate numerator/denominator units to the category's base unit (same as the material service). The dashboard allows clients to provide custom units in the create request. This is an intentional simplification in the Go API design — rates can be updated via the update endpoint.
- **Attributes in create request**: The dashboard accepts attributes in the create body, but the Go API does not. Attributes are managed separately via dedicated endpoints, which is consistent with the Go API's resource model.
