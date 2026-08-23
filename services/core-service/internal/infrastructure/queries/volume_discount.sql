-- name: ListVolumeDiscountsForward :many
SELECT
    qd.id,
    qd.name,
    qd.account_id,
    qd.created_at,
    qd.updated_at
FROM quantity_discount qd
WHERE qd.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR qd.name LIKE sqlc.narg('search_query')
    OR EXISTS (
        SELECT 1 FROM account_group_quantity_discount agqd
        JOIN account_group ag ON ag.id = agqd.account_group_id
        WHERE agqd.quantity_discount_id = qd.id
        AND ag.name LIKE sqlc.narg('search_query')
    )
    OR EXISTS (
        SELECT 1 FROM `_product_lines_quantity_discounts` plqd
        JOIN product_line pl ON pl.id = plqd.A
        WHERE plqd.B = qd.id
        AND pl.name LIKE sqlc.narg('search_query')
    )
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR qd.created_at < sqlc.narg('cursor_created_at')
    OR (qd.created_at = sqlc.narg('cursor_created_at') AND qd.id < sqlc.narg('cursor_id'))
)
ORDER BY qd.created_at DESC, qd.id DESC
LIMIT ?;

-- name: ListVolumeDiscountsBackward :many
SELECT
    qd.id,
    qd.name,
    qd.account_id,
    qd.created_at,
    qd.updated_at
FROM quantity_discount qd
WHERE qd.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR qd.name LIKE sqlc.narg('search_query')
    OR EXISTS (
        SELECT 1 FROM account_group_quantity_discount agqd
        JOIN account_group ag ON ag.id = agqd.account_group_id
        WHERE agqd.quantity_discount_id = qd.id
        AND ag.name LIKE sqlc.narg('search_query')
    )
    OR EXISTS (
        SELECT 1 FROM `_product_lines_quantity_discounts` plqd
        JOIN product_line pl ON pl.id = plqd.A
        WHERE plqd.B = qd.id
        AND pl.name LIKE sqlc.narg('search_query')
    )
)
AND (
    qd.created_at > sqlc.arg('cursor_created_at')
    OR (qd.created_at = sqlc.arg('cursor_created_at') AND qd.id > sqlc.arg('cursor_id'))
)
ORDER BY qd.created_at ASC, qd.id ASC
LIMIT ?;

-- name: ListVolumeDiscountsForCustomerForward :many
SELECT DISTINCT
    qd.id,
    qd.name,
    qd.account_id,
    qd.created_at,
    qd.updated_at
FROM quantity_discount qd
WHERE qd.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR qd.name LIKE sqlc.narg('search_query')
    OR EXISTS (
        SELECT 1 FROM account_group_quantity_discount agqd
        JOIN account_group ag ON ag.id = agqd.account_group_id
        WHERE agqd.quantity_discount_id = qd.id
        AND ag.name LIKE sqlc.narg('search_query')
    )
    OR EXISTS (
        SELECT 1 FROM `_product_lines_quantity_discounts` plqd
        JOIN product_line pl ON pl.id = plqd.A
        WHERE plqd.B = qd.id
        AND pl.name LIKE sqlc.narg('search_query')
    )
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR qd.created_at < sqlc.narg('cursor_created_at')
    OR (qd.created_at = sqlc.narg('cursor_created_at') AND qd.id < sqlc.narg('cursor_id'))
)
AND (
    NOT EXISTS (SELECT 1 FROM account_group_quantity_discount agqd2 WHERE agqd2.quantity_discount_id = qd.id)
    OR EXISTS (
        SELECT 1
        FROM account_group_quantity_discount agqd
        JOIN account_group ag ON ag.id = agqd.account_group_id
        WHERE agqd.quantity_discount_id = qd.id
        AND (
            EXISTS (
                SELECT 1 FROM account_relation_price_group arpg
                JOIN account_relation ar ON ar.id = arpg.account_relation_id
                WHERE arpg.account_group_id = ag.id
                AND ar.counterparty_account_id = sqlc.arg('customer_account_id')
                AND ar.owner_account_id = sqlc.arg('account_id')
            )
            OR EXISTS (
                SELECT 1 FROM account_relation ar
                WHERE ar.account_group_id = ag.id
                AND ar.counterparty_account_id = sqlc.arg('customer_account_id')
                AND ar.owner_account_id = sqlc.arg('account_id')
            )
        )
    )
)
ORDER BY qd.created_at DESC, qd.id DESC
LIMIT ?;

