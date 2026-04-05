# DELETE /v1/core/account-users/{id}

**Status: Issues found and fixed**

## What was compared

- **Validation rules**: Path param only (user ID) — matches
- **Permission checks**: Both check `checkIsAssignedActor`, internal users need `customers:delete`, external targets need edit access — matches
- **DB queries and logic**: Both soft-delete by setting `status_code = 'removed'` — matches after fix
- **Error handling**: Dashboard returns 404 if account user not found — Go was missing this (fixed)
- **Side effects**: Dashboard does fire-and-forget `syncSubscriptionSeatCount`; Go publishes `PublishReportSeatChange` + `PublishSyncSeats` via outbox — functionally equivalent (Go is more reliable)
- **Response shape**: Dashboard returns 200 with updated AccountUser object; Go returns 204 No Content — intentional design difference for Go API convention
- **Idempotency**: DELETE endpoint, idempotent by design — no idempotency keys needed (both agree)

## Issues found and fixed

### 1. Missing 404 when account user not found

**Dashboard behavior**: Queries for the account user first, throws `HttpError.notFound('Account user not found.')` if no record exists.

**Go behavior (before fix)**: SQL query used `:exec` which runs `UPDATE ... WHERE` — if no rows match, it silently succeeds with no error.

**Fix**: Changed SQL annotation from `:exec` to `:execresult` and updated `SoftDelete()` in `account_user_repository.go` to check `RowsAffected()` and return `apierror.NewResourceNotFoundError("Account user not found.")` when 0 rows affected.

**Files modified**:
- `services/core-service/internal/infrastructure/queries/account_user.sql` — `:exec` → `:execresult`
- `services/core-service/internal/infrastructure/repository/account_user_repository.go` — added rows affected check
- `services/core-service/internal/infrastructure/sqlc/account_user.sql.go` — regenerated

## Notes

- Response shape difference (200 + body vs 204 empty) is an intentional Go API convention, not a parity issue.
- Pre-existing compile errors on this branch in `shipment_repository.go` and `sales_order_repo.go` are unrelated to this endpoint.
