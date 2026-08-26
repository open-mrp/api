-- Lifecycle: hand edits, the deviation log, and publish/freeze/supersede.
--
-- Versions are immutable history, so publishing never rewrites the version it replaces: it stamps the old one superseded and points it at its replacement.

-- name: ListScheduleDeviationTypes :many
SELECT
    t.id,
    t.code,
    t.name,
    t.created_at,
    t.updated_at
FROM schedule_deviation_type t
ORDER BY t.code ASC;

-- name: CreateProductionScheduleDeviation :exec
INSERT INTO production_schedule_deviation (
    id, account_id, production_schedule_id, production_schedule_line_id,
    deviation_type_code, is_frozen_week,
    week_index, machine_id, item_id,
    before_json, after_json,
    delta_quantity, delta_run_hours,
    reason_code, reason_note, actor_id, created_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('production_schedule_id'), sqlc.narg('production_schedule_line_id'),
    sqlc.arg('deviation_type_code'), sqlc.arg('is_frozen_week'),
    sqlc.narg('week_index'), sqlc.narg('machine_id'), sqlc.narg('item_id'),
    sqlc.narg('before_json'), sqlc.narg('after_json'),
    sqlc.arg('delta_quantity'), sqlc.arg('delta_run_hours'),
    sqlc.narg('reason_code'), sqlc.narg('reason_note'), sqlc.arg('actor_id'), NOW(3)
);

-- name: ListProductionScheduleDeviationsForward :many
SELECT
    d.id,
    d.account_id,
    d.production_schedule_id,
    d.production_schedule_line_id,
    d.deviation_type_code,
    d.is_frozen_week,
    d.week_index,
    d.machine_id,
    d.item_id,
    -- Cast to text: a NULL JSON column cannot scan into json.RawMessage, and a removed line legitimately has no after snapshot.
    CAST(d.before_json AS CHAR) AS before_json,
    CAST(d.after_json AS CHAR) AS after_json,
    d.delta_quantity,
    d.delta_run_hours,
    d.reason_code,
    d.reason_note,
    d.actor_id,
    d.created_at
FROM production_schedule_deviation d
WHERE d.account_id = sqlc.arg('account_id')
AND d.production_schedule_id = sqlc.arg('production_schedule_id')
AND (sqlc.narg('frozen_only') IS NULL OR d.is_frozen_week = sqlc.narg('frozen_only'))
-- Free-text search runs against the reason note, the only prose a deviation carries.
AND (
    sqlc.narg('search_query') IS NULL
    OR d.reason_note LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR d.created_at < sqlc.narg('cursor_created_at')
    OR (d.created_at = sqlc.narg('cursor_created_at') AND d.id < sqlc.narg('cursor_id'))
)
ORDER BY d.created_at DESC, d.id DESC
LIMIT ?;

-- name: ListProductionScheduleDeviationsBackward :many
SELECT
    d.id,
    d.account_id,
    d.production_schedule_id,
    d.production_schedule_line_id,
    d.deviation_type_code,
    d.is_frozen_week,
    d.week_index,
    d.machine_id,
    d.item_id,
    -- Cast to text: a NULL JSON column cannot scan into json.RawMessage, and a removed line legitimately has no after snapshot.
    CAST(d.before_json AS CHAR) AS before_json,
    CAST(d.after_json AS CHAR) AS after_json,
    d.delta_quantity,
    d.delta_run_hours,
    d.reason_code,
    d.reason_note,
    d.actor_id,
    d.created_at
FROM production_schedule_deviation d
WHERE d.account_id = sqlc.arg('account_id')
AND d.production_schedule_id = sqlc.arg('production_schedule_id')
AND (sqlc.narg('frozen_only') IS NULL OR d.is_frozen_week = sqlc.narg('frozen_only'))
-- Free-text search runs against the reason note, the only prose a deviation carries.
AND (
    sqlc.narg('search_query') IS NULL
    OR d.reason_note LIKE sqlc.narg('search_query')
)
AND (
    d.created_at > sqlc.arg('cursor_created_at')
    OR (d.created_at = sqlc.arg('cursor_created_at') AND d.id > sqlc.arg('cursor_id'))
)
ORDER BY d.created_at ASC, d.id ASC
LIMIT ?;