-- name: ListVolumeDiscountsForCustomerBackward :many
SELECT DISTINCT
    qd.id,
    qd.name,
    qd.account_id,
    qd.created_at,
    qd.updated_at
FROM quantity_discount qd
WHERE qd.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR qd.name LIKE sqlc.narg('search_query')
    OR EXISTS (
        SELECT 1 FROM account_group_quantity_discount agqd
        JOIN account_group ag ON ag.id = agqd.account_group_id
        WHERE agqd.quantity_discount_id = qd.id
        AND ag.name LIKE sqlc.narg('search_query')
    )
    OR EXISTS (
        SELECT 1 FROM `_product_lines_quantity_discounts` plqd
        JOIN product_line pl ON pl.id = plqd.A
        WHERE plqd.B = qd.id
        AND pl.name LIKE sqlc.narg('search_query')
    )
)
AND (
    qd.created_at > sqlc.arg('cursor_created_at')
    OR (qd.created_at = sqlc.arg('cursor_created_at') AND qd.id > sqlc.arg('cursor_id'))
)
AND (
    NOT EXISTS (SELECT 1 FROM account_group_quantity_discount agqd2 WHERE agqd2.quantity_discount_id = qd.id)
    OR EXISTS (
        SELECT 1
        FROM account_group_quantity_discount agqd
        JOIN account_group ag ON ag.id = agqd.account_group_id
        WHERE agqd.quantity_discount_id = qd.id
        AND (
            EXISTS (
                SELECT 1 FROM account_relation_price_group arpg
                JOIN account_relation ar ON ar.id = arpg.account_relation_id
                WHERE arpg.account_group_id = ag.id
                AND ar.counterparty_account_id = sqlc.arg('customer_account_id')
                AND ar.owner_account_id = sqlc.arg('account_id')
            )
            OR EXISTS (
                SELECT 1 FROM account_relation ar
                WHERE ar.account_group_id = ag.id
                AND ar.counterparty_account_id = sqlc.arg('customer_account_id')
                AND ar.owner_account_id = sqlc.arg('account_id')
            )
        )
    )
)
ORDER BY qd.created_at ASC, qd.id ASC
LIMIT ?;

-- name: GetVolumeDiscount :one
SELECT
    qd.id,
    qd.name,
    qd.account_id,
    qd.created_at,
    qd.updated_at
FROM quantity_discount qd
WHERE qd.id = sqlc.arg('id')
AND qd.account_id = sqlc.arg('account_id');

-- name: InsertVolumeDiscount :exec
INSERT INTO quantity_discount (id, name, account_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('name'), sqlc.arg('account_id'), NOW(), NOW());

-- name: UpdateVolumeDiscount :execresult
UPDATE quantity_discount
SET
    name = COALESCE(sqlc.narg('name'), name),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteVolumeDiscount :execresult
DELETE FROM quantity_discount
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: CountVolumeDiscountsByName :one
SELECT COUNT(*) AS count
FROM quantity_discount
WHERE account_id = sqlc.arg('account_id')
AND name = sqlc.arg('name')
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: GetVolumeDiscountTiers :many
SELECT
    qdt.id,
    qdt.name,
    qdt.discount_percentage,
    qdt.threshold,
    qdt.parent_tier_id,
    qdt.created_at,
    qdt.updated_at
FROM quantity_discount_tier qdt
WHERE qdt.quantity_discount_id = sqlc.arg('quantity_discount_id')
ORDER BY qdt.threshold ASC;

