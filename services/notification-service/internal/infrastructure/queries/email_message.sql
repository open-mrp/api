-- name: TryInsertEmailMessage :execrows
-- At-least-once dedup: the unique rfc_message_id index collapses a redelivered inbound email to a
-- no-op (0 rows affected), so the caller can skip re-threading. INSERT IGNORE swallows the dup-key.
INSERT IGNORE INTO email_message (
    id, account_id, conversation_id, message_id, email_inbox_id, direction,
    rfc_message_id, in_reply_to, `references`, from_addr, to_addrs, cc_addrs,
    subject, raw_s3_key, ses_message_id, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3));

-- name: GetEmailMessageByRfcID :one
SELECT * FROM email_message
WHERE rfc_message_id = ?;

-- name: FindEmailThreadConversation :one
-- Threading: given the candidate rfc Message-IDs from an inbound mail's In-Reply-To + References
-- headers, find the most recent prior email in the same thread and return its conversation. Served
-- by email_message_rfc_uq.
SELECT conversation_id, email_inbox_id FROM email_message
WHERE rfc_message_id IN (sqlc.slice('rfc_message_ids'))
ORDER BY created_at DESC
LIMIT 1;

-- name: GetLatestInboundEmailMessage :one
-- Outbound threading: the newest inbound email in a conversation, whose rfc_message_id seeds the
-- In-Reply-To / References headers of an agent's reply.
SELECT * FROM email_message
WHERE conversation_id = ? AND direction = 'inbound'
ORDER BY created_at DESC
LIMIT 1;
