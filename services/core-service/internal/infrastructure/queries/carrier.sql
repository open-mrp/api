-- The carrier queries below select identical columns, so their rows convert to GetCarrierRow.

-- name: GetCarrier :one
SELECT
    carrier.id,
    carrier.name,
    carrier.code,
    carrier.scac,
    carrier.shippo_carrier_account_id,
    carrier.account_number,
    carrier.is_portal_enabled,
    carrier.is_active,
    carrier.default_carrier_option_id,
    carrier.account_id,
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
    scac,
    shippo_carrier_account_id,
    account_number,
    is_portal_enabled,
    is_active,
    account_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.arg('code'),
    sqlc.narg('scac'),
    sqlc.narg('shippo_carrier_account_id'),
    sqlc.narg('account_number'),
    sqlc.arg('is_portal_enabled'),
    sqlc.arg('is_active'),
    sqlc.arg('account_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateCarrier :execresult
-- scac and default_carrier_option_id are assigned rather than merged so they can be cleared; the service carries the
-- existing values forward when the caller leaves them out.
UPDATE carrier SET
    name = COALESCE(sqlc.narg('name'), name),
    is_portal_enabled = COALESCE(sqlc.narg('is_portal_enabled'), is_portal_enabled),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    scac = sqlc.narg('scac'),
    default_carrier_option_id = sqlc.narg('default_carrier_option_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id')
AND deleted_at IS NULL;

-- name: SetCarrierDefaultCarrierOption :exec
UPDATE carrier SET
    default_carrier_option_id = sqlc.narg('default_carrier_option_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id')
AND deleted_at IS NULL;

-- name: DeleteCarrier :execresult
DELETE FROM carrier
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteCarrierOptionsByCarrierID :exec
DELETE FROM carrier_option
WHERE carrier_id = sqlc.arg('carrier_id')
AND account_id = sqlc.arg('account_id');

-- name: CountCarriersByName :one
SELECT COUNT(*) FROM carrier
WHERE name = ? AND (account_id = ? OR account_id IS NULL)
AND deleted_at IS NULL
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: GetCarriersByIDs :many
-- Returns carriers matching the given IDs that the caller's account is
-- authorized to read (their own account plus system carriers with NULL
-- account_id). Used by BatchGetCarriersByIDs.
SELECT
    carrier.id,
    carrier.name,
    carrier.code,
    carrier.scac,
    carrier.shippo_carrier_account_id,
    carrier.account_number,
    carrier.is_portal_enabled,
    carrier.is_active,
    carrier.default_carrier_option_id,
    carrier.account_id,
    carrier.created_at,
    carrier.updated_at
FROM carrier
WHERE carrier.id IN (sqlc.slice('ids'))
AND (carrier.account_id = sqlc.arg('account_id') OR carrier.account_id IS NULL)
AND carrier.deleted_at IS NULL;
