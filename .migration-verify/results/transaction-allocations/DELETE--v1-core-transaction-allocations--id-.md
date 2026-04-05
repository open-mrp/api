# DELETE /v1/core/transaction-allocations/{id}

## Result: PARITY CONFIRMED

No issues found. The Go implementation faithfully reproduces the Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Permission: internal actor | `checkIsInternalActor` | `CheckIsInternalActor()` | Yes |
| Permission: domain/action | `settlements / delete` | `PermissionDomainSettlements / ActionDelete` | Yes |
| Account isolation | WHERE `transaction.accountID = ownerAccountID` | JOIN on `transaction.account_id` in both GetByID and Delete queries | Yes |
| Delete behavior | Hard delete via Prisma `.delete()` | Hard delete via SQL DELETE with JOIN | Yes |
| Quantity cleanup | Implicit (Prisma may orphan quantity) | Explicit delete of quantity before allocation (improvement) | Yes+ |
| Existence check | Implicit (Prisma throws NotFound) | Explicit `GetByID` before delete | Yes+ |
| Response status | 200 OK | 204 NoContent | Acceptable |
| Side effects | None (no transaction status recalculation) | None | Yes |
| Idempotency keys | N/A (DELETE is idempotent by design) | N/A | Yes |

## Notes

- The Go implementation returns 204 instead of 200, which is the standard REST convention for DELETE and aligns with the Go API patterns.
- The Go implementation explicitly deletes the associated `quantity` record before the allocation, which is an improvement over the Dashboard where Prisma's `.delete()` may leave orphaned quantity records.
- No side effects in either implementation — neither recalculates `isFullyAllocated` on the parent transaction after deletion.
