-- name: ListItemCategoriesForward :many
SELECT
    ic.id,
    ic.name,
    ic.notes,
    ic.item_category_type_code,
    ic.unit_group_id,
    ic.account_id,
    ic.created_at,
    ic.updated_at
FROM item_category ic
WHERE (ic.account_id = sqlc.arg('account_id') OR ic.account_id IS NULL)
AND (sqlc.narg('item_category_type_code') IS NULL OR ic.item_category_type_code = sqlc.narg('item_category_type_code'))
AND (
    sqlc.narg('search_query') IS NULL
    OR ic.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR ic.created_at < sqlc.narg('cursor_created_at')
    OR (ic.created_at = sqlc.narg('cursor_created_at') AND ic.id < sqlc.narg('cursor_id'))
)
ORDER BY ic.created_at DESC, ic.id DESC
LIMIT ?;

-- name: ListItemCategoriesBackward :many
SELECT
    ic.id,
    ic.name,
    ic.notes,
    ic.item_category_type_code,
    ic.unit_group_id,
    ic.account_id,
    ic.created_at,
    ic.updated_at
FROM item_category ic
WHERE (ic.account_id = sqlc.arg('account_id') OR ic.account_id IS NULL)
AND (sqlc.narg('item_category_type_code') IS NULL OR ic.item_category_type_code = sqlc.narg('item_category_type_code'))
AND (
    sqlc.narg('search_query') IS NULL
    OR ic.name LIKE sqlc.narg('search_query')
)
AND (
    ic.created_at > sqlc.arg('cursor_created_at')
    OR (ic.created_at = sqlc.arg('cursor_created_at') AND ic.id > sqlc.arg('cursor_id'))
)
ORDER BY ic.created_at ASC, ic.id ASC
LIMIT ?;

-- name: GetItemCategory :one
SELECT
    ic.id,
    ic.name,
    ic.notes,
    ic.item_category_type_code,
    ic.unit_group_id,
    ic.account_id,
    ic.created_at,
    ic.updated_at
FROM item_category ic
WHERE ic.id = sqlc.arg('id')
AND (ic.account_id = sqlc.arg('account_id') OR ic.account_id IS NULL);

