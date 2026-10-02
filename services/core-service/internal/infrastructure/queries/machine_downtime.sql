-- name: ListMachineDowntimeReasons :many
SELECT
    r.id,
    r.code,
    r.name,
    r.oee_bucket,
    r.is_planned,
    r.sort_order,
    r.created_at,
    r.updated_at
FROM machine_downtime_reason r
ORDER BY r.sort_order ASC, r.code ASC;

-- name: GetMachineDowntimeReason :one
SELECT
    r.id,
    r.code,
    r.name,
    r.oee_bucket,
    r.is_planned,
    r.sort_order,
    r.created_at,
    r.updated_at
FROM machine_downtime_reason r
WHERE r.code = sqlc.arg('code');

-- name: CreateMachineDowntimeEvent :exec
INSERT INTO machine_downtime_event (
    id,
    account_id,
    machine_id,
    department_id,
    production_step_id,
    reason_code,
    started_at,
    ended_at,
    duration_seconds,
    shift_date,
    shift_code,
    item_id,
    production_run_id,
    batch_id,
    schedule_line_id,
    note,
    reported_by_id,
    source_code,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_id'),
    sqlc.arg('machine_id'),
    sqlc.narg('department_id'),
    sqlc.narg('production_step_id'),
    sqlc.arg('reason_code'),
    sqlc.arg('started_at'),
    sqlc.narg('ended_at'),
    sqlc.narg('duration_seconds'),
    sqlc.arg('shift_date'),
    sqlc.narg('shift_code'),
    sqlc.narg('item_id'),
    sqlc.narg('production_run_id'),
    sqlc.narg('batch_id'),
    sqlc.narg('schedule_line_id'),
    sqlc.narg('note'),
    sqlc.arg('reported_by_id'),
    sqlc.arg('source_code'),
    NOW(3),
    NOW(3)
);

-- name: GetMachineDowntimeEvent :one
SELECT
    e.id,
    e.account_id,
    e.machine_id,
    e.department_id,
    e.production_step_id,
    e.reason_code,
    r.name AS reason_name,
    r.oee_bucket AS reason_oee_bucket,
    r.is_planned AS reason_is_planned,
    e.started_at,
    e.ended_at,
    e.duration_seconds,
    e.shift_date,
    e.shift_code,
    e.item_id,
    e.production_run_id,
    e.batch_id,
    e.schedule_line_id,
    e.note,
    e.reported_by_id,
    e.source_code,
    e.created_at,
    e.updated_at
FROM machine_downtime_event e
LEFT JOIN machine_downtime_reason r ON r.code = e.reason_code
WHERE e.account_id = sqlc.arg('account_id')
AND e.id = sqlc.arg('id');

-- GetOpenMachineDowntimeEventForMachine answers "is this machine down right now" as a seek on machine_downtime_open_idx rather than a scan of the machine's history. Used to stop a second open event being logged for a machine that is already down.
-- name: GetOpenMachineDowntimeEventForMachine :one
SELECT
    e.id,
    e.account_id,
    e.machine_id,
    e.department_id,
    e.production_step_id,
    e.reason_code,
    r.name AS reason_name,
    r.oee_bucket AS reason_oee_bucket,
    r.is_planned AS reason_is_planned,
    e.started_at,
    e.ended_at,
    e.duration_seconds,
    e.shift_date,
    e.shift_code,
    e.item_id,
    e.production_run_id,
    e.batch_id,
    e.schedule_line_id,
    e.note,
    e.reported_by_id,
    e.source_code,
    e.created_at,
    e.updated_at
FROM machine_downtime_event e
LEFT JOIN machine_downtime_reason r ON r.code = e.reason_code
WHERE e.account_id = sqlc.arg('account_id')
AND e.machine_id = sqlc.arg('machine_id')
AND e.ended_at IS NULL
ORDER BY e.started_at DESC, e.id DESC
LIMIT 1;

-- name: UpdateMachineDowntimeEvent :exec
UPDATE machine_downtime_event
SET
    -- The machine, and with it the department and step the stoppage is charged to, so a mis-logged event can be moved rather than deleted and re-entered.
    machine_id = sqlc.arg('machine_id'),
    department_id = sqlc.narg('department_id'),
    production_step_id = sqlc.narg('production_step_id'),
    reason_code = sqlc.arg('reason_code'),
    started_at = sqlc.arg('started_at'),
    ended_at = sqlc.narg('ended_at'),
    duration_seconds = sqlc.narg('duration_seconds'),
    shift_date = sqlc.arg('shift_date'),
    shift_code = sqlc.narg('shift_code'),
    item_id = sqlc.narg('item_id'),
    production_run_id = sqlc.narg('production_run_id'),
    batch_id = sqlc.narg('batch_id'),
    schedule_line_id = sqlc.narg('schedule_line_id'),
    note = sqlc.narg('note'),
    updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
AND id = sqlc.arg('id');

-- name: DeleteMachineDowntimeEvent :exec
DELETE FROM machine_downtime_event
WHERE account_id = sqlc.arg('account_id')
AND id = sqlc.arg('id');

-- name: GetMachineDowntimeEventsByIDs :many
SELECT
    e.id,
    e.account_id,
    e.machine_id,
    e.department_id,
    e.production_step_id,
    e.reason_code,
    r.name AS reason_name,
    r.oee_bucket AS reason_oee_bucket,
    r.is_planned AS reason_is_planned,
    e.started_at,
    e.ended_at,
    e.duration_seconds,
    e.shift_date,
    e.shift_code,
    e.item_id,
    e.production_run_id,
    e.batch_id,
    e.schedule_line_id,
    e.note,
    e.reported_by_id,
    e.source_code,
    e.created_at,
    e.updated_at
FROM machine_downtime_event e
LEFT JOIN machine_downtime_reason r ON r.code = e.reason_code
WHERE e.account_id = sqlc.arg('account_id')
AND e.id IN (sqlc.slice('ids'));
