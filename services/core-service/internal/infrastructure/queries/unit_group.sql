-- name: ListUnitGroupsForwardBase :many
SELECT
    ug.id,
    ug.name,
    ug.notes,
    ug.unit_type_code,
    ug.base_unit_id,
    ug.account_id,
    ug.created_at,
    ug.updated_at
FROM unit_group ug
WHERE (ug.account_id = sqlc.arg('account_id') OR ug.account_id IS NULL)
AND (sqlc.narg('unit_type_code') IS NULL OR ug.unit_type_code = sqlc.narg('unit_type_code'))
AND (
    sqlc.narg('search_query') IS NULL
    OR ug.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR ug.created_at < sqlc.narg('cursor_created_at')
    OR (ug.created_at = sqlc.narg('cursor_created_at') AND ug.id < sqlc.narg('cursor_id'))
)
ORDER BY ug.created_at DESC, ug.id DESC
LIMIT ?;

-- name: ListUnitGroupsBackwardBase :many
SELECT
    ug.id,
    ug.name,
    ug.notes,
    ug.unit_type_code,
    ug.base_unit_id,
    ug.account_id,
    ug.created_at,
    ug.updated_at
FROM unit_group ug
WHERE (ug.account_id = sqlc.arg('account_id') OR ug.account_id IS NULL)
AND (sqlc.narg('unit_type_code') IS NULL OR ug.unit_type_code = sqlc.narg('unit_type_code'))
AND (
    sqlc.narg('search_query') IS NULL
    OR ug.name LIKE sqlc.narg('search_query')
)
AND (
    ug.created_at > sqlc.arg('cursor_created_at')
    OR (ug.created_at = sqlc.arg('cursor_created_at') AND ug.id > sqlc.arg('cursor_id'))
)
ORDER BY ug.created_at ASC, ug.id ASC
LIMIT ?;

-- name: GetUnitGroupBase :one
SELECT
    ug.id,
    ug.name,
    ug.notes,
    ug.unit_type_code,
    ug.base_unit_id,
    ug.account_id,
    ug.created_at,
    ug.updated_at
FROM unit_group ug
WHERE ug.id = sqlc.arg('id')
AND (ug.account_id = sqlc.arg('account_id') OR ug.account_id IS NULL);

-- name: ListUnitGroupsForward :many
SELECT
    ug.id,
    ug.name,
    ug.notes,
    ug.unit_type_code,
    ug.base_unit_id,
    u.name AS base_unit_name,
    u.abbreviation AS base_unit_abbreviation,
    u.unit_dimension_code AS base_unit_type,
    u.ratio_numerator AS base_unit_ratio_numerator,
    u.ratio_denominator AS base_unit_ratio_denominator,
    u.offset_numerator AS base_unit_offset_numerator,
    u.offset_denominator AS base_unit_offset_denominator,
    u.is_base_unit AS base_unit_is_base_unit,
    u.created_at AS base_unit_created_at,
    u.updated_at AS base_unit_updated_at,
    u.account_id AS base_unit_account_id,
    ug.account_id,
    ug.created_at,
    ug.updated_at
FROM unit_group ug
JOIN unit u ON ug.base_unit_id = u.id
WHERE (ug.account_id = sqlc.arg('account_id') OR ug.account_id IS NULL)
AND (sqlc.narg('unit_type_code') IS NULL OR ug.unit_type_code = sqlc.narg('unit_type_code'))
AND (
    sqlc.narg('search_query') IS NULL
    OR ug.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR ug.created_at < sqlc.narg('cursor_created_at')
    OR (ug.created_at = sqlc.narg('cursor_created_at') AND ug.id < sqlc.narg('cursor_id'))
)
ORDER BY ug.created_at DESC, ug.id DESC
LIMIT ?;

-- name: ListUnitGroupsBackward :many
SELECT
    ug.id,
    ug.name,
    ug.notes,
    ug.unit_type_code,
    ug.base_unit_id,
    u.name AS base_unit_name,
    u.abbreviation AS base_unit_abbreviation,
    u.unit_dimension_code AS base_unit_type,
    u.ratio_numerator AS base_unit_ratio_numerator,
    u.ratio_denominator AS base_unit_ratio_denominator,
    u.offset_numerator AS base_unit_offset_numerator,
    u.offset_denominator AS base_unit_offset_denominator,
    u.is_base_unit AS base_unit_is_base_unit,
    u.created_at AS base_unit_created_at,
    u.updated_at AS base_unit_updated_at,
    u.account_id AS base_unit_account_id,
    ug.account_id,
    ug.created_at,
    ug.updated_at
FROM unit_group ug
JOIN unit u ON ug.base_unit_id = u.id
WHERE (ug.account_id = sqlc.arg('account_id') OR ug.account_id IS NULL)
AND (sqlc.narg('unit_type_code') IS NULL OR ug.unit_type_code = sqlc.narg('unit_type_code'))
AND (
    sqlc.narg('search_query') IS NULL
    OR ug.name LIKE sqlc.narg('search_query')
)
AND (
    ug.created_at > sqlc.arg('cursor_created_at')
    OR (ug.created_at = sqlc.arg('cursor_created_at') AND ug.id > sqlc.arg('cursor_id'))
)
ORDER BY ug.created_at ASC, ug.id ASC
LIMIT ?;