-- name: GetVolumeDiscountTiersByDiscountIDs :many
SELECT
    qdt.id,
    qdt.name,
    qdt.discount_percentage,
    qdt.threshold,
    qdt.parent_tier_id,
    qdt.quantity_discount_id,
    qdt.created_at,
    qdt.updated_at
FROM quantity_discount_tier qdt
WHERE qdt.quantity_discount_id IN (sqlc.slice('quantity_discount_ids'))
ORDER BY qdt.threshold ASC;

-- name: InsertVolumeDiscountTier :exec
INSERT INTO quantity_discount_tier (id, name, discount_percentage, threshold, parent_tier_id, quantity_discount_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('name'), sqlc.arg('discount_percentage'), sqlc.arg('threshold'), sqlc.narg('parent_tier_id'), sqlc.arg('quantity_discount_id'), NOW(), NOW());

-- name: UpdateVolumeDiscountTier :execresult
UPDATE quantity_discount_tier
SET
    name = sqlc.arg('name'),
    discount_percentage = sqlc.arg('discount_percentage'),
    threshold = sqlc.arg('threshold'),
    parent_tier_id = sqlc.narg('parent_tier_id'),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
AND quantity_discount_id = sqlc.arg('quantity_discount_id');

-- name: DeleteTiersByDiscountID :exec
DELETE FROM quantity_discount_tier
WHERE quantity_discount_id = sqlc.arg('quantity_discount_id');

-- name: DeleteTiersNotInIDs :exec
DELETE FROM quantity_discount_tier
WHERE quantity_discount_id = sqlc.arg('quantity_discount_id')
AND id NOT IN (sqlc.slice('keep_ids'));

-- name: GetVolumeDiscountCustomerGroups :many
SELECT
    agqd.id,
    agqd.account_group_id,
    ag.name,
    ag.commission_status_code,
    ag.freight_status_code,
    ag.account_group_type_code,
    ag.created_at,
    ag.updated_at
FROM account_group_quantity_discount agqd
JOIN account_group ag ON ag.id = agqd.account_group_id
WHERE agqd.quantity_discount_id = sqlc.arg('quantity_discount_id');

-- name: GetVolumeDiscountCustomerGroupsByDiscountIDs :many
SELECT
    agqd.id,
    agqd.account_group_id,
    ag.name,
    ag.commission_status_code,
    ag.freight_status_code,
    ag.account_group_type_code,
    ag.created_at,
    ag.updated_at,
    agqd.quantity_discount_id
FROM account_group_quantity_discount agqd
JOIN account_group ag ON ag.id = agqd.account_group_id
WHERE agqd.quantity_discount_id IN (sqlc.slice('quantity_discount_ids'));

-- name: DeleteCustomerGroupsByDiscountID :exec
DELETE FROM account_group_quantity_discount
WHERE quantity_discount_id = sqlc.arg('quantity_discount_id');

