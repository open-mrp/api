-- name: CreatePortalRegistrationSession :exec
INSERT INTO portal_registration_session (
    type_id,
    user_id,
    seller_account_id,
    seller_slug,
    is_existing_customer,
    step,
    session_data,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3));

-- name: GetPortalRegistrationSessionByTypeID :one
SELECT * FROM portal_registration_session
WHERE type_id = ?
LIMIT 1;

-- name: GetIncompletePortalRegistrationSession :one
SELECT * FROM portal_registration_session
WHERE user_id = ?
  AND seller_account_id = ?
  AND completed_at IS NULL
  AND abandoned_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: UpdatePortalRegistrationSession :exec
UPDATE portal_registration_session
SET step = ?,
    session_data = ?,
    is_existing_customer = ?,
    updated_at = NOW(3)
WHERE type_id = ?;

-- name: CompletePortalRegistrationSession :exec
UPDATE portal_registration_session
SET completed_at = NOW(3),
    customer_id = ?,
    step = ?,
    updated_at = NOW(3)
WHERE type_id = ?;

-- name: AbandonPortalRegistrationSession :exec
UPDATE portal_registration_session
SET abandoned_at = NOW(3),
    updated_at = NOW(3)
WHERE type_id = ?;

-- name: ListPortalRegistrationSessionsForward :many
-- Seller-facing registration activity, newest first. The optional status filter reproduces DeriveStatus in SQL: 'expired' vs 'in_progress' splits incomplete sessions on the created-at expiry threshold (now - TTL) supplied by the service. The optional search term matches the registrant's captured customer name/number.
SELECT * FROM portal_registration_session prs
WHERE prs.seller_account_id = sqlc.arg('seller_account_id')
AND (
    sqlc.narg('status') IS NULL
    OR (sqlc.narg('status') = 'completed'   AND prs.completed_at IS NOT NULL)
    OR (sqlc.narg('status') = 'abandoned'   AND prs.abandoned_at IS NOT NULL)
    OR (sqlc.narg('status') = 'expired'     AND prs.completed_at IS NULL AND prs.abandoned_at IS NULL AND prs.created_at <  sqlc.arg('expiry_threshold'))
    OR (sqlc.narg('status') = 'in_progress' AND prs.completed_at IS NULL AND prs.abandoned_at IS NULL AND prs.created_at >= sqlc.arg('expiry_threshold'))
)
AND (
    sqlc.narg('search') IS NULL
    OR prs.type_id LIKE CONCAT('%', sqlc.narg('search'), '%')
    OR prs.session_data->>'$.customer_name' LIKE CONCAT('%', sqlc.narg('search'), '%')
    OR prs.session_data->>'$.customer_number' LIKE CONCAT('%', sqlc.narg('search'), '%')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR prs.created_at < sqlc.narg('cursor_created_at')
    OR (prs.created_at = sqlc.narg('cursor_created_at') AND prs.type_id < sqlc.narg('cursor_id'))
)
ORDER BY prs.created_at DESC, prs.type_id DESC
LIMIT ?;

-- name: ListPortalRegistrationSessionsBackward :many
SELECT * FROM portal_registration_session prs
WHERE prs.seller_account_id = sqlc.arg('seller_account_id')
AND (
    sqlc.narg('status') IS NULL
    OR (sqlc.narg('status') = 'completed'   AND prs.completed_at IS NOT NULL)
    OR (sqlc.narg('status') = 'abandoned'   AND prs.abandoned_at IS NOT NULL)
    OR (sqlc.narg('status') = 'expired'     AND prs.completed_at IS NULL AND prs.abandoned_at IS NULL AND prs.created_at <  sqlc.arg('expiry_threshold'))
    OR (sqlc.narg('status') = 'in_progress' AND prs.completed_at IS NULL AND prs.abandoned_at IS NULL AND prs.created_at >= sqlc.arg('expiry_threshold'))
)
AND (
    sqlc.narg('search') IS NULL
    OR prs.type_id LIKE CONCAT('%', sqlc.narg('search'), '%')
    OR prs.session_data->>'$.customer_name' LIKE CONCAT('%', sqlc.narg('search'), '%')
    OR prs.session_data->>'$.customer_number' LIKE CONCAT('%', sqlc.narg('search'), '%')
)
AND (
    prs.created_at > sqlc.arg('cursor_created_at')
    OR (prs.created_at = sqlc.arg('cursor_created_at') AND prs.type_id > sqlc.arg('cursor_id'))
)
ORDER BY prs.created_at ASC, prs.type_id ASC
LIMIT ?;
