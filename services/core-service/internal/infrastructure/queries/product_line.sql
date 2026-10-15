-- name: GetProductLinesByIDs :many
SELECT
    pl.id,
    pl.name,
    pl.description,
    pl.notes,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    pl.unit_group_id,
    pl.supply_code,
    pl.is_active,
    pl.account_id,
    pl.created_at,
    pl.updated_at
FROM product_line pl
WHERE pl.id IN (sqlc.slice('ids'));

-- name: GetProductLinesByIDsScoped :many
SELECT
    pl.id,
    pl.name,
    pl.description,
    pl.notes,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    pl.unit_group_id,
    pl.supply_code,
    pl.is_active,
    pl.account_id,
    pl.created_at,
    pl.updated_at
FROM product_line pl
WHERE pl.id IN (sqlc.slice('ids'))
AND (pl.account_id = sqlc.arg('account_id') OR pl.account_id IS NULL);

-- name: GetProductLine :one
SELECT
    pl.id,
    pl.name,
    pl.description,
    pl.notes,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    pl.unit_group_id,
    pl.supply_code,
    pl.is_active,
    pl.account_id,
    pl.created_at,
    pl.updated_at
FROM product_line pl
WHERE pl.id = sqlc.arg('id')
AND (pl.account_id = sqlc.arg('account_id') OR pl.account_id IS NULL);

-- name: InsertProductLine :exec
INSERT INTO product_line (
    id,
    name,
    description,
    is_commission_exempt,
    is_freight_exempt,
    unit_group_id,
    supply_code,
    is_active,
    account_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.narg('description'),
    sqlc.arg('is_commission_exempt'),
    sqlc.arg('is_freight_exempt'),
    sqlc.narg('unit_group_id'),
    sqlc.narg('supply_code'),
    sqlc.arg('is_active'),
    sqlc.arg('account_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateProductLine :execresult
UPDATE product_line SET
    name = COALESCE(sqlc.narg('name'), name),
    description = IF(sqlc.arg('clear_description'), NULL, COALESCE(sqlc.narg('description'), description)),
    is_commission_exempt = COALESCE(sqlc.narg('is_commission_exempt'), is_commission_exempt),
    is_freight_exempt = COALESCE(sqlc.narg('is_freight_exempt'), is_freight_exempt),
    unit_group_id = COALESCE(sqlc.narg('unit_group_id'), unit_group_id),
    supply_code = IF(sqlc.arg('clear_supply'), NULL, COALESCE(sqlc.narg('supply_code'), supply_code)),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteProductLine :execresult
DELETE FROM product_line
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- Customer and account-group access to a line goes with the line.

-- name: DeleteAccountRelationProductLinesForProductLine :exec
DELETE FROM account_relation_product_line WHERE product_line_id = sqlc.arg('product_line_id');

-- name: DeleteAccountGroupProductLinesForProductLine :exec
DELETE FROM account_group_product_line WHERE product_line_id = sqlc.arg('product_line_id');

-- name: CountProductLinesByName :one
SELECT COUNT(*) FROM product_line
WHERE name = ? AND (account_id = ? OR account_id IS NULL)
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: FindProductLinesByNames :many
-- names must be pre-lowercased by the caller; the utf8mb4_unicode_ci collation makes the
-- IN comparison case-insensitive, so lowercasing on both sides is not required in SQL.
-- System product lines (account_id IS NULL) are included so the upsert can detect a
-- name collision with a platform-provided line and reject the modification.
SELECT
    pl.id,
    pl.name,
    pl.description,
    pl.notes,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    pl.unit_group_id,
    pl.supply_code,
    pl.is_active,
    pl.account_id,
    pl.created_at,
    pl.updated_at
FROM product_line pl
WHERE pl.name IN (sqlc.slice('names'))
AND (pl.account_id = sqlc.arg('account_id') OR pl.account_id IS NULL);

-- name: GetUnitGroupForProductLine :one
SELECT
    ug.id,
    ug.name,
    ug.base_unit_id,
    ug.unit_type_code,
    ug.created_at,
    ug.updated_at
FROM unit_group ug
WHERE ug.id = sqlc.arg('id')
AND (ug.account_id = sqlc.arg('account_id') OR ug.account_id IS NULL);

-- name: GetUnitGroupsForProductLinesByIDs :many
-- The batched form of GetUnitGroupForProductLine.
SELECT
    ug.id,
    ug.name,
    ug.base_unit_id,
    ug.unit_type_code,
    ug.created_at,
    ug.updated_at
FROM unit_group ug
WHERE ug.id IN (sqlc.slice('ids'))
AND (ug.account_id = sqlc.arg('account_id') OR ug.account_id IS NULL);

-- name: ExportProductLines :many
-- Unpaginated by design; the caller passes a row cap as the limit. System rows
-- (account_id IS NULL) are in scope, matching what the list endpoint returns.
SELECT
    pl.id,
    pl.name,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    ug.name AS unit_group_name,
    pl.created_at,
    pl.updated_at
FROM product_line pl
LEFT JOIN unit_group ug ON ug.id = pl.unit_group_id
WHERE (pl.account_id = sqlc.arg('account_id') OR pl.account_id IS NULL)
AND (
    sqlc.narg('search_query') IS NULL
    OR pl.name LIKE sqlc.narg('search_query')
)
ORDER BY pl.created_at DESC, pl.id DESC
LIMIT ?;

-- ListProductLineSupplies returns every product line in the account that sets a default supply.
-- name: ListProductLineSupplies :many
SELECT
    pl.id,
    pl.supply_code
FROM product_line pl
WHERE (pl.account_id = sqlc.arg('account_id') OR pl.account_id IS NULL)
  AND pl.supply_code IS NOT NULL
ORDER BY pl.id;

-- GetItemLotOverride reads a per-item lot size set by hand, which outranks the category's lot.
-- name: GetItemLotOverride :one
SELECT COALESCE(s.lot_multiple_units, 0) AS lot_multiple_units
FROM production_schedule_item_setting s
WHERE s.account_id = sqlc.arg('account_id')
AND s.item_id = sqlc.arg('item_id');
