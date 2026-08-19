-- name: ListAddressesForward :many
SELECT
    a.id,
    a.name,
    a.phone,
    a.email,
    a.is_drop_ship,
    a.receive_calendar_id,
    a.created_at,
    a.updated_at,
    g.id AS geolocation_id,
    g.street_line_1,
    g.street_line_2,
    g.locality,
    g.state,
    g.postal_code,
    g.country,
    g.google_place_id,
    g.latitude,
    g.longitude,
    g.timezone
FROM address a
JOIN geolocation g ON a.geolocation_id = g.id
JOIN account_address aa ON aa.address_id = a.id
WHERE aa.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR a.name LIKE sqlc.narg('search_query')
    OR g.street_line_1 LIKE sqlc.narg('search_query')
    OR g.street_line_2 LIKE sqlc.narg('search_query')
    OR g.locality LIKE sqlc.narg('search_query')
    OR g.state LIKE sqlc.narg('search_query')
    OR g.postal_code LIKE sqlc.narg('search_query')
    OR g.country LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('drop_ship') IS NULL
    OR a.is_drop_ship = sqlc.narg('drop_ship')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR a.created_at < sqlc.narg('cursor_created_at')
    OR (a.created_at = sqlc.narg('cursor_created_at') AND a.id < sqlc.narg('cursor_id'))
)
ORDER BY a.created_at DESC, a.id DESC
LIMIT ?;

-- name: ListAddressesBackward :many
SELECT
    a.id,
    a.name,
    a.phone,
    a.email,
    a.is_drop_ship,
    a.receive_calendar_id,
    a.created_at,
    a.updated_at,
    g.id AS geolocation_id,
    g.street_line_1,
    g.street_line_2,
    g.locality,
    g.state,
    g.postal_code,
    g.country,
    g.google_place_id,
    g.latitude,
    g.longitude,
    g.timezone
FROM address a
JOIN geolocation g ON a.geolocation_id = g.id
JOIN account_address aa ON aa.address_id = a.id
WHERE aa.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR a.name LIKE sqlc.narg('search_query')
    OR g.street_line_1 LIKE sqlc.narg('search_query')
    OR g.street_line_2 LIKE sqlc.narg('search_query')
    OR g.locality LIKE sqlc.narg('search_query')
    OR g.state LIKE sqlc.narg('search_query')
    OR g.postal_code LIKE sqlc.narg('search_query')
    OR g.country LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('drop_ship') IS NULL
    OR a.is_drop_ship = sqlc.narg('drop_ship')
)
AND (
    a.created_at > sqlc.arg('cursor_created_at')
    OR (a.created_at = sqlc.arg('cursor_created_at') AND a.id > sqlc.arg('cursor_id'))
)
ORDER BY a.created_at ASC, a.id ASC
LIMIT ?;

-- name: GetAddress :one
SELECT
    a.id,
    a.name,
    a.phone,
    a.email,
    a.is_drop_ship,
    a.receive_calendar_id,
    a.created_at,
    a.updated_at,
    g.id AS geolocation_id,
    g.street_line_1,
    g.street_line_2,
    g.locality,
    g.state,
    g.postal_code,
    g.country,
    g.google_place_id,
    g.latitude,
    g.longitude,
    g.timezone
FROM address a
JOIN geolocation g ON a.geolocation_id = g.id
JOIN account_address aa ON aa.address_id = a.id
WHERE a.id = sqlc.arg('id')
AND aa.account_id = sqlc.arg('account_id');

-- name: GetAddressesByIDs :many
-- Returns addresses matching the given IDs that belong to the caller's
-- account, via the account_address junction. Addresses are always
-- account-scoped (no system rows).
SELECT
    a.id,
    a.name,
    a.phone,
    a.email,
    a.is_drop_ship,
    a.receive_calendar_id,
    a.created_at,
    a.updated_at,
    g.id AS geolocation_id,
    g.street_line_1,
    g.street_line_2,
    g.locality,
    g.state,
    g.postal_code,
    g.country,
    g.google_place_id,
    g.latitude,
    g.longitude,
    g.timezone
FROM address a
JOIN geolocation g ON a.geolocation_id = g.id
JOIN account_address aa ON aa.address_id = a.id
WHERE a.id IN (sqlc.slice('ids'))
AND aa.account_id = sqlc.arg('account_id');

-- name: InsertGeolocation :exec
INSERT INTO geolocation (
    id,
    street_line_1,
    street_line_2,
    locality,
    state,
    postal_code,
    country,
    google_place_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.narg('street_line_1'),
    sqlc.narg('street_line_2'),
    sqlc.narg('locality'),
    sqlc.narg('state'),
    sqlc.narg('postal_code'),
    sqlc.arg('country'),
    sqlc.narg('google_place_id'),
    NOW(3),
    NOW(3)
);

