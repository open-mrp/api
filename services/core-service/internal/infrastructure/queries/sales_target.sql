-- name: ListSalesTargets :many
SELECT t.id, t.start_date, t.end_date, t.sales_rep_id, t.account_id,
       t.amount_id, t.created_at, t.updated_at,
       q.value AS amount_value, q.unit_id AS amount_unit_id,
       u.name AS amount_unit_name, u.abbreviation AS amount_unit_abbreviation,
       u.unit_dimension_code AS amount_unit_type,
       u.ratio_numerator AS amount_unit_ratio_numerator,
       u.ratio_denominator AS amount_unit_ratio_denominator,
       u.offset_numerator AS amount_unit_offset_numerator,
       u.offset_denominator AS amount_unit_offset_denominator,
       u.created_at AS amount_unit_created_at, u.updated_at AS amount_unit_updated_at
FROM target t
INNER JOIN quantity q ON q.id = t.amount_id
INNER JOIN unit u ON u.id = q.unit_id
WHERE t.sales_rep_id = ? AND t.account_id = ?
AND (
    sqlc.narg('search') IS NULL
    OR t.id LIKE CONCAT('%', sqlc.narg('search'), '%')
    OR q.value LIKE CONCAT('%', sqlc.narg('search'), '%')
)
ORDER BY t.start_date DESC
LIMIT ? OFFSET ?;

-- name: CountSalesTargets :one
SELECT COUNT(*) AS cnt
FROM target t
INNER JOIN quantity q ON q.id = t.amount_id
WHERE t.sales_rep_id = ? AND t.account_id = ?
AND (
    sqlc.narg('search') IS NULL
    OR t.id LIKE CONCAT('%', sqlc.narg('search'), '%')
    OR q.value LIKE CONCAT('%', sqlc.narg('search'), '%')
);

-- name: GetSalesTarget :one
SELECT t.id, t.start_date, t.end_date, t.sales_rep_id, t.account_id,
       t.amount_id, t.created_at, t.updated_at,
       q.value AS amount_value, q.unit_id AS amount_unit_id,
       u.name AS amount_unit_name, u.abbreviation AS amount_unit_abbreviation,
       u.unit_dimension_code AS amount_unit_type,
       u.ratio_numerator AS amount_unit_ratio_numerator,
       u.ratio_denominator AS amount_unit_ratio_denominator,
       u.offset_numerator AS amount_unit_offset_numerator,
       u.offset_denominator AS amount_unit_offset_denominator,
       u.created_at AS amount_unit_created_at, u.updated_at AS amount_unit_updated_at
FROM target t
INNER JOIN quantity q ON q.id = t.amount_id
INNER JOIN unit u ON u.id = q.unit_id
WHERE t.id = ?;

-- name: SalesTargetExists :one
SELECT EXISTS(SELECT 1 FROM target WHERE id = ?) AS does_exist;

-- name: SalesTargetIsInAccount :one
SELECT EXISTS(SELECT 1 FROM target WHERE id = ? AND account_id = ?) AS is_in_account;

-- name: SalesRepExistsInAccount :one
SELECT EXISTS(
    SELECT 1 FROM account_user
    WHERE id = ? AND account_id = ?
        AND (status_code = 'active' OR status_code IS NULL)
) AS does_exist;

-- name: InsertSalesTarget :exec
INSERT INTO target (id, start_date, end_date, sales_rep_id, account_id, amount_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, NOW(3), NOW(3));

-- name: UpdateSalesTarget :exec
UPDATE target SET
    start_date = sqlc.arg('start_date'),
    end_date = sqlc.arg('end_date'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');
