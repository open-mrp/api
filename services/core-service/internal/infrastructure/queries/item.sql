-- name: ListItemsForward :many
SELECT
    i.id,
    i.sku,
    i.description,
    i.notes,
    i.item_type_code,
    i.item_category_id,
    i.unit_value_id,
    i.unit_cost_id,
    i.burn_rate_id,
    i.account_id,
    i.is_dirty,
    i.created_at,
    i.updated_at,
    i.deleted_at,
    ic.name AS category_name,
    ic.item_category_type_code,
    ic.unit_group_id AS category_unit_group_id,
    rv.id AS unit_value_rate_id,
    rv.value AS unit_value_rate_value,
    rv.numerator_unit_id AS unit_value_numerator_unit_id,
    rv.denominator_unit_id AS unit_value_denominator_unit_id,
    rv.created_at AS unit_value_created_at,
    rv.updated_at AS unit_value_updated_at,
    rc.id AS unit_cost_rate_id,
    rc.value AS unit_cost_rate_value,
    rc.numerator_unit_id AS unit_cost_numerator_unit_id,
    rc.denominator_unit_id AS unit_cost_denominator_unit_id,
    rc.created_at AS unit_cost_created_at,
    rc.updated_at AS unit_cost_updated_at,
    rb.id AS burn_rate_id_joined,
    rb.value AS burn_rate_value,
    rb.numerator_unit_id AS burn_rate_numerator_unit_id,
    rb.denominator_unit_id AS burn_rate_denominator_unit_id,
    rb.created_at AS burn_rate_created_at,
    rb.updated_at AS burn_rate_updated_at
FROM item i
JOIN item_category ic ON ic.id = i.item_category_id
JOIN rate rv ON rv.id = i.unit_value_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN rate rb ON rb.id = i.burn_rate_id
WHERE i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL
AND (
    sqlc.arg('include_type_filter') = false
    OR i.item_type_code IN (sqlc.slice('item_type_codes'))
)
AND (
    sqlc.arg('include_category_filter') = false
    OR i.item_category_id IN (sqlc.slice('category_ids'))
)
AND (
    sqlc.arg('include_attribute_filter') = false
    OR EXISTS (
        SELECT 1 FROM _item_attributes ia
        WHERE ia.B = i.id
        AND ia.A IN (sqlc.slice('attribute_ids'))
    )
)
AND (
    sqlc.narg('supplier_id') IS NULL
    OR EXISTS (
        SELECT 1 FROM material m
        JOIN supplier_material sm ON sm.material_id = m.id
        WHERE m.item_id = i.id
        AND sm.supplier_account_id = sqlc.narg('supplier_id')
        AND sm.owner_account_id = sqlc.arg('account_id')
    )
)
AND (
    sqlc.narg('start_date') IS NULL
    OR i.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR i.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.narg('search_query') IS NULL
    OR (
        sqlc.arg('is_exact_match') = true AND (
            i.sku = sqlc.narg('search_exact')
            OR i.description LIKE sqlc.narg('search_query')
        )
    )
    OR (
        sqlc.arg('is_exact_match') = false AND (
            i.sku LIKE sqlc.narg('search_query')
            OR i.description LIKE sqlc.narg('search_query')
        )
    )
)
AND (
    NOT EXISTS (SELECT 1 FROM product p WHERE p.item_id = i.id)
    OR EXISTS (SELECT 1 FROM product p WHERE p.item_id = i.id AND p.product_type_code = 'sale')
)
AND (
    sqlc.arg('only_initial_subassemblies') = false
    OR EXISTS (
        SELECT 1 FROM production prd
        WHERE prd.item_id = i.id
        AND prd.production_step_id IS NOT NULL
        AND NOT EXISTS (
            SELECT 1 FROM _parent_child_production_steps pcps
            WHERE pcps.B = prd.production_step_id
        )
    )
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR i.created_at < sqlc.narg('cursor_created_at')
    OR (i.created_at = sqlc.narg('cursor_created_at') AND i.id < sqlc.narg('cursor_id'))
)
ORDER BY i.created_at DESC, i.id DESC
LIMIT ?;

