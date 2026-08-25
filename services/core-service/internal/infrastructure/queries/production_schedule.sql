-- Persistence for generated production schedules.
--
-- Versions are immutable history: a new generation is a new version, and publishing supersedes the previous one rather than mutating it, because attainment is measured against whichever version was live at the time.

-- AllocateNextProductionScheduleVersion atomically reserves the next version number.
--
-- The single upsert holds a row lock on the per-account counter, so two planners generating at once serialize instead of both reading the same MAX and colliding on production_schedule_account_version_key. Read the reserved number back from the statement result's LastInsertId(). Mirrors AllocateNextOrderNumber.
-- name: AllocateNextProductionScheduleVersion :execresult
INSERT INTO sys_property (id, account_id, sys_property_type_code, value, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), 'production_schedule_version', LAST_INSERT_ID(1), NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE value = LAST_INSERT_ID(value + 1), updated_at = NOW(3);

-- ProductionScheduleVersionCounterExists reports whether the account's counter has been primed.
-- A point lookup on the unique key, so the seed below — which scans the account's schedules —
-- runs once in an account's lifetime rather than on every generation.
-- name: ProductionScheduleVersionCounterExists :one
SELECT EXISTS (
    SELECT 1 FROM sys_property
    WHERE account_id = sqlc.arg('account_id')
    AND sys_property_type_code = 'production_schedule_version'
) AS counter_exists;

-- SeedProductionScheduleVersionCounter primes the counter from existing rows the first time an account allocates, so a database that already has versions does not restart numbering at 1.
--
-- INSERT ... SELECT takes shared locks over the production_schedule rows it aggregates. Running
-- it on every generation put those range locks ahead of the insert the same transaction was
-- about to make into that table, so two planners generating at once deadlocked reliably. Guard
-- it with the existence check above and the hot path touches only the counter row.
-- name: SeedProductionScheduleVersionCounter :exec
INSERT INTO sys_property (id, account_id, sys_property_type_code, value, created_at, updated_at)
SELECT sqlc.arg('id'), sqlc.arg('account_id'), 'production_schedule_version',
       COALESCE(MAX(s.version), 0), NOW(3), NOW(3)
FROM production_schedule s
WHERE s.account_id = sqlc.arg('account_id')
ON DUPLICATE KEY UPDATE updated_at = updated_at;

-- name: CreateProductionSchedule :exec
INSERT INTO production_schedule (
    id, account_id, version, status_code, name,
    planning_as_of, horizon_start_date, horizon_end_date, horizon_weeks, frozen_weeks,
    demand_basis_code, generation_source_code, solver_version,
    settings_snapshot, diagnostics,
    generated_by_id, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('version'), sqlc.arg('status_code'), sqlc.narg('name'),
    sqlc.arg('planning_as_of'), sqlc.arg('horizon_start_date'), sqlc.arg('horizon_end_date'),
    sqlc.arg('horizon_weeks'), sqlc.arg('frozen_weeks'),
    sqlc.arg('demand_basis_code'), sqlc.arg('generation_source_code'), sqlc.arg('solver_version'),
    sqlc.arg('settings_snapshot'), sqlc.arg('diagnostics'),
    sqlc.narg('generated_by_id'), NOW(3), NOW(3)
);

-- name: GetProductionSchedule :one
SELECT
    s.id,
    s.account_id,
    s.version,
    s.status_code,
    s.name,
    s.planning_as_of,
    s.horizon_start_date,
    s.horizon_end_date,
    s.horizon_weeks,
    s.frozen_weeks,
    s.frozen_through_date,
    s.demand_basis_code,
    s.generation_source_code,
    s.solver_version,
    s.settings_snapshot,
    s.diagnostics,
    s.error_message,
    s.frozen_line_count,
    s.frozen_planned_quantity,
    s.generated_by_id,
    s.published_by_id,
    s.published_at,
    s.superseded_by_id,
    s.created_at,
    s.updated_at
FROM production_schedule s
WHERE s.account_id = sqlc.arg('account_id')
AND s.id = sqlc.arg('id');

-- GetCurrentProductionSchedule returns the published version covering the given date. Newest publish wins, so a mid-horizon republish takes effect without rewriting the version it superseded.
-- name: GetCurrentProductionSchedule :one
SELECT
    s.id,
    s.account_id,
    s.version,
    s.status_code,
    s.name,
    s.planning_as_of,
    s.horizon_start_date,
    s.horizon_end_date,
    s.horizon_weeks,
    s.frozen_weeks,
    s.frozen_through_date,
    s.demand_basis_code,
    s.generation_source_code,
    s.solver_version,
    s.settings_snapshot,
    s.diagnostics,
    s.error_message,
    s.frozen_line_count,
    s.frozen_planned_quantity,
    s.generated_by_id,
    s.published_by_id,
    s.published_at,
    s.superseded_by_id,
    s.created_at,
    s.updated_at
