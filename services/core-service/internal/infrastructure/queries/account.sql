-- name: CreateAccount :exec
INSERT INTO account (
    id,
    name,
    account_type_code,
    onboarding_status_code,
    plan_code,
    created_at,
    updated_at
) VALUES (?, ?, ?, 'active', ?, NOW(3), NOW(3));

-- name: GetAccountPlanCode :one
SELECT plan_code FROM account WHERE id = ?;

-- name: DeleteAccountByID :exec
DELETE FROM account WHERE id = ?;

-- name: DeleteAccountByIDIfSandbox :execresult
DELETE FROM account WHERE id = ? AND account_type_code = 'sandbox';

-- name: GetAccountContext :one
SELECT
    a.id,
    a.account_type_code,
    sa.owner_account_id,
    CASE
        WHEN sa.owner_account_id IS NOT NULL THEN owner.subscription_status
        ELSE a.subscription_status
    END AS subscription_status
FROM account a
LEFT JOIN sandbox_account sa ON a.id = sa.account_id
LEFT JOIN account owner ON sa.owner_account_id = owner.id
WHERE a.id = ?;

-- name: GetAccountPlanTypeIDByCode :one
SELECT type_id FROM account_plan 
WHERE plan_type_code = ? 
AND effective_at <= NOW(3) 
AND (expires_at IS NULL OR expires_at > NOW(3))
ORDER BY effective_at DESC, version DESC
LIMIT 1;

-- name: CreateAccountForRegistration :exec
INSERT INTO account (
    id,
    name,
    account_type_code,
    onboarding_status_code,
    plan_code,
    account_plan_id,
    internal_stripe_customer_id,
    internal_stripe_subscription_id,
    subscription_status,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3));

-- name: GetAccountByStripeCustomerID :one
SELECT id, name, plan_code FROM account WHERE internal_stripe_customer_id = ?;

-- name: UpdateAccountSubscription :exec
UPDATE account SET
    subscription_status = ?,
    plan_code = ?,
    account_plan_id = ?,
    internal_stripe_subscription_id = ?,
    subscription_current_period_end = ?,
    internal_stripe_customer_id = COALESCE(?, internal_stripe_customer_id),
    updated_at = NOW(3)
WHERE id = ?;

-- name: GetSandboxLimitByAccountID :one
SELECT apl.value
FROM account a
JOIN account_plan ap ON ap.plan_type_code = a.plan_code
    AND ap.effective_at <= NOW(3)
    AND (ap.expires_at IS NULL OR ap.expires_at > NOW(3))
JOIN account_plan_limit apl ON apl.account_plan_id = ap.type_id
    AND apl.key = 'sandboxes_maximum'
WHERE a.id = ?
ORDER BY ap.effective_at DESC, ap.version DESC
LIMIT 1;

-- name: CountNonSandboxAccountsByPlanCode :one
SELECT COUNT(*) AS cnt
FROM account a
LEFT JOIN sandbox_account sa ON a.id = sa.account_id
WHERE a.plan_code = ?
  AND sa.id IS NULL;

-- name: GetSeatLimitByPlanCode :one
SELECT apl.value
FROM account_plan ap
JOIN account_plan_limit apl ON apl.account_plan_id = ap.type_id
    AND apl.key = 'seats_maximum'
WHERE ap.plan_type_code = ?
    AND ap.effective_at <= NOW(3)
    AND (ap.expires_at IS NULL OR ap.expires_at > NOW(3))
ORDER BY ap.effective_at DESC, ap.version DESC
LIMIT 1;

-- name: ClearAccountStripeCustomer :exec
UPDATE account SET
    internal_stripe_customer_id = NULL,
    internal_stripe_subscription_id = NULL,
    subscription_status = NULL,
    subscription_current_period_end = NULL,
    updated_at = NOW(3)
WHERE id = ?;

