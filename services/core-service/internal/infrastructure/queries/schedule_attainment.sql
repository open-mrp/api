-- Schedule attainment: what was planned versus what the floor actually built.
--
-- Every query here is scoped by the *baseline* schedule chosen per week, never by "the current schedule". Measuring against whatever happens to be live now would let a republish rewrite last month's performance.

-- SelectAttainmentBaselines returns, for each week start in the window, the published version that was live for that week.
--
-- `published_at <= week_start` is what stops a mid-horizon republish from rewriting history: a version published on Wednesday was not the plan the floor worked to on Monday. Newest qualifying publish wins.
-- name: SelectAttainmentBaselines :many
SELECT
    s.id AS schedule_id,
    s.version,
    s.horizon_start_date,
    s.horizon_end_date,
    s.published_at,
    s.frozen_through_date,
    s.frozen_line_count,
    s.frozen_planned_quantity
FROM production_schedule s
WHERE s.account_id = sqlc.arg('account_id')
AND s.status_code IN ('published', 'superseded', 'archived')
AND s.published_at IS NOT NULL
AND s.horizon_start_date <= sqlc.arg('window_end')
AND s.horizon_end_date >= sqlc.arg('window_start')
ORDER BY s.published_at DESC, s.id DESC;

-- SumPlannedByWeek returns planned quantity and run hours per (week, machine, item) for one baseline version.
-- name: SumPlannedByWeek :many
SELECT
    l.week_start_date,
    l.machine_id,
    l.item_id,
    l.department_id,
    SUM(l.planned_quantity) AS planned_quantity,
    SUM(l.planned_run_hours) AS planned_run_hours,
    COUNT(*) AS line_count
FROM production_schedule_line l
WHERE l.account_id = sqlc.arg('account_id')
AND l.production_schedule_id = sqlc.arg('production_schedule_id')
AND l.week_start_date >= sqlc.arg('window_start')
AND l.week_start_date <= sqlc.arg('window_end')
AND l.status_code != 'cancelled'
GROUP BY l.week_start_date, l.machine_id, l.item_id, l.department_id;

-- SumActualsByWeek returns what was actually produced, bucketed to the start of the scan's production week so it lines up with a schedule line's week_start_date.
--
-- The week start follows the account's configured week_start_day (0 = Sunday through 6 = Saturday), the same day schedule horizons are built on. A fixed Monday here would split one schedule week's scans across two buckets for any plant whose week does not start on Monday, and its planned quantity would then be judged against a fraction of its own output.
--
-- Department comes from the batch's production step, NOT from the scanning station the way AnalyzeOee does it. That is deliberate and the two are not interchangeable: a plan is expressed in the step's department, so attainment has to be measured there or a department would be judged against work it was never assigned.
-- name: SumActualsByWeek :many
SELECT
    DATE(DATE_SUB(b.scanned_at, INTERVAL ((DAYOFWEEK(b.scanned_at) + 6 - CAST(sqlc.arg('week_start_day') AS SIGNED)) % 7) DAY)) AS week_start_date,
    bm.B AS machine_id,
    b.item_id,
    ps.department_id,
    COALESCE(SUM(bq.value), 0) AS actual_quantity,
    COALESCE(SUM(wq.value), 0) AS waste_quantity,
    COUNT(*) AS batch_count
FROM batch b
JOIN quantity bq ON bq.id = b.quantity_id
LEFT JOIN quantity wq ON wq.id = b.waste_quantity_id
LEFT JOIN _batches_machines bm ON bm.A = b.id
LEFT JOIN production_step ps ON ps.id = b.production_step_id
WHERE b.account_id = sqlc.arg('account_id')
AND b.scanned_at >= sqlc.arg('window_start')
AND b.scanned_at < sqlc.arg('window_end')
GROUP BY week_start_date, bm.B, b.item_id, ps.department_id;

-- CountDeviationsForBaselines counts frozen-week changes per baseline version, which is the numerator of frozen adherence.
-- name: CountDeviationsForBaselines :many
SELECT
    d.production_schedule_id,
    COUNT(*) AS deviation_count,
    SUM(CASE WHEN d.deviation_type_code = 'line_added' THEN 1 ELSE 0 END) AS added_count,
    COALESCE(SUM(ABS(d.delta_quantity)), 0) AS abs_delta_quantity
FROM production_schedule_deviation d
WHERE d.account_id = sqlc.arg('account_id')
AND d.production_schedule_id IN (sqlc.slice('schedule_ids'))
AND d.is_frozen_week = true
GROUP BY d.production_schedule_id;
