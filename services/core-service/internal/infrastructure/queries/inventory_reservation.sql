-- name: FindReservedIssuesByOrderItem :many
SELECT
    ii.id,
    q.id AS quantity_id,
    q.value AS quantity_value,
    q.unit_id,
    ii.storage_location_id,
    ii.lot_id,
    ii.batch_id
FROM inventory_issue ii
JOIN quantity q ON q.id = ii.quantity_id
WHERE ii.order_id = sqlc.arg('order_id')
AND ii.account_id = sqlc.arg('account_id')
AND ii.item_id = sqlc.arg('item_id')
AND ii.status_code = 'reserved'
ORDER BY ii.created_at ASC;

-- name: DeleteInventoryIssueByID :exec
DELETE FROM inventory_issue WHERE id = sqlc.arg('id');

-- name: DeleteQuantityByID :exec
DELETE FROM quantity WHERE id = sqlc.arg('id');

-- name: UpdateQuantityValue :exec
UPDATE quantity SET value = sqlc.arg('value') WHERE id = sqlc.arg('id');

-- name: FindReservedIssuesWithAllocationSums :many
SELECT
    ii.id,
    q.id AS quantity_id,
    q.value AS quantity_value,
    q.unit_id,
    ii.storage_location_id,
    ii.lot_id,
    ii.batch_id,
    COALESCE(SUM(CAST(aq.value AS DECIMAL(65,30))), 0) AS allocated_sum
FROM inventory_issue ii
JOIN quantity q ON q.id = ii.quantity_id
LEFT JOIN inventory_allocation ia ON ia.inventory_issue_id = ii.id
LEFT JOIN quantity aq ON aq.id = ia.quantity_id
WHERE ii.order_id = sqlc.arg('order_id')
AND ii.account_id = sqlc.arg('account_id')
AND ii.item_id = sqlc.arg('item_id')
AND ii.status_code = 'reserved'
GROUP BY ii.id, q.id, q.value, q.unit_id, ii.storage_location_id, ii.lot_id, ii.batch_id
ORDER BY ii.created_at ASC;

-- UpdateInventoryIssueStatusToOpen consumes a whole reservation in place. The batch that consumed it
-- is stamped on the row so deleting that batch can find the reservation and hand it back; COALESCE
-- keeps whatever tag the row already carried when no batch is supplied.
-- name: UpdateInventoryIssueStatusToOpen :exec
UPDATE inventory_issue
SET status_code = 'open',
    issued_at = NOW(3),
    batch_id = COALESCE(sqlc.narg('batch_id'), batch_id),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: InsertInventoryIssueForReservation :exec
INSERT INTO inventory_issue (
    id, account_id, item_id, quantity_id,
    status_code, order_id, batch_id,
    storage_location_id, lot_id,
    issued_at, created_at, updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_id'),
    sqlc.arg('item_id'),
    sqlc.arg('quantity_id'),
    sqlc.arg('status_code'),
    sqlc.narg('order_id'),
    sqlc.narg('batch_id'),
    sqlc.narg('storage_location_id'),
    sqlc.narg('lot_id'),
    NOW(3),
    NOW(3),
    NOW(3)
);

-- FindReceiptsForAllocation lists the receipts an issue may draw from, oldest first.
--
-- FOR UPDATE holds the candidates until the allocating transaction commits. The caller decides how
-- much a receipt has left by reading its allocations and subtracting, so without the lock two
-- consumptions of the same item both saw the same receipt as free and each allocated the whole of
-- it — consuming stock that was never used. Only `available` receipts are candidates, so the lock
-- covers the few rows actually in play rather than the item's whole receipt history.
-- name: FindReceiptsForAllocation :many
SELECT
    ir.id,
    q.id AS quantity_id,
    q.value AS quantity_value,
    q.unit_id
FROM inventory_receipt ir
JOIN quantity q ON q.id = ir.quantity_id
WHERE ir.owner_account_id = sqlc.arg('account_id')
AND ir.item_id = sqlc.arg('item_id')
AND ir.status_code = 'available'
ORDER BY ir.received_at ASC
FOR UPDATE;

-- name: GetAllocationSumForReceipt :one
SELECT COALESCE(SUM(CAST(q.value AS DECIMAL(65,30))), 0) AS total_allocated
FROM inventory_allocation ia
JOIN quantity q ON q.id = ia.quantity_id
WHERE ia.inventory_receipt_id = sqlc.arg('receipt_id');

-- GetAllocationSumsForReceipts answers for a whole candidate set at once. Allocation walks receipts
-- oldest first and needs each one's drawn-down total; asking per receipt put a round trip inside that
-- loop, so an item with a long tail of open receipts cost a query apiece to find most of them full.
-- Receipts with no allocations are absent rather than zero — the caller treats a missing row as zero.
-- name: GetAllocationSumsForReceipts :many
SELECT ia.inventory_receipt_id, COALESCE(SUM(CAST(q.value AS DECIMAL(65,30))), 0) AS total_allocated
FROM inventory_allocation ia
JOIN quantity q ON q.id = ia.quantity_id
WHERE ia.inventory_receipt_id IN (sqlc.slice('receipt_ids'))
GROUP BY ia.inventory_receipt_id;

-- name: InsertInventoryAllocation :exec
INSERT INTO inventory_allocation (
    id, inventory_receipt_id, inventory_issue_id,
    quantity_id, unit_cost_id, total_cost_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('inventory_receipt_id'),
    sqlc.arg('inventory_issue_id'),
    sqlc.arg('quantity_id'),
    sqlc.arg('unit_cost_id'),
    sqlc.arg('total_cost_id'),
    NOW(3),
    NOW(3)
);