FROM production_schedule s
WHERE s.account_id = sqlc.arg('account_id')
AND s.status_code = 'published'
AND s.horizon_start_date <= sqlc.arg('as_of_date')
AND s.horizon_end_date >= sqlc.arg('as_of_date')
ORDER BY s.published_at DESC, s.id DESC
LIMIT 1;

-- name: ListProductionSchedulesForward :many
SELECT
    s.id,
    s.account_id,
    s.version,
    s.status_code,
    s.name,
    s.planning_as_of,
    s.horizon_start_date,
    s.horizon_end_date,
    s.horizon_weeks,
    s.frozen_weeks,
    s.frozen_through_date,
    s.demand_basis_code,
    s.generation_source_code,
    s.solver_version,
    s.settings_snapshot,
    s.diagnostics,
    s.error_message,
    s.frozen_line_count,
    s.frozen_planned_quantity,
    s.generated_by_id,
    s.published_by_id,
    s.published_at,
    s.superseded_by_id,
    s.created_at,
    s.updated_at
FROM production_schedule s
WHERE s.account_id = sqlc.arg('account_id')
AND (
    sqlc.arg('include_status_filter') = false
    OR s.status_code IN (sqlc.slice('status_codes'))
)
-- Free-text search runs against the version name, the only prose a schedule carries.
AND (
    sqlc.narg('search_query') IS NULL
    OR s.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR s.created_at < sqlc.narg('cursor_created_at')
    OR (s.created_at = sqlc.narg('cursor_created_at') AND s.id < sqlc.narg('cursor_id'))
)
ORDER BY s.created_at DESC, s.id DESC
LIMIT ?;

-- name: ListProductionSchedulesBackward :many
SELECT
    s.id,
    s.account_id,
    s.version,
    s.status_code,
    s.name,
    s.planning_as_of,
    s.horizon_start_date,
    s.horizon_end_date,
    s.horizon_weeks,
    s.frozen_weeks,
    s.frozen_through_date,
    s.demand_basis_code,
    s.generation_source_code,
    s.solver_version,
    s.settings_snapshot,
    s.diagnostics,
    s.error_message,
    s.frozen_line_count,
    s.frozen_planned_quantity,
    s.generated_by_id,
    s.published_by_id,
    s.published_at,
    s.superseded_by_id,
    s.created_at,
    s.updated_at
FROM production_schedule s
WHERE s.account_id = sqlc.arg('account_id')
AND (
    sqlc.arg('include_status_filter') = false
    OR s.status_code IN (sqlc.slice('status_codes'))
)
-- Free-text search runs against the version name, the only prose a schedule carries.
AND (
    sqlc.narg('search_query') IS NULL
    OR s.name LIKE sqlc.narg('search_query')
)
AND (
    s.created_at > sqlc.arg('cursor_created_at')
    OR (s.created_at = sqlc.arg('cursor_created_at') AND s.id > sqlc.arg('cursor_id'))
)
ORDER BY s.created_at ASC, s.id ASC
LIMIT ?;

-- name: DeleteProductionSchedule :exec
DELETE FROM production_schedule
WHERE account_id = sqlc.arg('account_id')
AND id = sqlc.arg('id');

-- name: DeleteProductionScheduleLines :exec
DELETE FROM production_schedule_line
WHERE account_id = sqlc.arg('account_id')
AND production_schedule_id = sqlc.arg('production_schedule_id');

-- name: DeleteProductionScheduleItemPolicies :exec
DELETE FROM production_schedule_item_policy
WHERE account_id = sqlc.arg('account_id')
AND production_schedule_id = sqlc.arg('production_schedule_id');