-- name: GetProductionScheduleLine :one
SELECT
    l.id,
    l.account_id,
    l.production_schedule_id,
    l.week_index,
    l.week_start_date,
    l.machine_id,
    l.production_step_id,
    l.department_id,
    l.item_id,
    l.planned_quantity,
    l.planned_unit_id,
    lu.abbreviation AS planned_unit_abbreviation,
    i.sku AS item_sku,
    l.planned_lots,
    l.planned_lot_units,
    l.planned_run_hours,
    l.planned_changeover_minutes,
    l.sequence_index,
    l.projected_on_hand_before,
    l.projected_on_hand_after,
    l.status_code,
    l.source_code,
    l.reason_code,
    l.is_frozen,
    l.production_run_id,
    l.created_at,
    l.updated_at
FROM production_schedule_line l
LEFT JOIN item i ON i.id = l.item_id
LEFT JOIN unit lu ON lu.id = l.planned_unit_id
WHERE l.account_id = sqlc.arg('account_id')
AND l.id = sqlc.arg('id');

-- name: UpdateProductionScheduleLine :exec
UPDATE production_schedule_line
SET
    machine_id = COALESCE(sqlc.narg('machine_id'), machine_id),
    week_index = COALESCE(sqlc.narg('week_index'), week_index),
    week_start_date = COALESCE(sqlc.narg('week_start_date'), week_start_date),
    planned_quantity = COALESCE(sqlc.narg('planned_quantity'), planned_quantity),
    planned_lots = COALESCE(sqlc.narg('planned_lots'), planned_lots),
    planned_run_hours = COALESCE(sqlc.narg('planned_run_hours'), planned_run_hours),
    sequence_index = COALESCE(sqlc.narg('sequence_index'), sequence_index),
    status_code = COALESCE(sqlc.narg('status_code'), status_code),
    reason_code = IF(sqlc.arg('clear_reason_code'), NULL, COALESCE(sqlc.narg('reason_code'), reason_code)),
    -- A hand-edited line stops being the solver's, so it can never silently revert to looking generated.
    source_code = 'manual',
    updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
AND id = sqlc.arg('id');

-- name: DeleteProductionScheduleLine :exec
DELETE FROM production_schedule_line
WHERE account_id = sqlc.arg('account_id')
AND id = sqlc.arg('id');

-- PublishProductionSchedule stamps the freeze in one statement so the frozen counts can never disagree with the lines they were counted from.
-- name: PublishProductionSchedule :exec
UPDATE production_schedule
SET
    status_code = 'published',
    frozen_through_date = sqlc.arg('frozen_through_date'),
    frozen_line_count = sqlc.arg('frozen_line_count'),
    frozen_planned_quantity = sqlc.arg('frozen_planned_quantity'),
    published_by_id = sqlc.narg('published_by_id'),
    published_at = NOW(3),
    updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
AND id = sqlc.arg('id');

-- name: FreezeProductionScheduleLines :exec
UPDATE production_schedule_line
SET is_frozen = true, updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
AND production_schedule_id = sqlc.arg('production_schedule_id')
AND week_start_date <= sqlc.arg('frozen_through_date');

-- name: SumFrozenProductionScheduleLines :one
SELECT
    COUNT(*) AS line_count,
    COALESCE(SUM(l.planned_quantity), 0) AS planned_quantity
FROM production_schedule_line l
WHERE l.account_id = sqlc.arg('account_id')
AND l.production_schedule_id = sqlc.arg('production_schedule_id')
AND l.week_start_date <= sqlc.arg('frozen_through_date');

-- name: SupersedeProductionSchedule :exec
UPDATE production_schedule
SET status_code = 'superseded', superseded_by_id = sqlc.arg('superseded_by_id'), updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
AND id = sqlc.arg('id');

-- ListPublishedProductionSchedulesOverlapping finds the versions a new publish replaces: published, and covering any part of the new horizon.
-- name: ListPublishedProductionSchedulesOverlapping :many
SELECT s.id
FROM production_schedule s
WHERE s.account_id = sqlc.arg('account_id')
AND s.status_code = 'published'
AND s.id != sqlc.arg('exclude_id')
AND s.horizon_start_date <= sqlc.arg('horizon_end_date')
AND s.horizon_end_date >= sqlc.arg('horizon_start_date');

