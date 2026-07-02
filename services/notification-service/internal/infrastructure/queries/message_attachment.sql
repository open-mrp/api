-- name: CreateMessageAttachment :exec
INSERT INTO message_attachment (
    id, message_id, account_id, kind, s3_key, url, filename, content_type, size_bytes,
    resource_type, resource_id, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3));

-- name: ListMessageAttachmentsByMessageIDs :many
SELECT * FROM message_attachment
WHERE message_id IN (sqlc.slice('message_ids'))
  AND deleted_at IS NULL
ORDER BY created_at ASC, id ASC;
