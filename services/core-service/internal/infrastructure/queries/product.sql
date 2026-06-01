-- name: SearchProductsBySKU :many
SELECT p.id as product_id, i.id as item_id, i.sku, i.description, r.value as unit_price
FROM product p
JOIN item i ON i.id = p.item_id
JOIN rate r ON r.id = i.unit_value_id
WHERE i.account_id = ? AND p.product_type_code = 'sale' AND i.sku LIKE ?
LIMIT 20;

-- name: GetAccountSystemProduct :one
-- Fetches the account's system product (e.g. credit, shipping) along with
-- the base unit of its item category, for use when synthesizing order lines.
SELECT
    p.id AS product_id,
    i.sku AS product_sku,
    ug.base_unit_id AS quantity_unit_id
FROM product p
JOIN item i ON i.id = p.item_id
JOIN item_category ic ON ic.id = i.item_category_id
JOIN unit_group ug ON ug.id = ic.unit_group_id
WHERE p.product_type_code = sqlc.arg('product_type_code')
AND i.account_id = sqlc.arg('account_id')
LIMIT 1;

-- name: ListProductsByAccount :many
SELECT p.id as product_id, i.id as item_id, i.sku, i.description, r.value as unit_price
FROM product p
JOIN item i ON i.id = p.item_id
JOIN rate r ON r.id = i.unit_value_id
WHERE i.account_id = ? AND p.product_type_code = 'sale'
LIMIT 100;

-- name: ListProductsFullForward :many
SELECT
    p.id,
    p.product_type_code,
    p.is_portal_ready,
    p.product_line_id,
    p.item_id,
    p.created_at,
    p.updated_at,
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
    ic.created_at AS category_created_at,
    ic.updated_at AS category_updated_at,
    cat_ug.name AS category_unit_group_name,
    cat_ug.unit_type_code AS category_unit_group_type,
    cat_ug.created_at AS category_unit_group_created_at,
    cat_ug.updated_at AS category_unit_group_updated_at,
    rv.id AS unit_value_rate_id,
    rv.value AS unit_value_rate_value,
    rv.numerator_unit_id AS unit_value_numerator_unit_id,
    nvu.name AS unit_value_numerator_unit_name,
    nvu.abbreviation AS unit_value_numerator_unit_abbreviation,
    nvu.unit_dimension_code AS unit_value_numerator_unit_type,
    nvu.ratio_numerator AS unit_value_numerator_unit_ratio_numerator,
    nvu.ratio_denominator AS unit_value_numerator_unit_ratio_denominator,
    nvu.offset_numerator AS unit_value_numerator_unit_offset_numerator,
    nvu.offset_denominator AS unit_value_numerator_unit_offset_denominator,
    nvu.created_at AS unit_value_numerator_unit_created_at,
    nvu.updated_at AS unit_value_numerator_unit_updated_at,
    rv.denominator_unit_id AS unit_value_denominator_unit_id,
    dvu.name AS unit_value_denominator_unit_name,
    dvu.abbreviation AS unit_value_denominator_unit_abbreviation,
    dvu.unit_dimension_code AS unit_value_denominator_unit_type,
    dvu.ratio_numerator AS unit_value_denominator_unit_ratio_numerator,
    dvu.ratio_denominator AS unit_value_denominator_unit_ratio_denominator,
    dvu.offset_numerator AS unit_value_denominator_unit_offset_numerator,
    dvu.offset_denominator AS unit_value_denominator_unit_offset_denominator,
    dvu.created_at AS unit_value_denominator_unit_created_at,
    dvu.updated_at AS unit_value_denominator_unit_updated_at,
    rv.created_at AS unit_value_created_at,
    rv.updated_at AS unit_value_updated_at,
    rc.id AS unit_cost_rate_id,
    rc.value AS unit_cost_rate_value,
    rc.numerator_unit_id AS unit_cost_numerator_unit_id,
    ncu.name AS unit_cost_numerator_unit_name,
    ncu.abbreviation AS unit_cost_numerator_unit_abbreviation,
    ncu.unit_dimension_code AS unit_cost_numerator_unit_type,
    ncu.ratio_numerator AS unit_cost_numerator_unit_ratio_numerator,
    ncu.ratio_denominator AS unit_cost_numerator_unit_ratio_denominator,
    ncu.offset_numerator AS unit_cost_numerator_unit_offset_numerator,
    ncu.offset_denominator AS unit_cost_numerator_unit_offset_denominator,
    ncu.created_at AS unit_cost_numerator_unit_created_at,
    ncu.updated_at AS unit_cost_numerator_unit_updated_at,
    rc.denominator_unit_id AS unit_cost_denominator_unit_id,
    dcu.name AS unit_cost_denominator_unit_name,
    dcu.abbreviation AS unit_cost_denominator_unit_abbreviation,
    dcu.unit_dimension_code AS unit_cost_denominator_unit_type,
    dcu.ratio_numerator AS unit_cost_denominator_unit_ratio_numerator,
    dcu.ratio_denominator AS unit_cost_denominator_unit_ratio_denominator,
    dcu.offset_numerator AS unit_cost_denominator_unit_offset_numerator,
    dcu.offset_denominator AS unit_cost_denominator_unit_offset_denominator,
    dcu.created_at AS unit_cost_denominator_unit_created_at,
    dcu.updated_at AS unit_cost_denominator_unit_updated_at,
    rc.created_at AS unit_cost_created_at,
    rc.updated_at AS unit_cost_updated_at,
    rb.id AS burn_rate_id_joined,
    rb.value AS burn_rate_value,
    rb.numerator_unit_id AS burn_rate_numerator_unit_id,
    nbr.name AS burn_rate_numerator_unit_name,
    nbr.abbreviation AS burn_rate_numerator_unit_abbreviation,
    nbr.unit_dimension_code AS burn_rate_numerator_unit_type,
    nbr.ratio_numerator AS burn_rate_numerator_unit_ratio_numerator,
    nbr.ratio_denominator AS burn_rate_numerator_unit_ratio_denominator,
    nbr.offset_numerator AS burn_rate_numerator_unit_offset_numerator,
    nbr.offset_denominator AS burn_rate_numerator_unit_offset_denominator,
    nbr.created_at AS burn_rate_numerator_unit_created_at,
    nbr.updated_at AS burn_rate_numerator_unit_updated_at,
    rb.denominator_unit_id AS burn_rate_denominator_unit_id,
    dbr.name AS burn_rate_denominator_unit_name,
    dbr.abbreviation AS burn_rate_denominator_unit_abbreviation,
    dbr.unit_dimension_code AS burn_rate_denominator_unit_type,
    dbr.ratio_numerator AS burn_rate_denominator_unit_ratio_numerator,
    dbr.ratio_denominator AS burn_rate_denominator_unit_ratio_denominator,
    dbr.offset_numerator AS burn_rate_denominator_unit_offset_numerator,
    dbr.offset_denominator AS burn_rate_denominator_unit_offset_denominator,
    dbr.created_at AS burn_rate_denominator_unit_created_at,
    dbr.updated_at AS burn_rate_denominator_unit_updated_at,
    rb.created_at AS burn_rate_created_at,
    rb.updated_at AS burn_rate_updated_at,
    pl.id AS product_line_id_joined,
    pl.name AS product_line_name,
    pl.description AS product_line_description,
    pl.notes AS product_line_notes,
    pl.is_commission_exempt AS product_line_is_commission_exempt,
    pl.is_freight_exempt AS product_line_is_freight_exempt,
    pl.unit_group_id AS product_line_unit_group_id,
    pl.account_id AS product_line_account_id,
    pl.created_at AS product_line_created_at,
    pl.updated_at AS product_line_updated_at,
    pt.id AS product_type_id,
    pt.name AS product_type_name,
    pt.code AS product_type_code_joined,
    pt.created_at AS product_type_created_at,
    pt.updated_at AS product_type_updated_at
