-- name: CreateGeolocation :exec
INSERT INTO geolocation (
    id,
    street_line_1,
    street_line_2,
    locality,
    state,
    postal_code,
    country,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW());

-- name: CreateAddress :exec
INSERT INTO address (
    id,
    name,
    geolocation_id,
    created_at,
    updated_at
) VALUES (?, ?, ?, NOW(), NOW());

-- name: CreateAccountAddress :exec
INSERT INTO account_address (
    id,
    account_id,
    address_id,
    created_at,
    updated_at
) VALUES (?, ?, ?, NOW(), NOW());

-- name: SetAccountDefaultBillingAddress :exec
UPDATE account SET default_billing_address_id = ?, updated_at = NOW() WHERE id = ?;

-- name: SetAccountDefaultShippingAddress :exec
UPDATE account SET default_shipping_address_id = ?, updated_at = NOW() WHERE id = ?;

-- name: CreateRole :exec
INSERT INTO role (
    id,
    name,
    role_type_code,
    account_id,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, NOW(), NOW());

-- name: CreateRolePermission :exec
INSERT INTO role_permission (
    id,
    role_id,
    permission_code,
    `create`,
    `read`,
    `update`,
    `delete`,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW());

-- name: GetAllPermissions :many
SELECT code FROM permission;

-- name: CreateAccountPortal :exec
INSERT INTO account_portal (
    id,
    owner_account_id,
    slug,
    created_at,
    updated_at
) VALUES (?, ?, ?, NOW(), NOW());
