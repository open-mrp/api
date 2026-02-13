-- name: CreateSandboxAccount :execresult
INSERT INTO sandbox_account (
    type_id,
    owner_account_id,
    account_id,
    created_at,
    updated_at
) VALUES (?, ?, ?, NOW(), NOW());

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

-- name: ListSandboxAccounts :many
SELECT
    sandbox_account.*,
    account.name
FROM sandbox_account
JOIN account ON sandbox_account.account_id = account.id
WHERE sandbox_account.owner_account_id = sqlc.arg('owner_account_id')
AND (sandbox_account.id > (SELECT sub.id FROM sandbox_account sub WHERE sub.type_id = sqlc.arg('cursor')) OR sqlc.arg('cursor') = '')
ORDER BY sandbox_account.id ASC
LIMIT ?;

-- name: CountSandboxAccounts :one
SELECT COUNT(*) FROM sandbox_account
WHERE sandbox_account.owner_account_id = ?;

-- name: DeleteSandboxAccountByID :exec
DELETE FROM sandbox_account WHERE id = ?;

-- name: FindFirstSandboxAccountByOwnerAccountID :one
SELECT sandbox_account.account_id
FROM sandbox_account
WHERE sandbox_account.owner_account_id = ?
ORDER BY sandbox_account.id ASC
LIMIT 1;

