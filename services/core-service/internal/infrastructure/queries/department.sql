-- name: GetDepartmentsByIDs :many
SELECT id, name, created_at, updated_at FROM department WHERE id IN (sqlc.slice('ids'));

-- name: GetDepartmentsFullByIDs :many
SELECT
    d.id,
    d.name,
    d.notes,
    d.location_id,
    d.account_id,
    d.created_at,
    d.updated_at
FROM department d
WHERE d.id IN (sqlc.slice('ids'))
AND d.account_id = sqlc.arg('account_id');

-- name: ListScanningStationsByDepartmentIDs :many
SELECT id, name, scanning_station_type_code, material_check_required, department_id, created_at, updated_at
FROM scanning_station
WHERE department_id IN (sqlc.slice('department_ids'))
AND account_id = sqlc.arg('account_id')
ORDER BY name ASC;

-- name: ListMachinesByDepartmentIDs :many
SELECT m.id, m.name, m.serial_number, m.department_id, m.created_at, m.updated_at
FROM machine m
JOIN department d ON d.id = m.department_id
WHERE m.department_id IN (sqlc.slice('department_ids'))
AND d.account_id = sqlc.arg('account_id')
ORDER BY m.name ASC;

-- name: ListDepartmentsForward :many
SELECT
    d.id,
    d.name,
    d.notes,
    d.location_id,
    sl.name AS location_name,
    sl.storage_location_type_code AS location_type_code,
    d.account_id,
    d.created_at,
    d.updated_at
FROM department d
LEFT JOIN storage_location sl ON sl.id = d.location_id
WHERE d.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR d.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR d.created_at < sqlc.narg('cursor_created_at')
    OR (d.created_at = sqlc.narg('cursor_created_at') AND d.id < sqlc.narg('cursor_id'))
)
ORDER BY d.created_at DESC, d.id DESC
LIMIT ?;

-- name: ListDepartmentsBackward :many
SELECT
    d.id,
    d.name,
    d.notes,
    d.location_id,
    sl.name AS location_name,
    sl.storage_location_type_code AS location_type_code,
    d.account_id,
    d.created_at,
    d.updated_at
FROM department d
LEFT JOIN storage_location sl ON sl.id = d.location_id
WHERE d.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR d.name LIKE sqlc.narg('search_query')
)
AND (
    d.created_at > sqlc.arg('cursor_created_at')
    OR (d.created_at = sqlc.arg('cursor_created_at') AND d.id > sqlc.arg('cursor_id'))
)
ORDER BY d.created_at ASC, d.id ASC
LIMIT ?;

-- name: GetDepartment :one
SELECT
    d.id,
    d.name,
    d.notes,
    d.location_id,
    sl.name AS location_name,
    sl.storage_location_type_code AS location_type_code,
    d.account_id,
    d.created_at,
    d.updated_at
FROM department d
LEFT JOIN storage_location sl ON sl.id = d.location_id
WHERE d.id = sqlc.arg('id')
AND d.account_id = sqlc.arg('account_id');

-- name: InsertDepartment :exec
INSERT INTO department (
    id,
    name,
    notes,
    location_id,
    account_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.narg('notes'),
    sqlc.narg('location_id'),
    sqlc.arg('account_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateDepartment :execresult
UPDATE department SET
    name = COALESCE(sqlc.narg('name'), name),
    notes = sqlc.narg('notes'),
    location_id = COALESCE(sqlc.narg('location_id'), location_id),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteDepartment :execresult
DELETE FROM department
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: CountDepartmentsByName :one
SELECT COUNT(*) FROM department
WHERE name = ? AND account_id = ?
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: ListScanningStationsByDepartmentID :many
SELECT id, name, scanning_station_type_code, material_check_required, created_at, updated_at
FROM scanning_station
WHERE department_id = sqlc.arg('department_id')
AND account_id = sqlc.arg('account_id')
ORDER BY name ASC;

-- name: ListMachinesByDepartmentID :many
SELECT m.id, m.name, m.serial_number, m.created_at, m.updated_at
FROM machine m
JOIN department d ON d.id = m.department_id
WHERE m.department_id = sqlc.arg('department_id')
AND d.account_id = sqlc.arg('account_id')
ORDER BY m.name ASC;

-- name: SetMachinesDepartmentID :exec
UPDATE machine
SET department_id = sqlc.arg('department_id')
WHERE id IN (sqlc.slice('machine_ids'));

-- name: SetScanningStationsDepartmentID :exec
UPDATE scanning_station
SET department_id = sqlc.arg('department_id')
WHERE id IN (sqlc.slice('scanning_station_ids'))
AND account_id = sqlc.arg('account_id');
