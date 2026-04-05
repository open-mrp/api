# POST /v1/core/batches/actions/close — Verification Result

**Status: PARITY CONFIRMED** — No issues found.

## What Was Compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| **Validation** | `batchID` required (zod string) | `batch_id` required (validate tag) | Yes |
| **Actor check** | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| **Permission** | `delete` on `batches` domain | `ActionDelete` on `PermissionDomainBatches` | Yes |
| **Account scoping** | `identity.targetAccountID` in WHERE | `*identity.TargetAccountID` in WHERE | Yes |
| **DB operation** | `UPDATE batch SET closedAt = new Date() WHERE id = ? AND accountID = ?` | `UPDATE batch SET closed_at = NOW(3), updated_at = NOW(3) WHERE id = ? AND account_id = ?` | Yes (Go also updates `updated_at`; Prisma likely does this via `@updatedAt`) |
| **Response status** | 200 OK | 200 OK | Yes |
| **Response shape** | `BaseBatch` with inline machines/lots | `Batch` resource with expandable sub-resources | Yes (follows Go API conventions) |
| **Side effects** | None | None | Yes |
| **Idempotency** | None (naturally idempotent) | gRPC tracking only, no keys in service (naturally idempotent) | Yes |
| **Error handling** | Prisma error on not found | `db.MapSQLError` on not found | Yes |

## Notes

- The Go version additionally updates `updated_at` alongside `closed_at`. This is correct behavior — Prisma's `@updatedAt` directive does the same implicitly.
- Response shape differences (expandable sub-resources vs inline) are by design per Go API conventions, not a parity gap.
- No idempotency key pattern in service is appropriate since `UPDATE SET closed_at = NOW(3)` is naturally idempotent.
- No fixes were needed.
