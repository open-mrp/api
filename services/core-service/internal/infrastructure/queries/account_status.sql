-- name: ListAccountStatusesForward :many
SELECT
    account_status.id,
    account_status.code,
    account_status.name,
    account_status.created_at,
    account_status.updated_at
FROM account_status
WHERE (
    sqlc.narg('search_query') IS NULL
    OR account_status.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR account_status.created_at < sqlc.narg('cursor_created_at')
    OR (account_status.created_at = sqlc.narg('cursor_created_at') AND account_status.id < sqlc.narg('cursor_id'))
)
ORDER BY account_status.created_at DESC, account_status.id DESC
LIMIT ?;

-- name: ListAccountStatusesBackward :many
SELECT
    account_status.id,
    account_status.code,
    account_status.name,
    account_status.created_at,
    account_status.updated_at
FROM account_status
WHERE (
    sqlc.narg('search_query') IS NULL
    OR account_status.name LIKE sqlc.narg('search_query')
)
AND (
    account_status.created_at > sqlc.arg('cursor_created_at')
    OR (account_status.created_at = sqlc.arg('cursor_created_at') AND account_status.id > sqlc.arg('cursor_id'))
)
ORDER BY account_status.created_at ASC, account_status.id ASC
LIMIT ?;

-- name: GetAccountStatus :one
SELECT
    account_status.id,
    account_status.code,
    account_status.name,
    account_status.created_at,
    account_status.updated_at
FROM account_status
WHERE account_status.id = sqlc.arg('id') OR account_status.code = sqlc.arg('code');
