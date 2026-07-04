-- name: FindCustomerAccountByExternalNumber :one
SELECT
    a.id
FROM account a
JOIN account_relation ar ON ar.counterparty_account_id = a.id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
AND TRIM(ar.external_number) = TRIM(sqlc.arg('external_number'))
AND ar.account_relation_role_code = 'customer'
LIMIT 1;

-- name: InsertAccountUserForCustomerRegistration :exec
INSERT INTO account_user (id, account_id, user_id, role_id, last_used_at, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('user_id'), sqlc.arg('role_id'), NOW(3), NOW(3), NOW(3));

-- name: GetNextCustomerNumber :one
SELECT COALESCE(
    (SELECT MAX(CAST(sp.value AS UNSIGNED)) + 1
     FROM sys_property sp
     WHERE sp.account_id = sqlc.arg('account_id')
     AND sp.sys_property_type_code = 'customer_number'),
    1
) AS next_number;

-- name: UpdateNextCustomerNumber :exec
INSERT INTO sys_property (id, account_id, sys_property_type_code, value, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), 'customer_number', sqlc.arg('value'), NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE value = sqlc.arg('value'), updated_at = NOW(3);

-- name: InsertGeolocationForCustomer :exec
INSERT INTO geolocation (
    id, street_line_1, street_line_2, locality, state, postal_code, country, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3));

-- name: InsertAddressForCustomer :exec
INSERT INTO address (
    id, name, geolocation_id, created_at, updated_at
) VALUES (?, ?, ?, NOW(3), NOW(3));

-- name: InsertAccountForCustomer :exec
INSERT INTO account (
    id, name, account_type_code, onboarding_status_code,
    default_billing_address_id, default_shipping_address_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('name'), 'company', 'unclaimed',
    sqlc.arg('default_billing_address_id'), sqlc.arg('default_shipping_address_id'),
    NOW(3), NOW(3)
);

-- name: InsertAccountBrandingForCustomer :exec
INSERT INTO account_branding (
    id, owner_account_id, support_email, phone_number, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('owner_account_id'), sqlc.arg('support_email'), sqlc.narg('phone_number'), NOW(3), NOW(3)
);

-- name: InsertAccountRelationForCustomer :exec
INSERT INTO account_relation (
    id, owner_account_id, counterparty_account_id, account_relation_role_code,
    alias, external_number, shipping_term_id, payment_term_id,
    default_billing_address_id, default_shipping_address_id,
    account_group_id, stripe_email,
    commission_status_code, freight_status_code,
    default_carrier_id, default_carrier_option_id,
    account_status_code, priority_code,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('owner_account_id'), sqlc.arg('counterparty_account_id'), 'customer',
    sqlc.arg('alias'), sqlc.arg('external_number'), sqlc.arg('shipping_term_id'), sqlc.arg('payment_term_id'),
    sqlc.arg('default_billing_address_id'), sqlc.arg('default_shipping_address_id'),
    sqlc.arg('account_group_id'), sqlc.arg('stripe_email'),
    'applied', 'billed',
    NULL, sqlc.narg('default_carrier_option_id'),
    'normal', 'normal',
    NOW(3), NOW(3)
);

-- name: InsertAccountAddressForCustomer :exec
INSERT INTO account_address (
    id, account_id, address_id, created_at, updated_at
) VALUES (?, ?, ?, NOW(3), NOW(3));

-- name: GetUserEmailByID :one
SELECT email FROM user WHERE id = sqlc.arg('id');
