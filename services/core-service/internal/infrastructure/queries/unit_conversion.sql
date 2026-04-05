-- name: GetUnitConversionFactors :one
SELECT id, ratio_numerator, ratio_denominator, offset_numerator, offset_denominator, is_base_unit
FROM unit
WHERE id = sqlc.arg('unit_id');

-- name: GetUnitConversionFactorsPair :many
SELECT id, ratio_numerator, ratio_denominator, offset_numerator, offset_denominator, is_base_unit
FROM unit
WHERE id IN (sqlc.arg('from_unit_id'), sqlc.arg('to_unit_id'));
