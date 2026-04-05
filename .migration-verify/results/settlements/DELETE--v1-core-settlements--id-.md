# DELETE /v1/core/settlements/{id} — Verification Result

## Status: PARITY CONFIRMED

No issues found. The Go implementation is functionally equivalent to the Dashboard and in some ways more robust.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission checks | `checkIsInternalActor` + `settlements:delete` | Same | Yes |
| Find settlement first | By ID + accountID, 404 if not found | Same | Yes |
| Delete orphaned adjustment txns | Prisma `every` filter (vacuous truth after allocation delete) | SQL `NOT EXISTS` subquery (before allocation delete) | Equivalent |
| Delete allocations | `transactionAllocation.deleteMany` | Deletes quantities first, then allocations | Yes (Go more thorough) |
| Delete settlement | Hard delete by ID | Hard delete by ID + account_id | Yes |
| Reset transaction flags | `isFullyAllocated = false` | Same | Yes |
| Reset invoice flags | `isOverPaid = false, isPaidInFull = false` | Same | Yes |
| Response | Returns original settlement object | Same | Yes |
| Side effects | None (no messages/webhooks) | None | Yes |
| HTTP status | 200 OK | 200 OK | Yes |

## Notes

- **Transaction safety**: Go wraps all mutations in a single DB transaction; Dashboard executes Prisma calls sequentially without an explicit transaction. Go is safer.
- **Quantity cleanup**: Go explicitly deletes `quantity` records tied to allocations before deleting allocations. Dashboard does not (may rely on DB cascades or leaves orphans).
- **Orphan detection ordering**: Dashboard deletes allocations first, then uses Prisma's `every` filter (which is vacuously true for empty sets) to find orphaned adjustment transactions. Go deletes orphaned transactions first while allocations still exist, using a `NOT EXISTS` subquery. Both produce the same result for normal cases.
- **Account ID scoping**: Dashboard adds `accountID` to the `UPDATE` queries for transactions and invoices. Go omits this since the IDs are already derived from account-scoped queries. No functional difference.
- **No idempotency key needed**: Correct — DELETE endpoints are idempotent by nature.