-- name: ListItemsBackward :many
SELECT
    i.id,
    i.sku,
    i.description,
    i.notes,
    i.item_type_code,
    i.item_category_id,
    i.unit_value_id,
    i.unit_cost_id,
    i.burn_rate_id,
    i.account_id,
    i.is_dirty,
    i.created_at,
    i.updated_at,
    i.deleted_at,
    ic.name AS category_name,
    ic.item_category_type_code,
    ic.unit_group_id AS category_unit_group_id,
    rv.id AS unit_value_rate_id,
    rv.value AS unit_value_rate_value,
    rv.numerator_unit_id AS unit_value_numerator_unit_id,
    rv.denominator_unit_id AS unit_value_denominator_unit_id,
    rv.created_at AS unit_value_created_at,
    rv.updated_at AS unit_value_updated_at,
    rc.id AS unit_cost_rate_id,
    rc.value AS unit_cost_rate_value,
    rc.numerator_unit_id AS unit_cost_numerator_unit_id,
    rc.denominator_unit_id AS unit_cost_denominator_unit_id,
    rc.created_at AS unit_cost_created_at,
    rc.updated_at AS unit_cost_updated_at,
    rb.id AS burn_rate_id_joined,
    rb.value AS burn_rate_value,
    rb.numerator_unit_id AS burn_rate_numerator_unit_id,
    rb.denominator_unit_id AS burn_rate_denominator_unit_id,
    rb.created_at AS burn_rate_created_at,
    rb.updated_at AS burn_rate_updated_at
FROM item i
JOIN item_category ic ON ic.id = i.item_category_id
JOIN rate rv ON rv.id = i.unit_value_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN rate rb ON rb.id = i.burn_rate_id
WHERE i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL
AND (
    sqlc.arg('include_type_filter') = false
    OR i.item_type_code IN (sqlc.slice('item_type_codes'))
)
AND (
    sqlc.arg('include_category_filter') = false
    OR i.item_category_id IN (sqlc.slice('category_ids'))
)
AND (
    sqlc.arg('include_attribute_filter') = false
    OR EXISTS (
        SELECT 1 FROM _item_attributes ia
        WHERE ia.B = i.id
        AND ia.A IN (sqlc.slice('attribute_ids'))
    )
)
AND (
    sqlc.narg('supplier_id') IS NULL
    OR EXISTS (
        SELECT 1 FROM material m
        JOIN supplier_material sm ON sm.material_id = m.id
        WHERE m.item_id = i.id
        AND sm.supplier_account_id = sqlc.narg('supplier_id')
        AND sm.owner_account_id = sqlc.arg('account_id')
    )
)
AND (
    sqlc.narg('start_date') IS NULL
    OR i.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR i.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.narg('search_query') IS NULL
    OR (
        sqlc.arg('is_exact_match') = true AND (
            i.sku = sqlc.narg('search_exact')
            OR i.description LIKE sqlc.narg('search_query')
        )
    )
    OR (
        sqlc.arg('is_exact_match') = false AND (
            i.sku LIKE sqlc.narg('search_query')
            OR i.description LIKE sqlc.narg('search_query')
        )
    )
)
AND (
    NOT EXISTS (SELECT 1 FROM product p WHERE p.item_id = i.id)
    OR EXISTS (SELECT 1 FROM product p WHERE p.item_id = i.id AND p.product_type_code = 'sale')
)
AND (
    sqlc.arg('only_initial_subassemblies') = false
    OR EXISTS (
        SELECT 1 FROM production prd
        WHERE prd.item_id = i.id
        AND prd.production_step_id IS NOT NULL
        AND NOT EXISTS (
            SELECT 1 FROM _parent_child_production_steps pcps
            WHERE pcps.B = prd.production_step_id
        )
    )
)
AND (
    i.created_at > sqlc.arg('cursor_created_at')
    OR (i.created_at = sqlc.arg('cursor_created_at') AND i.id > sqlc.arg('cursor_id'))
)
ORDER BY i.created_at ASC, i.id ASC
LIMIT ?;

