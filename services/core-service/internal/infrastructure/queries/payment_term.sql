-- name: GetPaymentTerm :one
SELECT
    payment_term.id,
    payment_term.is_active,
    payment_term.name,
    payment_term.account_id,
    payment_term.created_at,
    payment_term.updated_at
FROM payment_term
WHERE payment_term.id = sqlc.arg('id')
AND (payment_term.account_id = sqlc.arg('account_id') OR payment_term.account_id IS NULL);

-- name: InsertPaymentTerm :exec
INSERT INTO payment_term (
    id,
    name,
    is_active,
    account_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.arg('is_active'),
    sqlc.arg('account_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdatePaymentTerm :execresult
UPDATE payment_term SET
    name = COALESCE(sqlc.narg('name'), name),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeletePaymentTerm :execresult
DELETE FROM payment_term
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: GetPaymentTermsByIDs :many
-- Returns payment terms matching the given IDs that the caller's account is
-- authorized to read (their own account plus system payment terms).
SELECT
    payment_term.id,
    payment_term.is_active,
    payment_term.name,
    payment_term.account_id,
    payment_term.created_at,
    payment_term.updated_at
FROM payment_term
WHERE payment_term.id IN (sqlc.slice('ids'))
AND (payment_term.account_id = sqlc.arg('account_id') OR payment_term.account_id IS NULL);

-- name: CountPaymentTermsByName :one
SELECT COUNT(*) FROM payment_term
WHERE name = ? AND (account_id = ? OR account_id IS NULL)
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));
