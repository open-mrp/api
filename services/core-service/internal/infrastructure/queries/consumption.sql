-- name: GetConsumption :one
SELECT
    c.id,
    c.instructions,
    c.created_at,
    c.updated_at,
    ci.id AS item_id,
    ci.sku AS item_sku,
    ci.description AS item_description,
    ci.item_type_code,
    cq.id AS quantity_id,
    cq.value AS quantity_value,
    cu.id AS quantity_unit_id,
    cu.abbreviation AS quantity_unit_abbreviation,
    cu.unit_dimension_code AS quantity_unit_type,
    wq.id AS waste_quantity_id,
    wq.value AS waste_quantity_value,
    wu.id AS waste_unit_id,
    wu.abbreviation AS waste_unit_abbreviation,
    wu.unit_dimension_code AS waste_unit_type
FROM consumption c
JOIN item ci ON c.item_id = ci.id
JOIN quantity cq ON c.quantity_id = cq.id
JOIN unit cu ON cq.unit_id = cu.id
JOIN quantity wq ON c.waste_quantity_id = wq.id
JOIN unit wu ON wq.unit_id = wu.id
JOIN production_step ps ON c.production_step_id = ps.id
WHERE c.id = sqlc.arg('consumption_id')
AND ps.id = sqlc.arg('step_id')
AND ps.account_id = sqlc.arg('account_id');

-- name: InsertConsumption :exec
INSERT INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, instructions, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('item_id'), sqlc.arg('quantity_id'), sqlc.arg('waste_quantity_id'), sqlc.narg('production_step_id'), sqlc.narg('instructions'), NOW(3), NOW(3));

-- name: InsertConsumptionQuantity :exec
INSERT INTO quantity (id, value, unit_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('value'), sqlc.arg('unit_id'), NOW(3), NOW(3));

-- name: UpdateConsumptionQuantity :exec
UPDATE quantity SET value = sqlc.arg('value'), unit_id = sqlc.arg('unit_id'), updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: DeleteConsumptionQuantity :exec
DELETE FROM quantity WHERE id = sqlc.arg('id');

-- name: UpdateConsumptionItem :exec
UPDATE consumption SET item_id = sqlc.arg('item_id'), instructions = sqlc.narg('instructions'), updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: DeleteConsumptionRow :exec
DELETE FROM consumption WHERE id = sqlc.arg('id');

-- name: IsConsumptionInAccount :one
SELECT COUNT(*) FROM consumption c
JOIN production_step ps ON c.production_step_id = ps.id
WHERE c.id = sqlc.arg('consumption_id') AND ps.account_id = sqlc.arg('account_id');

-- name: GetConsumptionQuantityIDs :one
SELECT quantity_id, waste_quantity_id FROM consumption WHERE id = sqlc.arg('id');

-- name: GetConsumptionItemID :one
SELECT item_id FROM consumption WHERE id = sqlc.arg('id');

-- name: GetConsumptionInstructions :one
SELECT instructions FROM consumption WHERE id = sqlc.arg('id');

-- name: IsItemInAccount :one
SELECT COUNT(*) FROM item WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');
