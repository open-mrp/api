-- name: CreateRegistrationSession :execresult
INSERT INTO registration_session (
    type_id, email, plan_code, step, verification_token, is_email_verified,
    is_existing_user, session_data, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW());

-- name: GetRegistrationSessionByID :one
SELECT id, type_id, email, plan_code, step, verification_token, is_email_verified,
       is_existing_user, user_id, account_id, stripe_customer_id,
       stripe_checkout_session_id, stripe_subscription_id, payment_completed, session_data,
       completed_at, created_at, updated_at
FROM registration_session
WHERE id = ?
LIMIT 1;

-- name: GetRegistrationSessionByToken :one
SELECT id, type_id, email, plan_code, step, verification_token, is_email_verified,
       is_existing_user, user_id, account_id, stripe_customer_id,
       stripe_checkout_session_id, stripe_subscription_id, payment_completed, session_data,
       completed_at, created_at, updated_at
FROM registration_session
WHERE verification_token = ?
LIMIT 1;

-- name: GetRegistrationSessionByEmail :one
SELECT id, type_id, email, plan_code, step, verification_token, is_email_verified,
       is_existing_user, user_id, account_id, stripe_customer_id,
       stripe_checkout_session_id, stripe_subscription_id, payment_completed, session_data,
       completed_at, created_at, updated_at
FROM registration_session
WHERE email = ?
AND completed_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: GetRegistrationSessionByTypeID :one
SELECT id, type_id, email, plan_code, step, verification_token, is_email_verified,
       is_existing_user, user_id, account_id, stripe_customer_id,
       stripe_checkout_session_id, stripe_subscription_id, payment_completed, session_data,
       completed_at, created_at, updated_at
FROM registration_session
WHERE type_id = ?
LIMIT 1;

-- name: UpdateRegistrationSessionStep :exec
UPDATE registration_session
SET step = ?, session_data = ?, updated_at = NOW()
WHERE id = ?;

-- name: UpdateRegistrationSessionEmailVerified :exec
UPDATE registration_session
SET is_email_verified = ?, is_existing_user = ?, updated_at = NOW()
WHERE id = ?;

-- name: UpdateRegistrationSessionUser :exec
UPDATE registration_session
SET user_id = ?, session_data = ?, updated_at = NOW()
WHERE id = ?;

-- name: UpdateRegistrationSessionStripeCustomer :exec
UPDATE registration_session
SET stripe_customer_id = ?, stripe_checkout_session_id = ?, updated_at = NOW()
WHERE id = ?;

-- name: UpdateRegistrationSessionPaymentCompleted :exec
UPDATE registration_session
SET payment_completed = ?, stripe_subscription_id = ?, updated_at = NOW()
WHERE id = ?;

-- name: CompleteRegistrationSession :exec
UPDATE registration_session
SET completed_at = NOW(), account_id = ?, updated_at = NOW()
WHERE id = ?;

-- name: DeleteRegistrationSession :exec
DELETE FROM registration_session
WHERE id = ?;

-- name: UpdateRegistrationSessionToken :exec
UPDATE registration_session
SET verification_token = ?, updated_at = NOW()
WHERE id = ?;
