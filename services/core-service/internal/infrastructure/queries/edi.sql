-- name: ListDCLocationsForward :many
SELECT
    dcl.id,
    dcl.location,
    dcl.account_id,
    a.name AS customer_name,
    dcl.owner_account_id,
    dcl.created_at,
    dcl.updated_at
FROM dc_location dcl
LEFT JOIN account a ON a.id = dcl.account_id
WHERE dcl.owner_account_id = sqlc.arg('owner_account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR dcl.location LIKE sqlc.narg('search_query')
    OR a.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR dcl.created_at < sqlc.narg('cursor_created_at')
    OR (dcl.created_at = sqlc.narg('cursor_created_at') AND dcl.id < sqlc.narg('cursor_id'))
)
ORDER BY dcl.created_at DESC, dcl.id DESC
LIMIT ?;

-- name: ListDCLocationsBackward :many
SELECT
    dcl.id,
    dcl.location,
    dcl.account_id,
    a.name AS customer_name,
    dcl.owner_account_id,
    dcl.created_at,
    dcl.updated_at
FROM dc_location dcl
LEFT JOIN account a ON a.id = dcl.account_id
WHERE dcl.owner_account_id = sqlc.arg('owner_account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR dcl.location LIKE sqlc.narg('search_query')
    OR a.name LIKE sqlc.narg('search_query')
)
AND (
    dcl.created_at > sqlc.arg('cursor_created_at')
    OR (dcl.created_at = sqlc.arg('cursor_created_at') AND dcl.id > sqlc.arg('cursor_id'))
)
ORDER BY dcl.created_at ASC, dcl.id ASC
LIMIT ?;

-- name: GetDCLocation :one
SELECT
    dcl.id,
    dcl.location,
    dcl.account_id,
    a.name AS customer_name,
    dcl.owner_account_id,
    dcl.created_at,
    dcl.updated_at
FROM dc_location dcl
LEFT JOIN account a ON a.id = dcl.account_id
WHERE dcl.id = sqlc.arg('id')
AND dcl.owner_account_id = sqlc.arg('owner_account_id');

-- name: InsertDCLocation :exec
INSERT INTO dc_location (
    id,
    location,
    account_id,
    owner_account_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('location'),
    sqlc.arg('account_id'),
    sqlc.arg('owner_account_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateDCLocation :execresult
UPDATE dc_location SET
    location = sqlc.narg('location'),
    account_id = sqlc.narg('account_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND owner_account_id = sqlc.arg('owner_account_id');

-- name: DeleteDCLocation :execresult
DELETE FROM dc_location
WHERE id = sqlc.arg('id')
AND owner_account_id = sqlc.arg('owner_account_id');

-- name: GetDCLocationsByIDs :many
-- Returns DC locations matching the given IDs that belong to the caller's
-- account. Used by the api-gateway resourcekit resolver.
SELECT
    dcl.id,
    dcl.location,
    dcl.account_id,
    a.name AS customer_name,
    dcl.owner_account_id,
    dcl.created_at,
    dcl.updated_at
FROM dc_location dcl
LEFT JOIN account a ON a.id = dcl.account_id
WHERE dcl.id IN (sqlc.slice('ids'))
AND dcl.owner_account_id = sqlc.arg('owner_account_id');

-- name: ListEDIRunsForward :many
SELECT
    er.id,
    er.completed_at,
    er.has_succeeded,
    er.account_id,
    er.created_at,
    er.updated_at
FROM edi_run er
WHERE er.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('has_succeeded') IS NULL
    OR er.has_succeeded = sqlc.narg('has_succeeded')
)
AND (
    sqlc.narg('search') IS NULL
    OR er.id LIKE CONCAT('%', sqlc.narg('search'), '%')
)
AND (
    sqlc.narg('cursor_completed_at') IS NULL
    OR er.completed_at < sqlc.narg('cursor_completed_at')
    OR (er.completed_at = sqlc.narg('cursor_completed_at') AND er.id < sqlc.narg('cursor_id'))
)
ORDER BY er.completed_at DESC, er.id DESC
LIMIT ?;

-- name: ListEDIRunsBackward :many
SELECT
    er.id,
    er.completed_at,
    er.has_succeeded,
    er.account_id,
    er.created_at,
    er.updated_at
FROM edi_run er
WHERE er.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('has_succeeded') IS NULL
    OR er.has_succeeded = sqlc.narg('has_succeeded')
)
AND (
    sqlc.narg('search') IS NULL
    OR er.id LIKE CONCAT('%', sqlc.narg('search'), '%')
)
AND (
    er.completed_at > sqlc.arg('cursor_completed_at')
    OR (er.completed_at = sqlc.arg('cursor_completed_at') AND er.id > sqlc.arg('cursor_id'))
)
ORDER BY er.completed_at ASC, er.id ASC
LIMIT ?;

-- name: GetEDIRun :one
SELECT
    er.id,
    er.completed_at,
    er.has_succeeded,
    er.account_id,
    er.created_at,
    er.updated_at
FROM edi_run er
WHERE er.id = sqlc.arg('id')
AND er.account_id = sqlc.arg('account_id');

-- name: GetEDIRunsByIDs :many
-- Returns EDI runs matching the given IDs that belong to the caller's
-- account. Used by the api-gateway resourcekit resolver.
SELECT
    er.id,
    er.completed_at,
    er.has_succeeded,
    er.account_id,
    er.created_at,
    er.updated_at
FROM edi_run er
WHERE er.id IN (sqlc.slice('ids'))
AND er.account_id = sqlc.arg('account_id');
