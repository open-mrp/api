-- Machine status: what every machine is running right now, what is left on it, and what is queued behind that.
--
-- Read against the published schedule rather than the newest draft: the floor works to what was committed, and a draft regenerating underneath a wall display would make machines appear to change job on their own.

-- ListMachinesForStatus returns every machine that can carry work, whether or not the plan has given it any. A machine with nothing scheduled is idle, which is a state management needs to see rather than an absence from the list.
-- name: ListMachinesForStatus :many
SELECT
    m.id,
    m.name,
    m.department_id,
    d.name AS department_name
FROM machine m
LEFT JOIN department d ON d.id = m.department_id
WHERE m.account_id = sqlc.arg('account_id')
ORDER BY m.name ASC, m.id ASC;

-- ListScheduleLinesForStatus returns the plan from the current week forward, with how much of each campaign the floor has already scanned.
--
-- Progress is aggregated per (run, item) in a derived table: a run holds every SKU in its week, so joining batches directly would both multiply the line and credit each campaign with its neighbours' work.
-- name: ListScheduleLinesForStatus :many
SELECT
    l.id,
    l.machine_id,
    l.item_id,
    l.week_index,
    l.week_start_date,
    l.planned_quantity,
    l.planned_run_hours,
    l.status_code,
    l.production_run_id,
    lu.abbreviation AS planned_unit_abbreviation,
    p.sku,
    COALESCE(prog.released_batches, 0) AS released_batch_count,
    COALESCE(prog.scanned_batches, 0) AS scanned_batch_count,
    COALESCE(prog.scanned_quantity, 0) AS scanned_quantity
FROM production_schedule_line l
LEFT JOIN unit lu ON lu.id = l.planned_unit_id
LEFT JOIN production_schedule_item_policy p
    ON p.production_schedule_id = l.production_schedule_id AND p.item_id = l.item_id
LEFT JOIN (
    SELECT
        b.production_run_id,
        b.item_id,
        COUNT(*) AS released_batches,
        COALESCE(SUM(CASE WHEN b.scanned_at IS NOT NULL THEN 1 ELSE 0 END), 0) AS scanned_batches,
        COALESCE(SUM(CASE WHEN b.scanned_at IS NOT NULL THEN bq.value ELSE 0 END), 0) AS scanned_quantity
    FROM batch b
    JOIN quantity bq ON bq.id = b.quantity_id
    WHERE b.account_id = sqlc.arg('account_id')
    AND b.production_run_id IS NOT NULL
    -- Restrict the aggregate to the schedule's own runs: without this, the derived table scans and groups the tenant's entire batch history on every read of a polled status board, instead of only the runs the published schedule references.
    AND b.production_run_id IN (
        SELECT l2.production_run_id
        FROM production_schedule_line l2
        WHERE l2.account_id = sqlc.arg('account_id')
        AND l2.production_schedule_id = sqlc.arg('production_schedule_id')
        AND l2.production_run_id IS NOT NULL
    )
    GROUP BY b.production_run_id, b.item_id
) prog ON prog.production_run_id = l.production_run_id AND prog.item_id = l.item_id
WHERE l.account_id = sqlc.arg('account_id')
AND l.production_schedule_id = sqlc.arg('production_schedule_id')
AND l.week_start_date >= sqlc.arg('from_week')
AND l.status_code != 'cancelled'
ORDER BY l.machine_id ASC, l.week_start_date ASC, l.sequence_index ASC, l.id ASC;

-- ListOpenDowntimeForStatus returns the machines that are down right now.
--
-- One row per machine: a machine cannot be down twice at once, and the open-event guard enforces that on write.
-- name: ListOpenDowntimeForStatus :many
SELECT
    e.id,
    e.machine_id,
    e.reason_code,
    r.name AS reason_name,
    r.oee_bucket,
    e.started_at,
    e.note
FROM machine_downtime_event e
LEFT JOIN machine_downtime_reason r ON r.code = e.reason_code
WHERE e.account_id = sqlc.arg('account_id')
AND e.ended_at IS NULL;
