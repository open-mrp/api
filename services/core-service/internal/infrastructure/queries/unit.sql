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
    (sqlc.narg('search_query') IS NULL AND sqlc.narg('like_query') IS NULL)
    OR MATCH(unit.name, unit.abbreviation) AGAINST(sqlc.narg('search_query') IN BOOLEAN MODE)
    OR unit.name LIKE sqlc.narg('like_query')
    OR unit.abbreviation LIKE sqlc.narg('like_query')
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
    (sqlc.narg('search_query') IS NULL AND sqlc.narg('like_query') IS NULL)
    OR MATCH(unit.name, unit.abbreviation) AGAINST(sqlc.narg('search_query') IN BOOLEAN MODE)
    OR unit.name LIKE sqlc.narg('like_query')
    OR unit.abbreviation LIKE sqlc.narg('like_query')
)
AND (
    unit.created_at > sqlc.arg('cursor_created_at')
    OR (unit.created_at = sqlc.arg('cursor_created_at') AND unit.id > sqlc.arg('cursor_id'))
)
ORDER BY unit.created_at ASC, unit.id ASC
LIMIT ?;

-- name: GetUnit :one
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
WHERE unit.id = sqlc.arg('id')
AND (unit.account_id = sqlc.arg('account_id') OR unit.account_id IS NULL);

-- name: GetUnitsByIDs :many
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
WHERE unit.id IN (sqlc.slice('ids'));

-- name: GetUnitsByIDsScoped :many
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
WHERE unit.id IN (sqlc.slice('ids'))
AND (unit.account_id = sqlc.arg('account_id') OR unit.account_id IS NULL);

-- name: InsertUnit :exec
INSERT INTO unit (
    id,
    name,
    abbreviation,
    unit_dimension_code,
    ratio_numerator,
    ratio_denominator,
    offset_numerator,
    offset_denominator,
    is_base_unit,
    account_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.arg('abbreviation'),
    sqlc.arg('unit_dimension_code'),
    sqlc.arg('ratio_numerator'),
    sqlc.arg('ratio_denominator'),
    sqlc.arg('offset_numerator'),
    sqlc.arg('offset_denominator'),
    sqlc.arg('is_base_unit'),
    sqlc.arg('account_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateUnit :execresult
UPDATE unit SET
    name = COALESCE(sqlc.narg('name'), name),
    abbreviation = COALESCE(sqlc.narg('abbreviation'), abbreviation),
    ratio_numerator = COALESCE(sqlc.narg('ratio_numerator'), ratio_numerator),
    ratio_denominator = COALESCE(sqlc.narg('ratio_denominator'), ratio_denominator),
    offset_numerator = COALESCE(sqlc.narg('offset_numerator'), offset_numerator),
    offset_denominator = COALESCE(sqlc.narg('offset_denominator'), offset_denominator),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteUnitGroupUnitsByUnitID :exec
DELETE FROM unit_group_unit WHERE unit_id = sqlc.arg('unit_id');

-- name: DeleteUnit :execresult
DELETE FROM unit
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: CountUnitsByName :one
SELECT COUNT(*) FROM unit
WHERE name = ? AND (account_id = ? OR account_id IS NULL)
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: CountUnitsByAbbreviation :one
SELECT COUNT(*) FROM unit
WHERE abbreviation = ? AND (account_id = ? OR account_id IS NULL)
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: FindUnitsByAbbreviations :many
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
WHERE (unit.account_id = sqlc.arg('account_id') OR unit.account_id IS NULL);

-- name: GetCurrencyBaseUnit :one
-- Returns the global currency base unit ID (used as the numerator unit for price rates).
SELECT id
FROM unit
WHERE unit_dimension_code = 'currency'
AND is_base_unit = TRUE
AND account_id IS NULL
LIMIT 1;

-- name: GetUnitDimensionCodes :many
-- Returns (id, unit_dimension_code) pairs for the given unit IDs. Used to
-- validate the unit type on each side of a rate at write time.
SELECT id, unit_dimension_code
FROM unit
WHERE id IN (sqlc.slice('ids'));

-- name: FindUnitsByAbbreviationsOrNames :many
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
WHERE (unit.abbreviation IN (sqlc.slice('abbreviations')) OR unit.name IN (sqlc.slice('names')))
AND (unit.account_id = sqlc.arg('account_id') OR unit.account_id IS NULL);

-- name: ExportUnits :many
-- Unpaginated by design; the caller passes a row cap as the limit. System units
-- (account_id IS NULL) are in scope, matching what the list endpoint returns.
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
AND (
    sqlc.narg('search_query') IS NULL
    OR unit.name LIKE sqlc.narg('search_query')
)
ORDER BY unit.created_at DESC, unit.id DESC
LIMIT ?;
