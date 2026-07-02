-- name: CreateMessagingReport :exec
-- Persists a minimal abuse report filed by a participant against a conversation (optionally a
-- specific message). message_id is nullable.
INSERT INTO messaging_report (id, account_id, conversation_id, message_id, reporter_account_user_id, reason, created_at)
VALUES (?, ?, ?, ?, ?, ?, NOW(3));
