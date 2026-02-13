-- name: FindDocAPIKeyBySandboxAccountID :one
SELECT
    dak.id, dak.type_id, dak.api_key_id, dak.encrypted_secret,
    dak.created_at, dak.updated_at,
    ak.expires_at AS ak_expires_at, ak.revoked_at AS ak_revoked_at
FROM doc_api_key dak
JOIN api_key ak ON dak.api_key_id = ak.type_id
WHERE ak.owner_account_id = ?
ORDER BY dak.id DESC
LIMIT 1;

-- name: CreateDocAPIKey :execresult
INSERT INTO doc_api_key (type_id, api_key_id, encrypted_secret, created_at, updated_at)
VALUES (?, ?, ?, NOW(), NOW());

-- name: UpdateDocAPIKey :exec
UPDATE doc_api_key SET api_key_id = ?, encrypted_secret = ?, updated_at = NOW() WHERE id = ?;

-- name: DeleteDocAPIKeyByID :exec
DELETE FROM doc_api_key WHERE id = ?;

-- name: FindDocAPIKeyByAPIKeyID :one
SELECT id, type_id, api_key_id, encrypted_secret, created_at, updated_at
FROM doc_api_key WHERE api_key_id = ? LIMIT 1;

-- name: DeleteDocAPIKeyByAPIKeyID :exec
DELETE FROM doc_api_key WHERE api_key_id = ?;

-- name: DeleteDocAPIKeysBySandboxAccountID :exec
DELETE dak FROM doc_api_key dak
INNER JOIN api_key ak ON dak.api_key_id = ak.type_id
WHERE ak.owner_account_id = ?;
