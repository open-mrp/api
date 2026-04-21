-- name: CreateAccount :exec
INSERT INTO account (
    id,
    name,
    account_type_code,
    onboarding_status_code,
    created_at,
    updated_at
) VALUES (?, ?, ?, 'active', NOW(3), NOW(3));

-- name: GetAccountPlanCode :one
SELECT ap.plan_type_code AS plan_code
FROM account a
JOIN account_billing ab ON a.account_billing_id = ab.id
JOIN account_plan ap ON ab.account_plan_id = ap.type_id
WHERE a.id = ?;

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
        WHEN sa.owner_account_id IS NOT NULL THEN owner_ab.subscription_status
        ELSE ab.subscription_status
    END AS subscription_status,
    CASE
        WHEN sa.owner_account_id IS NOT NULL THEN owner_ap.plan_type_code
        ELSE ap.plan_type_code
    END AS plan_code,
    CASE
        WHEN sa.owner_account_id IS NOT NULL THEN owner_ab.agent_monthly_spending_cap_cents
        ELSE ab.agent_monthly_spending_cap_cents
    END AS agent_monthly_spending_cap_cents
FROM account a
LEFT JOIN account_billing ab ON a.account_billing_id = ab.id
LEFT JOIN account_plan ap ON ab.account_plan_id = ap.type_id
LEFT JOIN sandbox_account sa ON a.id = sa.account_id
LEFT JOIN account owner ON sa.owner_account_id = owner.id
LEFT JOIN account_billing owner_ab ON owner.account_billing_id = owner_ab.id
LEFT JOIN account_plan owner_ap ON owner_ab.account_plan_id = owner_ap.type_id
WHERE a.id = ?;

-- name: GetAccountPlanIDAndPeriodEnd :one
-- Returns the account's current plan id (used with ListAccountPlanLimits) and
-- subscription period end (used to derive the billing period start for usage limits).
SELECT ab.account_plan_id, ab.subscription_current_period_end
FROM account a
JOIN account_billing ab ON a.account_billing_id = ab.id
WHERE a.id = ?;

-- name: GetAccountPlanTypeIDByCode :one
SELECT type_id FROM account_plan
WHERE plan_type_code = ?
AND effective_at <= NOW(3)
AND (expires_at IS NULL OR expires_at > NOW(3))
ORDER BY effective_at DESC, version DESC
LIMIT 1;

-- name: CreateAccountBilling :exec
INSERT INTO account_billing (
    id,
    account_plan_id,
    internal_stripe_customer_id,
    internal_stripe_subscription_id,
    subscription_status,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, NOW(3), NOW(3));

-- name: CreateAccountForRegistration :exec
INSERT INTO account (
    id,
    name,
    account_type_code,
    onboarding_status_code,
    account_billing_id,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, NOW(3), NOW(3));

-- name: GetAccountByStripeCustomerID :one
SELECT a.id, a.name, ap.plan_type_code AS plan_code
FROM account a
JOIN account_billing ab ON a.account_billing_id = ab.id
JOIN account_plan ap ON ab.account_plan_id = ap.type_id
WHERE ab.internal_stripe_customer_id = ?;

-- name: UpdateAccountSubscription :exec
UPDATE account_billing SET
    subscription_status = COALESCE(?, subscription_status),
    account_plan_id = COALESCE(sqlc.narg('account_plan_id'), account_plan_id),
    internal_stripe_subscription_id = COALESCE(?, internal_stripe_subscription_id),
    subscription_current_period_end = COALESCE(?, subscription_current_period_end),
    internal_stripe_customer_id = COALESCE(?, internal_stripe_customer_id),
    stripe_billing_profile_id = COALESCE(?, stripe_billing_profile_id),
    stripe_billing_cadence_id = COALESCE(?, stripe_billing_cadence_id),
    stripe_pricing_plan_subscription_id = COALESCE(?, stripe_pricing_plan_subscription_id),
    servicing_status = COALESCE(?, servicing_status),
    collection_status = COALESCE(?, collection_status),
    updated_at = NOW(3)
WHERE account_billing.id = (SELECT account_billing_id FROM account WHERE account.id = sqlc.arg(account_id));

-- name: GetSandboxLimitByAccountID :one
SELECT apl.value
FROM account a
JOIN account_billing ab ON a.account_billing_id = ab.id
JOIN account_plan_limit apl ON apl.account_plan_id = ab.account_plan_id
    AND apl.key = 'sandboxes_maximum'
WHERE a.id = ?;

-- name: CountNonSandboxAccountsByPlanCode :one
SELECT COUNT(*) AS cnt
FROM account a
JOIN account_billing ab ON a.account_billing_id = ab.id
JOIN account_plan ap ON ab.account_plan_id = ap.type_id
LEFT JOIN sandbox_account sa ON a.id = sa.account_id
WHERE ap.plan_type_code = ?
  AND sa.id IS NULL
  AND a.onboarding_status_code = 'active';

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
UPDATE account_billing SET
    internal_stripe_customer_id = NULL,
    internal_stripe_subscription_id = NULL,
    subscription_status = NULL,
    subscription_current_period_end = NULL,
    stripe_billing_profile_id = NULL,
    stripe_billing_cadence_id = NULL,
    stripe_pricing_plan_subscription_id = NULL,
    servicing_status = NULL,
    collection_status = NULL,
    updated_at = NOW(3)