-- name: SetProductionScheduleStatus :exec
UPDATE production_schedule
SET status_code = sqlc.arg('status_code'), updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
AND id = sqlc.arg('id');

-- name: GetMaxSequenceIndexForWeek :one
SELECT COALESCE(MAX(l.sequence_index), -1) AS max_sequence_index
FROM production_schedule_line l
WHERE l.account_id = sqlc.arg('account_id')
AND l.production_schedule_id = sqlc.arg('production_schedule_id')
AND l.week_index = sqlc.arg('week_index');


-- Releasing a week to the floor.
--
-- A release turns planned campaigns into a production run with one batch per lot. Both queries below are scoped by week rather than by line id: a week is released as a unit, so a partial release would leave the floor holding half a plan.

-- CountReleasedLinesForWeek reports how much of a week has already been released, which is the guard against a double click creating a second run for work already issued.
-- name: CountReleasedLinesForWeek :one
SELECT
    COUNT(*) AS total_lines,
    COALESCE(SUM(CASE WHEN l.production_run_id IS NOT NULL THEN 1 ELSE 0 END), 0) AS released_lines,
    MIN(l.production_run_id) AS existing_production_run_id
FROM production_schedule_line l
WHERE l.account_id = sqlc.arg('account_id')
AND l.production_schedule_id = sqlc.arg('production_schedule_id')
AND l.week_index = sqlc.arg('week_index')
AND l.status_code != 'cancelled';

-- MarkScheduleLineReleased links one campaign to the run that now carries it.
--
-- source_code is deliberately NOT touched here, unlike a hand edit: releasing a line is not authoring it, and a released solver line must still read as the solver's work.
-- name: MarkScheduleLineReleased :exec
UPDATE production_schedule_line
SET
    production_run_id = sqlc.arg('production_run_id'),
    status_code = 'released',
    updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
AND id = sqlc.arg('id')
AND production_run_id IS NULL;

-- UnreleaseScheduleLinesForRun puts a week back to planned when the run holding its work is deleted.
--
-- Leaving the link behind would make the week read as issued while the run carrying it no longer exists, and the release guard would then refuse to ever issue it again.
-- name: UnreleaseScheduleLinesForRun :exec
UPDATE production_schedule_line
SET
    production_run_id = NULL,
    status_code = 'planned',
    updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
AND production_run_id = sqlc.arg('production_run_id');

-- ListCarryForwardBatchesForItem finds tickets an earlier week issued for an item that the floor never got to.
--
-- The plan already knows the week fell short — next week's quantity is smaller because the inventory it expected never arrived. What it cannot know is that the doffs it would issue for that shortfall are doffs somebody has already printed. Moving those tickets into the new run is what stops the floor being handed a second copy of work it is already holding.
--
-- Only batches a schedule release created are eligible: the run has to be linked to a campaign of a week that has already begun. A hand-built run is somebody's own work and is never raided, and a run for a week still ahead of this one is not late — it has simply not been worked yet.
--
-- Unscanned and unclosed is what "never got to" means. A scanned batch is production that happened, and a closed one has left the floor.
-- name: ListCarryForwardBatchesForItem :many
SELECT
    b.id,
    b.production_run_id,
    b.production_step_id,
    b.created_at,
    q.value AS quantity_value,
    q.unit_id AS quantity_unit_id,
    pr.number AS production_run_number
FROM batch b
JOIN quantity q ON q.id = b.quantity_id
JOIN production_run pr ON pr.id = b.production_run_id
WHERE b.account_id = sqlc.arg('account_id')
AND b.item_id = sqlc.arg('item_id')
AND b.scanned_at IS NULL
AND b.closed_at IS NULL
AND b.production_run_id IN (
    SELECT DISTINCT l.production_run_id
    FROM production_schedule_line l
    WHERE l.account_id = sqlc.arg('account_id')
    AND l.item_id = sqlc.arg('item_id')
    AND l.production_run_id IS NOT NULL
    AND l.week_start_date < sqlc.arg('week_start_date')
)
ORDER BY b.created_at ASC, b.id ASC
LIMIT ?;
