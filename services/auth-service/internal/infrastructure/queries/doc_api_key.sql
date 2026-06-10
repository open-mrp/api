-- name: FindDocAPIKeyBySandboxAccountID :one
SELECT
    dak.id, dak.type_id, dak.api_key_id, dak.owner_account_id, dak.encrypted_secret,
    dak.created_at, dak.updated_at,
    ak.expires_at AS ak_expires_at, ak.revoked_at AS ak_revoked_at
FROM doc_api_key dak
JOIN api_key ak ON dak.api_key_id = ak.type_id
WHERE dak.owner_account_id = ?
LIMIT 1;

-- name: CreateDocAPIKey :execresult
INSERT INTO doc_api_key (type_id, api_key_id, owner_account_id, encrypted_secret, created_at, updated_at)
VALUES (?, ?, ?, ?, NOW(3), NOW(3));

-- name: UpdateDocAPIKey :exec
UPDATE doc_api_key SET api_key_id = ?, encrypted_secret = ?, updated_at = NOW(3) WHERE id = ?;

-- name: DeleteDocAPIKeyByID :exec
DELETE FROM doc_api_key WHERE id = ?;

-- name: FindDocAPIKeyByAPIKeyID :one
SELECT id, type_id, api_key_id, owner_account_id, encrypted_secret, created_at, updated_at
FROM doc_api_key WHERE api_key_id = ? LIMIT 1;

-- name: DeleteDocAPIKeyByAPIKeyID :exec
DELETE FROM doc_api_key WHERE api_key_id = ?;
