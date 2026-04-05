-- name: GetProductionByID :one
SELECT
    p.id,
    p.item_id,
    pi.sku AS item_sku,
    pi.description AS item_description,
    pi.item_type_code,
    pq.id AS quantity_id,
    pq.value AS quantity_value,
    pu.id AS quantity_unit_id,
    pu.abbreviation AS quantity_unit_abbreviation,
    pu.unit_dimension_code AS quantity_unit_type,
    p.production_step_id,
    p.created_at,
    p.updated_at
FROM production p
JOIN item pi ON p.item_id = pi.id
JOIN quantity pq ON p.quantity_id = pq.id
JOIN unit pu ON pq.unit_id = pu.id
JOIN production_step ps ON p.production_step_id = ps.id
WHERE p.id = sqlc.arg('id')
AND p.production_step_id = sqlc.arg('production_step_id')
AND ps.account_id = sqlc.arg('account_id');

-- name: UpdateProductionItem :exec
UPDATE production SET
    item_id = sqlc.arg('item_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: UpdateProductionQuantity :exec
UPDATE quantity SET
    value = sqlc.arg('value'),
    unit_id = sqlc.arg('unit_id')
WHERE quantity.id = (SELECT p.quantity_id FROM production p WHERE p.id = sqlc.arg('production_id'));

-- name: GetProductionQuantityID :one
SELECT quantity_id FROM production WHERE id = sqlc.arg('production_id');