-- name: InsertAddress :exec
INSERT INTO address (
    id,
    name,
    phone,
    email,
    is_drop_ship,
    receive_calendar_id,
    geolocation_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.narg('phone'),
    sqlc.narg('email'),
    sqlc.arg('is_drop_ship'),
    sqlc.narg('receive_calendar_id'),
    sqlc.arg('geolocation_id'),
    NOW(3),
    NOW(3)
);

-- name: InsertAccountAddress :exec
INSERT INTO account_address (
    id,
    account_id,
    address_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_id'),
    sqlc.arg('address_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateAddress :execresult
UPDATE address SET
    name = COALESCE(sqlc.narg('name'), name),
    phone = sqlc.narg('phone'),
    email = sqlc.narg('email'),
    is_drop_ship = COALESCE(sqlc.narg('is_drop_ship'), is_drop_ship),
    receive_calendar_id = sqlc.narg('receive_calendar_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: UpdateGeolocation :execresult
UPDATE geolocation SET
    street_line_1 = COALESCE(sqlc.narg('street_line_1'), street_line_1),
    street_line_2 = sqlc.narg('street_line_2'),
    locality = COALESCE(sqlc.narg('locality'), locality),
    state = COALESCE(sqlc.narg('state'), state),
    postal_code = COALESCE(sqlc.narg('postal_code'), postal_code),
    country = COALESCE(sqlc.narg('country'), country),
    google_place_id = sqlc.narg('google_place_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: UpdateAddressGeolocationID :execresult
UPDATE address SET
    geolocation_id = sqlc.arg('geolocation_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: DeleteAccountAddressByAddressID :exec
DELETE FROM account_address WHERE address_id = sqlc.arg('address_id');

-- name: DeleteAddress :exec
DELETE FROM address WHERE id = sqlc.arg('id');

-- name: CountAddressesByGeolocationID :one
SELECT COUNT(*) AS count FROM address WHERE geolocation_id = sqlc.arg('geolocation_id');

-- name: GetGeolocationIDByAddressID :one
SELECT geolocation_id FROM address WHERE id = sqlc.arg('id');

-- name: CheckAddressInAccount :one
SELECT EXISTS(
    SELECT 1 FROM account_address
    WHERE account_id = sqlc.arg('account_id')
    AND address_id = sqlc.arg('address_id')
) AS `exists`;

-- name: CheckAddressUsedInSalesOrder :one
SELECT so.number FROM sales_order so
WHERE so.billing_address_id = sqlc.arg('address_id')
   OR so.shipping_address_id = sqlc.arg('address_id')
LIMIT 1;

-- name: CheckAddressUsedInInvoice :one
SELECT i.number FROM invoice i
WHERE i.billing_address_id = sqlc.arg('address_id')
LIMIT 1;

-- name: CheckAddressUsedInShipment :one
SELECT s.number FROM shipment s
WHERE s.shipping_address_id = sqlc.arg('address_id')
LIMIT 1;

-- name: CheckAddressUsedAsAccountDefault :one
-- Only an active account's default billing/shipping address blocks deletion. A
-- non-active (e.g. unclaimed, vendor-managed) account's default does not: the address
-- can be deleted and its account defaults are switched over to the account-relation
-- defaults by SwitchAccountDefaultAddressToRelation.
SELECT a.name FROM account a
WHERE (a.default_billing_address_id = sqlc.arg('address_id')
    OR a.default_shipping_address_id = sqlc.arg('address_id'))
  AND a.onboarding_status_code = 'active'
LIMIT 1;

-- name: SwitchAccountDefaultAddressToRelation :exec
-- When a non-active account's default billing/shipping address is deleted, realign each
-- affected default to the account-relation default (owner→this account), so the account
-- keeps a valid default instead of a dangling pointer (there are no FKs to cascade). The
-- relation default is only adopted when it exists and is not the address being deleted;
-- otherwise the pointer falls back to NULL. Only the column(s) that pointed at the deleted
-- address are touched. For a non-active (unclaimed) account there is exactly one relation.
UPDATE account a
SET default_billing_address_id = CASE
        WHEN a.default_billing_address_id = sqlc.arg('address_id') THEN (
            SELECT ar.default_billing_address_id FROM account_relation ar
            WHERE ar.counterparty_account_id = a.id
              AND ar.default_billing_address_id IS NOT NULL
              AND ar.default_billing_address_id <> sqlc.arg('address_id')
            ORDER BY ar.created_at ASC, ar.id ASC LIMIT 1)
        ELSE a.default_billing_address_id END,
    default_shipping_address_id = CASE
        WHEN a.default_shipping_address_id = sqlc.arg('address_id') THEN (
            SELECT ar.default_shipping_address_id FROM account_relation ar
            WHERE ar.counterparty_account_id = a.id
              AND ar.default_shipping_address_id IS NOT NULL
              AND ar.default_shipping_address_id <> sqlc.arg('address_id')
            ORDER BY ar.created_at ASC, ar.id ASC LIMIT 1)
        ELSE a.default_shipping_address_id END,
    updated_at = NOW(3)
WHERE a.default_billing_address_id = sqlc.arg('address_id')
   OR a.default_shipping_address_id = sqlc.arg('address_id');
