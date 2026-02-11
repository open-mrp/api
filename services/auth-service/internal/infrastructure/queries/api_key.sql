-- name: FindAPIKeyByID :one
SELECT * FROM api_key WHERE key_id = ? OR type_id = ?;

-- name: FindAPIKeyWithRoleByKeyID :one
SELECT 
    api_key.id,
    api_key.type_id,
    api_key.key_id,
    api_key.name,
    api_key.secret_hash,
    api_key.last_four,
    api_key.owner_account_id,
    api_key.role_id,
    api_key.created_at,
    api_key.updated_at,
    api_key.last_used_at,
    api_key.expires_at,
    api_key.revoked_at,
    role.name AS role_name,
    role.role_type_code
FROM api_key
LEFT JOIN role ON api_key.role_id = role.id
WHERE api_key.key_id = ?;

-- name: FindAPIKeyWithRoleByDatabaseID :one
SELECT 
    api_key.id,
    api_key.type_id,
    api_key.key_id,
    api_key.name,
    api_key.secret_hash,
    api_key.last_four,
    api_key.owner_account_id,
    api_key.role_id,
    api_key.created_at,
    api_key.updated_at,
    api_key.last_used_at,
    api_key.expires_at,
    api_key.revoked_at,
    role.name AS role_name,
    role.role_type_code
FROM api_key
LEFT JOIN role ON api_key.role_id = role.id
WHERE api_key.id = ?;

-- name: DeleteAPIKeyByID :exec
DELETE FROM api_key WHERE id = ?;

-- name: TouchAPIKeyByID :exec
UPDATE api_key SET last_used_at = NOW(), updated_at = NOW() WHERE id = ?;

-- name: CreateAPIKey :execresult
INSERT INTO api_key (
    type_id,
    key_id,
    name,
    secret_hash,
    last_four,
    owner_account_id,
    role_id,
    created_at,
    updated_at,
    expires_at
) VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW(), ?);

-- name: ListAPIKeys :many
SELECT
    api_key.*,
    role.name AS role_name,
    role.role_type_code
FROM api_key
LEFT JOIN role ON api_key.role_id = role.id
WHERE api_key.owner_account_id = sqlc.arg('owner_account_id')
AND (api_key.id > (SELECT sub.id FROM api_key sub WHERE sub.type_id = sqlc.arg('cursor')) OR sqlc.arg('cursor') = '')
AND (api_key.name LIKE CONCAT('%', sqlc.arg('query'), '%') OR sqlc.arg('query') = '')
ORDER BY api_key.id ASC
LIMIT ?;

-- name: CountAPIKeys :one
SELECT COUNT(*) FROM api_key
WHERE api_key.owner_account_id = sqlc.arg('owner_account_id')
AND (api_key.name LIKE CONCAT('%', sqlc.arg('query'), '%') OR sqlc.arg('query') = '');