-- name: GetItem :one
SELECT
    i.id,
    i.sku,
    i.description,
    i.notes,
    i.item_type_code,
    i.item_category_id,
    i.unit_value_id,
    i.unit_cost_id,
    i.burn_rate_id,
    i.account_id,
    i.is_dirty,
    i.created_at,
    i.updated_at,
    i.deleted_at,
    ic.name AS category_name,
    ic.item_category_type_code,
    ic.unit_group_id AS category_unit_group_id,
    rv.id AS unit_value_rate_id,
    rv.value AS unit_value_rate_value,
    rv.numerator_unit_id AS unit_value_numerator_unit_id,
    rv.denominator_unit_id AS unit_value_denominator_unit_id,
    rv.created_at AS unit_value_created_at,
    rv.updated_at AS unit_value_updated_at,
    rc.id AS unit_cost_rate_id,
    rc.value AS unit_cost_rate_value,
    rc.numerator_unit_id AS unit_cost_numerator_unit_id,
    rc.denominator_unit_id AS unit_cost_denominator_unit_id,
    rc.created_at AS unit_cost_created_at,
    rc.updated_at AS unit_cost_updated_at,
    rb.id AS burn_rate_id_joined,
    rb.value AS burn_rate_value,
    rb.numerator_unit_id AS burn_rate_numerator_unit_id,
    rb.denominator_unit_id AS burn_rate_denominator_unit_id,
    rb.created_at AS burn_rate_created_at,
    rb.updated_at AS burn_rate_updated_at
FROM item i
JOIN item_category ic ON ic.id = i.item_category_id
JOIN rate rv ON rv.id = i.unit_value_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN rate rb ON rb.id = i.burn_rate_id
WHERE i.id = sqlc.arg('id')
AND i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL;

-- name: GetItemAttributes :many
SELECT
    a.id,
    a.text,
    a.color_code,
    a.property_id
FROM _item_attributes ia
JOIN attribute a ON a.id = ia.A
WHERE ia.B = sqlc.arg('item_id');

-- name: GetItemInventory :one
SELECT
    COALESCE(receipt_totals.on_hand, 0) AS on_hand,
    COALESCE(issue_totals.reserved, 0) AS reserved,
    COALESCE(issue_totals.short, 0) AS short,
    CAST(COALESCE(receipt_totals.on_hand, 0) - COALESCE(issue_totals.reserved, 0) - COALESCE(issue_totals.short, 0) AS DECIMAL(65,30)) AS available_to_promise,
    COALESCE(rv.denominator_unit_id, '') AS unit_id
FROM item i
JOIN rate rv ON rv.id = i.unit_value_id
LEFT JOIN (
    SELECT
        ir.item_id,
        SUM(q.value - COALESCE(alloc.allocated, 0)) AS on_hand
    FROM inventory_receipt ir
    JOIN quantity q ON q.id = ir.quantity_id
    LEFT JOIN (
        SELECT ia.inventory_receipt_id, SUM(q.value) AS allocated
        FROM inventory_allocation ia
        JOIN quantity q ON q.id = ia.quantity_id
        GROUP BY ia.inventory_receipt_id
    ) alloc ON alloc.inventory_receipt_id = ir.id
    WHERE ir.item_id = sqlc.arg('item_id')
        AND (ir.owner_account_id = sqlc.arg('account_id') OR ir.holder_account_id = sqlc.arg('account_id'))
        AND ir.status_code = 'available'
    GROUP BY ir.item_id
) receipt_totals ON receipt_totals.item_id = i.id
LEFT JOIN (
    SELECT
        ii.item_id,
        SUM(CASE WHEN ii.status_code = 'reserved' THEN q.value - COALESCE(alloc.allocated, 0) ELSE 0 END) AS reserved,
        SUM(CASE WHEN ii.status_code = 'open' THEN q.value - COALESCE(alloc.allocated, 0) ELSE 0 END) AS short
    FROM inventory_issue ii
    JOIN quantity q ON q.id = ii.quantity_id
    LEFT JOIN (
        SELECT ia.inventory_issue_id, SUM(q.value) AS allocated
        FROM inventory_allocation ia
        JOIN quantity q ON q.id = ia.quantity_id
        GROUP BY ia.inventory_issue_id
    ) alloc ON alloc.inventory_issue_id = ii.id
    WHERE ii.item_id = sqlc.arg('item_id')
        AND ii.account_id = sqlc.arg('account_id')
        AND ii.status_code IN ('reserved', 'open')
    GROUP BY ii.item_id
) issue_totals ON issue_totals.item_id = i.id
WHERE i.id = sqlc.arg('item_id')
    AND i.account_id = sqlc.arg('account_id')
    AND i.deleted_at IS NULL;

