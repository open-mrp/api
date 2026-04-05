-- name: ListSupplierMaterialsForward :many
SELECT
    sm.id,
    sm.material_id,
    sm.supplier_account_id,
    sm.supplier_part_number,
    sm.supplier_description,
    sm.is_active,
    sm.owner_account_id,
    sm.created_at,
    sm.updated_at,
    m.id AS material_type_id,
    m.item_id,
    m.order_point_id,
    m.lead_time_id,
    m.created_at AS material_created_at,
    m.updated_at AS material_updated_at,
    i.sku,
    i.description AS item_description,
    i.notes AS item_notes,
    i.item_type_code,
    i.item_category_id,
    i.unit_value_id,
    i.unit_cost_id,
    i.burn_rate_id,
    i.account_id AS item_account_id,
    i.is_dirty,
    i.created_at AS item_created_at,
    i.updated_at AS item_updated_at,
    ic.name AS category_name,
    ic.item_category_type_code,
    ic.unit_group_id AS category_unit_group_id,
    op.value AS order_point_value,
    op.unit_id AS order_point_unit_id,
    op_u.abbreviation AS order_point_unit_abbreviation,
    op_u.unit_dimension_code AS order_point_unit_type,
    lt.value AS lead_time_value,
    lt.unit_id AS lead_time_unit_id,
    lt_u.abbreviation AS lead_time_unit_abbreviation,
    lt_u.unit_dimension_code AS lead_time_unit_type,
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
FROM supplier_material sm
JOIN material m ON m.id = sm.material_id
JOIN item i ON i.id = m.item_id
JOIN item_category ic ON ic.id = i.item_category_id
LEFT JOIN quantity op ON op.id = m.order_point_id
LEFT JOIN unit op_u ON op_u.id = op.unit_id
LEFT JOIN quantity lt ON lt.id = m.lead_time_id
LEFT JOIN unit lt_u ON lt_u.id = lt.unit_id
JOIN rate rv ON rv.id = i.unit_value_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN rate rb ON rb.id = i.burn_rate_id
WHERE sm.supplier_account_id = sqlc.arg('supplier_account_id')
  AND sm.owner_account_id = sqlc.arg('owner_account_id')
  AND i.deleted_at IS NULL
  AND (
    sqlc.narg('search_query') IS NULL
    OR sm.supplier_part_number LIKE sqlc.narg('search_query')
    OR sm.supplier_description LIKE sqlc.narg('search_query')
    OR i.sku LIKE sqlc.narg('search_query')
    OR i.description LIKE sqlc.narg('search_query')
  )
  AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR sm.created_at < sqlc.narg('cursor_created_at')
    OR (sm.created_at = sqlc.narg('cursor_created_at') AND sm.id < sqlc.narg('cursor_id'))
  )
ORDER BY sm.created_at DESC, sm.id DESC
LIMIT ?;

-- name: ListSupplierMaterialsBackward :many
SELECT
    sm.id,
    sm.material_id,
    sm.supplier_account_id,
    sm.supplier_part_number,
    sm.supplier_description,
    sm.is_active,
    sm.owner_account_id,
    sm.created_at,
    sm.updated_at,
    m.id AS material_type_id,
    m.item_id,
    m.order_point_id,
    m.lead_time_id,
    m.created_at AS material_created_at,
    m.updated_at AS material_updated_at,
    i.sku,
    i.description AS item_description,
    i.notes AS item_notes,
    i.item_type_code,
    i.item_category_id,
    i.unit_value_id,
    i.unit_cost_id,
    i.burn_rate_id,
    i.account_id AS item_account_id,
    i.is_dirty,
    i.created_at AS item_created_at,
    i.updated_at AS item_updated_at,
    ic.name AS category_name,
    ic.item_category_type_code,
    ic.unit_group_id AS category_unit_group_id,
    op.value AS order_point_value,
    op.unit_id AS order_point_unit_id,
    op_u.abbreviation AS order_point_unit_abbreviation,
    op_u.unit_dimension_code AS order_point_unit_type,
    lt.value AS lead_time_value,
    lt.unit_id AS lead_time_unit_id,
    lt_u.abbreviation AS lead_time_unit_abbreviation,
    lt_u.unit_dimension_code AS lead_time_unit_type,
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
FROM supplier_material sm
JOIN material m ON m.id = sm.material_id
JOIN item i ON i.id = m.item_id
JOIN item_category ic ON ic.id = i.item_category_id
LEFT JOIN quantity op ON op.id = m.order_point_id
LEFT JOIN unit op_u ON op_u.id = op.unit_id
LEFT JOIN quantity lt ON lt.id = m.lead_time_id
LEFT JOIN unit lt_u ON lt_u.id = lt.unit_id
JOIN rate rv ON rv.id = i.unit_value_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN rate rb ON rb.id = i.burn_rate_id
WHERE sm.supplier_account_id = sqlc.arg('supplier_account_id')
  AND sm.owner_account_id = sqlc.arg('owner_account_id')
  AND i.deleted_at IS NULL
  AND (
    sqlc.narg('search_query') IS NULL
    OR sm.supplier_part_number LIKE sqlc.narg('search_query')
    OR sm.supplier_description LIKE sqlc.narg('search_query')
    OR i.sku LIKE sqlc.narg('search_query')
    OR i.description LIKE sqlc.narg('search_query')
  )
  AND (
    sm.created_at > sqlc.arg('cursor_created_at')
    OR (sm.created_at = sqlc.arg('cursor_created_at') AND sm.id > sqlc.arg('cursor_id'))
  )
