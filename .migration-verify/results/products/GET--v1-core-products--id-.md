# GET /v1/core/products/{id} — Migration Verification

## Result: PARITY CONFIRMED

No issues found. The Go implementation matches the Dashboard business logic.

## What Was Compared

### Permission Checks
- **Dashboard**: `checkIsAssignedActor`, customer actors allowed without explicit permission, internal actors require `items` / `read`
- **Go**: `CheckIsAssignedActor()`, customer actors pass through (only internal users checked), internal actors require `PermissionDomainItems` / `ActionRead`
- **Verdict**: Match

### Database Query & Filtering
- **Dashboard**: Prisma `findFirst` on `product` where `item.id = itemID`, `item.accountID = accountID`, `item.deletedAt = null`
- **Go**: SQL `GetProductByItemID` with `WHERE p.item_id = ? AND i.account_id = ? AND i.deleted_at IS NULL`, 6-way JOIN to product_line (LEFT JOIN), product_type, item, item_category, and 3 rate tables
- **Verdict**: Match — both filter by item_id + account_id + soft-delete check

### Not Found Handling
- **Dashboard**: Returns `null` from repo, service throws `HttpError.notFound('Product not found.')`
- **Go**: sqlc `:one` query returns `sql.ErrNoRows` → `MapSQLError` maps to `NewResourceNotFoundError("Resource not found.")`
- **Verdict**: Match (both return 404)

### Response Shape
- **Dashboard**: Returns flat product with spread item fields, nested `productType`, `productLine` (nullable), `isPortalReady`, `margin: null`
- **Go**: Returns `Product` resource with expandable sub-resources: `Item`, `ProductLine` (nullable), `ProductType`, `IsPortalReady`, `CreatedAt`, `UpdatedAt`
- **Verdict**: Match (Go uses sub-resource pattern per conventions)

### Side Effects
- **Dashboard**: None
- **Go**: None
- **Verdict**: Match

### Idempotency
- Both are GET endpoints — no idempotency keys needed
- **Verdict**: Match

## No Issues Found
