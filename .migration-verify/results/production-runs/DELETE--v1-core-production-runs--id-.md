# DELETE /v1/core/production-runs/{id} — Migration Verification

## Result: PARITY CONFIRMED

No issues found. The Go implementation correctly mirrors the Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **Permission: internal actor** | `checkIsInternalActor()` | `CheckIsInternalActor()` | ✅ |
| **Permission: domain/action** | `productionRuns / delete` | `PermissionDomainProductionRuns / ActionDelete` | ✅ |
| **Existence check** | Fetches run, 404 if not found | `repo.Get()`, 404 if not found | ✅ |
| **Delete batches** | `db.batch.deleteMany({ where: { productionRunID } })` | `DeleteBatchesByProductionRunID` SQL | ✅ |
| **Find linked orders** | `db.order.findMany({ where: { productionRunID } })` | `FindSalesOrderIDsByProductionRunID` SQL | ✅ |
| **Delete production run** | `db.productionRun.delete()` | `DeleteProductionRunByID` SQL | ✅ |
| **Unlink orders** | `db.order.updateMany({ data: { productionRunID: null } })` | `UnlinkSalesOrdersFromProductionRun` SQL (sets `production_run_id = NULL`) | ✅ |
| **Release reserved inventory** | `inventoryIssueRepo.releaseMaterialsForProductionRun()` — deletes inventory issues where `statusCode = 'reserved'` | `DeleteReservedInventoryIssuesByOrderID` — deletes from `inventory_issue` where `status_code = 'reserved'` | ✅ |
| **Atomicity** | No transaction wrapper | All operations in a single DB transaction | ✅ (Go is stricter) |

## Response Shape

- **Dashboard**: Returns the deleted production run object (HTTP 200)
- **Go**: Returns `EmptyResource` (HTTP 200)

This is an intentional Go codebase convention — most DELETE endpoints return `EmptyResource`. Not a parity gap.

## Notes

- The Go implementation wraps all cascading operations in a transaction, which is an improvement over the Dashboard where operations could partially fail.
- The order of operations matches: delete batches → find orders → delete run → unlink orders → delete reserved inventory issues.
- `releaseMaterialsForProductionRun` in the Dashboard is confirmed to be a simple `deleteMany` on inventory issues with `statusCode = 'reserved'`, exactly matching the Go SQL query.
