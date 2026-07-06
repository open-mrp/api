-- name: CreatePortalDomain :exec
INSERT INTO portal_domain (
    id, account_id, domain, status, dns_records, created_at, updated_at
) VALUES (?, ?, ?, 'pending', NULL, NOW(3), NOW(3));

-- name: GetPortalDomainByID :one
SELECT * FROM portal_domain
WHERE id = ? AND account_id = ?;

-- name: GetPortalDomainByAccountID :one
SELECT * FROM portal_domain
WHERE account_id = ?;

-- name: GetPortalDomainByDomain :one
SELECT * FROM portal_domain
WHERE domain = ?;

-- name: ListPortalDomainsByAccount :many
SELECT * FROM portal_domain
WHERE account_id = ?
ORDER BY created_at DESC, id DESC;

-- name: BatchGetPortalDomainsByIDs :many
SELECT * FROM portal_domain
WHERE account_id = sqlc.arg('account_id') AND id IN (sqlc.slice('ids'));

-- name: UpdatePortalDomainProviderState :exec
UPDATE portal_domain
SET status = ?, dns_records = ?, updated_at = NOW(3)
WHERE id = ?;

-- name: MarkPortalDomainVerified :exec
UPDATE portal_domain
SET status = 'verified', verified_at = NOW(3), updated_at = NOW(3)
WHERE id = ?;

-- name: DeletePortalDomain :execrows
DELETE FROM portal_domain
WHERE id = ? AND account_id = ?;

-- name: ResolveVerifiedPortalHost :one
SELECT
    a.id,
    a.name,
    a.default_billing_address_id,
    ap.slug,
    ab.support_email,
    ab.logo_url,
    pd.domain
FROM portal_domain pd
JOIN account a ON a.id = pd.account_id
JOIN account_portal ap ON ap.owner_account_id = a.id
LEFT JOIN account_branding ab ON ab.owner_account_id = a.id
WHERE pd.domain = sqlc.arg('domain') AND pd.status = 'verified';
