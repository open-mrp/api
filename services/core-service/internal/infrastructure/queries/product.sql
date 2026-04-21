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
JOIN rate rv ON rv.id = i.unit_value_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN rate rb ON rb.id = i.burn_rate_id
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
JOIN rate rv ON rv.id = i.unit_value_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN rate rb ON rb.id = i.burn_rate_id
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

-- name: GetProductByItemID :one
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
JOIN rate rv ON rv.id = i.unit_value_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN rate rb ON rb.id = i.burn_rate_id
LEFT JOIN product_line pl ON pl.id = p.product_line_id
JOIN product_type pt ON pt.code = p.product_type_code
WHERE p.item_id = sqlc.arg('item_id')
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
WHERE item_id = sqlc.arg('item_id')
AND item_id IN (
    SELECT id FROM item WHERE account_id = sqlc.arg('account_id') AND deleted_at IS NULL
);

-- name: SoftDeleteProductByItemID :execresult
UPDATE item SET
    deleted_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id')
AND deleted_at IS NULL;

-- name: ChangeProductLineByItemID :execresult
UPDATE product SET
    product_line_id = sqlc.arg('product_line_id'),
    updated_at = NOW(3)
WHERE item_id = sqlc.arg('item_id')
AND item_id IN (
    SELECT id FROM item WHERE account_id = sqlc.arg('account_id') AND deleted_at IS NULL
);

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
JOIN rate rv ON rv.id = i.unit_value_id
JOIN rate rc ON rc.id = i.unit_cost_id
JOIN rate rb ON rb.id = i.burn_rate_id
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
    description = CASE WHEN sqlc.arg('update_description') = true THEN sqlc.narg('description') ELSE description END,
    notes = CASE WHEN sqlc.arg('update_notes') = true THEN sqlc.narg('notes') ELSE notes END,
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id')
AND deleted_at IS NULL;