-- name: CreateProductionScheduleLine :exec
INSERT INTO production_schedule_line (
    id, account_id, production_schedule_id,
    week_index, week_start_date,
    machine_id, production_step_id, department_id, item_id,
    planned_quantity, planned_unit_id, planned_lots, planned_lot_units, planned_run_hours,
    planned_changeover_minutes, sequence_index,
    projected_on_hand_before, projected_on_hand_after,
    status_code, source_code, reason_code, is_frozen,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('production_schedule_id'),
    sqlc.arg('week_index'), sqlc.arg('week_start_date'),
    sqlc.arg('machine_id'), sqlc.narg('production_step_id'), sqlc.narg('department_id'), sqlc.arg('item_id'),
    sqlc.arg('planned_quantity'), sqlc.narg('planned_unit_id'), sqlc.arg('planned_lots'), sqlc.arg('planned_lot_units'), sqlc.arg('planned_run_hours'),
    sqlc.arg('planned_changeover_minutes'), sqlc.arg('sequence_index'),
    sqlc.arg('projected_on_hand_before'), sqlc.arg('projected_on_hand_after'),
    sqlc.arg('status_code'), sqlc.arg('source_code'), sqlc.narg('reason_code'), sqlc.arg('is_frozen'),
    NOW(3), NOW(3)
);

-- ListProductionScheduleLines reads the plan FORWARD in time, matching prod_sched_line_sched_week_idx so the read is filesort-free.
-- name: ListProductionScheduleLines :many
SELECT
    l.id,
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
    COALESCE(prog.released_batches, 0) AS released_batch_count,
    COALESCE(prog.scanned_batches, 0) AS scanned_batch_count,
    COALESCE(prog.scanned_quantity, 0) AS scanned_quantity,
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
LEFT JOIN unit lu ON lu.id = l.planned_unit_id
-- Progress comes from the run the week was released as, matched on the item the campaign is for: a run holds every SKU in its week, so the run alone would credit one campaign with another's work. Aggregated in a derived table rather than joined directly, or the batch rows would multiply the line. The aggregate is bounded to the runs this schedule's lines were released as, so it scans per-run batches rather than the tenant's entire batch history.
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
AND (
    sqlc.arg('include_machine_filter') = false
    OR l.machine_id IN (sqlc.slice('machine_ids'))
)
AND (
    sqlc.narg('week_index') IS NULL
    OR l.week_index = sqlc.narg('week_index')
)
ORDER BY l.week_start_date ASC, l.sequence_index ASC, l.id ASC;

-- name: CreateProductionScheduleItemPolicy :exec
INSERT INTO production_schedule_item_policy (
    id, account_id, production_schedule_id, item_id, sku,
    production_step_id, primary_machine_id, unit_id,
    annual_demand, weekly_demand, seconds_per_unit, unit_cost,
    setup_cost, holding_cost, eoq_units,
    constraint_lead_time_weeks, finish_lead_time_weeks,
    sigma_weekly_pooled, sigma_downstream_sum,
    safety_stock_primary, safety_stock_downstream,
    reorder_point, order_up_to, on_hand_echelon,
    on_hand_greige, average_greige_inventory, max_greige_inventory,
    weeks_of_cover, projected_on_hand, projected_greige_on_hand, annual_run_hours,
    abc_class, was_eoq_capped, was_capacity_starved,
    fulfillment_policy_code, policy_source_code, firm_demand_units, forecast_demand_units,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('production_schedule_id'), sqlc.arg('item_id'), sqlc.arg('sku'),
    sqlc.narg('production_step_id'), sqlc.narg('primary_machine_id'), sqlc.narg('unit_id'),
    sqlc.arg('annual_demand'), sqlc.arg('weekly_demand'), sqlc.arg('seconds_per_unit'), sqlc.arg('unit_cost'),
    sqlc.arg('setup_cost'), sqlc.arg('holding_cost'), sqlc.arg('eoq_units'),
    sqlc.arg('constraint_lead_time_weeks'), sqlc.arg('finish_lead_time_weeks'),
    sqlc.arg('sigma_weekly_pooled'), sqlc.arg('sigma_downstream_sum'),
    sqlc.arg('safety_stock_primary'), sqlc.arg('safety_stock_downstream'),
    sqlc.arg('reorder_point'), sqlc.arg('order_up_to'), sqlc.arg('on_hand_echelon'),
    sqlc.arg('on_hand_greige'), sqlc.arg('average_greige_inventory'), sqlc.arg('max_greige_inventory'),
    sqlc.arg('weeks_of_cover'), sqlc.narg('projected_on_hand'), sqlc.narg('projected_greige_on_hand'), sqlc.arg('annual_run_hours'),
    sqlc.narg('abc_class'), sqlc.arg('was_eoq_capped'), sqlc.arg('was_capacity_starved'),
    sqlc.arg('fulfillment_policy_code'), sqlc.arg('policy_source_code'), sqlc.arg('firm_demand_units'), sqlc.arg('forecast_demand_units'),
    NOW(3), NOW(3)
);

