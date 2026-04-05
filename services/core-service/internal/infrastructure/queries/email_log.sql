-- name: ListEmailLogsForward :many
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
    el.created_at,
    el.updated_at
FROM email_log el
LEFT JOIN user u ON el.sent_by_id = u.id
WHERE el.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR el.subject LIKE sqlc.narg('search_query')
    OR EXISTS (
        SELECT 1 FROM email_recipient er
        WHERE er.email_log_id = el.id
        AND er.email LIKE sqlc.narg('search_query')
    )
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR el.created_at < sqlc.narg('cursor_created_at')
    OR (el.created_at = sqlc.narg('cursor_created_at') AND el.id < sqlc.narg('cursor_id'))
)
ORDER BY el.created_at DESC, el.id DESC
LIMIT ?;

-- name: ListEmailLogsBackward :many
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
    el.created_at,
    el.updated_at
FROM email_log el
LEFT JOIN user u ON el.sent_by_id = u.id
WHERE el.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR el.subject LIKE sqlc.narg('search_query')
    OR EXISTS (
        SELECT 1 FROM email_recipient er
        WHERE er.email_log_id = el.id
        AND er.email LIKE sqlc.narg('search_query')
    )
)
AND (
    el.created_at > sqlc.arg('cursor_created_at')
    OR (el.created_at = sqlc.arg('cursor_created_at') AND el.id > sqlc.arg('cursor_id'))
)
ORDER BY el.created_at ASC, el.id ASC
LIMIT ?;

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
    el.created_at,
    el.updated_at
FROM email_log el
LEFT JOIN user u ON el.sent_by_id = u.id
WHERE el.id = sqlc.arg('id')
AND el.account_id = sqlc.arg('account_id');

-- name: GetEmailRecipientsByEmailLogID :many
SELECT er.email
FROM email_recipient er
WHERE er.email_log_id = sqlc.arg('email_log_id')
ORDER BY er.created_at ASC;
