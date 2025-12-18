-- name: FindAPIKeyByID :one
SELECT * FROM api_key WHERE id = ?;

-- name: FindAPIKeyWithRoleByID :one
SELECT 
    api_key.id,
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
    role.role_type_code
FROM api_key
LEFT JOIN role ON api_key.role_id = role.id
WHERE api_key.id = ?;

-- name: DeleteAPIKeyByID :exec
DELETE FROM api_key WHERE id = ?;

-- name: TouchAPIKey :exec
UPDATE api_key SET last_used_at = NOW(), updated_at = NOW() WHERE id = ?;