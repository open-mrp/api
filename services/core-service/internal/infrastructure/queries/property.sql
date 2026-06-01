-- name: ListPropertiesForward :many
SELECT
    property.id,
    property.name,
    property.account_id,
    property.is_public,
    property.created_at,
    property.updated_at
FROM property
WHERE property.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR property.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR property.created_at < sqlc.narg('cursor_created_at')
    OR (property.created_at = sqlc.narg('cursor_created_at') AND property.id < sqlc.narg('cursor_id'))
)
ORDER BY property.created_at DESC, property.id DESC
LIMIT ?;

-- name: ListPropertiesBackward :many
SELECT
    property.id,
    property.name,
    property.account_id,
    property.is_public,
    property.created_at,
    property.updated_at
FROM property
WHERE property.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR property.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR property.created_at > sqlc.narg('cursor_created_at')
    OR (property.created_at = sqlc.narg('cursor_created_at') AND property.id > sqlc.narg('cursor_id'))
)
ORDER BY property.created_at ASC, property.id ASC
LIMIT ?;

-- name: GetProperty :one
SELECT
    property.id,
    property.name,
    property.account_id,
    property.is_public,
    property.created_at,
    property.updated_at
FROM property
WHERE property.id = sqlc.arg('id')
AND property.account_id = sqlc.arg('account_id');

-- name: InsertProperty :exec
INSERT INTO property (
    id,
    name,
    account_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.arg('account_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateProperty :execresult
UPDATE property SET
    name = COALESCE(sqlc.narg('name'), name),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeletePropertyAttributes :exec
DELETE FROM attribute
WHERE property_id = sqlc.arg('property_id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteProperty :execresult
DELETE FROM property
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: GetPropertiesByIDs :many
SELECT
    property.id,
    property.name,
    property.account_id,
    property.is_public,
    property.created_at,
    property.updated_at
FROM property
WHERE property.id IN (sqlc.slice('ids'))
AND property.account_id = sqlc.arg('account_id');

-- name: CountPropertiesByName :one
SELECT COUNT(*) FROM property
WHERE name = sqlc.arg('name') AND account_id = sqlc.arg('account_id')
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: CheckPropertyInAccount :one
SELECT COUNT(*) FROM property
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');
