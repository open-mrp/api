-- name: FindUnitGroupByItem :one
SELECT
    ug.id,
    u.id AS base_unit_id,
    u.abbreviation AS base_unit_abbreviation,
    u.unit_dimension_code AS base_unit_type
FROM item i
JOIN item_category ic ON i.item_category_id = ic.id
JOIN unit_group ug ON ic.unit_group_id = ug.id
JOIN unit u ON ug.base_unit_id = u.id
WHERE i.id = sqlc.arg('item_id')
AND (i.account_id = sqlc.arg('account_id') OR i.account_id IS NULL);
