-- name: CreateEmailInbox :exec
INSERT INTO email_inbox (
    id, account_id, email_domain_id, address, from_name, status,
    agent_config_id, agent_trigger_policy, agent_trigger_keywords, group_id, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, 'active', ?, ?, ?, ?, NOW(3), NOW(3));

-- name: GetEmailInboxByID :one
SELECT * FROM email_inbox
WHERE id = ? AND account_id = ?;

-- name: GetEmailInboxByAddress :one
-- Inbound routing: resolve the inbox a piece of mail was delivered to. Address is globally unique.
SELECT * FROM email_inbox
WHERE address = ?;

-- name: GetEmailInboxByIDSystem :one
-- Inbound routing (system, no account scope): resolve the inbox whose per-inbox forwarding address
-- (<inbox_id>@<inbound domain>) a forwarded email was delivered to. IDs are globally unique.
SELECT * FROM email_inbox
WHERE id = ?;

-- name: ListEmailInboxesByAccount :many
SELECT * FROM email_inbox
WHERE account_id = ?
ORDER BY created_at DESC, id DESC;

-- name: CountEmailInboxesByDomain :one
SELECT COUNT(*) FROM email_inbox
WHERE account_id = ? AND email_domain_id = ?;

-- name: UpdateEmailInbox :exec
UPDATE email_inbox
SET from_name = COALESCE(?, from_name), status = ?, agent_config_id = COALESCE(?, agent_config_id), agent_trigger_policy = COALESCE(?, agent_trigger_policy), agent_trigger_keywords = COALESCE(?, agent_trigger_keywords), group_id = COALESCE(?, group_id), updated_at = NOW(3)
WHERE id = ? AND account_id = ?;

-- name: DeleteEmailInbox :execrows
DELETE FROM email_inbox
WHERE id = ? AND account_id = ?;
