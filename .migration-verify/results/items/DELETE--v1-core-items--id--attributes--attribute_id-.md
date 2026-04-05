# DELETE /v1/core/items/{id}/attributes/{attribute_id}

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Internal actor + items/update permission
- **Account scoping**: Item must belong to the caller's target account
- **Soft-delete check**: Item must not be soft-deleted
- **DB query**: Junction table delete on `_item_attributes`
- **Error handling**: 404 when relationship not found
- **Response shape**: Returns full Item object with HTTP 200
- **Side effects**: None in either implementation

## Issues found

### 1. Missing account scoping and soft-delete check in SQL (FIXED)

**Dashboard behavior**: Prisma's `update` scopes to `{ accountID, id: itemID, deletedAt: null }` before disconnecting the attribute. This ensures:
- The item belongs to the caller's account
- The item is not soft-deleted

**Go behavior (before fix)**: The SQL was:
```sql
DELETE FROM _item_attributes WHERE A = ? AND B = ?
```
This deletes the junction row without verifying account ownership or soft-delete status. While the subsequent `Get` call scopes by account, the delete side effect already occurred.

**Fix applied** (`queries/item.sql` + `sqlc/item.sql.go` + `repository/item_repository.go`):
```sql
DELETE ia FROM _item_attributes ia
JOIN item i ON i.id = ia.B
WHERE ia.A = sqlc.arg('attribute_id')
  AND ia.B = sqlc.arg('item_id')
  AND i.account_id = sqlc.arg('account_id')
  AND i.deleted_at IS NULL;
```

The repository now passes `AccountID` to the sqlc params.

## No remaining concerns

All other aspects match:
- Permission checks are identical (internal actor + items/update)
- Response shape is consistent (full Item with 200 OK)
- No side effects in either implementation
- Error handling is adequate (rowsAffected == 0 returns 404)
