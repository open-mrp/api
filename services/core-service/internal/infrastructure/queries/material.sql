-- name: ListMaterialsForward :many
SELECT
    m.id,
    m.item_id,
    m.order_point_id,
    m.lead_time_id,
    m.created_at,
    m.updated_at,
    i.sku,
    i.description AS item_description,
    i.notes AS item_notes,
    i.item_type_code,
    i.item_category_id,
    i.unit_value_id,
    i.unit_cost_id,
    i.burn_rate_id,
    i.account_id,
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
FROM material m
JOIN item i ON i.id = m.item_id
JOIN item_category ic ON ic.id = i.item_category_id
JOIN quantity op ON op.id = m.order_point_id
JOIN unit op_u ON op_u.id = op.unit_id
JOIN quantity lt ON lt.id = m.lead_time_id
JOIN unit lt_u ON lt_u.id = lt.unit_id
JOIN rate rv ON rv.id = i.unit_value_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN rate rb ON rb.id = i.burn_rate_id
WHERE i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL
AND (
    sqlc.narg('search_query') IS NULL
    OR i.sku LIKE sqlc.narg('search_query')
    OR i.description LIKE sqlc.narg('search_query')
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
    sqlc.narg('start_date') IS NULL
    OR i.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR i.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR m.created_at < sqlc.narg('cursor_created_at')
    OR (m.created_at = sqlc.narg('cursor_created_at') AND m.id < sqlc.narg('cursor_id'))
)
ORDER BY m.created_at DESC, m.id DESC
LIMIT ?;

-- name: ListMaterialsBackward :many
SELECT
    m.id,
    m.item_id,
    m.order_point_id,
    m.lead_time_id,
    m.created_at,
    m.updated_at,
    i.sku,
    i.description AS item_description,
    i.notes AS item_notes,
    i.item_type_code,
    i.item_category_id,
    i.unit_value_id,
    i.unit_cost_id,
    i.burn_rate_id,
    i.account_id,
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
FROM material m
JOIN item i ON i.id = m.item_id
JOIN item_category ic ON ic.id = i.item_category_id
JOIN quantity op ON op.id = m.order_point_id
JOIN unit op_u ON op_u.id = op.unit_id
JOIN quantity lt ON lt.id = m.lead_time_id
JOIN unit lt_u ON lt_u.id = lt.unit_id
JOIN rate rv ON rv.id = i.unit_value_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN rate rb ON rb.id = i.burn_rate_id
WHERE i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL
AND (
    sqlc.narg('search_query') IS NULL
    OR i.sku LIKE sqlc.narg('search_query')
    OR i.description LIKE sqlc.narg('search_query')
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
    sqlc.narg('start_date') IS NULL
    OR i.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR i.created_at <= sqlc.narg('end_date')
)
AND (
    m.created_at > sqlc.arg('cursor_created_at')
    OR (m.created_at = sqlc.arg('cursor_created_at') AND m.id > sqlc.arg('cursor_id'))
)
ORDER BY m.created_at ASC, m.id ASC
LIMIT ?;

-- name: GetMaterialByID :one
SELECT
    m.id,
    m.item_id,
    m.order_point_id,
    m.lead_time_id,
    m.created_at,
    m.updated_at,
    i.sku,
    i.description AS item_description,
    i.notes AS item_notes,
    i.item_type_code,
    i.item_category_id,
    i.unit_value_id,
    i.unit_cost_id,
    i.burn_rate_id,
    i.account_id,
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
FROM material m
JOIN item i ON i.id = m.item_id
JOIN item_category ic ON ic.id = i.item_category_id
JOIN quantity op ON op.id = m.order_point_id
JOIN unit op_u ON op_u.id = op.unit_id
JOIN quantity lt ON lt.id = m.lead_time_id
JOIN unit lt_u ON lt_u.id = lt.unit_id
JOIN rate rv ON rv.id = i.unit_value_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN rate rb ON rb.id = i.burn_rate_id
WHERE m.id = sqlc.arg('id')
AND i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL;

-- name: GetMaterialByItemID :one
SELECT
    m.id,
    m.item_id,
    m.order_point_id,
    m.lead_time_id,
    m.created_at,
    m.updated_at,
    i.sku,
    i.description AS item_description,
    i.notes AS item_notes,
    i.item_type_code,
    i.item_category_id,
    i.unit_value_id,
    i.unit_cost_id,
    i.burn_rate_id,
    i.account_id,
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
FROM material m
JOIN item i ON i.id = m.item_id
JOIN item_category ic ON ic.id = i.item_category_id
JOIN quantity op ON op.id = m.order_point_id
JOIN unit op_u ON op_u.id = op.unit_id
JOIN quantity lt ON lt.id = m.lead_time_id
JOIN unit lt_u ON lt_u.id = lt.unit_id
JOIN rate rv ON rv.id = i.unit_value_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN rate rb ON rb.id = i.burn_rate_id
WHERE m.item_id = sqlc.arg('item_id')
AND i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL;

-- name: CreateMaterial :exec
INSERT INTO material (
    id,
    item_id,
    order_point_id,
    lead_time_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('item_id'),
    sqlc.arg('order_point_id'),
    sqlc.arg('lead_time_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateMaterial :execresult
UPDATE material SET
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: DeleteMaterialByID :execresult
UPDATE item i
JOIN material m ON m.item_id = i.id
SET i.deleted_at = NOW(3)
WHERE m.id = sqlc.arg('id')
AND i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL;

-- name: MaterialInsertQuantity :exec
INSERT INTO quantity (
    id,
    value,
    unit_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('value'),
    sqlc.arg('unit_id'),
    NOW(3),
    NOW(3)
);

-- name: MaterialUpdateQuantity :execresult
UPDATE quantity SET
    value = sqlc.arg('value'),
    unit_id = sqlc.arg('unit_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: MaterialInsertRate :exec
INSERT INTO rate (
    id,
    value,
    numerator_unit_id,
    denominator_unit_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('value'),
    sqlc.arg('numerator_unit_id'),
    sqlc.arg('denominator_unit_id'),
    NOW(3),
    NOW(3)
);

-- name: MaterialInsertItem :exec
INSERT INTO item (
    id,
    sku,
    description,
    notes,
    item_type_code,
    item_category_id,
    unit_value_id,
    unit_cost_id,
    burn_rate_id,
    account_id,
    is_dirty,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('sku'),
    sqlc.narg('description'),
    sqlc.narg('notes'),
    'material',
    sqlc.arg('item_category_id'),
    sqlc.arg('unit_value_id'),
    sqlc.arg('unit_cost_id'),
    sqlc.arg('burn_rate_id'),
    sqlc.arg('account_id'),
    sqlc.arg('is_dirty'),
    NOW(3),
    NOW(3)
);

-- name: MaterialUpdateItem :execresult
UPDATE item SET
    sku = COALESCE(sqlc.narg('sku'), sku),
    description = CASE WHEN sqlc.arg('update_description') = true THEN sqlc.narg('description') ELSE description END,
    notes = CASE WHEN sqlc.arg('update_notes') = true THEN sqlc.narg('notes') ELSE notes END,
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id')
AND deleted_at IS NULL;
