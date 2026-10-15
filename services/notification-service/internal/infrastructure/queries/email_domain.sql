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

-- name: ListEmailDomainsPage :many
-- One keyset page of an account's domains, newest first.
SELECT * FROM email_domain
WHERE account_id = sqlc.arg('account_id')
  AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR created_at < sqlc.narg('cursor_created_at')
    OR (created_at = sqlc.narg('cursor_created_at') AND id < sqlc.narg('cursor_id'))
  )
ORDER BY created_at DESC, id DESC
LIMIT ?;

-- name: ListPendingEmailDomains :many
-- Pending domains across every account not checked since the cutoff, least recently checked first, for the
-- background verifier. Served by email_domain_status_updated_idx.
SELECT * FROM email_domain
WHERE status = 'pending' AND updated_at < sqlc.arg('checked_before')
ORDER BY updated_at ASC, id ASC
LIMIT ?;

-- name: TouchEmailDomain :exec
-- Records that a pending domain was just re-checked, so the verifier rotates through the rest first.
UPDATE email_domain SET updated_at = NOW(3) WHERE id = ?;

-- name: MarkEmailDomainVerified :exec
UPDATE email_domain
SET status = 'verified', verified_at = NOW(3), updated_at = NOW(3)
WHERE id = ? AND account_id = ?;

-- name: UpdateEmailDomainStatus :exec
UPDATE email_domain
SET status = ?, dkim_tokens = ?, updated_at = NOW(3)
WHERE id = ? AND account_id = ?;

-- name: DeleteEmailDomain :execrows
DELETE FROM email_domain
WHERE id = ? AND account_id = ?;