-- name: GetCostFlowStepConsumptions :many
-- Fetches consumption data for a production step with item type and unit cost for cost calculation.
SELECT
    ci.item_type_code AS consumed_item_type,
    cq.value AS consumption_quantity_value,
    wq.value AS waste_quantity_value,
    COALESCE(ucr.value, 0) AS consumed_item_unit_cost
FROM consumption c
JOIN item ci ON c.item_id = ci.id
JOIN quantity cq ON c.quantity_id = cq.id
JOIN quantity wq ON c.waste_quantity_id = wq.id
LEFT JOIN rate ucr ON ucr.id = ci.unit_cost_id
WHERE c.production_step_id = sqlc.arg('production_step_id');

-- name: UpdateItemUnitCostRate :exec
-- Updates an item's unit cost rate value and denominator unit.
UPDATE rate r
SET r.value = sqlc.arg('value'),
    r.denominator_unit_id = sqlc.arg('denominator_unit_id')
WHERE r.id = (
    SELECT i.unit_cost_id FROM item i
    WHERE i.id = sqlc.arg('item_id')
    AND i.account_id = sqlc.arg('account_id')
    AND i.deleted_at IS NULL
);

-- name: ClearItemDirtyFlag :exec
-- Clears the dirty flag on an item after cost recalculation.
UPDATE item
SET is_dirty = 0
WHERE id = sqlc.arg('item_id')
AND account_id = sqlc.arg('account_id')
AND deleted_at IS NULL;

-- name: GetItemTrends :many
SELECT
    il.created_at AS date,
    q.value
FROM inventory_log il
JOIN quantity q ON q.id = il.quantity_id
WHERE il.item_id = sqlc.arg('item_id')
AND il.account_id = sqlc.arg('account_id')
AND il.created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)
ORDER BY il.created_at ASC;

-- name: ExportItemsWithInventory :many
SELECT
    i.id,
    i.sku,
    i.description,
    i.notes,
    i.item_type_code,
    i.item_category_id,
    i.account_id,
    i.created_at,
    i.updated_at,
    ic.name AS category_name,
    COALESCE(inv.on_hand, 0) AS on_hand_quantity,
    COALESCE(rv.denominator_unit_id, '') AS on_hand_unit_id
FROM item i
JOIN item_category ic ON ic.id = i.item_category_id
JOIN rate rv ON rv.id = i.unit_value_id
LEFT JOIN (
    SELECT
        ir.item_id,
        SUM(q.value - COALESCE(alloc.allocated, 0)) AS on_hand
    FROM inventory_receipt ir
    JOIN quantity q ON q.id = ir.quantity_id
    LEFT JOIN (
        SELECT ia.inventory_receipt_id, SUM(qa.value) AS allocated
        FROM inventory_allocation ia
        JOIN quantity qa ON qa.id = ia.quantity_id
        GROUP BY ia.inventory_receipt_id
    ) alloc ON alloc.inventory_receipt_id = ir.id
    WHERE ir.status_code = 'available'
    AND (ir.owner_account_id = sqlc.arg('account_id') OR ir.holder_account_id = sqlc.arg('account_id'))
    GROUP BY ir.item_id
) inv ON inv.item_id = i.id
WHERE i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL
ORDER BY i.sku ASC;

-- name: UpdateItem :exec
UPDATE item SET
  sku = COALESCE(sqlc.narg('sku'), sku),
  description = COALESCE(sqlc.narg('description'), description),
  notes = COALESCE(sqlc.narg('notes'), notes),
  updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id')
