-- name: CreateEmailDomain :exec
INSERT INTO email_domain (
    id, account_id, domain, status, dkim_tokens, created_at, updated_at
) VALUES (?, ?, ?, 'pending', ?, NOW(3), NOW(3));

-- name: GetEmailDomainByID :one
SELECT * FROM email_domain
WHERE id = ? AND account_id = ?;

-- name: GetEmailDomainByDomain :one
SELECT * FROM email_domain
WHERE domain = ?;

-- name: ListEmailDomainsByAccount :many
SELECT * FROM email_domain
WHERE account_id = ?
ORDER BY created_at DESC, id DESC;

-- name: MarkEmailDomainVerified :exec
UPDATE email_domain
SET status = 'verified', verified_at = NOW(3), updated_at = NOW(3)
WHERE id = ? AND account_id = ?;

-- name: UpdateEmailDomainStatus :exec
UPDATE email_domain
SET status = ?, dkim_tokens = ?, updated_at = NOW(3)
WHERE id = ? AND account_id = ?;
