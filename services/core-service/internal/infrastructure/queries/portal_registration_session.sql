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
