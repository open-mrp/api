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

-- name: AllocateNextCustomerNumber :execresult
-- Atomically reserves the next customer number for the account and returns it via LAST_INSERT_ID.
-- See AllocateNextOrderNumber: two people registering at once used to be handed the same number.
INSERT INTO sys_property (id, account_id, sys_property_type_code, value, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), 'customer_number', LAST_INSERT_ID(1), NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE value = LAST_INSERT_ID(value + 1), updated_at = NOW(3);

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
    'commission_applied', 'billed_freight',
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

-- name: FindCustomerAccountBySalesOrderNumber :one
SELECT so.buyer_account_id
FROM sales_order so
JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
  AND ar.counterparty_account_id = so.buyer_account_id
  AND ar.account_relation_role_code = 'customer'
WHERE so.owner_account_id = sqlc.arg('owner_account_id')
AND so.sales_order_type_code = 'sales_order'
AND so.number = sqlc.arg('number')
LIMIT 1;

-- name: FindCustomerAccountByInvoiceNumber :one
SELECT so.buyer_account_id
FROM invoice i
JOIN sales_order so ON so.id = i.sales_order_id
JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
  AND ar.counterparty_account_id = so.buyer_account_id
  AND ar.account_relation_role_code = 'customer'
WHERE i.account_id = sqlc.arg('owner_account_id')
AND i.number = sqlc.arg('number')
AND so.owner_account_id = sqlc.arg('owner_account_id')
LIMIT 1;

-- name: GetCustomerDefaultPostalCodes :one
SELECT gb.postal_code AS billing_postal_code, gs.postal_code AS shipping_postal_code
FROM account_relation ar
LEFT JOIN address ab ON ab.id = ar.default_billing_address_id
LEFT JOIN geolocation gb ON gb.id = ab.geolocation_id
LEFT JOIN address ads ON ads.id = ar.default_shipping_address_id
LEFT JOIN geolocation gs ON gs.id = ads.geolocation_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
AND ar.counterparty_account_id = sqlc.arg('customer_account_id')
AND ar.account_relation_role_code = 'customer';

-- name: ListCustomerDefaultAddressesInPostalArea :many
-- Customers whose default billing or shipping address sits in the given five-character postal prefix.
SELECT ar.counterparty_account_id, a.name, ar.external_number,
       gb.street_line_1 AS billing_street_line_1, gb.postal_code AS billing_postal_code,
       gs.street_line_1 AS shipping_street_line_1, gs.postal_code AS shipping_postal_code
FROM account_relation ar
JOIN account a ON a.id = ar.counterparty_account_id
LEFT JOIN address ab ON ab.id = ar.default_billing_address_id
LEFT JOIN geolocation gb ON gb.id = ab.geolocation_id
LEFT JOIN address ads ON ads.id = ar.default_shipping_address_id
LEFT JOIN geolocation gs ON gs.id = ads.geolocation_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
AND ar.account_relation_role_code = 'customer'
AND (LEFT(gb.postal_code, 5) = sqlc.arg('postal_prefix') OR LEFT(gs.postal_code, 5) = sqlc.arg('postal_prefix'));

-- name: ListCustomersWithUserEmailDomain :many
SELECT DISTINCT ar.counterparty_account_id, a.name, ar.external_number
FROM account_relation ar
JOIN account a ON a.id = ar.counterparty_account_id
JOIN account_user au ON au.account_id = ar.counterparty_account_id
JOIN user u ON u.id = au.user_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
AND ar.account_relation_role_code = 'customer'
AND SUBSTRING_INDEX(u.email, '@', -1) = sqlc.arg('email_domain')
LIMIT 5;
