-- name: ListAddressesForward :many
SELECT
    a.id,
    a.name,
    a.phone,
    a.email,
    a.is_drop_ship,
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
    g.longitude
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
    g.longitude
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
    g.longitude
FROM address a
JOIN geolocation g ON a.geolocation_id = g.id
JOIN account_address aa ON aa.address_id = a.id
WHERE a.id = sqlc.arg('id')
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
    geolocation_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.narg('phone'),
    sqlc.narg('email'),
    sqlc.arg('is_drop_ship'),
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
SELECT a.name FROM account a
WHERE a.default_billing_address_id = sqlc.arg('address_id')
   OR a.default_shipping_address_id = sqlc.arg('address_id')
LIMIT 1;
