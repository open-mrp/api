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

-- name: UpdateRateByID :execresult
UPDATE rate SET
    value = COALESCE(sqlc.narg('value'), value),
    numerator_unit_id = COALESCE(sqlc.narg('numerator_unit_id'), numerator_unit_id),
    denominator_unit_id = COALESCE(sqlc.narg('denominator_unit_id'), denominator_unit_id),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');
