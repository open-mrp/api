-- name: GetLocation :one
SELECT
    sl.id,
    sl.name,
    sl.storage_location_type_code AS type_code,
    sl.parent_id,
    p.name AS parent_name,
    p.storage_location_type_code AS parent_type_code,
    sl.is_active,
    sl.address_id,
    sl.created_at,
    sl.updated_at
FROM storage_location sl
LEFT JOIN storage_location p ON p.id = sl.parent_id
WHERE sl.id = sqlc.arg('id')
AND sl.account_id = sqlc.arg('account_id');

-- name: GetLocationsByIDs :many
SELECT
    sl.id,
    sl.name,
    sl.storage_location_type_code AS type_code,
    sl.parent_id,
    p.name AS parent_name,
    p.storage_location_type_code AS parent_type_code,
    sl.is_active,
    sl.address_id,
    sl.created_at,
    sl.updated_at
FROM storage_location sl
LEFT JOIN storage_location p ON p.id = sl.parent_id
WHERE sl.id IN (sqlc.slice('ids'))
AND sl.account_id = sqlc.arg('account_id');

-- name: ListLocationChildrenByParentIDs :many
SELECT
    sl.id,
    sl.name,
    sl.storage_location_type_code AS type_code,
    sl.parent_id,
    sl.created_at
FROM storage_location sl
WHERE sl.parent_id IN (sqlc.slice('parent_ids'))
AND sl.account_id = sqlc.arg('account_id')
ORDER BY sl.created_at DESC, sl.id DESC;

-- name: InsertLocation :exec
INSERT INTO storage_location (
    id,
    account_id,
    name,
    storage_location_type_code,
    parent_id,
    address_id,
    is_active,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_id'),
    sqlc.arg('name'),
    sqlc.arg('storage_location_type_code'),
    sqlc.narg('parent_id'),
    sqlc.narg('address_id'),
    sqlc.arg('is_active'),
    NOW(3),
    NOW(3)
);

-- name: ConnectLocationChildren :exec
UPDATE storage_location
SET parent_id = sqlc.arg('parent_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('child_id')
AND account_id = sqlc.arg('account_id');

-- name: DisconnectLocationChildren :exec
UPDATE storage_location
SET parent_id = NULL,
    updated_at = NOW(3)
WHERE parent_id = sqlc.arg('parent_id')
AND account_id = sqlc.arg('account_id');

-- name: UpdateLocation :execresult
UPDATE storage_location SET
    name = COALESCE(sqlc.narg('name'), name),
    storage_location_type_code = COALESCE(sqlc.narg('storage_location_type_code'), storage_location_type_code),
    parent_id = sqlc.narg('parent_id'),
    address_id = sqlc.narg('address_id'),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteLocation :exec
DELETE FROM storage_location
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: FindLocationsByNames :many
SELECT
    sl.id,
    sl.name,
    sl.storage_location_type_code AS type_code,
    sl.parent_id,
    p.name AS parent_name,
    p.storage_location_type_code AS parent_type_code,
    sl.is_active,
    sl.address_id,
    sl.created_at,
    sl.updated_at
FROM storage_location sl
LEFT JOIN storage_location p ON p.id = sl.parent_id
-- names must be pre-lowercased by the caller; the utf8mb4_unicode_ci collation makes the
-- IN comparison case-insensitive, so lowercasing on both sides is not required in SQL.
WHERE sl.name IN (sqlc.slice('names'))
AND sl.account_id = sqlc.arg('account_id');

-- name: CheckLocationInAccount :one
SELECT EXISTS(
    SELECT 1 FROM storage_location
    WHERE id = sqlc.arg('id')
    AND account_id = sqlc.arg('account_id')
) AS `exists`;

-- name: GetLocationType :one
SELECT
    slt.id,
    slt.code,
    slt.name,
    slt.created_at,
    slt.updated_at
FROM storage_location_type slt
WHERE slt.id = sqlc.arg('id') OR slt.code = sqlc.arg('code');

-- name: ListLocationTypesForward :many
SELECT
    slt.id,
    slt.code,
    slt.name,
    slt.created_at,
    slt.updated_at
FROM storage_location_type slt
WHERE (
    sqlc.narg('search_query') IS NULL
    OR slt.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR slt.created_at < sqlc.narg('cursor_created_at')
    OR (slt.created_at = sqlc.narg('cursor_created_at') AND slt.id < sqlc.narg('cursor_id'))
)
ORDER BY slt.created_at DESC, slt.id DESC
LIMIT ?;

-- name: ListLocationTypesBackward :many
SELECT
    slt.id,
    slt.code,
    slt.name,
    slt.created_at,
    slt.updated_at
FROM storage_location_type slt
WHERE (
    sqlc.narg('search_query') IS NULL
    OR slt.name LIKE sqlc.narg('search_query')
)
AND (
    slt.created_at > sqlc.arg('cursor_created_at')
    OR (slt.created_at = sqlc.arg('cursor_created_at') AND slt.id > sqlc.arg('cursor_id'))
)
ORDER BY slt.created_at ASC, slt.id ASC
LIMIT ?;

-- name: ExportLocations :many
-- Unpaginated by design; the caller passes a row cap as the limit.
SELECT
    sl.id,
    sl.name,
    sl.storage_location_type_code AS type_code,
    sl.parent_id,
    p.name AS parent_name,
    sl.created_at,
    sl.updated_at
FROM storage_location sl
LEFT JOIN storage_location p ON p.id = sl.parent_id
WHERE sl.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR sl.name LIKE sqlc.narg('search_query')
)
ORDER BY sl.created_at DESC, sl.id DESC
LIMIT ?;


-- name: GetAccountDefaultSiteID :one
SELECT default_site_id FROM account WHERE id = sqlc.arg('account_id');

-- name: SetAccountDefaultSiteIfUnset :execresult
UPDATE account SET default_site_id = sqlc.arg('site_id'), updated_at = NOW(3)
WHERE id = sqlc.arg('account_id') AND default_site_id IS NULL;

-- name: ListLocationTreeNodes :many
-- Every location's place in the tree, for working out which site each belongs to. An account holds a few hundred.
SELECT id, parent_id, storage_location_type_code AS type_code
FROM storage_location
WHERE account_id = sqlc.arg('account_id');