WHERE account_billing.id = (SELECT account_billing_id FROM account WHERE account.id = ?);

-- name: ClearAccountPricingPlanSubscription :exec
UPDATE account_billing SET
    stripe_pricing_plan_subscription_id = NULL,
    servicing_status = NULL,
    collection_status = NULL,
    updated_at = NOW(3)
WHERE account_billing.id = (SELECT account_billing_id FROM account WHERE account.id = ?);

-- name: UpdateAgentSpendingCap :exec
UPDATE account_billing SET agent_monthly_spending_cap_cents = ?, updated_at = NOW(3)
WHERE account_billing.id = (SELECT account_billing_id FROM account WHERE account.id = sqlc.arg(account_id));

-- name: GetAgentSpendingCap :one
SELECT ab.agent_monthly_spending_cap_cents
FROM account a
JOIN account_billing ab ON a.account_billing_id = ab.id
WHERE a.id = ?;

-- name: HasActiveBillingPlan :one
SELECT EXISTS(
    SELECT 1 FROM account a
    JOIN account_billing ab ON a.account_billing_id = ab.id
    WHERE a.id = ? AND ab.account_plan_id IS NOT NULL
) AS has_plan;

-- name: ListAccountPlanLimits :many
SELECT `key`, value
FROM account_plan_limit
WHERE account_plan_id = ?;

-- name: ListAccountPlanFeatures :many
SELECT `key`, enabled
FROM account_plan_feature
WHERE account_plan_id = ?;

-- name: GetAccountNameByID :one
SELECT name FROM account WHERE id = ?;

-- name: GetAccountBrandingByAccountID :one
SELECT logo_url FROM account_branding WHERE owner_account_id = ?;

-- name: GetAccountPortalSlugByAccountID :one
SELECT slug FROM account_portal WHERE owner_account_id = ?;

-- name: GetAccountByID :one
SELECT
    a.id,
    a.name,
    a.default_billing_address_id,
    a.default_shipping_address_id,
    a.created_at,
    a.updated_at,
    ab.id AS branding_id,
    ab.support_email AS branding_support_email,
    ab.phone_number AS branding_phone_number,
    ab.logo_url AS branding_logo_url,
    ab.facebook_handle AS branding_facebook_handle,
    ab.instagram_handle AS branding_instagram_handle,
    ab.linkedin_handle AS branding_linkedin_handle,
    ab.twitter_handle AS branding_twitter_handle,
    ab.website_url AS branding_website_url,
    ab.created_at AS branding_created_at,
    ab.updated_at AS branding_updated_at,
    ap.id AS portal_id,
    ap.slug AS portal_slug,
    ap.created_at AS portal_created_at,
    ap.updated_at AS portal_updated_at
FROM account a
LEFT JOIN account_branding ab ON ab.owner_account_id = a.id
LEFT JOIN account_portal ap ON ap.owner_account_id = a.id
WHERE a.id = sqlc.arg('account_id');

-- name: GetPublicAccountBySlug :one
SELECT
    a.id,
    a.name,
    a.default_billing_address_id,
    ap.slug,
    ab.support_email,
    ab.logo_url
FROM account_portal ap
JOIN account a ON a.id = ap.owner_account_id
LEFT JOIN account_branding ab ON ab.owner_account_id = a.id
WHERE ap.slug = sqlc.arg('slug');

-- name: UpdateAccountName :execresult
UPDATE account SET name = sqlc.arg('name'), updated_at = NOW(3) WHERE id = sqlc.arg('account_id');

-- name: UpdateAccountBranding :execresult
UPDATE account_branding SET
    support_email = COALESCE(sqlc.narg('support_email'), support_email),
    phone_number = COALESCE(sqlc.narg('phone_number'), phone_number),
    facebook_handle = COALESCE(sqlc.narg('facebook_handle'), facebook_handle),
    instagram_handle = COALESCE(sqlc.narg('instagram_handle'), instagram_handle),
    linkedin_handle = COALESCE(sqlc.narg('linkedin_handle'), linkedin_handle),
    twitter_handle = COALESCE(sqlc.narg('twitter_handle'), twitter_handle),
    website_url = COALESCE(sqlc.narg('website_url'), website_url),
    updated_at = NOW(3)
WHERE owner_account_id = sqlc.arg('account_id');

-- name: UpdateAccountPortalSlug :execresult
UPDATE account_portal SET slug = sqlc.arg('slug'), updated_at = NOW(3) WHERE owner_account_id = sqlc.arg('account_id');

-- name: ExistsPortalSlug :one
SELECT EXISTS(
    SELECT 1 FROM account_portal
    WHERE slug = sqlc.arg('slug')
    AND owner_account_id != sqlc.arg('exclude_account_id')
) AS slug_exists;

-- name: UpdateAccountBrandingLogoURL :exec
UPDATE account_branding SET logo_url = sqlc.arg('logo_url'), updated_at = NOW(3) WHERE owner_account_id = sqlc.arg('account_id');

-- name: GetAccountBrandingLogoKey :one
SELECT logo_url FROM account_branding WHERE owner_account_id = sqlc.arg('account_id');
