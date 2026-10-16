-- name: GetItemUnitSources :many
-- An item's stored units, and what its units derive from while they are not stored yet: its category's unit group,
-- its product line's unit group and its selling price.
SELECT
    i.id,
    i.account_id,
    i.item_type_code,
    i.stocking_unit_id,
    i.default_order_unit_id,
    ic.unit_group_id AS category_unit_group_id,
    cug.base_unit_id AS category_base_unit_id,
    pl.unit_group_id AS product_line_unit_group_id,
    plug.base_unit_id AS product_line_base_unit_id,
    rv.value AS unit_value,
    rv.denominator_unit_id AS unit_value_per_unit_id
FROM item i
JOIN item_category ic ON ic.id = i.item_category_id
JOIN unit_group cug ON cug.id = ic.unit_group_id
JOIN rate rv ON rv.id = i.unit_value_id
LEFT JOIN product p ON p.item_id = i.id
LEFT JOIN product_line pl ON pl.id = p.product_line_id
LEFT JOIN unit_group plug ON plug.id = pl.unit_group_id
WHERE i.id IN (sqlc.slice('item_ids'));

-- name: GetItemOrderUnitsByItemIDs :many
SELECT
    item_id, unit_id, position, portal_visibility, discount_type, discount_percent_off,
    discount_amount_off_amount, discount_amount_off_currency, stocking_quantity
FROM item_order_unit
WHERE item_id IN (sqlc.slice('item_ids'))
ORDER BY item_id, position, unit_id;

-- name: GetUnitGroupMembersForItemUnits :many
SELECT ugu.unit_group_id, ugu.unit_id, ugu.discount_percentage, ugu.discount_fixed, ugu.is_visible
FROM unit_group_unit ugu
WHERE ugu.unit_group_id IN (sqlc.slice('unit_group_ids'))
ORDER BY ugu.unit_group_id, ugu.created_at, ugu.id;

-- name: GetUnitFactorsForItemUnits :many
SELECT u.id, u.ratio_numerator, u.ratio_denominator, u.unit_dimension_code
FROM unit u
WHERE u.id IN (sqlc.slice('unit_ids'));

-- name: SetItemUnitsIfUnstored :execrows
-- Stores derived units only on an item that has none stored, so a backfill never overwrites a writer.
UPDATE item SET
  stocking_unit_id = sqlc.narg('stocking_unit_id'),
  default_order_unit_id = sqlc.arg('default_order_unit_id')
WHERE id = sqlc.arg('id')
AND default_order_unit_id IS NULL;

-- name: SetItemStockingUnit :exec
UPDATE item SET stocking_unit_id = sqlc.arg('stocking_unit_id'), updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: SetItemDefaultOrderUnit :exec
UPDATE item SET default_order_unit_id = sqlc.arg('default_order_unit_id'), updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: DeleteItemOrderUnits :exec
DELETE FROM item_order_unit WHERE item_id = sqlc.arg('item_id');

-- name: DeleteItemOrderUnitsByUnitIDs :exec
DELETE FROM item_order_unit WHERE item_id = sqlc.arg('item_id') AND unit_id IN (sqlc.slice('unit_ids'));

-- name: InsertItemOrderUnit :exec
INSERT INTO item_order_unit (
    item_id, unit_id, account_id, position, portal_visibility, discount_type, discount_percent_off,
    discount_amount_off_amount, discount_amount_off_currency, stocking_quantity, created_at, updated_at
) VALUES (
    sqlc.arg('item_id'), sqlc.arg('unit_id'), sqlc.arg('account_id'), sqlc.arg('position'), sqlc.arg('portal_visibility'),
    sqlc.narg('discount_type'), sqlc.narg('discount_percent_off'), sqlc.narg('discount_amount_off_amount'),
    sqlc.narg('discount_amount_off_currency'), sqlc.narg('stocking_quantity'), NOW(3), NOW(3)
);

-- name: ListUnstoredItemIDsAfter :many
-- The backfill's page: items by id after the cursor.
SELECT id, default_order_unit_id IS NULL AS unstored FROM item
WHERE id > sqlc.arg('cursor')
ORDER BY id
LIMIT ?;
