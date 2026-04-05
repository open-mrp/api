-- name: ListPaymentTermsForward :many
SELECT
    payment_term.id,
    payment_term.is_active,
    payment_term.name,
    payment_term.account_id,
    payment_term.created_at,
    payment_term.updated_at
FROM payment_term
WHERE (payment_term.account_id = sqlc.arg('account_id') OR payment_term.account_id IS NULL)
AND (
    sqlc.narg('search_query') IS NULL
    OR payment_term.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR payment_term.created_at < sqlc.narg('cursor_created_at')
    OR (payment_term.created_at = sqlc.narg('cursor_created_at') AND payment_term.id < sqlc.narg('cursor_id'))
)
ORDER BY payment_term.created_at DESC, payment_term.id DESC
LIMIT ?;

-- name: ListPaymentTermsBackward :many
SELECT
    payment_term.id,
    payment_term.is_active,
    payment_term.name,
    payment_term.account_id,
    payment_term.created_at,
    payment_term.updated_at
FROM payment_term
WHERE (payment_term.account_id = sqlc.arg('account_id') OR payment_term.account_id IS NULL)
AND (
    sqlc.narg('search_query') IS NULL
    OR payment_term.name LIKE sqlc.narg('search_query')
)
AND (
    payment_term.created_at > sqlc.arg('cursor_created_at')
    OR (payment_term.created_at = sqlc.arg('cursor_created_at') AND payment_term.id > sqlc.arg('cursor_id'))
)
ORDER BY payment_term.created_at ASC, payment_term.id ASC
LIMIT ?;

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
    account_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.arg('account_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdatePaymentTerm :execresult
UPDATE payment_term SET
    name = COALESCE(sqlc.narg('name'), name),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeletePaymentTerm :execresult
DELETE FROM payment_term
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: CountPaymentTermsByName :one
SELECT COUNT(*) FROM payment_term
WHERE name = ? AND (account_id = ? OR account_id IS NULL)
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));
