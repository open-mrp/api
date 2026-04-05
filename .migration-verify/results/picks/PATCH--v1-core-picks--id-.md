# PATCH /v1/core/picks/{id} — Migration Verification

## Result: Issues found and fixed

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Permission: internal actor | `checkIsInternalActor` | `CheckIsInternalActor` | ✅ |
| Permission: domain/action | `picks` / `update` | `PermissionDomainPicks` / `ActionUpdate` | ✅ |
| Target account scoping | `this.identity.targetAccountID` | `identity.TargetAccountID` | ✅ |
| Updatable field: `number` | ✅ updated in repo | ✅ `UpdateNumber` repo method | ✅ |
| Updatable field: `finishedAt` | ✅ updated in repo | ❌ **Missing** — only `number` was wired | ❌ → Fixed |
| Idempotency keys | Not used | Used (PATCH pattern) | ✅ (Go improvement) |
| Transaction | Not used | Used via `withTx` | ✅ (Go improvement) |
| Response shape | Full pick with relations | `PickDetail` with sub-resources | ✅ |
| Error handling | Prisma errors | `db.MapSQLError` | ✅ |
| Side effects | None | None | ✅ |

## Issue found and fixed

**Missing `finishedAt` support in the update endpoint.**

The Dashboard `PickRepo.update()` updates both `number` and `finishedAt`:
```typescript
data: {
    number: data.number,
    finishedAt: data.finishedAt,
}
```

The Go implementation only supported updating `number`. The domain model already had `FinishedAt **time.Time` on `UpdatePickParams` and the SQL queries (`UpdatePickFinishedAt`, `ClearPickFinishedAt`) already existed, but they were not wired through the transport layers.

### Changes made

1. **`proto/core_picking.proto`**: Added `optional string finished_at = 3` to `UpdatePickRequest`
2. **`services/api-gateway/endpoints/picks/endpoint_update_pick.go`**: Added `FinishedAt *string` to `UpdatePickRequest`
3. **`services/api-gateway/endpoints/picks/service.go`**: Pass `FinishedAt` to proto request
4. **`services/core-service/internal/infrastructure/grpc/grpc_picking_handler.go`**: Parse `finished_at` string — empty string means clear (set to null), valid RFC3339 sets the timestamp
5. **`services/core-service/internal/domain/repositories.go`**: Added `UpdateFinishedAt` to `PickRepo` interface
6. **`services/core-service/internal/infrastructure/repository/pick_repository.go`**: Implemented `UpdateFinishedAt` using existing `sqlc.UpdatePickFinishedAt`
7. **`services/core-service/internal/service/pick_svc.go`**: Added `FinishedAt` handling in the update transaction — calls `ClearFinishedAt` when set to null, `UpdateFinishedAt` when set to a timestamp
8. Regenerated proto bindings and mocks

## Remaining concerns

- Pre-existing compilation errors in `pick_repository.go` (lines 455-463) and `pick_line_repository.go` (line 160) related to `GetPickShipmentNumbers` and `HasUnpackedPickLineForOrderLine` — these are unrelated to this endpoint and exist on the branch already.
