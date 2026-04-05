-- name: GetScanningStation :one
SELECT
    ss.id,
    ss.name,
    ss.notes,
    ss.scanning_station_type_code,
    ss.label_size_code,
    ss.label_type_code,
    ss.material_check_required,
    ss.department_id,
    d.name AS department_name,
    ss.account_id,
    ss.created_at,
    ss.updated_at
FROM scanning_station ss
LEFT JOIN department d ON d.id = ss.department_id
WHERE ss.id = sqlc.arg('id')
AND ss.account_id = sqlc.arg('account_id');

-- name: ListProductionStepsByScanningStationID :many
SELECT id, name
FROM production_step
WHERE scanning_station_id = sqlc.arg('scanning_station_id')
AND account_id = sqlc.arg('account_id')
ORDER BY name ASC;

-- name: ListScanningStationsForward :many
SELECT
    ss.id,
    ss.name,
    ss.notes,
    ss.scanning_station_type_code,
    ss.label_size_code,
    ss.label_type_code,
    ss.material_check_required,
    ss.department_id,
    d.name AS department_name,
    ss.account_id,
    ss.created_at,
    ss.updated_at
FROM scanning_station ss
LEFT JOIN department d ON d.id = ss.department_id
WHERE ss.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR ss.name LIKE sqlc.narg('search_query')
    OR d.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR ss.created_at < sqlc.narg('cursor_created_at')
    OR (ss.created_at = sqlc.narg('cursor_created_at') AND ss.id < sqlc.narg('cursor_id'))
)
ORDER BY ss.created_at DESC, ss.id DESC
LIMIT ?;

-- name: ListScanningStationsBackward :many
SELECT
    ss.id,
    ss.name,
    ss.notes,
    ss.scanning_station_type_code,
    ss.label_size_code,
    ss.label_type_code,
    ss.material_check_required,
    ss.department_id,
    d.name AS department_name,
    ss.account_id,
    ss.created_at,
    ss.updated_at
FROM scanning_station ss
LEFT JOIN department d ON d.id = ss.department_id
WHERE ss.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR ss.name LIKE sqlc.narg('search_query')
    OR d.name LIKE sqlc.narg('search_query')
)
AND (
    ss.created_at > sqlc.arg('cursor_created_at')
    OR (ss.created_at = sqlc.arg('cursor_created_at') AND ss.id > sqlc.arg('cursor_id'))
)
ORDER BY ss.created_at ASC, ss.id ASC
LIMIT ?;

-- name: InsertScanningStation :exec
INSERT INTO scanning_station (
    id,
    name,
    notes,
    scanning_station_type_code,
    material_check_required,
    department_id,
    account_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.narg('notes'),
    sqlc.arg('scanning_station_type_code'),
    sqlc.arg('material_check_required'),
    sqlc.arg('department_id'),
    sqlc.arg('account_id'),
    NOW(3),
    NOW(3)
);

-- name: CountScanningStationsByName :one
SELECT COUNT(*) FROM scanning_station
WHERE name = sqlc.arg('name') AND account_id = sqlc.arg('account_id')
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: UpdateScanningStation :execresult
UPDATE scanning_station SET
    name = COALESCE(sqlc.narg('name'), name),
    notes = COALESCE(sqlc.narg('notes'), notes),
    label_size_code = COALESCE(sqlc.narg('label_size_code'), label_size_code),
    label_type_code = COALESCE(sqlc.narg('label_type_code'), label_type_code),
    material_check_required = COALESCE(sqlc.narg('material_check_required'), material_check_required),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: IsScanningStationInAccount :one
SELECT COUNT(*) FROM scanning_station
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: DeleteScanningStation :execresult
DELETE FROM scanning_station
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: ConnectProductionStepsByName :execresult
UPDATE production_step
SET scanning_station_id = sqlc.arg('scanning_station_id'),
    updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
AND name LIKE CONCAT('%', sqlc.arg('name'), '%');

-- name: FindScanningStationIDByName :one
SELECT id FROM scanning_station
WHERE name = sqlc.arg('name') AND account_id = sqlc.arg('account_id')
LIMIT 1;

-- name: GetScanningStationType :one
SELECT scanning_station_type_code FROM scanning_station
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');