FROM product p
JOIN item i ON i.id = p.item_id
JOIN item_category ic ON ic.id = i.item_category_id
JOIN unit_group cat_ug ON cat_ug.id = ic.unit_group_id
JOIN rate rv ON rv.id = i.unit_value_id
JOIN unit nvu ON nvu.id = rv.numerator_unit_id
JOIN unit dvu ON dvu.id = rv.denominator_unit_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN unit ncu ON ncu.id = rc.numerator_unit_id
JOIN unit dcu ON dcu.id = rc.denominator_unit_id
JOIN rate rb ON rb.id = i.burn_rate_id
JOIN unit nbr ON nbr.id = rb.numerator_unit_id
JOIN unit dbr ON dbr.id = rb.denominator_unit_id
LEFT JOIN product_line pl ON pl.id = p.product_line_id
JOIN product_type pt ON pt.code = p.product_type_code
WHERE i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL
AND (
    sqlc.narg('search_query') IS NULL
    OR i.sku LIKE sqlc.narg('search_query')
    OR i.description LIKE sqlc.narg('search_query')
)
AND (
    (sqlc.arg('include_product_line_filter') = false AND sqlc.arg('include_customer_filter') = false)
    OR (sqlc.arg('include_product_line_filter') = true AND p.product_line_id IN (sqlc.slice('product_line_ids')))
    OR (sqlc.arg('include_customer_filter') = true AND (
        -- Path 1: Direct account relation product lines
        p.product_line_id IN (
            SELECT arpl.product_line_id
            FROM account_relation_product_line arpl
            JOIN account_relation ar ON ar.id = arpl.account_relation_id
            WHERE ar.owner_account_id = i.account_id
            AND ar.counterparty_account_id IN (sqlc.slice('customer_ids'))
            AND ar.account_relation_role_code = 'customer'
        )
        -- Path 2: Account group product lines via account group on account relation
        OR p.product_line_id IN (
            SELECT agpl.product_line_id
            FROM account_group_product_line agpl
            JOIN account_relation ar ON ar.account_group_id = agpl.account_group_id
            WHERE ar.owner_account_id = i.account_id
            AND ar.counterparty_account_id IN (sqlc.slice('customer_ids'))
            AND ar.account_relation_role_code = 'customer'
        )
        -- Path 3: Account group product lines via price group on account relation
        OR p.product_line_id IN (
            SELECT agpl.product_line_id
            FROM account_group_product_line agpl
            JOIN account_relation_price_group arpg ON arpg.account_group_id = agpl.account_group_id
            JOIN account_relation ar ON ar.id = arpg.account_relation_id
            WHERE ar.owner_account_id = i.account_id
            AND ar.counterparty_account_id IN (sqlc.slice('customer_ids'))
            AND ar.account_relation_role_code = 'customer'
        )
    ))
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
AND p.product_type_code = 'sale'
AND (
    sqlc.narg('is_portal_ready') IS NULL
    OR p.is_portal_ready = sqlc.narg('is_portal_ready')
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
    OR p.created_at < sqlc.narg('cursor_created_at')
    OR (p.created_at = sqlc.narg('cursor_created_at') AND p.id < sqlc.narg('cursor_id'))
)
ORDER BY p.created_at DESC, p.id DESC
LIMIT ?;

-- name: ListProductsFullBackward :many
SELECT
    p.id,
    p.product_type_code,
    p.is_portal_ready,
    p.product_line_id,
    p.item_id,
    p.created_at,
    p.updated_at,
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
    ic.created_at AS category_created_at,
    ic.updated_at AS category_updated_at,
    cat_ug.name AS category_unit_group_name,
    cat_ug.unit_type_code AS category_unit_group_type,
    cat_ug.created_at AS category_unit_group_created_at,
    cat_ug.updated_at AS category_unit_group_updated_at,
    rv.id AS unit_value_rate_id,
    rv.value AS unit_value_rate_value,
    rv.numerator_unit_id AS unit_value_numerator_unit_id,
    nvu.name AS unit_value_numerator_unit_name,
    nvu.abbreviation AS unit_value_numerator_unit_abbreviation,
    nvu.unit_dimension_code AS unit_value_numerator_unit_type,
    nvu.ratio_numerator AS unit_value_numerator_unit_ratio_numerator,
    nvu.ratio_denominator AS unit_value_numerator_unit_ratio_denominator,
    nvu.offset_numerator AS unit_value_numerator_unit_offset_numerator,
    nvu.offset_denominator AS unit_value_numerator_unit_offset_denominator,
    nvu.created_at AS unit_value_numerator_unit_created_at,
    nvu.updated_at AS unit_value_numerator_unit_updated_at,
    rv.denominator_unit_id AS unit_value_denominator_unit_id,
    dvu.name AS unit_value_denominator_unit_name,
    dvu.abbreviation AS unit_value_denominator_unit_abbreviation,
    dvu.unit_dimension_code AS unit_value_denominator_unit_type,
    dvu.ratio_numerator AS unit_value_denominator_unit_ratio_numerator,
    dvu.ratio_denominator AS unit_value_denominator_unit_ratio_denominator,
    dvu.offset_numerator AS unit_value_denominator_unit_offset_numerator,
    dvu.offset_denominator AS unit_value_denominator_unit_offset_denominator,
    dvu.created_at AS unit_value_denominator_unit_created_at,
    dvu.updated_at AS unit_value_denominator_unit_updated_at,
    rv.created_at AS unit_value_created_at,
    rv.updated_at AS unit_value_updated_at,
    rc.id AS unit_cost_rate_id,
    rc.value AS unit_cost_rate_value,
    rc.numerator_unit_id AS unit_cost_numerator_unit_id,
    ncu.name AS unit_cost_numerator_unit_name,
    ncu.abbreviation AS unit_cost_numerator_unit_abbreviation,
    ncu.unit_dimension_code AS unit_cost_numerator_unit_type,
    ncu.ratio_numerator AS unit_cost_numerator_unit_ratio_numerator,
    ncu.ratio_denominator AS unit_cost_numerator_unit_ratio_denominator,
    ncu.offset_numerator AS unit_cost_numerator_unit_offset_numerator,
    ncu.offset_denominator AS unit_cost_numerator_unit_offset_denominator,
    ncu.created_at AS unit_cost_numerator_unit_created_at,
    ncu.updated_at AS unit_cost_numerator_unit_updated_at,
    rc.denominator_unit_id AS unit_cost_denominator_unit_id,
    dcu.name AS unit_cost_denominator_unit_name,
    dcu.abbreviation AS unit_cost_denominator_unit_abbreviation,
    dcu.unit_dimension_code AS unit_cost_denominator_unit_type,
    dcu.ratio_numerator AS unit_cost_denominator_unit_ratio_numerator,
    dcu.ratio_denominator AS unit_cost_denominator_unit_ratio_denominator,
    dcu.offset_numerator AS unit_cost_denominator_unit_offset_numerator,
    dcu.offset_denominator AS unit_cost_denominator_unit_offset_denominator,
    dcu.created_at AS unit_cost_denominator_unit_created_at,
    dcu.updated_at AS unit_cost_denominator_unit_updated_at,
    rc.created_at AS unit_cost_created_at,
    rc.updated_at AS unit_cost_updated_at,
    rb.id AS burn_rate_id_joined,
    rb.value AS burn_rate_value,
    rb.numerator_unit_id AS burn_rate_numerator_unit_id,
    nbr.name AS burn_rate_numerator_unit_name,
    nbr.abbreviation AS burn_rate_numerator_unit_abbreviation,
    nbr.unit_dimension_code AS burn_rate_numerator_unit_type,
    nbr.ratio_numerator AS burn_rate_numerator_unit_ratio_numerator,
    nbr.ratio_denominator AS burn_rate_numerator_unit_ratio_denominator,
    nbr.offset_numerator AS burn_rate_numerator_unit_offset_numerator,
    nbr.offset_denominator AS burn_rate_numerator_unit_offset_denominator,
    nbr.created_at AS burn_rate_numerator_unit_created_at,
    nbr.updated_at AS burn_rate_numerator_unit_updated_at,
    rb.denominator_unit_id AS burn_rate_denominator_unit_id,
    dbr.name AS burn_rate_denominator_unit_name,
    dbr.abbreviation AS burn_rate_denominator_unit_abbreviation,
    dbr.unit_dimension_code AS burn_rate_denominator_unit_type,
    dbr.ratio_numerator AS burn_rate_denominator_unit_ratio_numerator,
    dbr.ratio_denominator AS burn_rate_denominator_unit_ratio_denominator,
    dbr.offset_numerator AS burn_rate_denominator_unit_offset_numerator,
    dbr.offset_denominator AS burn_rate_denominator_unit_offset_denominator,
    dbr.created_at AS burn_rate_denominator_unit_created_at,
    dbr.updated_at AS burn_rate_denominator_unit_updated_at,
    rb.created_at AS burn_rate_created_at,
    rb.updated_at AS burn_rate_updated_at,
    pl.id AS product_line_id_joined,
    pl.name AS product_line_name,
    pl.description AS product_line_description,
    pl.notes AS product_line_notes,
    pl.is_commission_exempt AS product_line_is_commission_exempt,
    pl.is_freight_exempt AS product_line_is_freight_exempt,
    pl.unit_group_id AS product_line_unit_group_id,
    pl.account_id AS product_line_account_id,
    pl.created_at AS product_line_created_at,
    pl.updated_at AS product_line_updated_at,
    pt.id AS product_type_id,
    pt.name AS product_type_name,
    pt.code AS product_type_code_joined,
    pt.created_at AS product_type_created_at,
    pt.updated_at AS product_type_updated_at
FROM product p
JOIN item i ON i.id = p.item_id
JOIN item_category ic ON ic.id = i.item_category_id
JOIN unit_group cat_ug ON cat_ug.id = ic.unit_group_id
JOIN rate rv ON rv.id = i.unit_value_id
JOIN unit nvu ON nvu.id = rv.numerator_unit_id
JOIN unit dvu ON dvu.id = rv.denominator_unit_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN unit ncu ON ncu.id = rc.numerator_unit_id
JOIN unit dcu ON dcu.id = rc.denominator_unit_id
JOIN rate rb ON rb.id = i.burn_rate_id
JOIN unit nbr ON nbr.id = rb.numerator_unit_id
JOIN unit dbr ON dbr.id = rb.denominator_unit_id
LEFT JOIN product_line pl ON pl.id = p.product_line_id
JOIN product_type pt ON pt.code = p.product_type_code
WHERE i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL
AND (
    sqlc.narg('search_query') IS NULL
    OR i.sku LIKE sqlc.narg('search_query')
    OR i.description LIKE sqlc.narg('search_query')
)
AND (
    (sqlc.arg('include_product_line_filter') = false AND sqlc.arg('include_customer_filter') = false)
    OR (sqlc.arg('include_product_line_filter') = true AND p.product_line_id IN (sqlc.slice('product_line_ids')))
    OR (sqlc.arg('include_customer_filter') = true AND (
        -- Path 1: Direct account relation product lines
        p.product_line_id IN (
            SELECT arpl.product_line_id
            FROM account_relation_product_line arpl
            JOIN account_relation ar ON ar.id = arpl.account_relation_id
            WHERE ar.owner_account_id = i.account_id
            AND ar.counterparty_account_id IN (sqlc.slice('customer_ids'))
            AND ar.account_relation_role_code = 'customer'
        )
        -- Path 2: Account group product lines via account group on account relation
        OR p.product_line_id IN (
            SELECT agpl.product_line_id
            FROM account_group_product_line agpl
            JOIN account_relation ar ON ar.account_group_id = agpl.account_group_id
            WHERE ar.owner_account_id = i.account_id
            AND ar.counterparty_account_id IN (sqlc.slice('customer_ids'))
            AND ar.account_relation_role_code = 'customer'
        )
        -- Path 3: Account group product lines via price group on account relation
        OR p.product_line_id IN (
            SELECT agpl.product_line_id
            FROM account_group_product_line agpl
            JOIN account_relation_price_group arpg ON arpg.account_group_id = agpl.account_group_id
            JOIN account_relation ar ON ar.id = arpg.account_relation_id
            WHERE ar.owner_account_id = i.account_id
            AND ar.counterparty_account_id IN (sqlc.slice('customer_ids'))
            AND ar.account_relation_role_code = 'customer'
        )
    ))
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
AND p.product_type_code = 'sale'
AND (
    sqlc.narg('is_portal_ready') IS NULL
    OR p.is_portal_ready = sqlc.narg('is_portal_ready')
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
    p.created_at > sqlc.arg('cursor_created_at')
    OR (p.created_at = sqlc.arg('cursor_created_at') AND p.id > sqlc.arg('cursor_id'))
)
ORDER BY p.created_at ASC, p.id ASC
LIMIT ?;

-- name: ListProductsFullForwardBase :many
SELECT
    p.id,
    p.product_type_code,
    p.is_portal_ready,
    p.product_line_id,
    p.item_id,
    p.created_at,
    p.updated_at,
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
    ic.created_at AS category_created_at,
    ic.updated_at AS category_updated_at,
    pt.id AS product_type_id,
    pt.name AS product_type_name,
    pt.code AS product_type_code_joined,
    pt.created_at AS product_type_created_at,
    pt.updated_at AS product_type_updated_at
FROM product p
JOIN item i ON i.id = p.item_id
JOIN item_category ic ON ic.id = i.item_category_id
JOIN product_type pt ON pt.code = p.product_type_code
WHERE i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL
AND (
    sqlc.narg('search_query') IS NULL
    OR i.sku LIKE sqlc.narg('search_query')
    OR i.description LIKE sqlc.narg('search_query')
)
AND (
    (sqlc.arg('include_product_line_filter') = false AND sqlc.arg('include_customer_filter') = false)
    OR (sqlc.arg('include_product_line_filter') = true AND p.product_line_id IN (sqlc.slice('product_line_ids')))
    OR (sqlc.arg('include_customer_filter') = true AND (
        -- Path 1: Direct account relation product lines
        p.product_line_id IN (
            SELECT arpl.product_line_id
            FROM account_relation_product_line arpl
            JOIN account_relation ar ON ar.id = arpl.account_relation_id
            WHERE ar.owner_account_id = i.account_id
            AND ar.counterparty_account_id IN (sqlc.slice('customer_ids'))
            AND ar.account_relation_role_code = 'customer'
        )
        -- Path 2: Account group product lines via account group on account relation
        OR p.product_line_id IN (
            SELECT agpl.product_line_id
            FROM account_group_product_line agpl
            JOIN account_relation ar ON ar.account_group_id = agpl.account_group_id
            WHERE ar.owner_account_id = i.account_id
            AND ar.counterparty_account_id IN (sqlc.slice('customer_ids'))
            AND ar.account_relation_role_code = 'customer'
        )
        -- Path 3: Account group product lines via price group on account relation
        OR p.product_line_id IN (
            SELECT agpl.product_line_id
            FROM account_group_product_line agpl
            JOIN account_relation_price_group arpg ON arpg.account_group_id = agpl.account_group_id
            JOIN account_relation ar ON ar.id = arpg.account_relation_id
            WHERE ar.owner_account_id = i.account_id
            AND ar.counterparty_account_id IN (sqlc.slice('customer_ids'))
            AND ar.account_relation_role_code = 'customer'
        )
    ))
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
AND p.product_type_code = 'sale'
AND (
    sqlc.narg('is_portal_ready') IS NULL
    OR p.is_portal_ready = sqlc.narg('is_portal_ready')
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
    OR (
        (sqlc.narg('cursor_match_tier') IS NULL AND (
            p.created_at < sqlc.narg('cursor_created_at')
            OR (p.created_at = sqlc.narg('cursor_created_at') AND p.id < sqlc.narg('cursor_id'))
        ))
        OR (sqlc.narg('cursor_match_tier') IS NOT NULL AND (
            (CASE
                WHEN CAST(sqlc.narg('search_exact') AS CHAR) IS NULL THEN 0
                WHEN i.sku COLLATE utf8mb4_general_ci = CAST(sqlc.narg('search_exact') AS CHAR) THEN 0
                WHEN i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR), ' %')
                    OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT(CAST(sqlc.narg('search_exact') AS CHAR), ' %')
                    OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR)) THEN 1
                WHEN sqlc.narg('search_prefix') IS NOT NULL AND i.sku COLLATE utf8mb4_general_ci LIKE sqlc.narg('search_prefix') THEN 2
                ELSE 3
            END) > CAST(sqlc.narg('cursor_match_tier') AS SIGNED)
            OR (
                (CASE
                    WHEN CAST(sqlc.narg('search_exact') AS CHAR) IS NULL THEN 0
                    WHEN i.sku COLLATE utf8mb4_general_ci = CAST(sqlc.narg('search_exact') AS CHAR) THEN 0
                    WHEN i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR), ' %')
                        OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT(CAST(sqlc.narg('search_exact') AS CHAR), ' %')
                        OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR)) THEN 1
                    WHEN sqlc.narg('search_prefix') IS NOT NULL AND i.sku COLLATE utf8mb4_general_ci LIKE sqlc.narg('search_prefix') THEN 2
                    ELSE 3
                END) = CAST(sqlc.narg('cursor_match_tier') AS SIGNED)
                AND (
                    p.created_at < sqlc.narg('cursor_created_at')
                    OR (p.created_at = sqlc.narg('cursor_created_at') AND p.id < sqlc.narg('cursor_id'))
                )
            )
        ))
    )
)
ORDER BY
    CASE
        WHEN CAST(sqlc.narg('search_exact') AS CHAR) IS NULL THEN 0
        WHEN i.sku COLLATE utf8mb4_general_ci = CAST(sqlc.narg('search_exact') AS CHAR) THEN 0
        WHEN i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR), ' %')
            OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT(CAST(sqlc.narg('search_exact') AS CHAR), ' %')
            OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR)) THEN 1
        WHEN sqlc.narg('search_prefix') IS NOT NULL AND i.sku COLLATE utf8mb4_general_ci LIKE sqlc.narg('search_prefix') THEN 2
        ELSE 3
    END ASC,
    p.created_at DESC,
    p.id DESC
