-- name: ListMachineItemsByMachineIDs :many
SELECT
    mi.machine_id,
    mi.item_id,
    i.sku
FROM machine_item mi
JOIN item i ON i.id = mi.item_id AND i.account_id = mi.account_id
WHERE mi.account_id = sqlc.arg('account_id')
AND mi.machine_id IN (sqlc.slice('machine_ids'))
AND i.deleted_at IS NULL
ORDER BY mi.item_id, mi.machine_id;

-- name: CountMachineItemsForMachines :one
SELECT COUNT(*)
FROM machine_item mi
JOIN item i ON i.id = mi.item_id AND i.account_id = mi.account_id
WHERE mi.account_id = sqlc.arg('account_id')
AND mi.machine_id IN (sqlc.slice('machine_ids'))
AND i.deleted_at IS NULL;

-- name: DeleteMachineItems :exec
DELETE FROM machine_item
WHERE account_id = sqlc.arg('account_id')
AND machine_id = sqlc.arg('machine_id');

-- name: InsertMachineItem :exec
INSERT INTO machine_item (account_id, machine_id, item_id, created_at, updated_at)
VALUES (
    sqlc.arg('account_id'),
    sqlc.arg('machine_id'),
    sqlc.arg('item_id'),
    CURRENT_TIMESTAMP(3),
    CURRENT_TIMESTAMP(3)
);

-- name: SuggestMachineItemsFromScans :many
SELECT
    i.id AS item_id,
    i.sku,
    i.description,
    MAX(b.scanned_at) AS last_scanned_at,
    COUNT(*) AS batch_count
FROM batch b
JOIN `_batches_machines` bm ON bm.A = b.id
JOIN item i ON i.id = b.item_id AND i.account_id = b.account_id
WHERE b.account_id = sqlc.arg('account_id')
AND bm.B = sqlc.arg('machine_id')
AND b.scanned_at IS NOT NULL
AND b.scanned_at >= sqlc.arg('window_start')
AND b.scanned_at <= sqlc.arg('window_end')
AND i.deleted_at IS NULL
GROUP BY i.id, i.sku, i.description
ORDER BY last_scanned_at DESC, i.sku ASC, i.id ASC;
