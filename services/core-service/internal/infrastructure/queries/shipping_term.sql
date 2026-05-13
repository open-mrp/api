-- name: ListShippingTermsForward :many
SELECT
    st.id,
    st.name,
    st.is_freight_exempt,
    st.is_carrier_rate,
    st.account_id,
    st.flat_rate_id,
    st.minimum_order_id,
    st.created_at,
    st.updated_at,
    fr.id AS flat_rate_quantity_id,
    fr.value AS flat_rate_value,
    fr.unit_id AS flat_rate_unit_id,
    fr.created_at AS flat_rate_quantity_created_at,
    fr.updated_at AS flat_rate_quantity_updated_at,
    fr_u.name AS flat_rate_unit_name,
    fr_u.abbreviation AS flat_rate_unit_abbreviation,
    fr_u.unit_dimension_code AS flat_rate_unit_type,
    fr_u.ratio_numerator AS flat_rate_unit_ratio_numerator,
    fr_u.ratio_denominator AS flat_rate_unit_ratio_denominator,
    fr_u.offset_numerator AS flat_rate_unit_offset_numerator,
    fr_u.offset_denominator AS flat_rate_unit_offset_denominator,
    fr_u.is_base_unit AS flat_rate_unit_is_base_unit,
    fr_u.account_id AS flat_rate_unit_account_id,
    fr_u.created_at AS flat_rate_unit_created_at,
    fr_u.updated_at AS flat_rate_unit_updated_at,
    mo.id AS minimum_order_quantity_id,
    mo.value AS minimum_order_value,
    mo.unit_id AS minimum_order_unit_id,
    mo.created_at AS minimum_order_quantity_created_at,
    mo.updated_at AS minimum_order_quantity_updated_at,
    mo_u.name AS minimum_order_unit_name,
    mo_u.abbreviation AS minimum_order_unit_abbreviation,
    mo_u.unit_dimension_code AS minimum_order_unit_type,
    mo_u.ratio_numerator AS minimum_order_unit_ratio_numerator,
    mo_u.ratio_denominator AS minimum_order_unit_ratio_denominator,
    mo_u.offset_numerator AS minimum_order_unit_offset_numerator,
    mo_u.offset_denominator AS minimum_order_unit_offset_denominator,
    mo_u.is_base_unit AS minimum_order_unit_is_base_unit,
    mo_u.account_id AS minimum_order_unit_account_id,
    mo_u.created_at AS minimum_order_unit_created_at,
    mo_u.updated_at AS minimum_order_unit_updated_at
FROM shipping_term st
LEFT JOIN quantity fr ON st.flat_rate_id = fr.id
LEFT JOIN unit fr_u ON fr.unit_id = fr_u.id
LEFT JOIN quantity mo ON st.minimum_order_id = mo.id
LEFT JOIN unit mo_u ON mo.unit_id = mo_u.id
WHERE (st.account_id = sqlc.arg('account_id') OR st.account_id IS NULL)
AND (
    sqlc.narg('search_query') IS NULL
    OR st.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR st.created_at < sqlc.narg('cursor_created_at')
    OR (st.created_at = sqlc.narg('cursor_created_at') AND st.id < sqlc.narg('cursor_id'))
)
ORDER BY st.created_at DESC, st.id DESC
LIMIT ?;

-- name: ListShippingTermsBackward :many
SELECT
    st.id,
    st.name,
    st.is_freight_exempt,
    st.is_carrier_rate,
    st.account_id,
    st.flat_rate_id,
    st.minimum_order_id,
    st.created_at,
    st.updated_at,
    fr.id AS flat_rate_quantity_id,
    fr.value AS flat_rate_value,
    fr.unit_id AS flat_rate_unit_id,
    fr.created_at AS flat_rate_quantity_created_at,
    fr.updated_at AS flat_rate_quantity_updated_at,
    fr_u.name AS flat_rate_unit_name,
    fr_u.abbreviation AS flat_rate_unit_abbreviation,
    fr_u.unit_dimension_code AS flat_rate_unit_type,
    fr_u.ratio_numerator AS flat_rate_unit_ratio_numerator,
    fr_u.ratio_denominator AS flat_rate_unit_ratio_denominator,
    fr_u.offset_numerator AS flat_rate_unit_offset_numerator,
    fr_u.offset_denominator AS flat_rate_unit_offset_denominator,
    fr_u.is_base_unit AS flat_rate_unit_is_base_unit,
    fr_u.account_id AS flat_rate_unit_account_id,
    fr_u.created_at AS flat_rate_unit_created_at,
    fr_u.updated_at AS flat_rate_unit_updated_at,
    mo.id AS minimum_order_quantity_id,
    mo.value AS minimum_order_value,
    mo.unit_id AS minimum_order_unit_id,
    mo.created_at AS minimum_order_quantity_created_at,
    mo.updated_at AS minimum_order_quantity_updated_at,
    mo_u.name AS minimum_order_unit_name,
    mo_u.abbreviation AS minimum_order_unit_abbreviation,
    mo_u.unit_dimension_code AS minimum_order_unit_type,
    mo_u.ratio_numerator AS minimum_order_unit_ratio_numerator,
    mo_u.ratio_denominator AS minimum_order_unit_ratio_denominator,
    mo_u.offset_numerator AS minimum_order_unit_offset_numerator,
    mo_u.offset_denominator AS minimum_order_unit_offset_denominator,
    mo_u.is_base_unit AS minimum_order_unit_is_base_unit,
    mo_u.account_id AS minimum_order_unit_account_id,
    mo_u.created_at AS minimum_order_unit_created_at,
    mo_u.updated_at AS minimum_order_unit_updated_at