LIMIT ?;

-- name: ListProductsFullBackwardBase :many
SELECT
    p.id,
    p.product_type_code,
    p.is_portal_ready,
    p.product_line_id,
    p.item_id,
    p.created_at,
    p.updated_at,
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
    ic.created_at AS category_created_at,
    ic.updated_at AS category_updated_at,
    pt.id AS product_type_id,
    pt.name AS product_type_name,
    pt.code AS product_type_code_joined,
    pt.created_at AS product_type_created_at,
    pt.updated_at AS product_type_updated_at
FROM product p
JOIN item i ON i.id = p.item_id
JOIN item_category ic ON ic.id = i.item_category_id
JOIN product_type pt ON pt.code = p.product_type_code
WHERE i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL
AND (
    sqlc.narg('search_query') IS NULL
    OR i.sku LIKE sqlc.narg('search_query')
    OR i.description LIKE sqlc.narg('search_query')
)
AND (
    (sqlc.arg('include_product_line_filter') = false AND sqlc.arg('include_customer_filter') = false)
    OR (sqlc.arg('include_product_line_filter') = true AND p.product_line_id IN (sqlc.slice('product_line_ids')))
    OR (sqlc.arg('include_customer_filter') = true AND (
        -- Path 1: Direct account relation product lines
        p.product_line_id IN (
            SELECT arpl.product_line_id
            FROM account_relation_product_line arpl
            JOIN account_relation ar ON ar.id = arpl.account_relation_id
            WHERE ar.owner_account_id = i.account_id
            AND ar.counterparty_account_id IN (sqlc.slice('customer_ids'))
            AND ar.account_relation_role_code = 'customer'
        )
        -- Path 2: Account group product lines via account group on account relation
        OR p.product_line_id IN (
            SELECT agpl.product_line_id
            FROM account_group_product_line agpl
            JOIN account_relation ar ON ar.account_group_id = agpl.account_group_id
            WHERE ar.owner_account_id = i.account_id
            AND ar.counterparty_account_id IN (sqlc.slice('customer_ids'))
            AND ar.account_relation_role_code = 'customer'
        )
        -- Path 3: Account group product lines via price group on account relation
        OR p.product_line_id IN (
            SELECT agpl.product_line_id
            FROM account_group_product_line agpl
            JOIN account_relation_price_group arpg ON arpg.account_group_id = agpl.account_group_id
            JOIN account_relation ar ON ar.id = arpg.account_relation_id
            WHERE ar.owner_account_id = i.account_id
            AND ar.counterparty_account_id IN (sqlc.slice('customer_ids'))
            AND ar.account_relation_role_code = 'customer'
        )
    ))
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
AND p.product_type_code = 'sale'
AND (
    sqlc.narg('is_portal_ready') IS NULL
    OR p.is_portal_ready = sqlc.narg('is_portal_ready')
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
    (sqlc.narg('cursor_match_tier') IS NULL AND (
        p.created_at > sqlc.arg('cursor_created_at')
        OR (p.created_at = sqlc.arg('cursor_created_at') AND p.id > sqlc.arg('cursor_id'))
    ))
    OR (sqlc.narg('cursor_match_tier') IS NOT NULL AND (
        (CASE
            WHEN CAST(sqlc.narg('search_exact') AS CHAR) IS NULL THEN 0
            WHEN i.sku COLLATE utf8mb4_general_ci = CAST(sqlc.narg('search_exact') AS CHAR) THEN 0
            WHEN i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR), ' %')
                OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT(CAST(sqlc.narg('search_exact') AS CHAR), ' %')
                OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR)) THEN 1
            WHEN sqlc.narg('search_prefix') IS NOT NULL AND i.sku COLLATE utf8mb4_general_ci LIKE sqlc.narg('search_prefix') THEN 2
            ELSE 3
        END) < CAST(sqlc.narg('cursor_match_tier') AS SIGNED)
        OR (
            (CASE
                WHEN CAST(sqlc.narg('search_exact') AS CHAR) IS NULL THEN 0
                WHEN i.sku COLLATE utf8mb4_general_ci = CAST(sqlc.narg('search_exact') AS CHAR) THEN 0
                WHEN i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR), ' %')
                    OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT(CAST(sqlc.narg('search_exact') AS CHAR), ' %')
                    OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR)) THEN 1
                WHEN sqlc.narg('search_prefix') IS NOT NULL AND i.sku COLLATE utf8mb4_general_ci LIKE sqlc.narg('search_prefix') THEN 2
                ELSE 3
            END) = CAST(sqlc.narg('cursor_match_tier') AS SIGNED)
            AND (
                p.created_at > sqlc.arg('cursor_created_at')
                OR (p.created_at = sqlc.arg('cursor_created_at') AND p.id > sqlc.arg('cursor_id'))
            )
        )
    ))
)
ORDER BY
    CASE
        WHEN CAST(sqlc.narg('search_exact') AS CHAR) IS NULL THEN 0
        WHEN i.sku COLLATE utf8mb4_general_ci = CAST(sqlc.narg('search_exact') AS CHAR) THEN 0
        WHEN i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR), ' %')
            OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT(CAST(sqlc.narg('search_exact') AS CHAR), ' %')
            OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR)) THEN 1
        WHEN sqlc.narg('search_prefix') IS NOT NULL AND i.sku COLLATE utf8mb4_general_ci LIKE sqlc.narg('search_prefix') THEN 2
        ELSE 3
    END DESC,
    p.created_at ASC,
    p.id ASC
