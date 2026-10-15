-- name: InsertItem :exec
INSERT INTO item (
    id, sku, description, item_type_code, item_category_id, unit_value_id, unit_cost_id, burn_rate_id,
    account_id, is_dirty, is_active, supply_code, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('sku'), sqlc.narg('description'), sqlc.arg('item_type_code'), sqlc.arg('item_category_id'),
    sqlc.arg('unit_value_id'), sqlc.arg('unit_cost_id'), sqlc.arg('burn_rate_id'), sqlc.arg('account_id'), false,
    sqlc.arg('is_active'), sqlc.narg('supply_code'), NOW(3), NOW(3)
);

-- An item made active again is no longer soft-deleted either.
-- name: SetItemFields :exec
UPDATE item SET
  sku = COALESCE(sqlc.narg('sku'), sku),
  description = IF(sqlc.arg('set_description'), sqlc.narg('description'), description),
  item_category_id = COALESCE(sqlc.narg('item_category_id'), item_category_id),
  deleted_at = IF(sqlc.narg('is_active') = true, NULL, deleted_at),
  is_active = COALESCE(sqlc.narg('is_active'), is_active),
  supply_code = IF(sqlc.arg('set_supply'), sqlc.narg('supply_code'), supply_code),
  updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteItemAttributes :exec
DELETE FROM _item_attributes WHERE B = sqlc.arg('item_id');

-- name: GetItemSalesByItemIDs :many
SELECT
    p.id, p.item_id, p.product_line_id, p.is_portal_ready, p.lead_time_id,
    lq.value AS lead_time_value, lq.unit_id AS lead_time_unit_id, lu.unit_dimension_code AS lead_time_unit_type
FROM product p
LEFT JOIN quantity lq ON lq.id = p.lead_time_id
LEFT JOIN unit lu ON lu.id = lq.unit_id
WHERE p.item_id IN (sqlc.slice('item_ids'));

-- name: GetItemMaterialsByItemIDs :many
SELECT
    m.id, m.item_id,
    m.order_point_id, op.value AS order_point_value, op.unit_id AS order_point_unit_id, opu.unit_dimension_code AS order_point_unit_type,
    m.lead_time_id, lt.value AS lead_time_value, lt.unit_id AS lead_time_unit_id, ltu.unit_dimension_code AS lead_time_unit_type
FROM material m
JOIN quantity op ON op.id = m.order_point_id
JOIN unit opu ON opu.id = op.unit_id
JOIN quantity lt ON lt.id = m.lead_time_id
JOIN unit ltu ON ltu.id = lt.unit_id
WHERE m.item_id IN (sqlc.slice('item_ids'));

-- name: InsertItemProduct :exec
INSERT INTO product (id, item_id, product_type_code, product_line_id, is_portal_ready, lead_time_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('item_id'), 'sale', sqlc.narg('product_line_id'), sqlc.arg('is_portal_ready'),
        sqlc.narg('lead_time_id'), NOW(3), NOW(3));

-- name: SetItemProductFields :exec
UPDATE product SET
  product_line_id = IF(sqlc.arg('set_product_line'), sqlc.narg('product_line_id'), product_line_id),
  is_portal_ready = COALESCE(sqlc.narg('is_portal_ready'), is_portal_ready),
  lead_time_id = IF(sqlc.arg('set_lead_time'), sqlc.narg('lead_time_id'), lead_time_id),
  updated_at = NOW(3)
WHERE item_id = sqlc.arg('item_id');

-- name: InsertItemPart :exec
INSERT INTO part (id, item_id, created_at, updated_at) VALUES (sqlc.arg('id'), sqlc.arg('item_id'), NOW(3), NOW(3));

-- name: DeleteItemProduct :exec
DELETE FROM product WHERE item_id = sqlc.arg('item_id');

-- name: DeleteItemMaterial :exec
DELETE FROM material WHERE item_id = sqlc.arg('item_id');

-- name: DeleteItemPart :exec
DELETE FROM part WHERE item_id = sqlc.arg('item_id');

-- An item nothing references has only the levels and changes logged without moving stock, such as the zero opening
-- level logged at its creation, each over its own quantity row.
-- name: ListItemInventoryLogQuantityIDs :many
SELECT il.quantity_id FROM inventory_log il WHERE il.item_id = sqlc.arg('item_id')
UNION ALL
SELECT icl.quantity_id FROM inventory_change_log icl WHERE icl.item_id = sqlc.arg('item_id');

-- name: DeleteItemInventoryChangeLogs :exec
DELETE FROM inventory_change_log WHERE item_id = sqlc.arg('item_id');

-- name: DeleteItemInventoryLogs :exec
DELETE FROM inventory_log WHERE item_id = sqlc.arg('item_id');

-- name: DeleteItemInventoryLock :exec
DELETE FROM inventory_item_lock WHERE item_id = sqlc.arg('item_id');

-- name: DeleteItemMachineItems :exec
DELETE FROM machine_item WHERE account_id = sqlc.arg('account_id') AND item_id = sqlc.arg('item_id');

-- name: DeleteItemPlanningSettings :exec
DELETE FROM production_schedule_item_setting WHERE account_id = sqlc.arg('account_id') AND item_id = sqlc.arg('item_id');

-- name: DeleteItemPlanningPolicies :exec
DELETE FROM production_schedule_item_policy WHERE account_id = sqlc.arg('account_id') AND item_id = sqlc.arg('item_id');

-- name: DeleteItemFinishedPolicies :exec
DELETE FROM production_schedule_finished_policy WHERE account_id = sqlc.arg('account_id') AND item_id = sqlc.arg('item_id');

-- name: DeleteItemDemandOverrides :exec
DELETE FROM demand_override
WHERE account_id = sqlc.arg('account_id') AND scope_code = 'item' AND scope_ref_id = sqlc.arg('item_id');

-- name: DeleteItemByID :execresult
DELETE FROM item WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- Whether each item may be newly assigned: archived, or soft-deleted before archiving existed, it may not.
-- name: GetItemAssignabilityByIDs :many
SELECT i.id, i.sku, CAST(i.is_active = 1 AND i.deleted_at IS NULL AS SIGNED) AS is_assignable
FROM item i
WHERE i.id IN (sqlc.slice('ids')) AND i.account_id = sqlc.arg('account_id');