FROM shipping_term st
LEFT JOIN quantity fr ON st.flat_rate_id = fr.id
LEFT JOIN unit fr_u ON fr.unit_id = fr_u.id
LEFT JOIN quantity mo ON st.minimum_order_id = mo.id
LEFT JOIN unit mo_u ON mo.unit_id = mo_u.id
WHERE (st.account_id = sqlc.arg('account_id') OR st.account_id IS NULL)
AND (
    sqlc.narg('search_query') IS NULL
    OR st.name LIKE sqlc.narg('search_query')
)
AND (
    st.created_at > sqlc.arg('cursor_created_at')
    OR (st.created_at = sqlc.arg('cursor_created_at') AND st.id > sqlc.arg('cursor_id'))
)
ORDER BY st.created_at ASC, st.id ASC
LIMIT ?;

-- name: GetShippingTerm :one
SELECT
    st.id,
    st.name,
    st.is_freight_exempt,
    st.is_carrier_rate,
    st.account_id,
    st.flat_rate_id,
    st.minimum_order_id,
    st.created_at,
    st.updated_at,
    fr.id AS flat_rate_quantity_id,
    fr.value AS flat_rate_value,
    fr.unit_id AS flat_rate_unit_id,
    fr.created_at AS flat_rate_quantity_created_at,
    fr.updated_at AS flat_rate_quantity_updated_at,
    fr_u.name AS flat_rate_unit_name,
    fr_u.abbreviation AS flat_rate_unit_abbreviation,
    fr_u.unit_dimension_code AS flat_rate_unit_type,
    fr_u.ratio_numerator AS flat_rate_unit_ratio_numerator,
    fr_u.ratio_denominator AS flat_rate_unit_ratio_denominator,
    fr_u.offset_numerator AS flat_rate_unit_offset_numerator,
    fr_u.offset_denominator AS flat_rate_unit_offset_denominator,
    fr_u.is_base_unit AS flat_rate_unit_is_base_unit,
    fr_u.account_id AS flat_rate_unit_account_id,
    fr_u.created_at AS flat_rate_unit_created_at,
    fr_u.updated_at AS flat_rate_unit_updated_at,
    mo.id AS minimum_order_quantity_id,
    mo.value AS minimum_order_value,
    mo.unit_id AS minimum_order_unit_id,
    mo.created_at AS minimum_order_quantity_created_at,
    mo.updated_at AS minimum_order_quantity_updated_at,
    mo_u.name AS minimum_order_unit_name,
    mo_u.abbreviation AS minimum_order_unit_abbreviation,
    mo_u.unit_dimension_code AS minimum_order_unit_type,
    mo_u.ratio_numerator AS minimum_order_unit_ratio_numerator,
    mo_u.ratio_denominator AS minimum_order_unit_ratio_denominator,
    mo_u.offset_numerator AS minimum_order_unit_offset_numerator,
    mo_u.offset_denominator AS minimum_order_unit_offset_denominator,
    mo_u.is_base_unit AS minimum_order_unit_is_base_unit,
    mo_u.account_id AS minimum_order_unit_account_id,
    mo_u.created_at AS minimum_order_unit_created_at,
    mo_u.updated_at AS minimum_order_unit_updated_at
FROM shipping_term st
LEFT JOIN quantity fr ON st.flat_rate_id = fr.id
LEFT JOIN unit fr_u ON fr.unit_id = fr_u.id
LEFT JOIN quantity mo ON st.minimum_order_id = mo.id
LEFT JOIN unit mo_u ON mo.unit_id = mo_u.id
WHERE st.id = sqlc.arg('id')
AND (st.account_id = sqlc.arg('account_id') OR st.account_id IS NULL);

-- name: InsertShippingTerm :exec
INSERT INTO shipping_term (
    id,
    name,
    is_freight_exempt,
    is_carrier_rate,
    account_id,
    flat_rate_id,
    minimum_order_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.arg('is_freight_exempt'),
    sqlc.arg('is_carrier_rate'),
    sqlc.narg('account_id'),
    sqlc.narg('flat_rate_id'),
    sqlc.narg('minimum_order_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateShippingTerm :execresult
UPDATE shipping_term SET
    name = COALESCE(sqlc.narg('name'), name),
    is_freight_exempt = COALESCE(sqlc.narg('is_freight_exempt'), is_freight_exempt),
    is_carrier_rate = COALESCE(sqlc.narg('is_carrier_rate'), is_carrier_rate),
    flat_rate_id = sqlc.narg('flat_rate_id'),
    minimum_order_id = sqlc.narg('minimum_order_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteShippingTerm :execresult
DELETE FROM shipping_term
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: ListFreeShippingCarrierOptionsByShippingTermID :many
SELECT
    co.id,
    co.code,
    co.name,
    co.service_level_token,
    co.is_portal_enabled,
    co.is_default,
    co.carrier_id,
    co.account_id,
    co.created_at,
    co.updated_at
FROM shipping_term_free_shipping_rule stfsr
INNER JOIN carrier_option co ON co.id = stfsr.carrier_option_id
WHERE stfsr.shipping_term_id = sqlc.arg('shipping_term_id')
ORDER BY co.name ASC;

-- name: InsertFreeShippingRule :exec
INSERT INTO shipping_term_free_shipping_rule (
    id,
    shipping_term_id,
    carrier_option_id,
    created_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('shipping_term_id'),
    sqlc.arg('carrier_option_id'),
    NOW(3)
);

-- name: DeleteFreeShippingRulesByShippingTermID :exec
DELETE FROM shipping_term_free_shipping_rule
WHERE shipping_term_id = ?;

-- name: InsertQuantity :exec
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

-- name: UpdateQuantity :execresult
UPDATE quantity SET
    value = sqlc.arg('value'),
    unit_id = sqlc.arg('unit_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: DeleteQuantity :exec
DELETE FROM quantity WHERE id = ?;