-- name: GetUnitGroup :one
SELECT
    ug.id,
    ug.name,
    ug.notes,
    ug.unit_type_code,
    ug.base_unit_id,
    u.name AS base_unit_name,
    u.abbreviation AS base_unit_abbreviation,
    u.unit_dimension_code AS base_unit_type,
    u.ratio_numerator AS base_unit_ratio_numerator,
    u.ratio_denominator AS base_unit_ratio_denominator,
    u.offset_numerator AS base_unit_offset_numerator,
    u.offset_denominator AS base_unit_offset_denominator,
    u.is_base_unit AS base_unit_is_base_unit,
    u.created_at AS base_unit_created_at,
    u.updated_at AS base_unit_updated_at,
    u.account_id AS base_unit_account_id,
    ug.account_id,
    ug.created_at,
    ug.updated_at
FROM unit_group ug
JOIN unit u ON ug.base_unit_id = u.id
WHERE ug.id = sqlc.arg('id')
AND (ug.account_id = sqlc.arg('account_id') OR ug.account_id IS NULL);

-- name: GetUnitGroupsByIDs :many
SELECT
    ug.id,
    ug.name,
    ug.unit_type_code,
    ug.base_unit_id,
    ug.created_at,
    ug.updated_at
FROM unit_group ug
WHERE ug.id IN (sqlc.slice('ids'));

-- name: GetUnitGroupsByIDsScoped :many
SELECT
    ug.id,
    ug.name,
    ug.notes,
    ug.unit_type_code,
    ug.base_unit_id,
    u.name AS base_unit_name,
    u.abbreviation AS base_unit_abbreviation,
    u.unit_dimension_code AS base_unit_type,
    u.ratio_numerator AS base_unit_ratio_numerator,
    u.ratio_denominator AS base_unit_ratio_denominator,
    u.offset_numerator AS base_unit_offset_numerator,
    u.offset_denominator AS base_unit_offset_denominator,
    u.is_base_unit AS base_unit_is_base_unit,
    u.created_at AS base_unit_created_at,
    u.updated_at AS base_unit_updated_at,
    u.account_id AS base_unit_account_id,
    ug.account_id,
    ug.created_at,
    ug.updated_at
FROM unit_group ug
JOIN unit u ON ug.base_unit_id = u.id
WHERE ug.id IN (sqlc.slice('ids'))
AND (ug.account_id = sqlc.arg('account_id') OR ug.account_id IS NULL);

-- name: GetUnitGroupUnitsByIDsScoped :many
SELECT
    ugu.id,
    ugu.unit_id,
    ugu.unit_group_id,
    ugu.discount_percentage,
    ugu.discount_fixed,
    ugu.is_visible,
    ugu.created_at,
    ugu.updated_at,
    u.name AS unit_name,
    u.abbreviation AS unit_abbreviation,
    u.unit_dimension_code AS unit_type,
    u.ratio_numerator AS unit_ratio_numerator,
    u.ratio_denominator AS unit_ratio_denominator,
    u.offset_numerator AS unit_offset_numerator,
    u.offset_denominator AS unit_offset_denominator,
    u.is_base_unit AS unit_is_base_unit,
    u.created_at AS unit_created_at,
    u.updated_at AS unit_updated_at,
    u.account_id AS unit_account_id
FROM unit_group_unit ugu
JOIN unit u ON ugu.unit_id = u.id
JOIN unit_group ug ON ugu.unit_group_id = ug.id
WHERE ugu.id IN (sqlc.slice('ids'))
AND (ug.account_id = sqlc.arg('account_id') OR ug.account_id IS NULL);

-- name: ListUnitGroupUnitsByUnitGroupIDs :many
SELECT
    ugu.id,
    ugu.unit_id,
    ugu.unit_group_id,
    ugu.discount_percentage,
    ugu.discount_fixed,
    ugu.is_visible,
    ugu.created_at,
    ugu.updated_at,
    u.name AS unit_name,
    u.abbreviation AS unit_abbreviation,
    u.unit_dimension_code AS unit_type,
    u.ratio_numerator AS unit_ratio_numerator,
    u.ratio_denominator AS unit_ratio_denominator,
    u.offset_numerator AS unit_offset_numerator,
    u.offset_denominator AS unit_offset_denominator,
    u.is_base_unit AS unit_is_base_unit,
    u.created_at AS unit_created_at,
    u.updated_at AS unit_updated_at,
    u.account_id AS unit_account_id
FROM unit_group_unit ugu
JOIN unit u ON ugu.unit_id = u.id
WHERE ugu.unit_group_id IN (sqlc.slice('unit_group_ids'));