-- name: InsertItemCategory :exec
INSERT INTO item_category (
    id,
    name,
    notes,
    item_category_type_code,
    unit_group_id,
    account_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.narg('notes'),
    sqlc.arg('item_category_type_code'),
    sqlc.arg('unit_group_id'),
    sqlc.arg('account_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateItemCategory :execresult
UPDATE item_category SET
    name       = COALESCE(sqlc.narg('name'), name),
    notes      = COALESCE(sqlc.narg('notes'), notes),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: UpdateItemCategoryWithUnitGroup :execresult
UPDATE item_category SET
    name          = COALESCE(sqlc.narg('name'), name),
    notes         = COALESCE(sqlc.narg('notes'), notes),
    unit_group_id = sqlc.arg('unit_group_id'),
    updated_at    = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteItemCategory :execresult
DELETE FROM item_category
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: CountItemCategoryInAccount :one
SELECT COUNT(*) FROM item_category
WHERE id = sqlc.arg('id')
AND (account_id = sqlc.arg('account_id') OR account_id IS NULL);

-- name: InsertItemCategoryProperty :exec
INSERT INTO _item_categories_properties (A, B)
VALUES (sqlc.arg('item_category_id'), sqlc.arg('property_id'));

-- name: UpsertItemCategoryProperty :exec
INSERT IGNORE INTO _item_categories_properties (A, B)
VALUES (sqlc.arg('item_category_id'), sqlc.arg('property_id'));

-- name: DeleteItemCategoryProperty :exec
DELETE FROM _item_categories_properties
WHERE A = sqlc.arg('item_category_id') AND B = sqlc.arg('property_id');

-- name: UpdateItemCategoryUnitGroup :execresult
UPDATE item_category SET
    unit_group_id = sqlc.arg('unit_group_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: ListItemCategoryProperties :many
SELECT
    p.id,
    p.name,
    p.created_at,
    p.updated_at
FROM property p
INNER JOIN _item_categories_properties icp ON icp.B = p.id
WHERE icp.A = sqlc.arg('item_category_id');

-- name: ListItemCategoryPropertiesForCategories :many
SELECT
    icp.A AS item_category_id,
    p.id AS property_id,
    p.name AS property_name,
    p.created_at AS property_created_at,
    p.updated_at AS property_updated_at
FROM property p
INNER JOIN _item_categories_properties icp ON icp.B = p.id
WHERE icp.A IN (sqlc.slice('item_category_ids'))
ORDER BY icp.A, p.name;

-- name: CountUnitGroupVisibleToAccount :one
SELECT COUNT(*) FROM unit_group
WHERE id = sqlc.arg('id')
AND (account_id = sqlc.arg('account_id') OR account_id IS NULL);

-- name: CountPropertiesInCategoryByName :one
SELECT COUNT(*) FROM property p
INNER JOIN _item_categories_properties icp ON icp.B = p.id
WHERE icp.A = sqlc.arg('item_category_id')
AND p.name = sqlc.arg('name')
AND p.account_id = sqlc.arg('account_id')
AND (sqlc.narg('exclude_property_id') IS NULL OR p.id != sqlc.narg('exclude_property_id'));

-- name: CountPropertyInAccount :one
SELECT COUNT(*) FROM property
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: GetItemCategoriesByIDs :many
SELECT
    ic.id,
    ic.name,
    ic.notes,
    ic.item_category_type_code,
    ic.unit_group_id,
    ic.account_id,
    ic.created_at,
    ic.updated_at
FROM item_category ic
WHERE ic.id IN (sqlc.slice('ids'))
AND (ic.account_id = sqlc.arg('account_id') OR ic.account_id IS NULL);

-- name: FindItemCategoriesByNames :many
SELECT
    ic.id,
    ic.name,
    ic.notes,
    ic.item_category_type_code,
    ic.unit_group_id,
    ic.account_id,
    ic.created_at,
    ic.updated_at
FROM item_category ic
WHERE ic.name IN (sqlc.slice('names'))
AND (ic.account_id = sqlc.arg('account_id') OR ic.account_id IS NULL);

-- name: GetUnitGroupForCategory :one
SELECT
    ug.id,
    ug.name,
    ug.base_unit_id,
    ug.unit_type_code,
    ug.created_at,
    ug.updated_at
FROM unit_group ug
WHERE ug.id = sqlc.arg('id');

-- name: GetCategoryBaseUnitID :one
-- The base unit is the one configured on the category's unit group
-- (unit_group.base_unit_id), NOT a group member flagged is_base_unit — that flag
-- marks the canonical base of a unit type and is not what a group's base unit is.
-- Also returns the category type so create paths can enforce that materials only use
-- material categories and parts/products only product categories.
SELECT
  ug.base_unit_id AS base_unit_id,
  ic.item_category_type_code
FROM item_category ic
JOIN unit_group ug ON ug.id = ic.unit_group_id
WHERE ic.id = sqlc.arg('category_id');

-- name: GetCategoryBaseUnits :many
-- Batched counterpart to GetCategoryBaseUnitID used by bulk upsert to validate
-- create-row categories up front. Returns one row per category that exists, with its
-- unit group's base_unit_id and its category type (materials may only use
-- material_category; parts and products only product_category). Categories absent from
-- the result do not exist. Mirrors GetCategoryBaseUnitID (no account scope) so the
-- verdict matches the create path.
SELECT
  ic.id AS category_id,
  ic.item_category_type_code,
  ug.base_unit_id AS base_unit_id
FROM item_category ic
JOIN unit_group ug ON ug.id = ic.unit_group_id
WHERE ic.id IN (sqlc.slice('category_ids'));

-- name: ExportItemCategories :many
-- Unpaginated by design; the caller passes a row cap as the limit. System rows
-- (account_id IS NULL) are in scope, matching what the list endpoint returns.
SELECT
    ic.id,
    ic.name,
    ic.notes,
    ic.item_category_type_code,
    ic.unit_group_id,
    ug.name AS unit_group_name,
    ic.account_id,
    ic.created_at,
    ic.updated_at
FROM item_category ic
LEFT JOIN unit_group ug ON ug.id = ic.unit_group_id
WHERE (ic.account_id = sqlc.arg('account_id') OR ic.account_id IS NULL)
AND (
    sqlc.narg('search_query') IS NULL
    OR ic.name LIKE sqlc.narg('search_query')
)
ORDER BY ic.created_at DESC, ic.id DESC
LIMIT ?;