LIMIT ?;

-- name: GetProductByIDBase :one
SELECT
    p.id,
    p.product_type_code,
    p.is_portal_ready,
    p.product_line_id,
    p.item_id,
    p.created_at,
    p.updated_at,
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
    ic.created_at AS category_created_at,
    ic.updated_at AS category_updated_at,
    pt.id AS product_type_id,
    pt.name AS product_type_name,
    pt.code AS product_type_code_joined,
    pt.created_at AS product_type_created_at,
    pt.updated_at AS product_type_updated_at
FROM product p
JOIN item i ON i.id = p.item_id
JOIN item_category ic ON ic.id = i.item_category_id
JOIN product_type pt ON pt.code = p.product_type_code
WHERE p.id = sqlc.arg('id')
AND i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL;

-- name: GetProductByID :one
SELECT
    p.id,
    p.product_type_code,
    p.is_portal_ready,
    p.product_line_id,
    p.item_id,
    p.created_at,
    p.updated_at,
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
    ic.created_at AS category_created_at,
    ic.updated_at AS category_updated_at,
    cat_ug.name AS category_unit_group_name,
    cat_ug.unit_type_code AS category_unit_group_type,
    cat_ug.created_at AS category_unit_group_created_at,
    cat_ug.updated_at AS category_unit_group_updated_at,
    rv.id AS unit_value_rate_id,
    rv.value AS unit_value_rate_value,
    rv.numerator_unit_id AS unit_value_numerator_unit_id,
    nvu.name AS unit_value_numerator_unit_name,
    nvu.abbreviation AS unit_value_numerator_unit_abbreviation,
    nvu.unit_dimension_code AS unit_value_numerator_unit_type,
    nvu.ratio_numerator AS unit_value_numerator_unit_ratio_numerator,
    nvu.ratio_denominator AS unit_value_numerator_unit_ratio_denominator,
    nvu.offset_numerator AS unit_value_numerator_unit_offset_numerator,
    nvu.offset_denominator AS unit_value_numerator_unit_offset_denominator,
    nvu.created_at AS unit_value_numerator_unit_created_at,
    nvu.updated_at AS unit_value_numerator_unit_updated_at,
    rv.denominator_unit_id AS unit_value_denominator_unit_id,
    dvu.name AS unit_value_denominator_unit_name,
    dvu.abbreviation AS unit_value_denominator_unit_abbreviation,
    dvu.unit_dimension_code AS unit_value_denominator_unit_type,
    dvu.ratio_numerator AS unit_value_denominator_unit_ratio_numerator,
    dvu.ratio_denominator AS unit_value_denominator_unit_ratio_denominator,
    dvu.offset_numerator AS unit_value_denominator_unit_offset_numerator,
    dvu.offset_denominator AS unit_value_denominator_unit_offset_denominator,
    dvu.created_at AS unit_value_denominator_unit_created_at,
    dvu.updated_at AS unit_value_denominator_unit_updated_at,
    rv.created_at AS unit_value_created_at,
    rv.updated_at AS unit_value_updated_at,
    rc.id AS unit_cost_rate_id,
    rc.value AS unit_cost_rate_value,
    rc.numerator_unit_id AS unit_cost_numerator_unit_id,
    ncu.name AS unit_cost_numerator_unit_name,
    ncu.abbreviation AS unit_cost_numerator_unit_abbreviation,
    ncu.unit_dimension_code AS unit_cost_numerator_unit_type,
    ncu.ratio_numerator AS unit_cost_numerator_unit_ratio_numerator,
    ncu.ratio_denominator AS unit_cost_numerator_unit_ratio_denominator,
    ncu.offset_numerator AS unit_cost_numerator_unit_offset_numerator,
    ncu.offset_denominator AS unit_cost_numerator_unit_offset_denominator,
    ncu.created_at AS unit_cost_numerator_unit_created_at,
    ncu.updated_at AS unit_cost_numerator_unit_updated_at,
    rc.denominator_unit_id AS unit_cost_denominator_unit_id,
    dcu.name AS unit_cost_denominator_unit_name,
    dcu.abbreviation AS unit_cost_denominator_unit_abbreviation,
    dcu.unit_dimension_code AS unit_cost_denominator_unit_type,
    dcu.ratio_numerator AS unit_cost_denominator_unit_ratio_numerator,
    dcu.ratio_denominator AS unit_cost_denominator_unit_ratio_denominator,
    dcu.offset_numerator AS unit_cost_denominator_unit_offset_numerator,
    dcu.offset_denominator AS unit_cost_denominator_unit_offset_denominator,
    dcu.created_at AS unit_cost_denominator_unit_created_at,
    dcu.updated_at AS unit_cost_denominator_unit_updated_at,
    rc.created_at AS unit_cost_created_at,
    rc.updated_at AS unit_cost_updated_at,
    rb.id AS burn_rate_id_joined,
    rb.value AS burn_rate_value,
    rb.numerator_unit_id AS burn_rate_numerator_unit_id,
    nbr.name AS burn_rate_numerator_unit_name,
    nbr.abbreviation AS burn_rate_numerator_unit_abbreviation,
    nbr.unit_dimension_code AS burn_rate_numerator_unit_type,
    nbr.ratio_numerator AS burn_rate_numerator_unit_ratio_numerator,
    nbr.ratio_denominator AS burn_rate_numerator_unit_ratio_denominator,
    nbr.offset_numerator AS burn_rate_numerator_unit_offset_numerator,
    nbr.offset_denominator AS burn_rate_numerator_unit_offset_denominator,
    nbr.created_at AS burn_rate_numerator_unit_created_at,
    nbr.updated_at AS burn_rate_numerator_unit_updated_at,
    rb.denominator_unit_id AS burn_rate_denominator_unit_id,
    dbr.name AS burn_rate_denominator_unit_name,
    dbr.abbreviation AS burn_rate_denominator_unit_abbreviation,
    dbr.unit_dimension_code AS burn_rate_denominator_unit_type,
    dbr.ratio_numerator AS burn_rate_denominator_unit_ratio_numerator,
    dbr.ratio_denominator AS burn_rate_denominator_unit_ratio_denominator,
    dbr.offset_numerator AS burn_rate_denominator_unit_offset_numerator,
    dbr.offset_denominator AS burn_rate_denominator_unit_offset_denominator,
    dbr.created_at AS burn_rate_denominator_unit_created_at,
    dbr.updated_at AS burn_rate_denominator_unit_updated_at,
    rb.created_at AS burn_rate_created_at,
    rb.updated_at AS burn_rate_updated_at,
    pl.id AS product_line_id_joined,
    pl.name AS product_line_name,
    pl.description AS product_line_description,
    pl.notes AS product_line_notes,
    pl.is_commission_exempt AS product_line_is_commission_exempt,
    pl.is_freight_exempt AS product_line_is_freight_exempt,
    pl.unit_group_id AS product_line_unit_group_id,
    pl.account_id AS product_line_account_id,
    pl.created_at AS product_line_created_at,
    pl.updated_at AS product_line_updated_at,
    pt.id AS product_type_id,
    pt.name AS product_type_name,
    pt.code AS product_type_code_joined,
    pt.created_at AS product_type_created_at,
    pt.updated_at AS product_type_updated_at
