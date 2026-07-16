-- name: CreateEmailLog :exec
INSERT INTO email_log (id, has_sent, account_id, sent_by_id, subject, filename, ses_message_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3));

-- name: CreateEmailRecipient :exec
INSERT INTO email_recipient (id, email, email_log_id, created_at, updated_at)
VALUES (?, ?, ?, NOW(3), NOW(3));

-- name: FindEmailLogBySesMessageID :one
SELECT id, has_sent, account_id, sent_by_id, subject, filename, ses_message_id, created_at, updated_at
FROM email_log
WHERE ses_message_id = ?;