# Verification: GET /v1/core/settlements/{id}

## Result: Issue found and fixed

## What was compared

- **Permission checks**: Both implementations check internal actor + `settlements` read permission. **Parity confirmed.**
- **Account scoping**: Both scope by `identity.targetAccountID`. **Parity confirmed.**
- **Not found handling**: Dashboard checks for null and throws `HttpError.notFound('Settlement not found.')`. Go uses `db.MapSQLError` which converts `sql.ErrNoRows` to a not-found error. **Parity confirmed.**
- **DB query / JOINs**: Dashboard uses Prisma `findUnique` by `{ accountID, id }` with eager loading of allocations and responsible user via relations. Go uses a SQL query with LEFT JOINs. **Issue found (see below).**
- **Response shape**: Go correctly adds `object` field, uses expandable include pattern for `allocations`, and returns sub-resource stubs for `responsible_user`. These are intentional convention differences. **Parity confirmed.**
- **Side effects**: None for GET. **Parity confirmed.**
- **Idempotency**: GET endpoint, no idempotency key needed. **Parity confirmed.**
- **Validation**: Dashboard validates `settlementID` param via Zod schema. Go extracts `id` from path param. **Parity confirmed.**

## Issue found and fixed

### `responsible_user_id` JOIN was incorrect

The Prisma schema shows that `settlement.responsible_user_id` stores an **account_user ID** (not a user ID):

```prisma
responsibleAccountUserID  String?  @map("responsible_user_id")
responsibleUser           AccountUser?  @relation(fields: [responsibleAccountUserID], references: [id])
```

The Go `GetSettlement` SQL query had incorrect JOINs:

```sql
-- BEFORE (wrong): treats responsible_user_id as a user ID
LEFT JOIN account_user au ON au.user_id = s.responsible_user_id AND au.account_id = s.account_id
LEFT JOIN `user` u ON u.id = s.responsible_user_id

-- AFTER (fixed): treats responsible_user_id as an account_user ID
LEFT JOIN account_user au ON au.id = s.responsible_user_id
LEFT JOIN `user` u ON u.id = au.user_id
```

The old query would fail to match any account_user rows (and user rows) for existing data written by the Dashboard, causing `responsible_user` to always be null.

**Fix applied to**: `services/core-service/internal/infrastructure/queries/settlement.sql`
**sqlc regenerated**: `services/core-service/internal/infrastructure/sqlc/settlement.sql.go`

## Remaining concerns

- None for this endpoint. The fix is straightforward and the rest of the implementation has full parity.
- Note: pre-existing build errors exist on this branch in unrelated files (transaction handler, sales_order_repo, shipment_repository). These are not caused by this change.