-- name: InsertUnitGroup :exec
INSERT INTO unit_group (
    id,
    name,
    notes,
    unit_type_code,
    base_unit_id,
    account_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.narg('notes'),
    sqlc.arg('unit_type_code'),
    sqlc.arg('base_unit_id'),
    sqlc.arg('account_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateUnitGroup :execresult
UPDATE unit_group SET
    name = COALESCE(sqlc.narg('name'), name),
    notes = CASE WHEN sqlc.arg('update_notes') = true THEN sqlc.narg('notes') ELSE notes END,
    base_unit_id = COALESCE(sqlc.narg('base_unit_id'), base_unit_id),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteUnitGroup :execresult
DELETE FROM unit_group
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: CountUnitGroupsByName :one
SELECT COUNT(*) FROM unit_group
WHERE name = ? AND (account_id = ? OR account_id IS NULL)
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: ListUnitGroupUnitsBase :many
SELECT
    ugu.id,
    ugu.unit_id,
    ugu.unit_group_id,
    ugu.discount_percentage,
    ugu.discount_fixed,
    ugu.is_visible,
    ugu.created_at,
    ugu.updated_at
FROM unit_group_unit ugu
WHERE ugu.unit_group_id = sqlc.arg('unit_group_id');

-- name: ListUnitGroupUnits :many
SELECT
    ugu.id,
    ugu.unit_id,
    ugu.unit_group_id,
    ugu.discount_percentage,
    ugu.discount_fixed,
    ugu.is_visible,
    ugu.created_at,
    ugu.updated_at,
    u.name AS unit_name,
    u.abbreviation AS unit_abbreviation,
    u.unit_dimension_code AS unit_type,
    u.ratio_numerator AS unit_ratio_numerator,
    u.ratio_denominator AS unit_ratio_denominator,
    u.offset_numerator AS unit_offset_numerator,
    u.offset_denominator AS unit_offset_denominator,
    u.is_base_unit AS unit_is_base_unit,
    u.created_at AS unit_created_at,
    u.updated_at AS unit_updated_at,
    u.account_id AS unit_account_id
FROM unit_group_unit ugu
JOIN unit u ON ugu.unit_id = u.id
WHERE ugu.unit_group_id = sqlc.arg('unit_group_id');

-- name: UpsertUnitGroupUnit :exec
INSERT INTO unit_group_unit (
    id,
    unit_group_id,
    unit_id,
    discount_percentage,
    discount_fixed,
    is_visible,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('unit_group_id'),
    sqlc.arg('unit_id'),
    sqlc.arg('discount_percentage'),
    sqlc.arg('discount_fixed'),
    sqlc.arg('is_visible'),
    NOW(3),
    NOW(3)
) ON DUPLICATE KEY UPDATE
    discount_percentage = VALUES(discount_percentage),
    discount_fixed = VALUES(discount_fixed),
    is_visible = VALUES(is_visible),
    updated_at = NOW(3);

-- name: GetUnitGroupUnitBase :one
SELECT
    ugu.id,
    ugu.unit_id,
    ugu.unit_group_id,
    ugu.discount_percentage,
    ugu.discount_fixed,
    ugu.is_visible,
    ugu.created_at,
    ugu.updated_at
FROM unit_group_unit ugu
WHERE ugu.id = sqlc.arg('id')
AND ugu.unit_group_id = sqlc.arg('unit_group_id');

-- name: GetUnitGroupUnit :one
SELECT
    ugu.id,
    ugu.unit_id,
    ugu.unit_group_id,
    ugu.discount_percentage,
    ugu.discount_fixed,
    ugu.is_visible,
    ugu.created_at,
    ugu.updated_at,
    u.name AS unit_name,
    u.abbreviation AS unit_abbreviation,
    u.unit_dimension_code AS unit_type,
    u.ratio_numerator AS unit_ratio_numerator,
    u.ratio_denominator AS unit_ratio_denominator,
    u.offset_numerator AS unit_offset_numerator,
    u.offset_denominator AS unit_offset_denominator,
    u.is_base_unit AS unit_is_base_unit,
    u.created_at AS unit_created_at,
    u.updated_at AS unit_updated_at,
    u.account_id AS unit_account_id
FROM unit_group_unit ugu
JOIN unit u ON ugu.unit_id = u.id
WHERE ugu.id = sqlc.arg('id')
AND ugu.unit_group_id = sqlc.arg('unit_group_id');

-- name: DeleteUnitGroupUnitByID :execresult
DELETE FROM unit_group_unit
WHERE id = sqlc.arg('id')
AND unit_group_id = sqlc.arg('unit_group_id');

-- name: DeleteAllUnitGroupUnits :exec
DELETE FROM unit_group_unit
WHERE unit_group_id = sqlc.arg('unit_group_id');

-- CountUnitInGroup reports whether a unit belongs to a unit group, counting the group's
-- base unit as a member: the base unit is implicit rather than a unit_group_unit row.
-- name: CountUnitInGroup :one
SELECT COUNT(*) FROM unit_group ug
LEFT JOIN unit_group_unit ugu ON ugu.unit_group_id = ug.id AND ugu.unit_id = sqlc.arg('unit_id')
WHERE ug.id = sqlc.arg('unit_group_id')
AND (ug.base_unit_id = sqlc.arg('unit_id') OR ugu.id IS NOT NULL);
