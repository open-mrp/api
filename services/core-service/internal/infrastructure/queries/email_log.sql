-- name: GetEmailLog :one
SELECT
    el.id,
    el.has_sent,
    el.subject,
    el.filename,
    el.ses_message_id,
    el.sent_by_id,
    u.name AS sent_by_name,
    u.username AS sent_by_username,
    u.email AS sent_by_email,
    ak.type_id AS sent_by_api_key_id,
    ak.name AS sent_by_api_key_name,
    el.related_type,
    el.related_id,
    el.created_at,
    el.updated_at
FROM email_log el
LEFT JOIN user u ON el.sent_by_id = u.id
LEFT JOIN api_key ak ON el.sent_by_id = ak.type_id
WHERE el.id = sqlc.arg('id')
AND el.account_id = sqlc.arg('account_id');

-- name: GetEmailRecipientsByEmailLogID :many
SELECT er.email
FROM email_recipient er
WHERE er.email_log_id = sqlc.arg('email_log_id')
ORDER BY er.created_at ASC;
