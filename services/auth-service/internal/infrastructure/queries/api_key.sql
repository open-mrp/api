-- name: FindAPIKeyByID :one
SELECT * FROM api_key WHERE key_id = ? OR type_id = ?;

-- name: FindAPIKeyWithRoleByKeyID :one
SELECT
    api_key.id,
    api_key.type_id,
    api_key.key_id,
    api_key.name,
    api_key.secret_hash,
    api_key.redacted_value,
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
    api_key.redacted_value,
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

-- name: FindAPIKeyWithRoleByTypeID :one
SELECT
    api_key.id,
    api_key.type_id,
    api_key.key_id,
    api_key.name,
    api_key.secret_hash,
    api_key.redacted_value,
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
WHERE api_key.type_id = ?;

-- name: FindAPIKeyBaseByTypeID :one
SELECT api_key.* FROM api_key WHERE api_key.type_id = ?;

-- name: FindAPIKeyBaseByDatabaseID :one
SELECT api_key.* FROM api_key WHERE api_key.id = ?;

-- name: ListAPIKeysBaseForward :many
SELECT api_key.*
FROM api_key
WHERE api_key.owner_account_id = sqlc.arg('owner_account_id')
AND (api_key.name LIKE CONCAT('%', sqlc.arg('query'), '%') OR sqlc.arg('query') = '')
AND (
    (sqlc.arg('include_active') = true AND api_key.revoked_at IS NULL AND (api_key.expires_at IS NULL OR api_key.expires_at > NOW(3)))
    OR (sqlc.arg('include_expired') = true AND api_key.expires_at IS NOT NULL AND api_key.expires_at <= NOW(3) AND api_key.revoked_at IS NULL AND api_key.expires_at >= DATE_SUB(NOW(3), INTERVAL 30 DAY))
    OR (sqlc.arg('include_revoked') = true AND api_key.revoked_at IS NOT NULL AND api_key.revoked_at >= DATE_SUB(NOW(3), INTERVAL 30 DAY))
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR api_key.created_at < sqlc.narg('cursor_created_at')
    OR (api_key.created_at = sqlc.narg('cursor_created_at') AND api_key.id < sqlc.narg('cursor_id'))
)
ORDER BY api_key.created_at DESC, api_key.id DESC
LIMIT ?;

-- name: ListAPIKeysBaseBackward :many
SELECT api_key.*
FROM api_key
WHERE api_key.owner_account_id = sqlc.arg('owner_account_id')
AND (api_key.name LIKE CONCAT('%', sqlc.arg('query'), '%') OR sqlc.arg('query') = '')
AND (
    (sqlc.arg('include_active') = true AND api_key.revoked_at IS NULL AND (api_key.expires_at IS NULL OR api_key.expires_at > NOW(3)))
    OR (sqlc.arg('include_expired') = true AND api_key.expires_at IS NOT NULL AND api_key.expires_at <= NOW(3) AND api_key.revoked_at IS NULL AND api_key.expires_at >= DATE_SUB(NOW(3), INTERVAL 30 DAY))
    OR (sqlc.arg('include_revoked') = true AND api_key.revoked_at IS NOT NULL AND api_key.revoked_at >= DATE_SUB(NOW(3), INTERVAL 30 DAY))
)
AND (
    api_key.created_at > sqlc.arg('cursor_created_at')
    OR (api_key.created_at = sqlc.arg('cursor_created_at') AND api_key.id > sqlc.arg('cursor_id'))
)
ORDER BY api_key.created_at ASC, api_key.id ASC
LIMIT ?;

-- name: GetAPIKeysByIDs :many
SELECT
    api_key.id,
    api_key.type_id,
    api_key.key_id,
    api_key.name,
    api_key.secret_hash,
    api_key.redacted_value,
    api_key.owner_account_id,
    api_key.role_id,
    api_key.created_at,
    api_key.updated_at,
    api_key.last_used_at,
    api_key.expires_at,
    api_key.revoked_at
FROM api_key
WHERE api_key.type_id IN (sqlc.slice('ids'))
AND api_key.owner_account_id = sqlc.arg('owner_account_id');

-- name: RevokeAPIKeyByTypeID :execresult
UPDATE api_key SET revoked_at = NOW(3), updated_at = NOW(3) WHERE type_id = ? AND owner_account_id = ?;

-- name: TouchAPIKeyByID :exec
UPDATE api_key SET last_used_at = NOW(3), updated_at = NOW(3) WHERE id = ?;

-- name: CreateAPIKey :execresult
INSERT INTO api_key (
    type_id,
    key_id,
    name,
    secret_hash,
    redacted_value,
    owner_account_id,
    role_id,
    created_at,
    updated_at,
    expires_at
) VALUES (?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3), ?);

-- name: ListAPIKeysForward :many
SELECT
    api_key.*,
    role.name AS role_name,
    role.role_type_code
FROM api_key
LEFT JOIN role ON api_key.role_id = role.id
WHERE api_key.owner_account_id = sqlc.arg('owner_account_id')
AND (api_key.name LIKE CONCAT('%', sqlc.arg('query'), '%') OR sqlc.arg('query') = '')
AND (
    (sqlc.arg('include_active') = true AND api_key.revoked_at IS NULL AND (api_key.expires_at IS NULL OR api_key.expires_at > NOW(3)))
    OR (sqlc.arg('include_expired') = true AND api_key.expires_at IS NOT NULL AND api_key.expires_at <= NOW(3) AND api_key.revoked_at IS NULL AND api_key.expires_at >= DATE_SUB(NOW(3), INTERVAL 30 DAY))
    OR (sqlc.arg('include_revoked') = true AND api_key.revoked_at IS NOT NULL AND api_key.revoked_at >= DATE_SUB(NOW(3), INTERVAL 30 DAY))
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR api_key.created_at < sqlc.narg('cursor_created_at')
    OR (api_key.created_at = sqlc.narg('cursor_created_at') AND api_key.id < sqlc.narg('cursor_id'))
)
ORDER BY api_key.created_at DESC, api_key.id DESC
LIMIT ?;

-- name: ListAPIKeysBackward :many
SELECT
    api_key.*,
    role.name AS role_name,
    role.role_type_code
FROM api_key
LEFT JOIN role ON api_key.role_id = role.id
WHERE api_key.owner_account_id = sqlc.arg('owner_account_id')
AND (api_key.name LIKE CONCAT('%', sqlc.arg('query'), '%') OR sqlc.arg('query') = '')
AND (
    (sqlc.arg('include_active') = true AND api_key.revoked_at IS NULL AND (api_key.expires_at IS NULL OR api_key.expires_at > NOW(3)))
    OR (sqlc.arg('include_expired') = true AND api_key.expires_at IS NOT NULL AND api_key.expires_at <= NOW(3) AND api_key.revoked_at IS NULL AND api_key.expires_at >= DATE_SUB(NOW(3), INTERVAL 30 DAY))
    OR (sqlc.arg('include_revoked') = true AND api_key.revoked_at IS NOT NULL AND api_key.revoked_at >= DATE_SUB(NOW(3), INTERVAL 30 DAY))
)
AND (
    api_key.created_at > sqlc.arg('cursor_created_at')
    OR (api_key.created_at = sqlc.arg('cursor_created_at') AND api_key.id > sqlc.arg('cursor_id'))
)
ORDER BY api_key.created_at ASC, api_key.id ASC
LIMIT ?;