-- name: ListProductionScheduleItemPolicies :many
SELECT
    p.id,
    p.production_schedule_id,
    p.item_id,
    p.sku,
    p.production_step_id,
    p.primary_machine_id,
    p.unit_id,
    pu.abbreviation AS unit_abbreviation,
    p.annual_demand,
    p.weekly_demand,
    p.seconds_per_unit,
    p.unit_cost,
    p.fulfillment_policy_code,
    p.policy_source_code,
    p.firm_demand_units,
    p.forecast_demand_units,
    p.setup_cost,
    p.holding_cost,
    p.eoq_units,
    p.constraint_lead_time_weeks,
    p.finish_lead_time_weeks,
    p.sigma_weekly_pooled,
    p.sigma_downstream_sum,
    p.safety_stock_primary,
    p.safety_stock_downstream,
    p.reorder_point,
    p.order_up_to,
    p.on_hand_echelon,
    p.on_hand_greige,
    p.average_greige_inventory,
    p.max_greige_inventory,
    p.weeks_of_cover,
    CAST(p.projected_on_hand AS CHAR) AS projected_on_hand,
    CAST(p.projected_greige_on_hand AS CHAR) AS projected_greige_on_hand,
    p.annual_run_hours,
    p.abc_class,
    p.was_eoq_capped,
    p.was_capacity_starved,
    p.created_at,
    p.updated_at
FROM production_schedule_item_policy p
LEFT JOIN unit pu ON pu.id = p.unit_id
WHERE p.account_id = sqlc.arg('account_id')
AND p.production_schedule_id = sqlc.arg('production_schedule_id')
ORDER BY p.annual_run_hours DESC, p.id ASC;

-- Finished-goods policy: the per-SKU decomposition of the pooled greige echelon.

-- name: DeleteProductionScheduleFinishedPolicies :exec
DELETE FROM production_schedule_finished_policy
WHERE account_id = sqlc.arg('account_id') AND production_schedule_id = sqlc.arg('production_schedule_id');

-- name: CreateProductionScheduleFinishedPolicy :exec
INSERT INTO production_schedule_finished_policy (
    id, account_id, production_schedule_id,
    item_id, sku, greige_item_id, greige_sku, product_line_id,
    annual_demand, weekly_demand, sigma_weekly,
    safety_stock, reorder_point, on_hand, weeks_of_cover,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('production_schedule_id'),
    sqlc.arg('item_id'), sqlc.arg('sku'), sqlc.arg('greige_item_id'), sqlc.arg('greige_sku'), sqlc.narg('product_line_id'),
    sqlc.arg('annual_demand'), sqlc.arg('weekly_demand'), sqlc.arg('sigma_weekly'),
    sqlc.arg('safety_stock'), sqlc.arg('reorder_point'), sqlc.arg('on_hand'), sqlc.arg('weeks_of_cover'),
    NOW(3), NOW(3)
);

-- ListProductionScheduleFinishedPolicies returns a version's finished-goods targets, grouped under the greige each one is made from. The ORDER BY (greige_sku, sku, id) has no matching index, so the read filesorts; this is an accepted residual bounded by the schedule's finished-SKU count (one row per SKU, rewritten wholesale on each regeneration).
-- name: ListProductionScheduleFinishedPolicies :many
SELECT
    f.id,
    f.production_schedule_id,
    f.item_id,
    f.sku,
    f.greige_item_id,
    f.greige_sku,
    f.product_line_id,
    f.annual_demand,
    f.weekly_demand,
    f.sigma_weekly,
    f.safety_stock,
    f.reorder_point,
    f.on_hand,
    f.weeks_of_cover,
    f.created_at,
    f.updated_at
FROM production_schedule_finished_policy f
WHERE f.account_id = sqlc.arg('account_id')
  AND f.production_schedule_id = sqlc.arg('production_schedule_id')
ORDER BY f.greige_sku, f.sku, f.id;

-- CreateProductionScheduleLineOrder links one campaign to one order it is building.
-- name: CreateProductionScheduleLineOrder :exec
INSERT INTO production_schedule_line_order (
    id, account_id, production_schedule_id,
    production_schedule_line_id, sales_order_id, sales_order_line_id,
    allocated_quantity, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('production_schedule_id'),
    sqlc.arg('production_schedule_line_id'), sqlc.arg('sales_order_id'), sqlc.arg('sales_order_line_id'),
    sqlc.arg('allocated_quantity'), NOW(3), NOW(3)
);

