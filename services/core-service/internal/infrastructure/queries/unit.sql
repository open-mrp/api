-- name: ListUnitsForward :many
SELECT
    unit.id,
    unit.name,
    unit.abbreviation,
    unit.unit_dimension_code,
    unit.ratio_numerator,
    unit.ratio_denominator,
    unit.offset_numerator,
    unit.offset_denominator,
    unit.is_base_unit,
    unit.account_id,
    unit.created_at,
    unit.updated_at
FROM unit
WHERE (unit.account_id = sqlc.arg('account_id') OR unit.account_id IS NULL)
AND (sqlc.narg('unit_dimension_code') IS NULL OR unit.unit_dimension_code = sqlc.narg('unit_dimension_code'))
AND (
    sqlc.arg('include_unit_group_filter') = false
    OR EXISTS (
        SELECT 1 FROM unit_group_unit
        JOIN unit_group ON unit_group_unit.unit_group_id = unit_group.id
        WHERE unit_group_unit.unit_id = unit.id
        AND unit_group_unit.unit_group_id IN (sqlc.slice('unit_group_ids'))
        AND (unit_group.account_id = sqlc.arg('account_id') OR unit_group.account_id IS NULL)
    )
)
AND (
    sqlc.narg('search_query') IS NULL
    OR unit.name LIKE sqlc.narg('search_query')
    OR unit.abbreviation = sqlc.narg('search_exact')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR unit.created_at < sqlc.narg('cursor_created_at')
    OR (unit.created_at = sqlc.narg('cursor_created_at') AND unit.id < sqlc.narg('cursor_id'))
)
ORDER BY unit.created_at DESC, unit.id DESC
LIMIT ?;

-- name: ListUnitsBackward :many
SELECT
    unit.id,
    unit.name,
    unit.abbreviation,
    unit.unit_dimension_code,
    unit.ratio_numerator,
    unit.ratio_denominator,
    unit.offset_numerator,
    unit.offset_denominator,
    unit.is_base_unit,
    unit.account_id,
    unit.created_at,
    unit.updated_at
FROM unit
WHERE (unit.account_id = sqlc.arg('account_id') OR unit.account_id IS NULL)
AND (sqlc.narg('unit_dimension_code') IS NULL OR unit.unit_dimension_code = sqlc.narg('unit_dimension_code'))
AND (
    sqlc.arg('include_unit_group_filter') = false
    OR EXISTS (
        SELECT 1 FROM unit_group_unit
        JOIN unit_group ON unit_group_unit.unit_group_id = unit_group.id
        WHERE unit_group_unit.unit_id = unit.id
        AND unit_group_unit.unit_group_id IN (sqlc.slice('unit_group_ids'))
        AND (unit_group.account_id = sqlc.arg('account_id') OR unit_group.account_id IS NULL)
    )
)
AND (
    sqlc.narg('search_query') IS NULL
    OR unit.name LIKE sqlc.narg('search_query')
    OR unit.abbreviation = sqlc.narg('search_exact')
)
AND (
    unit.created_at > sqlc.arg('cursor_created_at')
    OR (unit.created_at = sqlc.arg('cursor_created_at') AND unit.id > sqlc.arg('cursor_id'))
)
ORDER BY unit.created_at ASC, unit.id ASC
LIMIT ?;