AND deleted_at IS NULL;

-- name: SetItemDescription :exec
UPDATE item SET
  description = sqlc.narg('description'),
  updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id')
AND deleted_at IS NULL;

-- name: SetItemNotes :exec
UPDATE item SET
  notes = sqlc.narg('notes'),
  updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id')
AND deleted_at IS NULL;

-- name: CheckItemSKUExists :one
SELECT EXISTS(
  SELECT 1 FROM item
  WHERE sku = sqlc.arg('sku')
  AND account_id = sqlc.arg('account_id')
  AND id != sqlc.arg('exclude_id')
  AND deleted_at IS NULL
) AS sku_exists;

-- name: AddItemAttribute :exec
INSERT INTO _item_attributes (A, B) VALUES (sqlc.arg('attribute_id'), sqlc.arg('item_id'))
ON DUPLICATE KEY UPDATE A = A;

-- name: RemoveItemAttribute :execresult
DELETE ia FROM _item_attributes ia
JOIN item i ON i.id = ia.B
WHERE ia.A = sqlc.arg('attribute_id')
  AND ia.B = sqlc.arg('item_id')
  AND i.account_id = sqlc.arg('account_id')
  AND i.deleted_at IS NULL;

-- name: ChangeItemCategory :exec
UPDATE item SET
  item_category_id = sqlc.arg('category_id'),
  updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id')
AND deleted_at IS NULL;

-- name: UpdateItemRateUnitValue :exec
UPDATE rate r
JOIN item i ON i.unit_value_id = r.id
SET r.denominator_unit_id = sqlc.arg('new_unit_id')
WHERE i.id = sqlc.arg('item_id')
AND i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL;

-- name: UpdateItemRateUnitCost :exec
UPDATE rate r
JOIN item i ON i.unit_cost_id = r.id
SET r.denominator_unit_id = sqlc.arg('new_unit_id')
WHERE i.id = sqlc.arg('item_id')
AND i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL;

-- name: UpdateItemRateBurnRate :exec
UPDATE rate r
JOIN item i ON i.burn_rate_id = r.id
SET r.numerator_unit_id = sqlc.arg('new_unit_id')
WHERE i.id = sqlc.arg('item_id')
AND i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL;

-- name: GetCategoryBaseUnitID :one
SELECT
  ugu.unit_id AS base_unit_id
FROM item_category ic
JOIN unit_group_unit ugu ON ugu.unit_group_id = ic.unit_group_id
JOIN unit u ON u.id = ugu.unit_id AND u.is_base_unit = true
WHERE ic.id = sqlc.arg('category_id');

-- name: UpdateMaterialOrderPointUnit :exec
UPDATE quantity q
JOIN material m ON m.order_point_id = q.id
JOIN item i ON i.id = m.item_id
SET q.unit_id = sqlc.arg('new_unit_id')
WHERE i.id = sqlc.arg('item_id')
AND i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL
AND i.item_type_code = 'material';

-- name: UpdateItemConsumptionQuantityUnits :exec
UPDATE quantity q
JOIN consumption c ON (c.quantity_id = q.id OR c.waste_quantity_id = q.id)
JOIN item i ON i.id = c.item_id
SET q.unit_id = sqlc.arg('new_unit_id')
WHERE i.id = sqlc.arg('item_id')
AND i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL;

-- name: UpdateItemProductionQuantityUnits :exec
UPDATE quantity q
JOIN production p ON p.quantity_id = q.id
JOIN item i ON i.id = p.item_id
SET q.unit_id = sqlc.arg('new_unit_id')
WHERE i.id = sqlc.arg('item_id')
AND i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL;

-- name: FetchItemsBySKU :many
SELECT
    i.id AS item_id,
    i.sku,
    ug.base_unit_id
FROM item i
JOIN item_category ic ON i.item_category_id = ic.id
JOIN unit_group ug ON ic.unit_group_id = ug.id
WHERE i.account_id = sqlc.arg('account_id')
  AND i.sku IN (sqlc.slice('skus'))
  AND i.deleted_at IS NULL;
