-- name: ListCarrierOptionsForward :many
SELECT
    carrier_option.id,
    carrier_option.name,
    carrier_option.code,
    carrier_option.service_level_token,
    carrier_option.is_portal_enabled,
    carrier_option.is_default,
    carrier_option.carrier_id,
    carrier_option.account_id,
    carrier_option.created_at,
    carrier_option.updated_at
FROM carrier_option
WHERE carrier_option.carrier_id = sqlc.arg('carrier_id')
AND (carrier_option.account_id = sqlc.arg('account_id') OR carrier_option.account_id IS NULL)
AND (
    sqlc.narg('search_query') IS NULL
    OR carrier_option.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR carrier_option.created_at < sqlc.narg('cursor_created_at')
    OR (carrier_option.created_at = sqlc.narg('cursor_created_at') AND carrier_option.id < sqlc.narg('cursor_id'))
)
ORDER BY carrier_option.created_at DESC, carrier_option.id DESC
LIMIT ?;

-- name: ListCarrierOptionsBackward :many
SELECT
    carrier_option.id,
    carrier_option.name,
    carrier_option.code,
    carrier_option.service_level_token,
    carrier_option.is_portal_enabled,
    carrier_option.is_default,
    carrier_option.carrier_id,
    carrier_option.account_id,
    carrier_option.created_at,
    carrier_option.updated_at
FROM carrier_option
WHERE carrier_option.carrier_id = sqlc.arg('carrier_id')
AND (carrier_option.account_id = sqlc.arg('account_id') OR carrier_option.account_id IS NULL)
AND (
    sqlc.narg('search_query') IS NULL
    OR carrier_option.name LIKE sqlc.narg('search_query')
)
AND (
    carrier_option.created_at > sqlc.arg('cursor_created_at')
    OR (carrier_option.created_at = sqlc.arg('cursor_created_at') AND carrier_option.id > sqlc.arg('cursor_id'))
)
ORDER BY carrier_option.created_at ASC, carrier_option.id ASC
LIMIT ?;

-- name: GetCarrierOption :one
SELECT
    carrier_option.id,
    carrier_option.name,
    carrier_option.code,
    carrier_option.service_level_token,
    carrier_option.is_portal_enabled,
    carrier_option.is_default,
    carrier_option.carrier_id,
    carrier_option.account_id,
    carrier_option.created_at,
    carrier_option.updated_at
FROM carrier_option
WHERE carrier_option.id = sqlc.arg('id')
AND (carrier_option.account_id = sqlc.arg('account_id') OR carrier_option.account_id IS NULL);

-- name: InsertCarrierOption :exec
INSERT INTO carrier_option (
    id,
    name,
    code,
    service_level_token,
    is_portal_enabled,
    is_default,
    carrier_id,
    account_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.arg('code'),
    sqlc.narg('service_level_token'),
    sqlc.arg('is_portal_enabled'),
    sqlc.arg('is_default'),
    sqlc.arg('carrier_id'),
    sqlc.arg('account_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateCarrierOption :execresult
UPDATE carrier_option SET
    name = COALESCE(sqlc.narg('name'), name),
    code = COALESCE(sqlc.narg('code'), code),
    is_portal_enabled = COALESCE(sqlc.narg('is_portal_enabled'), is_portal_enabled),
    is_default = COALESCE(sqlc.narg('is_default'), is_default),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteCarrierOption :execresult
DELETE FROM carrier_option
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: CheckCarrierOptionInCarrier :one
SELECT EXISTS(
    SELECT 1 FROM carrier_option
    WHERE id = sqlc.arg('id')
    AND carrier_id = sqlc.arg('carrier_id')
) AS `exists`;

-- name: CountCarrierOptionsByCodeInCarrier :one
SELECT COUNT(*) FROM carrier_option
WHERE code = ? AND carrier_id = ?
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: ClearDefaultServiceLevelsForCarrier :exec
UPDATE carrier_option SET
    is_default = false,
    updated_at = NOW(3)
WHERE carrier_id = sqlc.arg('carrier_id')
AND account_id = sqlc.arg('account_id')
AND is_default = true;

-- name: ListCarrierOptionsByCarrierID :many
SELECT
    carrier_option.id,
    carrier_option.name,
    carrier_option.code,
    carrier_option.service_level_token,
    carrier_option.is_portal_enabled,
    carrier_option.is_default,
    carrier_option.carrier_id,
    carrier_option.account_id,
    carrier_option.created_at,
    carrier_option.updated_at
FROM carrier_option
WHERE carrier_option.carrier_id = sqlc.arg('carrier_id')
AND (carrier_option.account_id = sqlc.arg('account_id') OR carrier_option.account_id IS NULL);

-- name: ListCarrierOptionIDsForCarriers :many
-- Returns all carrier_option IDs for the given carriers, deterministically
-- ordered. The api-gateway groups results by carrier_id and truncates to
-- `service_levels_limit` per carrier in Go (sqlc's MySQL engine has limited
-- CTE / window-function support, so per-group truncation is done client-side).
-- Authorization: the parent carrier must be the caller's own or a system
-- carrier; only options visible to the caller's account (or system options)
-- are returned.
SELECT
    carrier_option.id,
    carrier_option.carrier_id
FROM carrier_option
INNER JOIN carrier c ON c.id = carrier_option.carrier_id
WHERE carrier_option.carrier_id IN (sqlc.slice('carrier_ids'))
AND (carrier_option.account_id = sqlc.arg('account_id') OR carrier_option.account_id IS NULL)
AND (c.account_id = sqlc.arg('account_id') OR c.account_id IS NULL)
AND c.deleted_at IS NULL
ORDER BY carrier_option.carrier_id, carrier_option.created_at ASC, carrier_option.id ASC;

-- name: GetCarrierOptionsByIDs :many
-- Returns carrier_options (service levels) matching the given IDs that the
-- caller's account is authorized to read (the option's own account_id
-- scoping plus the parent carrier's account scoping).
SELECT
    carrier_option.id,
    carrier_option.name,
    carrier_option.code,
    carrier_option.service_level_token,
    carrier_option.is_portal_enabled,
    carrier_option.is_default,
    carrier_option.carrier_id,
    carrier_option.account_id,
    carrier_option.created_at,
    carrier_option.updated_at
FROM carrier_option
INNER JOIN carrier c ON c.id = carrier_option.carrier_id
WHERE carrier_option.id IN (sqlc.slice('ids'))
AND (carrier_option.account_id = sqlc.arg('account_id') OR carrier_option.account_id IS NULL)
AND (c.account_id = sqlc.arg('account_id') OR c.account_id IS NULL)
AND c.deleted_at IS NULL;
