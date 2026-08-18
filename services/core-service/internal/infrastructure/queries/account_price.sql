-- name: ListAccountPricesForward :many
SELECT
    ap.id,
    ap.owner_account_id,
    ap.recipient_account_id,
    ra.name AS recipient_account_name,
    rec_ar.external_number AS recipient_account_number,
    rec_ar.account_status_code AS recipient_account_status,
    rec_ar.is_edi_enabled AS recipient_account_is_edi_enabled,
    rec_ar.commission_status_code AS recipient_account_commission_policy,
    CASE
        WHEN rec_ar.parent_account_relation_id IS NOT NULL THEN 'child'
        WHEN EXISTS (SELECT 1 FROM account_relation car WHERE car.parent_account_relation_id = rec_ar.id) THEN 'parent'
        ELSE 'standalone'
    END AS recipient_account_relationship_type,
    ra.created_at AS recipient_account_created_at,
    ra.updated_at AS recipient_account_updated_at,
    ap.product_line_id,
    pl.name AS product_line_name,
    pl.is_commission_exempt AS product_line_is_commission_exempt,
    pl.is_freight_exempt AS product_line_is_freight_exempt,
    pl.created_at AS product_line_created_at,
    pl.updated_at AS product_line_updated_at,
    r.id AS rate_id,
    r.value AS rate_value,
    r.created_at AS rate_created_at,
    r.updated_at AS rate_updated_at,
    r.numerator_unit_id,
    nu.name AS numerator_unit_name,
    nu.abbreviation AS numerator_unit_abbreviation,
    nu.unit_dimension_code AS numerator_unit_type,
    nu.ratio_numerator AS numerator_unit_ratio_numerator,
    nu.ratio_denominator AS numerator_unit_ratio_denominator,
    nu.offset_numerator AS numerator_unit_offset_numerator,
    nu.offset_denominator AS numerator_unit_offset_denominator,
    nu.created_at AS numerator_unit_created_at,
    nu.updated_at AS numerator_unit_updated_at,
    r.denominator_unit_id,
    du.name AS denominator_unit_name,
    du.abbreviation AS denominator_unit_abbreviation,
    du.unit_dimension_code AS denominator_unit_type,
    du.ratio_numerator AS denominator_unit_ratio_numerator,
    du.ratio_denominator AS denominator_unit_ratio_denominator,
    du.offset_numerator AS denominator_unit_offset_numerator,
    du.offset_denominator AS denominator_unit_offset_denominator,
    du.created_at AS denominator_unit_created_at,
    du.updated_at AS denominator_unit_updated_at,
    ap.created_at,
    ap.updated_at
FROM account_price ap
JOIN rate r ON ap.unit_value_id = r.id
JOIN account ra ON ap.recipient_account_id = ra.id
JOIN account_relation rec_ar ON rec_ar.counterparty_account_id = ap.recipient_account_id
    AND rec_ar.owner_account_id = ap.owner_account_id
    AND rec_ar.account_relation_role_code = 'customer'
JOIN product_line pl ON ap.product_line_id = pl.id
JOIN unit nu ON r.numerator_unit_id = nu.id
JOIN unit du ON r.denominator_unit_id = du.id
WHERE ap.owner_account_id = sqlc.arg('owner_account_id')
AND (
    sqlc.arg('include_recipient_filter') = false
    OR ap.recipient_account_id IN (sqlc.slice('recipient_account_ids'))
)
AND (
    sqlc.narg('search_query') IS NULL
    OR ra.name LIKE sqlc.narg('search_query')
    OR rec_ar.external_number LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR ap.created_at < sqlc.narg('cursor_created_at')
    OR (ap.created_at = sqlc.narg('cursor_created_at') AND ap.id < sqlc.narg('cursor_id'))
)
ORDER BY ap.created_at DESC, ap.id DESC
LIMIT ?;