FROM product p
JOIN item i ON i.id = p.item_id
JOIN item_category ic ON ic.id = i.item_category_id
JOIN unit_group cat_ug ON cat_ug.id = ic.unit_group_id
JOIN rate rv ON rv.id = i.unit_value_id
JOIN unit nvu ON nvu.id = rv.numerator_unit_id
JOIN unit dvu ON dvu.id = rv.denominator_unit_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN unit ncu ON ncu.id = rc.numerator_unit_id
JOIN unit dcu ON dcu.id = rc.denominator_unit_id
JOIN rate rb ON rb.id = i.burn_rate_id
JOIN unit nbr ON nbr.id = rb.numerator_unit_id
JOIN unit dbr ON dbr.id = rb.denominator_unit_id
LEFT JOIN product_line pl ON pl.id = p.product_line_id
JOIN product_type pt ON pt.code = p.product_type_code
WHERE p.id = sqlc.arg('id')
AND i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL;

-- name: InsertProduct :exec
INSERT INTO product (
    id,
    item_id,
    product_type_code,
    product_line_id,
    is_portal_ready,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('item_id'),
    sqlc.arg('product_type_code'),
    sqlc.narg('product_line_id'),
    sqlc.arg('is_portal_ready'),
    NOW(3),
    NOW(3)
);

