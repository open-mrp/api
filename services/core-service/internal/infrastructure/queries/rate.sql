-- name: GetRateWithUnits :one
SELECT
    r.id,
    r.value,
    r.numerator_unit_id,
    nu.name AS numerator_unit_name,
    nu.abbreviation AS numerator_unit_abbreviation,
    nu.unit_dimension_code AS numerator_unit_type,
    r.denominator_unit_id,
    du.name AS denominator_unit_name,
    du.abbreviation AS denominator_unit_abbreviation,
    du.unit_dimension_code AS denominator_unit_type,
    r.created_at,
    r.updated_at
FROM rate r
JOIN unit nu ON r.numerator_unit_id = nu.id
JOIN unit du ON r.denominator_unit_id = du.id
WHERE r.id = sqlc.arg('id');

-- name: GetRatesByIDs :many
SELECT
    r.id,
    r.value,
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
    r.created_at,
    r.updated_at
FROM rate r
JOIN unit nu ON nu.id = r.numerator_unit_id
JOIN unit du ON du.id = r.denominator_unit_id
WHERE r.id IN (sqlc.slice('ids'));

-- name: UpdateRateByID :execresult
UPDATE rate SET
    value = COALESCE(sqlc.narg('value'), value),
    numerator_unit_id = COALESCE(sqlc.narg('numerator_unit_id'), numerator_unit_id),
    denominator_unit_id = COALESCE(sqlc.narg('denominator_unit_id'), denominator_unit_id),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');