-- name: ListAccountPricesBackward :many
SELECT
    ap.id,
    ap.owner_account_id,
    ap.recipient_account_id,
    ra.name AS recipient_account_name,
    rec_ar.external_number AS recipient_account_number,
    rec_ar.account_status_code AS recipient_account_status,
    rec_ar.is_edi_enabled AS recipient_account_is_edi_enabled,
    rec_ar.commission_status_code AS recipient_account_commission_policy,
    CASE
        WHEN rec_ar.parent_account_relation_id IS NOT NULL THEN 'child'
        WHEN EXISTS (SELECT 1 FROM account_relation car WHERE car.parent_account_relation_id = rec_ar.id) THEN 'parent'
        ELSE 'standalone'
    END AS recipient_account_relationship_type,
    ra.created_at AS recipient_account_created_at,
    ra.updated_at AS recipient_account_updated_at,
    ap.product_line_id,
    pl.name AS product_line_name,
    pl.is_commission_exempt AS product_line_is_commission_exempt,
    pl.is_freight_exempt AS product_line_is_freight_exempt,
    pl.created_at AS product_line_created_at,
    pl.updated_at AS product_line_updated_at,
    r.id AS rate_id,
    r.value AS rate_value,
    r.created_at AS rate_created_at,
    r.updated_at AS rate_updated_at,
    r.numerator_unit_id,
    nu.name AS numerator_unit_name,
    nu.abbreviation AS numerator_unit_abbreviation,
    nu.unit_dimension_code AS numerator_unit_type,
    nu.ratio_numerator AS numerator_unit_ratio_numerator,
    nu.ratio_denominator AS numerator_unit_ratio_denominator,
    nu.offset_numerator AS numerator_unit_offset_numerator,
    nu.offset_denominator AS numerator_unit_offset_denominator,
    nu.created_at AS numerator_unit_created_at,
    nu.updated_at AS numerator_unit_updated_at,
    r.denominator_unit_id,
    du.name AS denominator_unit_name,
    du.abbreviation AS denominator_unit_abbreviation,
    du.unit_dimension_code AS denominator_unit_type,
    du.ratio_numerator AS denominator_unit_ratio_numerator,
    du.ratio_denominator AS denominator_unit_ratio_denominator,
    du.offset_numerator AS denominator_unit_offset_numerator,
    du.offset_denominator AS denominator_unit_offset_denominator,
    du.created_at AS denominator_unit_created_at,
    du.updated_at AS denominator_unit_updated_at,
    ap.created_at,
    ap.updated_at
FROM account_price ap
JOIN rate r ON ap.unit_value_id = r.id
JOIN account ra ON ap.recipient_account_id = ra.id
JOIN account_relation rec_ar ON rec_ar.counterparty_account_id = ap.recipient_account_id
    AND rec_ar.owner_account_id = ap.owner_account_id
    AND rec_ar.account_relation_role_code = 'customer'
JOIN product_line pl ON ap.product_line_id = pl.id
JOIN unit nu ON r.numerator_unit_id = nu.id
JOIN unit du ON r.denominator_unit_id = du.id
WHERE ap.owner_account_id = sqlc.arg('owner_account_id')
AND (
    sqlc.arg('include_recipient_filter') = false
    OR ap.recipient_account_id IN (sqlc.slice('recipient_account_ids'))
)
AND (
    sqlc.narg('search_query') IS NULL
    OR ra.name LIKE sqlc.narg('search_query')
    OR rec_ar.external_number LIKE sqlc.narg('search_query')
)
AND (
    ap.created_at > sqlc.arg('cursor_created_at')
    OR (ap.created_at = sqlc.arg('cursor_created_at') AND ap.id > sqlc.arg('cursor_id'))
)
ORDER BY ap.created_at ASC, ap.id ASC
LIMIT ?;

-- name: GetAccountPrice :one
SELECT
    ap.id,
    ap.owner_account_id,
    ap.recipient_account_id,
    ra.name AS recipient_account_name,
    rec_ar.external_number AS recipient_account_number,
    rec_ar.account_status_code AS recipient_account_status,
    rec_ar.is_edi_enabled AS recipient_account_is_edi_enabled,
    rec_ar.commission_status_code AS recipient_account_commission_policy,
    CASE
        WHEN rec_ar.parent_account_relation_id IS NOT NULL THEN 'child'
        WHEN EXISTS (SELECT 1 FROM account_relation car WHERE car.parent_account_relation_id = rec_ar.id) THEN 'parent'
        ELSE 'standalone'
    END AS recipient_account_relationship_type,
    ra.created_at AS recipient_account_created_at,
    ra.updated_at AS recipient_account_updated_at,
    ap.product_line_id,
    pl.name AS product_line_name,
    pl.is_commission_exempt AS product_line_is_commission_exempt,
    pl.is_freight_exempt AS product_line_is_freight_exempt,
    pl.created_at AS product_line_created_at,
    pl.updated_at AS product_line_updated_at,
    r.id AS rate_id,
    r.value AS rate_value,
    r.created_at AS rate_created_at,
    r.updated_at AS rate_updated_at,
    r.numerator_unit_id,
    nu.name AS numerator_unit_name,
    nu.abbreviation AS numerator_unit_abbreviation,
    nu.unit_dimension_code AS numerator_unit_type,
    nu.ratio_numerator AS numerator_unit_ratio_numerator,
    nu.ratio_denominator AS numerator_unit_ratio_denominator,
    nu.offset_numerator AS numerator_unit_offset_numerator,
    nu.offset_denominator AS numerator_unit_offset_denominator,
    nu.created_at AS numerator_unit_created_at,
    nu.updated_at AS numerator_unit_updated_at,
    r.denominator_unit_id,
    du.name AS denominator_unit_name,
    du.abbreviation AS denominator_unit_abbreviation,
    du.unit_dimension_code AS denominator_unit_type,
    du.ratio_numerator AS denominator_unit_ratio_numerator,
    du.ratio_denominator AS denominator_unit_ratio_denominator,
    du.offset_numerator AS denominator_unit_offset_numerator,
    du.offset_denominator AS denominator_unit_offset_denominator,
    du.created_at AS denominator_unit_created_at,
    du.updated_at AS denominator_unit_updated_at,
    ap.created_at,
    ap.updated_at