-- name: UpdateProductFields :execresult
UPDATE product SET
    is_portal_ready = COALESCE(sqlc.narg('is_portal_ready'), is_portal_ready),
    updated_at = NOW(3)
WHERE product.id = sqlc.arg('id')
AND item_id IN (
    SELECT id FROM item WHERE account_id = sqlc.arg('account_id') AND deleted_at IS NULL
);

-- name: SoftDeleteProductByID :execresult
UPDATE item i
JOIN product p ON p.item_id = i.id
SET i.deleted_at = NOW(3)
WHERE p.id = sqlc.arg('id')
AND i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL;

-- name: ChangeProductLineByID :execresult
UPDATE product SET
    product_line_id = sqlc.arg('product_line_id'),
    updated_at = NOW(3)
WHERE (product.id = sqlc.arg('id') OR product.item_id = sqlc.arg('id'))
AND item_id IN (
    SELECT id FROM item WHERE account_id = sqlc.arg('account_id') AND deleted_at IS NULL
);

-- name: GetProductsByIDs :many
SELECT
    p.id,
    p.product_type_code,
    p.is_portal_ready,
    p.product_line_id,
    p.item_id,
    p.created_at,
    p.updated_at
FROM product p
JOIN item i ON i.id = p.item_id
WHERE p.id IN (sqlc.slice('ids'))
AND i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL;

