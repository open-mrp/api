-- name: InsertInventoryReceipt :exec
INSERT INTO inventory_receipt (
    id, owner_account_id, holder_account_id, item_id,
    quantity_id, unit_cost_id, status_code,
    batch_id, storage_location_id, lot_id,
    received_at, created_at, updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('owner_account_id'),
    sqlc.arg('holder_account_id'),
    sqlc.arg('item_id'),
    sqlc.arg('quantity_id'),
    sqlc.arg('unit_cost_id'),
    'available',
    sqlc.narg('batch_id'),
    sqlc.narg('storage_location_id'),
    sqlc.narg('lot_id'),
    NOW(3),
    NOW(3),
    NOW(3)
);

-- name: InsertInventoryIssue :exec
INSERT INTO inventory_issue (
    id, account_id, item_id,
    quantity_id, status_code,
    batch_id, storage_location_id, lot_id,
    issued_at, created_at, updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_id'),
    sqlc.arg('item_id'),
    sqlc.arg('quantity_id'),
    'open',
    sqlc.narg('batch_id'),
    sqlc.narg('storage_location_id'),
    sqlc.narg('lot_id'),
    NOW(3),
    NOW(3),
    NOW(3)
);

-- name: InsertQuantityForInventory :exec
INSERT INTO quantity (id, value, unit_id)
VALUES (sqlc.arg('id'), sqlc.arg('value'), sqlc.arg('unit_id'));

-- name: InsertRateForInventory :exec
INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id)
VALUES (sqlc.arg('id'), sqlc.arg('value'), sqlc.arg('numerator_unit_id'), sqlc.arg('denominator_unit_id'));

-- name: GetItemUnitCost :one
SELECT r.id, r.value, r.numerator_unit_id, r.denominator_unit_id
FROM item i
JOIN rate r ON r.id = i.unit_cost_id
WHERE i.id = sqlc.arg('item_id')
  AND i.account_id = sqlc.arg('account_id')
  AND i.deleted_at IS NULL;

-- name: FindBatchProductionRunIDAncestry :many
SELECT b.id, b.production_run_id,
       bf.B as parent_id
FROM batch b
LEFT JOIN _batch_flow bf ON bf.A = b.id
WHERE b.id IN (sqlc.slice('batch_ids'));

-- name: InsertInventoryLog :exec
INSERT INTO inventory_log (id, item_id, quantity_id, account_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('item_id'), sqlc.arg('quantity_id'), sqlc.arg('account_id'), NOW(3), NOW(3));

-- name: InsertInventoryChangeLog :exec
INSERT INTO inventory_change_log (
    id, item_id, quantity_id, action_type_code,
    scanning_station_id, account_id, inventory_log_id,
    responsible_user_id, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('item_id'), sqlc.arg('quantity_id'), sqlc.arg('action_type_code'),
    sqlc.narg('scanning_station_id'), sqlc.arg('account_id'), sqlc.narg('inventory_log_id'),
    sqlc.narg('responsible_user_id'), NOW(3), NOW(3)
);

-- name: FetchPhysicalInventoryForItem :one
SELECT CAST(
    COALESCE(
        (SELECT SUM(CAST(q.value AS DECIMAL(65,30)))
         FROM inventory_receipt ir
         JOIN quantity q ON ir.quantity_id = q.id
         WHERE ir.item_id = sqlc.arg('item_id')
         AND (ir.owner_account_id = sqlc.arg('owner_account_id') OR ir.holder_account_id = sqlc.arg('owner_account_id'))
         AND ir.status_code = 'available'), 0
    ) - COALESCE(
        (SELECT SUM(CAST(q.value AS DECIMAL(65,30)))
         FROM inventory_issue ii
         JOIN quantity q ON ii.quantity_id = q.id
         WHERE ii.item_id = sqlc.arg('item_id')
         AND ii.account_id = sqlc.arg('owner_account_id')
         AND ii.status_code = 'open'), 0
    )
AS DECIMAL(65,30)) AS physical_inventory;

-- name: GetBatchSecondsAndWaste :one
SELECT
    sq.value AS seconds_value,
    sq.unit_id AS seconds_unit_id,
    wq.value AS waste_value,
    wq.unit_id AS waste_unit_id
FROM batch b
LEFT JOIN quantity sq ON sq.id = b.seconds_quantity_id
LEFT JOIN quantity wq ON wq.id = b.waste_quantity_id
WHERE b.id = sqlc.arg('batch_id');
