-- name: GetLimitsByAccountID :many
SELECT apl.`key`, apl.value
FROM account_plan_limit apl
JOIN account_plan ap ON ap.type_id = apl.account_plan_id
JOIN account a ON a.account_plan_id = ap.type_id
WHERE a.id = ?
ORDER BY apl.`key` ASC;

-- name: CountUsersByAccountID :one
SELECT COUNT(*) AS cnt FROM account_user WHERE account_id = ?;

-- name: CountSandboxesByAccountID :one
SELECT COUNT(*) AS cnt FROM sandbox_account WHERE owner_account_id = ?;

-- name: CountInvoicesByAccountID :one
SELECT COUNT(*) AS cnt FROM invoice WHERE account_id = ?;

-- name: CountInvoicesByAccountIDInPeriod :one
SELECT COUNT(*) AS cnt FROM invoice WHERE account_id = ? AND created_at >= ?;

-- name: CountBatchesByAccountID :one
SELECT COUNT(*) AS cnt FROM batch WHERE account_id = ?;

-- name: CountBatchesByAccountIDInPeriod :one
SELECT COUNT(*) AS cnt FROM batch WHERE account_id = ? AND created_at >= ?;

-- name: GetAccountSubscriptionInfo :one
SELECT subscription_status, subscription_current_period_end, internal_stripe_subscription_id
FROM account
WHERE id = ?;

-- name: GetStripeCustomerIDByAccountID :one
SELECT internal_stripe_customer_id
FROM account
WHERE id = ?;

-- name: GetAccountNameAndPlanCode :one
SELECT name, plan_code
FROM account
WHERE id = ?;

-- name: GetUserEmailByID :one
SELECT email, name AS display_name
FROM `user`
WHERE id = ?;

-- name: GetAdminEmailByAccountID :one
SELECT u.email
FROM account_user au
JOIN `user` u ON u.id = au.user_id
JOIN role r ON r.id = au.role_id
WHERE au.account_id = ? AND r.role_type_code = 'admin'
LIMIT 1;

-- name: UpdateStripeCustomerIDByAccountID :exec
UPDATE account SET internal_stripe_customer_id = ? WHERE id = ?;
