-- name: ListMachineIDsByProductionStep :many
SELECT m.id
FROM machine m
WHERE m.production_step_id = sqlc.arg('production_step_id')
AND m.account_id = sqlc.arg('account_id')
ORDER BY m.id;

-- name: ListMachinesForward :many
SELECT
    m.id,
    m.name,
    m.serial_number,
    m.notes,
    m.department_id,
    d.name AS department_name,
    d.created_at AS department_created_at,
    d.updated_at AS department_updated_at,
    m.production_step_id,
    m.created_at,
    m.updated_at
FROM machine m
JOIN department d ON d.id = m.department_id
WHERE m.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR m.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR m.created_at < sqlc.narg('cursor_created_at')
    OR (m.created_at = sqlc.narg('cursor_created_at') AND m.id < sqlc.narg('cursor_id'))
)
ORDER BY m.created_at DESC, m.id DESC
LIMIT ?;

-- name: ListMachinesBackward :many
SELECT
    m.id,
    m.name,
    m.serial_number,
    m.notes,
    m.department_id,
    d.name AS department_name,
    d.created_at AS department_created_at,
    d.updated_at AS department_updated_at,
    m.production_step_id,
    m.created_at,
    m.updated_at
FROM machine m
JOIN department d ON d.id = m.department_id
WHERE m.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR m.name LIKE sqlc.narg('search_query')
)
AND (
    m.created_at > sqlc.arg('cursor_created_at')
    OR (m.created_at = sqlc.arg('cursor_created_at') AND m.id > sqlc.arg('cursor_id'))
)
ORDER BY m.created_at ASC, m.id ASC
LIMIT ?;

-- name: GetMachine :one
SELECT
    m.id,
    m.name,
    m.serial_number,
    m.notes,
    m.department_id,
    d.name AS department_name,
    d.created_at AS department_created_at,
    d.updated_at AS department_updated_at,
    m.production_step_id,
    m.created_at,
    m.updated_at
FROM machine m
JOIN department d ON d.id = m.department_id
WHERE m.id = sqlc.arg('id')
AND m.account_id = sqlc.arg('account_id');

-- name: InsertMachine :exec
INSERT INTO machine (
    id,
    account_id,
    name,
    serial_number,
    notes,
    department_id,
    production_step_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_id'),
    sqlc.arg('name'),
    sqlc.arg('serial_number'),
    sqlc.narg('notes'),
    sqlc.arg('department_id'),
    NULL,
    NOW(3),
    NOW(3)
);

-- name: UpdateMachine :execresult
UPDATE machine m
SET
    m.name = COALESCE(sqlc.narg('name'), m.name),
    m.serial_number = COALESCE(sqlc.narg('serial_number'), m.serial_number),
    m.notes = COALESCE(sqlc.narg('notes'), m.notes),
    m.updated_at = NOW(3)
WHERE m.id = sqlc.arg('id')
AND m.account_id = sqlc.arg('account_id');

-- name: DeleteMachine :execresult
DELETE FROM machine
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: GetMachinesByIDs :many
SELECT
    m.id,
    m.name,
    m.serial_number,
    m.notes,
    m.department_id,
    d.name AS department_name,
    d.created_at AS department_created_at,
    d.updated_at AS department_updated_at,
    m.production_step_id,
    m.created_at,
    m.updated_at
FROM machine m
JOIN department d ON d.id = m.department_id
WHERE m.id IN (sqlc.slice('ids'))
AND m.account_id = sqlc.arg('account_id');

-- name: CountMachinesByName :one
SELECT COUNT(*) FROM machine m
WHERE m.name = ? AND m.account_id = ?
AND (sqlc.narg('exclude_id') IS NULL OR m.id != sqlc.narg('exclude_id'));

-- name: CountMachinesBySerialNumber :one
SELECT COUNT(*) FROM machine m
WHERE m.serial_number = ? AND m.account_id = ?
AND (sqlc.narg('exclude_id') IS NULL OR m.id != sqlc.narg('exclude_id'));

-- name: FindMachinesBySerialNumbers :many
-- Used by bulk upsert to enforce account-wide serial number uniqueness in one query.
-- Matching is case-insensitive via the column collation.
SELECT
    m.id,
    m.name,
    m.serial_number,
    m.notes,
    m.department_id,
    d.name AS department_name,
    d.created_at AS department_created_at,
    d.updated_at AS department_updated_at,
    m.production_step_id,
    m.created_at,
    m.updated_at
FROM machine m
JOIN department d ON d.id = m.department_id
WHERE m.serial_number IN (sqlc.slice('serial_numbers'))
AND m.account_id = sqlc.arg('account_id');

-- name: FindMachinesByNames :many
-- Names must be pre-lowercased by the caller; the utf8mb4_unicode_ci collation makes the IN
-- comparison case-insensitive, so lowercasing on both sides is not required in SQL.
SELECT
    m.id,
    m.name,
    m.serial_number,
    m.notes,
    m.department_id,
    d.name AS department_name,
    d.created_at AS department_created_at,
    d.updated_at AS department_updated_at,
    m.production_step_id,
    m.created_at,
    m.updated_at
FROM machine m
JOIN department d ON d.id = m.department_id
WHERE m.name IN (sqlc.slice('names'))
AND m.account_id = sqlc.arg('account_id');

-- name: ExportMachines :many
-- Unpaginated by design; the caller passes a row cap as the limit.
SELECT
    m.id,
    m.name,
    m.serial_number,
    m.notes,
    m.department_id,
    d.name AS department_name,
    m.created_at,
    m.updated_at
FROM machine m
JOIN department d ON d.id = m.department_id
WHERE m.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR m.name LIKE sqlc.narg('search_query')
)
ORDER BY m.created_at DESC, m.id DESC
LIMIT ?;
