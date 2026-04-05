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

-- name: UpdateInventoryIssueStatusToOpen :exec
UPDATE inventory_issue
SET status_code = 'open', issued_at = NOW(3), updated_at = NOW(3)
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
ORDER BY ir.received_at ASC;

-- name: GetAllocationSumForReceipt :one
SELECT COALESCE(SUM(CAST(q.value AS DECIMAL(65,30))), 0) AS total_allocated
FROM inventory_allocation ia
JOIN quantity q ON q.id = ia.quantity_id
WHERE ia.inventory_receipt_id = sqlc.arg('receipt_id');

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
