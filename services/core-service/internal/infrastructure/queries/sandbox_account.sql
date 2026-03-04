-- name: CreateSandboxAccount :execresult
INSERT INTO sandbox_account (
    type_id,
    owner_account_id,
    account_id,
    created_at,
    updated_at
) VALUES (?, ?, ?, NOW(3), NOW(3));

-- name: FindSandboxAccountByTypeID :one
SELECT 
    sandbox_account.*,
    account.name
FROM sandbox_account
JOIN account ON sandbox_account.account_id = account.id
WHERE sandbox_account.type_id = ?;

-- name: FindSandboxAccountByAccountID :one
SELECT 
    sandbox_account.*,
    account.name
FROM sandbox_account
JOIN account ON sandbox_account.account_id = account.id
WHERE sandbox_account.account_id = ?;

-- name: ListSandboxAccountsForward :many
SELECT
    sandbox_account.*,
    account.name
FROM sandbox_account
JOIN account ON sandbox_account.account_id = account.id
WHERE sandbox_account.owner_account_id = sqlc.arg('owner_account_id')
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR sandbox_account.created_at < sqlc.narg('cursor_created_at')
    OR (sandbox_account.created_at = sqlc.narg('cursor_created_at') AND sandbox_account.id < sqlc.narg('cursor_id'))
)
ORDER BY sandbox_account.created_at DESC, sandbox_account.id DESC
LIMIT ?;

-- name: ListSandboxAccountsBackward :many
SELECT
    sandbox_account.*,
    account.name
FROM sandbox_account
JOIN account ON sandbox_account.account_id = account.id
WHERE sandbox_account.owner_account_id = sqlc.arg('owner_account_id')
AND (
    sandbox_account.created_at > sqlc.arg('cursor_created_at')
    OR (sandbox_account.created_at = sqlc.arg('cursor_created_at') AND sandbox_account.id > sqlc.arg('cursor_id'))
)
ORDER BY sandbox_account.created_at ASC, sandbox_account.id ASC
LIMIT ?;

-- name: FindSandboxAccountWithOwnerByTypeID :one
SELECT
    sandbox_account.*,
    account.name,
    owner_account.name AS owner_account_name
FROM sandbox_account
JOIN account ON sandbox_account.account_id = account.id
LEFT JOIN account owner_account ON sandbox_account.owner_account_id = owner_account.id
WHERE sandbox_account.type_id = ?;

-- name: ListSandboxAccountsWithOwnerForward :many
SELECT
    sandbox_account.*,
    account.name,
    owner_account.name AS owner_account_name
FROM sandbox_account
JOIN account ON sandbox_account.account_id = account.id
LEFT JOIN account owner_account ON sandbox_account.owner_account_id = owner_account.id
WHERE sandbox_account.owner_account_id = sqlc.arg('owner_account_id')
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR sandbox_account.created_at < sqlc.narg('cursor_created_at')
    OR (sandbox_account.created_at = sqlc.narg('cursor_created_at') AND sandbox_account.id < sqlc.narg('cursor_id'))
)
ORDER BY sandbox_account.created_at DESC, sandbox_account.id DESC
LIMIT ?;

-- name: ListSandboxAccountsWithOwnerBackward :many
SELECT
    sandbox_account.*,
    account.name,
    owner_account.name AS owner_account_name
FROM sandbox_account
JOIN account ON sandbox_account.account_id = account.id
LEFT JOIN account owner_account ON sandbox_account.owner_account_id = owner_account.id
WHERE sandbox_account.owner_account_id = sqlc.arg('owner_account_id')
AND (
    sandbox_account.created_at > sqlc.arg('cursor_created_at')
    OR (sandbox_account.created_at = sqlc.arg('cursor_created_at') AND sandbox_account.id > sqlc.arg('cursor_id'))
)
ORDER BY sandbox_account.created_at ASC, sandbox_account.id ASC
LIMIT ?;

-- name: CountSandboxAccounts :one
SELECT COUNT(*) FROM sandbox_account
WHERE sandbox_account.owner_account_id = ?;

-- name: DeleteSandboxAccountByID :exec
DELETE FROM sandbox_account WHERE id = ?;

-- name: DeleteSandboxAccountByTypeID :exec
DELETE FROM sandbox_account WHERE type_id = ?;

-- name: FindFirstSandboxAccountByOwnerAccountID :one
SELECT sandbox_account.account_id
FROM sandbox_account
WHERE sandbox_account.owner_account_id = ?
ORDER BY sandbox_account.id ASC
LIMIT 1;

