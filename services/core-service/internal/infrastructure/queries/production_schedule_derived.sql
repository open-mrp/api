-- Derived department work: what each department has to do because of the constraint plan.
--
-- Rows are regenerated with their schedule rather than patched, so every write path here is delete-then-insert for a whole version.

-- name: CreateProductionScheduleDerivedLine :exec
INSERT INTO production_schedule_derived_line (
    id, account_id, production_schedule_id, source_line_id,
    production_step_id, department_id, item_id,
    week_index, week_start_date, quantity, planned_unit_id,
    explosion_depth, offset_weeks, status_code, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('production_schedule_id'), sqlc.arg('source_line_id'),
    sqlc.arg('production_step_id'), sqlc.narg('department_id'), sqlc.arg('item_id'),
    sqlc.arg('week_index'), sqlc.arg('week_start_date'), sqlc.arg('quantity'), sqlc.narg('planned_unit_id'),
    sqlc.arg('explosion_depth'), sqlc.arg('offset_weeks'), sqlc.arg('status_code'), NOW(3), NOW(3)
);

-- name: DeleteProductionScheduleDerivedLines :exec
DELETE FROM production_schedule_derived_line
WHERE account_id = sqlc.arg('account_id')
AND production_schedule_id = sqlc.arg('production_schedule_id');

-- ListProductionScheduleDerivedLines reads a work list FORWARD in time. The sort (week_start_date, explosion_depth, id) has no fully matching index — the closest, prod_sched_derived_sched_week_step_idx, carries production_step_id rather than explosion_depth — so the read filesorts a set bounded by one schedule's derived lines. A matching prod_sched_derived_sched_week_depth_idx (production_schedule_id, week_start_date, explosion_depth, id) would remove the filesort.
-- name: ListProductionScheduleDerivedLines :many
SELECT
    d.id,
    d.production_schedule_id,
    d.source_line_id,
    d.production_step_id,
    d.department_id,
    d.item_id,
    d.week_index,
    d.week_start_date,
    d.quantity,
    d.planned_unit_id,
    d.explosion_depth,
    d.offset_weeks,
    d.status_code,
    d.created_at,
    d.updated_at
FROM production_schedule_derived_line d
WHERE d.account_id = sqlc.arg('account_id')
AND d.production_schedule_id = sqlc.arg('production_schedule_id')
AND (
    sqlc.arg('include_department_filter') = false
    OR d.department_id IN (sqlc.slice('department_ids'))
)
AND (sqlc.narg('week_index') IS NULL OR d.week_index = sqlc.narg('week_index'))
ORDER BY d.week_start_date ASC, d.explosion_depth ASC, d.id ASC;

-- GetProductionStepGraph returns every step edge plus the metadata the explosion needs, in one pass. The edge orientation is upstream=B, downstream=A; see docs/patterns/production-step-graph-patterns.md.
-- name: GetProductionStepGraph :many
SELECT
    pcps.B AS upstream_step_id,
    pcps.A AS downstream_step_id
FROM _parent_child_production_steps pcps
JOIN production_step ps ON pcps.B = ps.id
WHERE ps.account_id = sqlc.arg('account_id');

-- name: GetProductionStepsForExplosion :many
SELECT
    ps.id,
    ps.name,
    ps.department_id
FROM production_step ps
WHERE ps.account_id = sqlc.arg('account_id');

-- ListProductionScheduleResourceSettings returns every per-resource override. The explosion only needs the production-step scoped rows, but reading them all in one query beats a round trip per step.
-- name: ListProductionScheduleResourceSettings :many
SELECT
    rs.id,
    rs.scope_code,
    rs.scope_ref_id,
    rs.is_excluded,
    rs.lead_time_offset_weeks
FROM production_schedule_resource_setting rs
WHERE rs.account_id = sqlc.arg('account_id');