-- name: InsertVolumeDiscountCustomerGroup :exec
INSERT INTO account_group_quantity_discount (id, account_group_id, quantity_discount_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_group_id'), sqlc.arg('quantity_discount_id'), NOW(), NOW());

-- name: GetVolumeDiscountProductLines :many
SELECT
    pl.id,
    pl.name,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    pl.created_at,
    pl.updated_at
FROM `_product_lines_quantity_discounts` plqd
JOIN product_line pl ON pl.id = plqd.A
WHERE plqd.B = sqlc.arg('quantity_discount_id');

-- name: GetVolumeDiscountProductLinesByDiscountIDs :many
SELECT
    pl.id,
    pl.name,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    pl.created_at,
    pl.updated_at,
    plqd.B AS quantity_discount_id
FROM `_product_lines_quantity_discounts` plqd
JOIN product_line pl ON pl.id = plqd.A
WHERE plqd.B IN (sqlc.slice('quantity_discount_ids'));

-- name: DeleteProductLinesByDiscountID :exec
DELETE FROM `_product_lines_quantity_discounts`
WHERE B = sqlc.arg('quantity_discount_id');

-- name: InsertVolumeDiscountProductLine :exec
INSERT INTO `_product_lines_quantity_discounts` (A, B)
VALUES (sqlc.arg('product_line_id'), sqlc.arg('quantity_discount_id'));

-- name: GetVolumeDiscountCategories :many
SELECT
    ic.id,
    ic.name,
    ic.item_category_type_code,
    ic.created_at,
    ic.updated_at
FROM `_item_categories_quantity_discounts` icqd
JOIN item_category ic ON ic.id = icqd.A
WHERE icqd.B = sqlc.arg('quantity_discount_id');

-- name: GetVolumeDiscountCategoriesByDiscountIDs :many
SELECT
    ic.id,
    ic.name,
    ic.item_category_type_code,
    ic.created_at,
    ic.updated_at,
    icqd.B AS quantity_discount_id
FROM `_item_categories_quantity_discounts` icqd
JOIN item_category ic ON ic.id = icqd.A
WHERE icqd.B IN (sqlc.slice('quantity_discount_ids'));

-- name: DeleteCategoriesByDiscountID :exec
DELETE FROM `_item_categories_quantity_discounts`
WHERE B = sqlc.arg('quantity_discount_id');

-- name: InsertVolumeDiscountCategory :exec
INSERT INTO `_item_categories_quantity_discounts` (A, B)
VALUES (sqlc.arg('item_category_id'), sqlc.arg('quantity_discount_id'));

-- name: GetVolumeDiscountAttributes :many
SELECT
    a.id,
    a.text AS name,
    a.color_code,
    a.property_id,
    a.created_at,
    a.updated_at
FROM `_quantity_discounts_attributes` qda
JOIN attribute a ON a.id = qda.A
WHERE qda.B = sqlc.arg('quantity_discount_id');

-- name: GetVolumeDiscountAttributesByDiscountIDs :many
SELECT
    a.id,
    a.text AS name,
    a.color_code,
    a.property_id,
    a.created_at,
    a.updated_at,
    qda.B AS quantity_discount_id
FROM `_quantity_discounts_attributes` qda
JOIN attribute a ON a.id = qda.A
WHERE qda.B IN (sqlc.slice('quantity_discount_ids'));

-- name: DeleteAttributesByDiscountID :exec
DELETE FROM `_quantity_discounts_attributes`
WHERE B = sqlc.arg('quantity_discount_id');

-- name: InsertVolumeDiscountAttribute :exec
INSERT INTO `_quantity_discounts_attributes` (A, B)
VALUES (sqlc.arg('attribute_id'), sqlc.arg('quantity_discount_id'));

-- name: GetVolumeDiscountUnits :many
SELECT
    u.id,
    u.name,
    u.abbreviation,
    u.unit_dimension_code AS type,
    u.ratio_numerator,
    u.ratio_denominator,
    u.offset_numerator,
    u.offset_denominator,
    u.created_at,
    u.updated_at
FROM `_quantity_discounts_units` qdu
JOIN unit u ON u.id = qdu.B
WHERE qdu.A = sqlc.arg('quantity_discount_id');

-- name: GetVolumeDiscountUnitsByDiscountIDs :many
SELECT
    u.id,
    u.name,
    u.abbreviation,
    u.unit_dimension_code AS type,
    u.ratio_numerator,
    u.ratio_denominator,
    u.offset_numerator,
    u.offset_denominator,
    u.created_at,
    u.updated_at,
    qdu.A AS quantity_discount_id
FROM `_quantity_discounts_units` qdu
JOIN unit u ON u.id = qdu.B
WHERE qdu.A IN (sqlc.slice('quantity_discount_ids'));

-- name: DeleteUnitsByDiscountID :exec
DELETE FROM `_quantity_discounts_units`
WHERE A = sqlc.arg('quantity_discount_id');

-- name: InsertVolumeDiscountUnit :exec
INSERT INTO `_quantity_discounts_units` (A, B)
VALUES (sqlc.arg('quantity_discount_id'), sqlc.arg('unit_id'));