ORDER BY sm.created_at ASC, sm.id ASC
LIMIT ?;

-- name: GetSupplierMaterialBySupplierAndItemID :one
SELECT
    sm.id,
    sm.material_id,
    sm.supplier_account_id,
    sm.supplier_part_number,
    sm.supplier_description,
    sm.is_active,
    sm.owner_account_id,
    sm.created_at,
    sm.updated_at,
    m.id AS material_type_id,
    m.item_id,
    m.order_point_id,
    m.lead_time_id,
    m.created_at AS material_created_at,
    m.updated_at AS material_updated_at,
    i.sku,
    i.description AS item_description,
    i.notes AS item_notes,
    i.item_type_code,
    i.item_category_id,
    i.unit_value_id,
    i.unit_cost_id,
    i.burn_rate_id,
    i.account_id AS item_account_id,
    i.is_dirty,
    i.created_at AS item_created_at,
    i.updated_at AS item_updated_at,
    ic.name AS category_name,
    ic.item_category_type_code,
    ic.unit_group_id AS category_unit_group_id,
    op.value AS order_point_value,
    op.unit_id AS order_point_unit_id,
    op_u.abbreviation AS order_point_unit_abbreviation,
    op_u.unit_dimension_code AS order_point_unit_type,
    lt.value AS lead_time_value,
    lt.unit_id AS lead_time_unit_id,
    lt_u.abbreviation AS lead_time_unit_abbreviation,
    lt_u.unit_dimension_code AS lead_time_unit_type,
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
FROM supplier_material sm
JOIN material m ON m.id = sm.material_id
JOIN item i ON i.id = m.item_id
JOIN item_category ic ON ic.id = i.item_category_id
LEFT JOIN quantity op ON op.id = m.order_point_id
LEFT JOIN unit op_u ON op_u.id = op.unit_id
LEFT JOIN quantity lt ON lt.id = m.lead_time_id
LEFT JOIN unit lt_u ON lt_u.id = lt.unit_id
JOIN rate rv ON rv.id = i.unit_value_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN rate rb ON rb.id = i.burn_rate_id
WHERE sm.supplier_account_id = sqlc.arg('supplier_account_id')
  AND m.item_id = sqlc.arg('item_id')
  AND sm.owner_account_id = sqlc.arg('owner_account_id')
  AND i.deleted_at IS NULL;

-- name: CreateSupplierMaterial :exec
INSERT INTO supplier_material (
    id, material_id, supplier_account_id, supplier_part_number, supplier_description,
    is_active, owner_account_id, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('material_id'), sqlc.arg('supplier_account_id'),
    sqlc.arg('supplier_part_number'), sqlc.narg('supplier_description'),
    sqlc.arg('is_active'), sqlc.arg('owner_account_id'), NOW(3), NOW(3)
);

-- name: UpdateSupplierMaterial :execresult
UPDATE supplier_material SET
    supplier_part_number = COALESCE(sqlc.narg('supplier_part_number'), supplier_part_number),
    supplier_description = CASE WHEN sqlc.arg('update_description') THEN sqlc.narg('supplier_description') ELSE supplier_description END,
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
  AND owner_account_id = sqlc.arg('owner_account_id');

-- name: DeleteSupplierMaterial :execresult
DELETE FROM supplier_material
WHERE id = sqlc.arg('id')
  AND owner_account_id = sqlc.arg('owner_account_id');

-- name: ExistsSupplierMaterialByMaterialAndSupplier :one
SELECT COUNT(*) AS total
FROM supplier_material
WHERE material_id = sqlc.arg('material_id')
  AND supplier_account_id = sqlc.arg('supplier_account_id')
  AND owner_account_id = sqlc.arg('owner_account_id');

-- name: GetMaterialIDByItemID :one
SELECT m.id
FROM material m
JOIN item i ON i.id = m.item_id
WHERE m.item_id = sqlc.arg('item_id')
  AND i.account_id = sqlc.arg('account_id')
  AND i.deleted_at IS NULL;

-- name: GetItemIDByMaterialID :one
SELECT m.item_id
FROM material m
WHERE m.id = sqlc.arg('material_id');
