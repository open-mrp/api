-- Queries backing the undo of a batch scan: the guard the delete runs synchronously, and the
-- ledger reversal the undo_batch_scan consumer runs behind it.
--
-- Everything a scan wrote carries the batch on its `batch_id`, which is what makes the reversal
-- addressable after the batch row itself is gone. There are no foreign keys on these columns, so a
-- deleted batch leaves its tag behind rather than nulling it.

-- CountDownstreamBatches counts the batches that consume the given one. Per the Prisma orientation
-- of _batch_flow, row (A, B) means A consumes B, so the downstream batches are the A side.
-- name: CountDownstreamBatches :one
SELECT COUNT(*) FROM _batch_flow WHERE B = sqlc.arg('batch_id');

-- CountAllocatedReceiptsForBatch counts how many of the batch's produced receipts have already been
-- drawn against. A non-zero count means the output left the building and the scan can no longer be
-- undone.
-- name: CountAllocatedReceiptsForBatch :one
SELECT COUNT(*)
FROM inventory_receipt ir
JOIN inventory_allocation ia ON ia.inventory_receipt_id = ir.id
WHERE ir.batch_id = sqlc.arg('batch_id')
AND (ir.owner_account_id = sqlc.arg('account_id') OR ir.holder_account_id = sqlc.arg('account_id'));

-- name: FindReceiptsForBatchReversal :many
SELECT
    ir.id,
    ir.item_id,
    ir.quantity_id,
    ir.unit_cost_id,
    q.value AS quantity_value,
    q.unit_id,
    (SELECT COUNT(*) FROM inventory_allocation ia WHERE ia.inventory_receipt_id = ir.id) AS allocation_count
FROM inventory_receipt ir
JOIN quantity q ON q.id = ir.quantity_id
WHERE ir.batch_id = sqlc.arg('batch_id')
AND (ir.owner_account_id = sqlc.arg('account_id') OR ir.holder_account_id = sqlc.arg('account_id'));

-- name: FindIssuesForBatchReversal :many
SELECT
    ii.id,
    ii.item_id,
    ii.order_id,
    ii.quantity_id,
    q.value AS quantity_value,
    q.unit_id
FROM inventory_issue ii
JOIN quantity q ON q.id = ii.quantity_id
WHERE ii.batch_id = sqlc.arg('batch_id')
AND ii.account_id = sqlc.arg('account_id');

-- name: FindAllocationsByIssueIDs :many
SELECT id, inventory_receipt_id, inventory_issue_id, quantity_id, unit_cost_id, total_cost_id
FROM inventory_allocation
WHERE inventory_issue_id IN (sqlc.slice('issue_ids'));

-- name: DeleteAllocationsByIDs :exec
DELETE FROM inventory_allocation WHERE id IN (sqlc.slice('ids'));

-- FreeReleasedReceipts returns receipts to `available` once the allocations that closed them out are
-- gone. Run after the allocations are deleted: the join sees only the surviving ones, so a receipt
-- another issue still fills stays as it was.
-- name: FreeReleasedReceipts :exec
UPDATE inventory_receipt ir
JOIN quantity q ON q.id = ir.quantity_id
LEFT JOIN (
    SELECT ia.inventory_receipt_id, SUM(CAST(aq.value AS DECIMAL(65,30))) AS allocated
    FROM inventory_allocation ia
    JOIN quantity aq ON aq.id = ia.quantity_id
    GROUP BY ia.inventory_receipt_id
) alloc ON alloc.inventory_receipt_id = ir.id
SET ir.status_code = 'available', ir.updated_at = NOW(3)
WHERE ir.id IN (sqlc.slice('ids'))
AND ir.status_code <> 'available'
AND COALESCE(alloc.allocated, 0) < CAST(q.value AS DECIMAL(65,30));

-- RestoreIssuesToReserved hands an order-linked issue back to the reservation it came out of. The
-- batch tag goes with it: the row is no longer anything the scan owns.
-- name: RestoreIssuesToReserved :exec
UPDATE inventory_issue
SET status_code = 'reserved', issued_at = NULL, batch_id = NULL, updated_at = NOW(3)
WHERE id IN (sqlc.slice('ids'));

-- name: DeleteInventoryIssuesByIDs :exec
DELETE FROM inventory_issue WHERE id IN (sqlc.slice('ids'));

-- name: DeleteInventoryReceiptsByIDs :exec
DELETE FROM inventory_receipt WHERE id IN (sqlc.slice('ids'));

-- name: DeleteQuantitiesByIDs :exec
DELETE FROM quantity WHERE id IN (sqlc.slice('ids'));

-- name: DeleteRatesByIDs :exec
DELETE FROM rate WHERE id IN (sqlc.slice('ids'));

-- UnscanBatch returns a batch to the state it was in before an init station stamped it, leaving the
-- row in place so the production run that created it still holds that unit of work.
-- name: UnscanBatch :exec
UPDATE batch
SET scanned_at = NULL, closed_at = NULL, production_step_id = NULL, scanning_station_id = NULL, updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: ReopenBatch :exec
UPDATE batch SET closed_at = NULL, updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- ReopenProductionRun undoes a completion after one of the run's batches goes back to unscanned. A
-- run with no scanned batches left never started, so started_at is cleared too — otherwise the run
-- would report as in progress with nothing having happened.
-- name: ReopenProductionRun :exec
UPDATE production_run pr
SET pr.completed_at = NULL,
    pr.started_at = CASE
        WHEN (SELECT COUNT(*) FROM batch b WHERE b.production_run_id = pr.id AND b.scanned_at IS NOT NULL) = 0
        THEN NULL
        ELSE pr.started_at
    END,
    pr.updated_at = NOW(3)
WHERE pr.id = sqlc.arg('id') AND pr.account_id = sqlc.arg('account_id');
