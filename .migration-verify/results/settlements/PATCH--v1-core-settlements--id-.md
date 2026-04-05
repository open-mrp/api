# PATCH /v1/core/settlements/{id} — Migration Verification

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Internal actor + settlements domain + update action
- **Updatable fields**: number, note, responsible_user_id
- **Validation**: Duplicate number uniqueness check (excluding current settlement)
- **DB queries**: UPDATE with COALESCE for partial updates, scoped by account_id
- **Error handling**: Conflict error for duplicate number
- **Side effects**: None in update (consistent across both implementations)
- **Response shape**: Full Settlement object with responsible_user, allocations, timestamps
- **Idempotency**: PATCH uses idempotency keys via recovery points

## Issues found and fixed

### Missing `responsible_user_id` update support

The Dashboard update method persists `responsibleUser` via `PrismaUtils.connect(data.responsibleUser)`, but the Go implementation only supported updating `number` and `note`.

**Fixed by adding `responsible_user_id` across the entire update flow:**
- `proto/core.proto` — added `optional string responsible_user_id = 4` to `UpdateSettlementRequest`
- `services/core-service/internal/domain/settlement_models.go` — added `ResponsibleUserID *string` to `UpdateSettlementParams`
- `services/api-gateway/endpoints/settlements/endpoint_update_settlement.go` — added `ResponsibleUserID *string` to request struct
- `services/api-gateway/endpoints/settlements/service.go` — pass `ResponsibleUserID` to proto request
- `services/core-service/internal/infrastructure/grpc/grpc_settlement_handler.go` — map proto field to domain params
- `services/core-service/internal/infrastructure/queries/settlement.sql` — added `responsible_user_id = COALESCE(...)` to UPDATE query
- `services/core-service/internal/infrastructure/repository/settlement_repository.go` — handle `ResponsibleUserID` in Update method
- Regenerated proto and sqlc

## Notes

- The Dashboard has a quirk where the `number` field is used in the WHERE clause of the Prisma update (not in the data), meaning it doesn't actually update the number. The Go implementation correctly supports number updates via COALESCE, which is the intended behavior.
- The Dashboard's schema allows `allocations` in the partial update, but the repository never actually updates them — only `note` and `responsibleUser` are persisted. The Go implementation is consistent with this actual behavior.
- No side effects (like `updateSettlementPaymentStatus`) are triggered on update in either implementation — that only happens on create.
