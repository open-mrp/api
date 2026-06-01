-- name: ListAccountIntegrationsForward :many
SELECT
    account_integration.id,
    account_integration.account_id,
    account_integration.integration_code,
    account_integration.name,
    account_integration.is_active,
    account_integration.created_at,
    account_integration.updated_at
FROM account_integration
WHERE account_integration.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR account_integration.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR account_integration.created_at < sqlc.narg('cursor_created_at')
    OR (account_integration.created_at = sqlc.narg('cursor_created_at') AND account_integration.id < sqlc.narg('cursor_id'))
)
ORDER BY account_integration.created_at DESC, account_integration.id DESC
LIMIT ?;

-- name: ListAccountIntegrationsBackward :many
SELECT
    account_integration.id,
    account_integration.account_id,
    account_integration.integration_code,
    account_integration.name,
    account_integration.is_active,
    account_integration.created_at,
    account_integration.updated_at
FROM account_integration
WHERE account_integration.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR account_integration.name LIKE sqlc.narg('search_query')
)
AND (
    account_integration.created_at > sqlc.arg('cursor_created_at')
    OR (account_integration.created_at = sqlc.arg('cursor_created_at') AND account_integration.id > sqlc.arg('cursor_id'))
)
ORDER BY account_integration.created_at ASC, account_integration.id ASC
LIMIT ?;

-- name: GetAccountIntegration :one
SELECT
    account_integration.id,
    account_integration.account_id,
    account_integration.integration_code,
    account_integration.name,
    account_integration.is_active,
    account_integration.created_at,
    account_integration.updated_at
FROM account_integration
WHERE account_integration.id = sqlc.arg('id')
AND account_integration.account_id = sqlc.arg('account_id');

-- name: FindAccountIntegrationByCode :one
SELECT
    account_integration.id,
    account_integration.account_id,
    account_integration.integration_code,
    account_integration.name,
    account_integration.is_active,
    account_integration.created_at,
    account_integration.updated_at
FROM account_integration
WHERE account_integration.account_id = sqlc.arg('account_id')
AND account_integration.integration_code = sqlc.arg('integration_code');

-- name: InsertAccountIntegration :exec
INSERT INTO account_integration (
    id,
    account_id,
    integration_code,
    name,
    credentials_v2,
    is_active,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_id'),
    sqlc.arg('integration_code'),
    sqlc.arg('name'),
    sqlc.arg('credentials_v2'),
    1,
    NOW(3),
    NOW(3)
);

-- name: UpdateAccountIntegrationCredentials :execresult
UPDATE account_integration SET
    name = sqlc.arg('name'),
    credentials_v2 = sqlc.arg('credentials_v2'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: UpdateAccountIntegration :execresult
UPDATE account_integration SET
    name = COALESCE(sqlc.narg('name'), name),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteAccountIntegration :execresult
DELETE FROM account_integration
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: GetAccountIntegrationsByIDs :many
-- Returns account integrations matching the given IDs that belong to the
-- caller's account. Account integrations are always account-scoped.
SELECT
    account_integration.id,
    account_integration.account_id,
    account_integration.integration_code,
    account_integration.name,
    account_integration.is_active,
    account_integration.created_at,
    account_integration.updated_at
FROM account_integration
WHERE account_integration.id IN (sqlc.slice('ids'))
AND account_integration.account_id = sqlc.arg('account_id');

-- name: GetAccountIntegrationCredentials :one
SELECT
    account_integration.credentials_v2,
    account_integration.is_active
FROM account_integration
WHERE account_integration.account_id = sqlc.arg('account_id')
AND account_integration.integration_code = sqlc.arg('integration_code');

-- name: CountAccountIntegrationByCode :one
SELECT COUNT(*) FROM account_integration
WHERE account_id = sqlc.arg('account_id')
AND integration_code = sqlc.arg('integration_code');
