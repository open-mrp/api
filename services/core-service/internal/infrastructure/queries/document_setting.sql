-- name: ListDocumentSettings :many
SELECT * FROM document_setting
WHERE account_id = sqlc.arg('account_id')
ORDER BY document_type;

-- name: GetDocumentSetting :one
SELECT * FROM document_setting
WHERE account_id = sqlc.arg('account_id')
  AND document_type = sqlc.arg('document_type');

-- The setting as it stands after an update: the service applies the update to what was there, so a cleared
-- field is written as NULL. id is used only when the account has no row for the type yet.
-- name: UpsertDocumentSetting :exec
INSERT INTO document_setting (
    id,
    account_id,
    document_type,
    process_owner,
    document_number,
    revision,
    footer_text,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_id'),
    sqlc.arg('document_type'),
    sqlc.narg('process_owner'),
    sqlc.narg('document_number'),
    sqlc.narg('revision'),
    sqlc.narg('footer_text'),
    NOW(3),
    NOW(3)
)
ON DUPLICATE KEY UPDATE
    process_owner = VALUES(process_owner),
    document_number = VALUES(document_number),
    revision = VALUES(revision),
    footer_text = VALUES(footer_text),
    updated_at = NOW(3);
