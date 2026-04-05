-- name: ListCarriersForward :many
SELECT
    carrier.id,
    carrier.name,
    carrier.code,
    carrier.shippo_carrier_account_id,
    carrier.account_number,
    carrier.is_portal_enabled,
    carrier.account_id,
    carrier.deleted_at,
    carrier.created_at,
    carrier.updated_at
FROM carrier
WHERE (carrier.account_id = sqlc.arg('account_id') OR carrier.account_id IS NULL)
AND carrier.deleted_at IS NULL
AND (
    sqlc.narg('search_query') IS NULL
    OR carrier.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR carrier.created_at < sqlc.narg('cursor_created_at')
    OR (carrier.created_at = sqlc.narg('cursor_created_at') AND carrier.id < sqlc.narg('cursor_id'))
)
ORDER BY carrier.created_at DESC, carrier.id DESC
LIMIT ?;

-- name: ListCarriersBackward :many
SELECT
    carrier.id,
    carrier.name,
    carrier.code,
    carrier.shippo_carrier_account_id,
    carrier.account_number,
    carrier.is_portal_enabled,
    carrier.account_id,
    carrier.deleted_at,
    carrier.created_at,
    carrier.updated_at
FROM carrier
WHERE (carrier.account_id = sqlc.arg('account_id') OR carrier.account_id IS NULL)
AND carrier.deleted_at IS NULL
AND (
    sqlc.narg('search_query') IS NULL
    OR carrier.name LIKE sqlc.narg('search_query')
)
AND (
    carrier.created_at > sqlc.arg('cursor_created_at')
    OR (carrier.created_at = sqlc.arg('cursor_created_at') AND carrier.id > sqlc.arg('cursor_id'))
)
ORDER BY carrier.created_at ASC, carrier.id ASC
LIMIT ?;

-- name: GetCarrier :one
SELECT
    carrier.id,
    carrier.name,
    carrier.code,
    carrier.shippo_carrier_account_id,
    carrier.account_number,
    carrier.is_portal_enabled,
    carrier.account_id,
    carrier.deleted_at,
    carrier.created_at,
    carrier.updated_at
FROM carrier
WHERE carrier.id = sqlc.arg('id')
AND (carrier.account_id = sqlc.arg('account_id') OR carrier.account_id IS NULL)
AND carrier.deleted_at IS NULL;

-- name: InsertCarrier :exec
INSERT INTO carrier (
    id,
    name,
    code,
    shippo_carrier_account_id,
    account_number,
    is_portal_enabled,
    account_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.narg('code'),
    sqlc.narg('shippo_carrier_account_id'),
    sqlc.narg('account_number'),
    sqlc.arg('is_portal_enabled'),
    sqlc.arg('account_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateCarrier :execresult
UPDATE carrier SET
    name = COALESCE(sqlc.narg('name'), name),
    is_portal_enabled = COALESCE(sqlc.narg('is_portal_enabled'), is_portal_enabled),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id')
AND deleted_at IS NULL;

-- name: SoftDeleteCarrier :execresult
UPDATE carrier SET
    deleted_at = NOW(3),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id')
AND deleted_at IS NULL;

-- name: DeleteCarrierOptionsByCarrierID :exec
DELETE FROM carrier_option
WHERE carrier_id = sqlc.arg('carrier_id')
AND account_id = sqlc.arg('account_id');

-- name: CountCarriersByName :one
SELECT COUNT(*) FROM carrier
WHERE name = ? AND (account_id = ? OR account_id IS NULL)
AND deleted_at IS NULL
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));