-- name: FindProductsBySKUs :many
SELECT
    p.id,
    p.product_type_code,
    p.is_portal_ready,
    p.product_line_id,
    p.item_id,
    p.created_at,
    p.updated_at,
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
    ic.created_at AS category_created_at,
    ic.updated_at AS category_updated_at,
    cat_ug.name AS category_unit_group_name,
    cat_ug.unit_type_code AS category_unit_group_type,
    cat_ug.created_at AS category_unit_group_created_at,
    cat_ug.updated_at AS category_unit_group_updated_at,
    rv.id AS unit_value_rate_id,
    rv.value AS unit_value_rate_value,
    rv.numerator_unit_id AS unit_value_numerator_unit_id,
    nvu.name AS unit_value_numerator_unit_name,
    nvu.abbreviation AS unit_value_numerator_unit_abbreviation,
    nvu.unit_dimension_code AS unit_value_numerator_unit_type,
    nvu.ratio_numerator AS unit_value_numerator_unit_ratio_numerator,
    nvu.ratio_denominator AS unit_value_numerator_unit_ratio_denominator,
    nvu.offset_numerator AS unit_value_numerator_unit_offset_numerator,
    nvu.offset_denominator AS unit_value_numerator_unit_offset_denominator,
    nvu.created_at AS unit_value_numerator_unit_created_at,
    nvu.updated_at AS unit_value_numerator_unit_updated_at,
    rv.denominator_unit_id AS unit_value_denominator_unit_id,
    dvu.name AS unit_value_denominator_unit_name,
    dvu.abbreviation AS unit_value_denominator_unit_abbreviation,
    dvu.unit_dimension_code AS unit_value_denominator_unit_type,
    dvu.ratio_numerator AS unit_value_denominator_unit_ratio_numerator,
    dvu.ratio_denominator AS unit_value_denominator_unit_ratio_denominator,
    dvu.offset_numerator AS unit_value_denominator_unit_offset_numerator,
    dvu.offset_denominator AS unit_value_denominator_unit_offset_denominator,
    dvu.created_at AS unit_value_denominator_unit_created_at,
    dvu.updated_at AS unit_value_denominator_unit_updated_at,
    rv.created_at AS unit_value_created_at,
    rv.updated_at AS unit_value_updated_at,
    rc.id AS unit_cost_rate_id,
    rc.value AS unit_cost_rate_value,
    rc.numerator_unit_id AS unit_cost_numerator_unit_id,
    ncu.name AS unit_cost_numerator_unit_name,
    ncu.abbreviation AS unit_cost_numerator_unit_abbreviation,
    ncu.unit_dimension_code AS unit_cost_numerator_unit_type,
    ncu.ratio_numerator AS unit_cost_numerator_unit_ratio_numerator,
    ncu.ratio_denominator AS unit_cost_numerator_unit_ratio_denominator,
    ncu.offset_numerator AS unit_cost_numerator_unit_offset_numerator,
    ncu.offset_denominator AS unit_cost_numerator_unit_offset_denominator,
    ncu.created_at AS unit_cost_numerator_unit_created_at,
    ncu.updated_at AS unit_cost_numerator_unit_updated_at,
    rc.denominator_unit_id AS unit_cost_denominator_unit_id,
    dcu.name AS unit_cost_denominator_unit_name,
    dcu.abbreviation AS unit_cost_denominator_unit_abbreviation,
    dcu.unit_dimension_code AS unit_cost_denominator_unit_type,
    dcu.ratio_numerator AS unit_cost_denominator_unit_ratio_numerator,
    dcu.ratio_denominator AS unit_cost_denominator_unit_ratio_denominator,
    dcu.offset_numerator AS unit_cost_denominator_unit_offset_numerator,
    dcu.offset_denominator AS unit_cost_denominator_unit_offset_denominator,
    dcu.created_at AS unit_cost_denominator_unit_created_at,
    dcu.updated_at AS unit_cost_denominator_unit_updated_at,
    rc.created_at AS unit_cost_created_at,
    rc.updated_at AS unit_cost_updated_at,
    rb.id AS burn_rate_id_joined,
    rb.value AS burn_rate_value,
    rb.numerator_unit_id AS burn_rate_numerator_unit_id,
    nbr.name AS burn_rate_numerator_unit_name,
    nbr.abbreviation AS burn_rate_numerator_unit_abbreviation,
    nbr.unit_dimension_code AS burn_rate_numerator_unit_type,
    nbr.ratio_numerator AS burn_rate_numerator_unit_ratio_numerator,
    nbr.ratio_denominator AS burn_rate_numerator_unit_ratio_denominator,
    nbr.offset_numerator AS burn_rate_numerator_unit_offset_numerator,
    nbr.offset_denominator AS burn_rate_numerator_unit_offset_denominator,
    nbr.created_at AS burn_rate_numerator_unit_created_at,
    nbr.updated_at AS burn_rate_numerator_unit_updated_at,
    rb.denominator_unit_id AS burn_rate_denominator_unit_id,
    dbr.name AS burn_rate_denominator_unit_name,
    dbr.abbreviation AS burn_rate_denominator_unit_abbreviation,
    dbr.unit_dimension_code AS burn_rate_denominator_unit_type,
    dbr.ratio_numerator AS burn_rate_denominator_unit_ratio_numerator,
    dbr.ratio_denominator AS burn_rate_denominator_unit_ratio_denominator,
    dbr.offset_numerator AS burn_rate_denominator_unit_offset_numerator,
    dbr.offset_denominator AS burn_rate_denominator_unit_offset_denominator,
    dbr.created_at AS burn_rate_denominator_unit_created_at,
    dbr.updated_at AS burn_rate_denominator_unit_updated_at,
    rb.created_at AS burn_rate_created_at,
    rb.updated_at AS burn_rate_updated_at,
    pl.id AS product_line_id_joined,
    pl.name AS product_line_name,
    pl.description AS product_line_description,
    pl.notes AS product_line_notes,
    pl.is_commission_exempt AS product_line_is_commission_exempt,
    pl.is_freight_exempt AS product_line_is_freight_exempt,
    pl.unit_group_id AS product_line_unit_group_id,
    pl.account_id AS product_line_account_id,
    pl.created_at AS product_line_created_at,
    pl.updated_at AS product_line_updated_at,
    pt.id AS product_type_id,
    pt.name AS product_type_name,
    pt.code AS product_type_code_joined,
    pt.created_at AS product_type_created_at,
    pt.updated_at AS product_type_updated_at