-- DeleteProductionScheduleLineOrders clears a version's links, so a re-solve replaces them wholesale rather than accumulating.
-- name: DeleteProductionScheduleLineOrders :exec
DELETE FROM production_schedule_line_order
WHERE account_id = sqlc.arg('account_id') AND production_schedule_id = sqlc.arg('production_schedule_id');

-- ListProductionScheduleLineOrders returns which campaigns are building which orders for one version.
-- name: ListProductionScheduleLineOrders :many
SELECT
    lo.id,
    lo.production_schedule_line_id,
    lo.sales_order_id,
    lo.sales_order_line_id,
    lo.allocated_quantity,
    so.number AS sales_order_number,
    so.ship_by_date,
    l.item_id,
    l.week_index,
    l.machine_id,
    i.sku
FROM production_schedule_line_order lo
JOIN production_schedule_line l ON l.id = lo.production_schedule_line_id
JOIN sales_order so ON so.id = lo.sales_order_id
JOIN item i ON i.id = l.item_id
WHERE lo.account_id = sqlc.arg('account_id')
  AND lo.production_schedule_id = sqlc.arg('production_schedule_id')
ORDER BY l.week_index, so.number, lo.id;

-- The finishing plan: stage two, which decides how many of which finished good to make from the knitted parts.
--
-- Written wholesale on every solve like the rest of the plan, never patched. The mix is a pure function of the knit plan, the order book and each SKU's position, so a partial update could leave a week holding lines for a campaign the re-solve no longer produces.

-- name: DeleteProductionScheduleFinishingLines :exec
DELETE FROM production_schedule_finishing_line
WHERE account_id = sqlc.arg('account_id') AND production_schedule_id = sqlc.arg('production_schedule_id');

-- name: CreateProductionScheduleFinishingLine :exec
INSERT INTO production_schedule_finishing_line (
    id, account_id, production_schedule_id,
    week_index, week_start_date,
    item_id, sku, greige_item_id, greige_sku,
    department_id, production_step_id,
    planned_quantity, planned_unit_id, planned_lots, planned_lot_units, planned_run_hours,
    greige_consumed, firm_units,
    projected_on_hand_before, projected_on_hand_after,
    status_code, source_code, is_frozen,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('production_schedule_id'),
    sqlc.arg('week_index'), sqlc.arg('week_start_date'),
    sqlc.arg('item_id'), sqlc.arg('sku'), sqlc.arg('greige_item_id'), sqlc.arg('greige_sku'),
    sqlc.narg('department_id'), sqlc.narg('production_step_id'),
    sqlc.arg('planned_quantity'), sqlc.narg('planned_unit_id'), sqlc.arg('planned_lots'), sqlc.arg('planned_lot_units'), sqlc.arg('planned_run_hours'),
    sqlc.arg('greige_consumed'), sqlc.arg('firm_units'),
    sqlc.arg('projected_on_hand_before'), sqlc.arg('projected_on_hand_after'),
    sqlc.arg('status_code'), sqlc.arg('source_code'), sqlc.arg('is_frozen'),
    NOW(3), NOW(3)
);

-- ListProductionScheduleFinishingLines returns a version's finishing plan, forward in time.
--
-- Ordered to match prod_sched_fin_line_sched_week_idx so the read is an index walk rather than a filesort, which matters because this is one row per finished SKU per week rather than one per SKU.
-- name: ListProductionScheduleFinishingLines :many
SELECT
    f.id,
    f.production_schedule_id,
    f.week_index,
    f.week_start_date,
    f.item_id,
    f.sku,
    f.greige_item_id,
    f.greige_sku,
    f.department_id,
    f.production_step_id,
    f.planned_quantity,
    f.planned_unit_id,
    u.abbreviation AS planned_unit_abbreviation,
    f.planned_lots,
    f.planned_lot_units,
    f.planned_run_hours,
    f.greige_consumed,
    f.firm_units,
    f.projected_on_hand_before,
    f.projected_on_hand_after,
    f.status_code,
    f.source_code,
    f.is_frozen,
    f.created_at,
    f.updated_at
FROM production_schedule_finishing_line f
LEFT JOIN unit u ON u.id = f.planned_unit_id
WHERE f.account_id = sqlc.arg('account_id')
  AND f.production_schedule_id = sqlc.arg('production_schedule_id')
  AND (sqlc.narg('week_index') IS NULL OR f.week_index = sqlc.narg('week_index'))
  AND (sqlc.narg('item_id') IS NULL OR f.item_id = sqlc.narg('item_id'))
ORDER BY f.week_start_date, f.sku, f.id;
