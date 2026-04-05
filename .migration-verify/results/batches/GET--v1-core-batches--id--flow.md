# GET /v1/core/batches/{id}/flow — Migration Verification

**Status: Parity confirmed — no fixes needed**

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Permission: internal actor check | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| Permission: domain + action | `batches` / `read` | `PermissionDomainBatches` / `ActionRead` | Yes |
| Account scoping | `targetAccountID` required | `CheckTargetAccountSet()` | Yes |
| BFS algorithm | Queue-based BFS over `in`/`out` relations | Queue-based BFS over `_batch_flow` table | Yes |
| BFS bidirectional traversal | Traverses both `in` and `out` edges | Traverses both outgoing (A→B) and incoming (B→A) | Yes |
| Cycle prevention | `visited` Set | `visited` map | Yes |
| Batch detail fetch | Bulk fetch all discovered IDs via Prisma | Individual fetch per ID via `GetBatch` query | Yes (functional equivalent) |
| Account filtering on batch fetch | `accountID` filter on Prisma query | `account_id` param on `GetBatch` SQL | Yes |
| Machines included | Via `BatchAdapter.select` includes machines | Separate `GetBatchMachines` query | Yes |
| Null safety on edge arrays | Prisma returns empty arrays by default | Explicit nil→empty slice conversion | Yes |

## Minor differences (acceptable, no fix needed)

1. **No upfront existence check in Go**: The Dashboard calls `checkExistence({ id: batchID, accountID })` before starting BFS. The Go code omits this — if the batch doesn't exist, it will still fail when `GetBatch` returns `sql.ErrNoRows` for the initial batch ID (which is always in the `visited` set). End result is the same (not-found error), just at a slightly different point in execution.

2. **Response shape**: The Dashboard returns a flat `Batch[]` where each batch object has `inputBatchIDs` and `outputBatchIDs` inlined. The Go API returns `List[BatchFlowNode]` where each node wraps a `Batch` object alongside `input_batch_ids` and `output_batch_ids`. This is consistent with Go API conventions (sub-resources, List wrapper) and is expected for the new `v1/core/` API surface.

3. **Dashboard batch includes `lots` and `departmentName`**: These fields exist in the Dashboard Batch model but are not in the Go Batch resource. This is a broader Batch resource concern (not specific to the flow endpoint) and is tracked separately. The Go Batch resource includes all other fields: item, quantity, seconds, waste, scanning_station, production_step, production_run, machines, closed_at, scanned_at, created_at, updated_at.

4. **BFS account filtering**: The Dashboard filters BFS traversal nodes by `accountID` (Prisma `where: { id, accountID }`). The Go BFS queries (`GetBatchFlowOutgoing`/`GetBatchFlowIncoming`) only filter by `batch_id` on the `_batch_flow` join table. Since batch flow relationships are created within account context and `GetBatch` validates `account_id`, this is functionally equivalent.

## Conclusion

The core business logic — BFS graph traversal of batch flow relationships with proper permission checks and account scoping — has full parity between the Dashboard and Go implementations.