FROM product p
JOIN item i ON i.id = p.item_id
JOIN item_category ic ON ic.id = i.item_category_id
JOIN unit_group cat_ug ON cat_ug.id = ic.unit_group_id
JOIN rate rv ON rv.id = i.unit_value_id
JOIN unit nvu ON nvu.id = rv.numerator_unit_id
JOIN unit dvu ON dvu.id = rv.denominator_unit_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN unit ncu ON ncu.id = rc.numerator_unit_id
JOIN unit dcu ON dcu.id = rc.denominator_unit_id
JOIN rate rb ON rb.id = i.burn_rate_id
JOIN unit nbr ON nbr.id = rb.numerator_unit_id
JOIN unit dbr ON dbr.id = rb.denominator_unit_id
LEFT JOIN product_line pl ON pl.id = p.product_line_id
JOIN product_type pt ON pt.code = p.product_type_code
WHERE i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL
AND i.sku IN (sqlc.slice('skus'));

-- name: CheckProductSKUExists :one
SELECT COUNT(*) AS cnt FROM item i
JOIN product p ON p.item_id = i.id
WHERE i.account_id = sqlc.arg('account_id')
AND i.sku = sqlc.arg('sku')
AND i.deleted_at IS NULL
AND (sqlc.narg('exclude_item_id') IS NULL OR i.id != sqlc.narg('exclude_item_id'));

-- name: ProductInsertRate :exec
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

-- name: ProductInsertItem :exec
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
    'product',
    sqlc.arg('item_category_id'),
    sqlc.arg('unit_value_id'),
    sqlc.arg('unit_cost_id'),
    sqlc.arg('burn_rate_id'),
    sqlc.arg('account_id'),
    false,
    NOW(3),
    NOW(3)
);

-- name: ProductUpdateItem :execresult
UPDATE item SET
    sku = COALESCE(sqlc.narg('sku'), sku),
    description = sqlc.narg('description'),
    notes = sqlc.narg('notes'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id')
AND deleted_at IS NULL;

-- name: ExportProductsWithFilters :many
SELECT
    p.id,
    p.product_type_code,
    p.is_portal_ready,
    p.product_line_id,
    p.item_id,
    p.created_at,
    p.updated_at,
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
    ic.created_at AS category_created_at,
    ic.updated_at AS category_updated_at,
    pt.id AS product_type_id,
    pt.name AS product_type_name,
    pt.code AS product_type_code_joined,
    pt.created_at AS product_type_created_at,
    pt.updated_at AS product_type_updated_at
FROM product p
JOIN item i ON i.id = p.item_id
JOIN item_category ic ON ic.id = i.item_category_id
JOIN product_type pt ON pt.code = p.product_type_code
WHERE i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL
AND (
    sqlc.narg('search_query') IS NULL
    OR i.sku LIKE sqlc.narg('search_query')
    OR i.description LIKE sqlc.narg('search_query')
)
AND (
    (sqlc.arg('include_product_line_filter') = false AND sqlc.arg('include_customer_filter') = false)
    OR (sqlc.arg('include_product_line_filter') = true AND p.product_line_id IN (sqlc.slice('product_line_ids')))
    OR (sqlc.arg('include_customer_filter') = true AND (
        p.product_line_id IN (
            SELECT arpl.product_line_id
            FROM account_relation_product_line arpl
            JOIN account_relation ar ON ar.id = arpl.account_relation_id
            WHERE ar.owner_account_id = i.account_id
            AND ar.counterparty_account_id IN (sqlc.slice('customer_ids'))
            AND ar.account_relation_role_code = 'customer'
        )
        OR p.product_line_id IN (
            SELECT agpl.product_line_id
            FROM account_group_product_line agpl
            JOIN account_relation ar ON ar.account_group_id = agpl.account_group_id
            WHERE ar.owner_account_id = i.account_id
            AND ar.counterparty_account_id IN (sqlc.slice('customer_ids'))
            AND ar.account_relation_role_code = 'customer'
        )
        OR p.product_line_id IN (
            SELECT agpl.product_line_id
            FROM account_group_product_line agpl
            JOIN account_relation_price_group arpg ON arpg.account_group_id = agpl.account_group_id
            JOIN account_relation ar ON ar.id = arpg.account_relation_id
            WHERE ar.owner_account_id = i.account_id
            AND ar.counterparty_account_id IN (sqlc.slice('customer_ids'))
            AND ar.account_relation_role_code = 'customer'
        )
    ))
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
AND p.product_type_code = 'sale'
AND (
    sqlc.narg('is_portal_ready') IS NULL
    OR p.is_portal_ready = sqlc.narg('is_portal_ready')
)
AND (
    sqlc.narg('start_date') IS NULL
    OR i.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR i.created_at <= sqlc.narg('end_date')
)
ORDER BY
    CASE
        WHEN CAST(sqlc.narg('search_exact') AS CHAR) IS NULL THEN 0
        WHEN i.sku COLLATE utf8mb4_general_ci = CAST(sqlc.narg('search_exact') AS CHAR) THEN 0
        WHEN i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR), ' %')
            OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT(CAST(sqlc.narg('search_exact') AS CHAR), ' %')
            OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR)) THEN 1
        WHEN sqlc.narg('search_prefix') IS NOT NULL AND i.sku COLLATE utf8mb4_general_ci LIKE sqlc.narg('search_prefix') THEN 2
        ELSE 3
    END ASC,
    p.created_at DESC,
    p.id DESC;