FROM account_price ap
JOIN rate r ON ap.unit_value_id = r.id
JOIN account ra ON ap.recipient_account_id = ra.id
JOIN account_relation rec_ar ON rec_ar.counterparty_account_id = ap.recipient_account_id
    AND rec_ar.owner_account_id = ap.owner_account_id
    AND rec_ar.account_relation_role_code = 'customer'
JOIN product_line pl ON ap.product_line_id = pl.id
JOIN unit nu ON r.numerator_unit_id = nu.id
JOIN unit du ON r.denominator_unit_id = du.id
WHERE ap.id = sqlc.arg('id')
AND ap.owner_account_id = sqlc.arg('owner_account_id');

-- name: GetAccountPriceCategories :many
SELECT
    ic.id,
    ic.name,
    ic.item_category_type_code AS type,
    ic.created_at,
    ic.updated_at
FROM account_price_item_category apic
JOIN item_category ic ON apic.item_category_id = ic.id
WHERE apic.account_price_id = sqlc.arg('account_price_id');

-- name: GetAccountPriceAttributes :many
SELECT
    a.id,
    a.text,
    a.color_code,
    a.created_at,
    a.updated_at
FROM account_price_attribute apa
JOIN attribute a ON apa.attribute_id = a.id
WHERE apa.account_price_id = sqlc.arg('account_price_id');

-- name: InsertRate :exec
INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('value'), sqlc.arg('numerator_unit_id'), sqlc.arg('denominator_unit_id'), NOW(3), NOW(3));

-- name: UpdateRate :exec
UPDATE rate SET
    value = COALESCE(sqlc.narg('value'), value),
    numerator_unit_id = COALESCE(sqlc.narg('numerator_unit_id'), numerator_unit_id),
    denominator_unit_id = COALESCE(sqlc.narg('denominator_unit_id'), denominator_unit_id),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: DeleteRate :exec
DELETE FROM rate WHERE id = sqlc.arg('id');

-- name: GetRateIDByAccountPriceID :one
SELECT unit_value_id FROM account_price WHERE id = sqlc.arg('id') AND owner_account_id = sqlc.arg('owner_account_id');

-- name: InsertAccountPrice :exec
INSERT INTO account_price (id, owner_account_id, recipient_account_id, product_line_id, unit_value_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('owner_account_id'), sqlc.arg('recipient_account_id'), sqlc.arg('product_line_id'), sqlc.arg('unit_value_id'), NOW(3), NOW(3));

-- name: UpdateAccountPrice :exec
UPDATE account_price SET
    recipient_account_id = COALESCE(sqlc.narg('recipient_account_id'), recipient_account_id),
    product_line_id = COALESCE(sqlc.narg('product_line_id'), product_line_id),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND owner_account_id = sqlc.arg('owner_account_id');

-- name: DeleteAccountPrice :exec
DELETE FROM account_price WHERE id = sqlc.arg('id') AND owner_account_id = sqlc.arg('owner_account_id');

-- name: InsertAccountPriceCategory :exec
INSERT INTO account_price_item_category (id, account_price_id, item_category_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_price_id'), sqlc.arg('item_category_id'), NOW(3), NOW(3));

-- name: DeleteAccountPriceCategoriesByPriceID :exec
DELETE FROM account_price_item_category WHERE account_price_id = sqlc.arg('account_price_id');

-- name: InsertAccountPriceAttribute :exec
INSERT INTO account_price_attribute (id, account_price_id, attribute_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_price_id'), sqlc.arg('attribute_id'), NOW(3), NOW(3));

-- name: DeleteAccountPriceAttributesByPriceID :exec
DELETE FROM account_price_attribute WHERE account_price_id = sqlc.arg('account_price_id');
