-- name: CreateAccount :exec
INSERT INTO account (
    id,
    name,
    account_type_code,
    onboarding_status_code,
    plan_code,
    created_at,
    updated_at
) VALUES (?, ?, ?, 'active', ?, NOW(), NOW());

-- name: GetAccountPlanCode :one
SELECT plan_code FROM account WHERE id = ?;

-- name: DeleteAccountByID :exec
DELETE FROM account WHERE id = ?;

-- name: GetAccountContext :one
SELECT 
    account.id,
    account.account_type_code,
    sandbox_account.owner_account_id
FROM account
LEFT JOIN sandbox_account ON account.id = sandbox_account.account_id
WHERE account.id = ?;

-- name: GetAccountPlanTypeIDByCode :one
SELECT type_id FROM account_plan 
WHERE plan_type_code = ? 
AND effective_at <= NOW() 
AND (expires_at IS NULL OR expires_at > NOW())
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
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW());

-- name: GetAccountByStripeCustomerID :one
SELECT id, name, plan_code FROM account WHERE internal_stripe_customer_id = ?;

