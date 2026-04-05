-- name: FetchCurrentInventoryForItem :one
SELECT
    CAST(COALESCE(
        (SELECT SUM(q.value) FROM inventory_receipt ir
         JOIN quantity q ON ir.quantity_id = q.id
         WHERE ir.item_id = sqlc.arg('item_id')
         AND ir.owner_account_id = sqlc.arg('owner_account_id')
         AND ir.status_code = 'available'), 0
    ) - COALESCE(
        (SELECT SUM(q.value) FROM inventory_issue ii
         JOIN quantity q ON ii.quantity_id = q.id
         WHERE ii.item_id = sqlc.arg('item_id')
         AND ii.account_id = sqlc.arg('owner_account_id')
         AND ii.status_code = 'committed'), 0
    ) AS SIGNED) AS available_to_promise,
    COALESCE(
        (SELECT qu.abbreviation FROM item i
         JOIN item_category ic ON i.item_category_id = ic.id
         JOIN unit_group ug ON ic.unit_group_id = ug.id
         JOIN unit qu ON ug.base_unit_id = qu.id
         WHERE i.id = sqlc.arg('item_id') LIMIT 1), ''
    ) AS unit_abbreviation;

-- name: FetchOnHandInventoryBulk :many
SELECT
    i.id AS item_id,
    CAST(COALESCE(
        (SELECT SUM(CAST(q.value AS DECIMAL(65,30)))
         FROM inventory_receipt ir
         JOIN quantity q ON ir.quantity_id = q.id
         LEFT JOIN inventory_allocation ia ON ia.inventory_receipt_id = ir.id
         WHERE ir.item_id = i.id
         AND (ir.owner_account_id = sqlc.arg('owner_account_id') OR ir.holder_account_id = sqlc.arg('owner_account_id'))
         AND ir.status_code = 'available')
    , 0) - COALESCE(
        (SELECT SUM(CAST(aq.value AS DECIMAL(65,30)))
         FROM inventory_receipt ir2
         JOIN inventory_allocation ia2 ON ia2.inventory_receipt_id = ir2.id
         JOIN quantity aq ON aq.id = ia2.quantity_id
         WHERE ir2.item_id = i.id
         AND (ir2.owner_account_id = sqlc.arg('owner_account_id') OR ir2.holder_account_id = sqlc.arg('owner_account_id'))
         AND ir2.status_code = 'available')
    , 0) AS SIGNED) AS on_hand_quantity,
    bu.id AS unit_id,
    bu.abbreviation AS unit_abbreviation,
    ug.unit_type_code AS unit_type
FROM item i
JOIN item_category ic ON i.item_category_id = ic.id
JOIN unit_group ug ON ic.unit_group_id = ug.id
JOIN unit bu ON ug.base_unit_id = bu.id
WHERE i.id IN (sqlc.slice('item_ids'))
  AND i.account_id = sqlc.arg('owner_account_id')
  AND i.deleted_at IS NULL;
