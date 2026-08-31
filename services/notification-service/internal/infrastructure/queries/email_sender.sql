-- name: UpsertEmailSender :exec
INSERT INTO email_sender (
    id, account_id, email_domain_id, local_part, from_name, reply_to, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE
    email_domain_id = VALUES(email_domain_id),
    local_part = VALUES(local_part),
    from_name = VALUES(from_name),
    reply_to = VALUES(reply_to),
    updated_at = NOW(3);

-- name: GetEmailSenderByAccount :one
SELECT sqlc.embed(s), d.domain, d.status, d.mail_from_domain
FROM email_sender s
JOIN email_domain d ON d.id = s.email_domain_id
WHERE s.account_id = ?;

-- name: DeleteEmailSender :execrows
DELETE FROM email_sender
WHERE account_id = ?;

-- name: DeleteEmailSenderByDomain :execrows
DELETE FROM email_sender
WHERE email_domain_id = ? AND account_id = ?;

-- name: SetEmailDomainMailFrom :exec
UPDATE email_domain
SET mail_from_domain = ?, updated_at = NOW(3)
WHERE id = ? AND account_id = ?;
