-- Solver inputs for the production schedule (internal/scheduling).
--
-- These are the direct translations of the reads the knit-scheduling TS script did through Prisma. Everything the solver needs is loaded up front by these queries so the solver itself stays pure and testable against a fixture.

-- GetConstraintMachines returns every machine in the constraint department.
--
-- The department is the selection, not the machine: the knitting room sets the pace of the factory, so a machine added to it is planned without anyone remembering to tick a box. A machine that must sit out — down for a rebuild — is excluded by an explicit resource-setting row; the LEFT JOIN means the absence of a row means "planned", which is the safe default when the department is what was chosen.
-- name: GetConstraintMachines :many
SELECT
    m.id,
    m.name,
    m.department_id,
    m.production_step_id
FROM machine m
LEFT JOIN production_schedule_resource_setting rs
    ON rs.scope_ref_id = m.id
   AND rs.scope_code = 'machine'
   AND rs.account_id = sqlc.arg('account_id')
WHERE m.account_id = sqlc.arg('account_id')
  AND m.department_id = sqlc.arg('department_id')
  AND COALESCE(rs.is_excluded, 0) = 0
  AND COALESCE(rs.is_enabled, 1) = 1
ORDER BY m.name, m.id;

-- GetConstraintDepartmentStepCoverage reports how much of the constraint department can actually carry a plan downstream.
--
-- A campaign explodes into department work through the machine's own production step, so a machine with no step assigned produces campaigns that derive nothing. That is a data problem worth naming rather than guessing at: picking a step for the machine because it happens to share a department would put work in front of a room that was never asked to do it.
-- name: GetConstraintDepartmentStepCoverage :one
SELECT
    COUNT(*) AS machine_count,
    SUM(CASE WHEN m.production_step_id IS NULL THEN 1 ELSE 0 END) AS machines_without_step
FROM machine m
WHERE m.account_id = sqlc.arg('account_id')
  AND m.department_id = sqlc.arg('department_id');

-- GetConstraintBatchMeasurements returns one row per historical batch produced on the constraint machines, which is what the run rate, cost, lot count, machine affinity and measured lead time are all derived from.
--
-- labor_time is a Rate whose numerator unit decides its scale (min/hr/sec); the caller converts. production_run.created_at paired with scanned_at gives the observed lead time for that batch.
-- name: GetConstraintBatchMeasurements :many
SELECT
    b.id AS batch_id,
    b.item_id,
    i.sku,
    b.scanned_at,
    q.value AS quantity_value,
    u.abbreviation AS quantity_unit,
    q.unit_id AS quantity_unit_id,
    u.ratio_numerator,
    u.ratio_denominator,
    b.production_step_id,
    bm.B AS machine_id,
    mc.name AS machine_name,
    cost_rate.value AS unit_cost,
    labor_time.value AS labor_time_value,
    labor_time_unit.abbreviation AS labor_time_unit,
    labor_rate.value AS labor_rate,
    overhead_rate.value AS overhead_rate,
    pr.created_at AS run_created_at
FROM batch b
JOIN _batches_machines bm ON bm.A = b.id
JOIN machine mc ON mc.id = bm.B
JOIN item i ON i.id = b.item_id
LEFT JOIN quantity q ON q.id = b.quantity_id
LEFT JOIN unit u ON u.id = q.unit_id
LEFT JOIN rate cost_rate ON cost_rate.id = i.unit_cost_id
LEFT JOIN production_step ps ON ps.id = b.production_step_id
LEFT JOIN scanning_station ss ON ss.id = ps.scanning_station_id
LEFT JOIN rate labor_time ON labor_time.id = ps.labor_time_id
LEFT JOIN unit labor_time_unit ON labor_time_unit.id = labor_time.numerator_unit_id
LEFT JOIN rate labor_rate ON labor_rate.id = ps.labor_rate_id
LEFT JOIN rate overhead_rate ON overhead_rate.id = ps.overhead_rate_id
LEFT JOIN production_run pr ON pr.id = b.production_run_id
WHERE b.account_id = sqlc.arg('account_id')
  AND b.scanned_at IS NOT NULL
  AND b.scanned_at >= sqlc.arg('window_start')
  AND b.scanned_at <= sqlc.arg('window_end')
  AND bm.B IN (sqlc.slice('machine_ids'))
  -- A constraint machine can carry scans from other stages (a sewing step recorded against a knitting machine). Only steps that belong to the constraint department are measurements of the constraint; a step names its department directly or through the scanning station it is scanned at.
  AND COALESCE(ps.department_id, ss.department_id) = sqlc.arg('constraint_department_id')
ORDER BY bm.B, b.scanned_at, b.id;

-- GetStepConsumptionItems returns the input items each production step consumes. The count of inputs a product introduces relative to the previous one is what drives the changeover model.
-- name: GetStepConsumptionItems :many
SELECT
    c.production_step_id,
    c.item_id,
    i.sku
FROM consumption c
JOIN item i ON i.id = c.item_id
WHERE c.production_step_id IN (sqlc.slice('production_step_ids'))
ORDER BY c.production_step_id, i.sku;

-- GetBatchFlowChildren returns the immediate downstream batches for a set of batches.
--
-- The caller walks the genealogy one level at a time, passing the whole frontier each round. That is O(depth) queries total regardless of how many items are being planned, where the TS script issued one query per depth PER item. A recursive CTE would be nicer still, but sqlc's MySQL parser cannot resolve the self-reference. Per the Prisma orientation of _batch_flow (row (A, B): A = downstream/target, B = upstream/source; see docs/patterns/production-step-graph-patterns.md and InsertBatchFlow in batch.sql), a batch's children are the A side of rows where it is B.
-- name: GetBatchFlowChildren :many
SELECT
    bf.B AS parent_batch_id,
    child.id AS batch_id,
    child.item_id
FROM _batch_flow bf
JOIN batch child ON child.id = bf.A
WHERE child.account_id = sqlc.arg('account_id')
  AND bf.B IN (sqlc.slice('parent_batch_ids'))
ORDER BY bf.B, child.id;

-- GetSeedBatchesForItems returns the batches to start the genealogy walk from. Capped per item by the caller: a handful of recent batches is enough to discover which finished goods an item becomes.
-- name: GetSeedBatchesForItems :many
SELECT
    b.id AS batch_id,
    b.item_id
FROM batch b
WHERE b.account_id = sqlc.arg('account_id')
  AND b.item_id IN (sqlc.slice('item_ids'))
  AND b.scanned_at IS NOT NULL
ORDER BY b.item_id, b.scanned_at DESC, b.id DESC;

-- GetProductsForItems returns the sellable products for a set of items, with the SKU and product line each one carries so a finished good can be reported by name rather than by ID. Only items with a product carry order demand.
-- name: GetProductsForItems :many
SELECT
    p.id AS product_id,
    p.item_id,
    i.sku,
    p.product_line_id
FROM product p
JOIN item i ON i.id = p.item_id
WHERE i.account_id = sqlc.arg('account_id')
  AND p.item_id IN (sqlc.slice('item_ids'))
ORDER BY p.item_id;

-- GetEchelonOnHand returns available inventory per item, net of what is already allocated. Quantities are normalized through the unit ratio so items stocked in different units are comparable.
-- name: GetEchelonOnHand :many
SELECT
    ir.item_id,
    CAST(COALESCE(SUM(
        (q.value - COALESCE(alloc.allocated, 0)) * (u.ratio_numerator / u.ratio_denominator)
    ), 0) AS DECIMAL(65,30)) AS on_hand
FROM inventory_receipt ir
JOIN quantity q ON q.id = ir.quantity_id
JOIN unit u ON u.id = q.unit_id
LEFT JOIN (
    SELECT ia.inventory_receipt_id, SUM(aq.value) AS allocated
    FROM inventory_allocation ia
    JOIN quantity aq ON aq.id = ia.quantity_id
    GROUP BY ia.inventory_receipt_id
) alloc ON alloc.inventory_receipt_id = ir.id
WHERE ir.owner_account_id = sqlc.arg('account_id')
  AND ir.status_code = 'available'
  AND ir.item_id IN (sqlc.slice('item_ids'))
GROUP BY ir.item_id;

-- GetPooledOrderDemandByProduct returns monthly sold quantity per product, which is pooled back onto the constraint item that produces it.
--
-- Estimates are excluded: an unissued quote is not demand.
-- name: GetPooledOrderDemandByProduct :many
SELECT
    sol.product_id,
    YEAR(so.issued_at) AS demand_year,
    MONTH(so.issued_at) AS demand_month,
    CAST(COALESCE(SUM(q.value * (u.ratio_numerator / u.ratio_denominator)), 0) AS DECIMAL(65,30)) AS quantity
FROM sales_order_line sol
JOIN sales_order so ON so.id = sol.sales_order_id
JOIN quantity q ON q.id = sol.quantity_id
JOIN unit u ON u.id = q.unit_id
WHERE so.owner_account_id = sqlc.arg('account_id')
  AND so.sales_order_type_code = 'sales_order'
  AND so.sales_order_status_code <> 'estimate'
  AND so.issued_at IS NOT NULL
  AND so.issued_at >= sqlc.arg('window_start')
  AND so.issued_at <= sqlc.arg('window_end')
  AND sol.product_id IN (sqlc.slice('product_ids'))
GROUP BY sol.product_id, YEAR(so.issued_at), MONTH(so.issued_at)
ORDER BY sol.product_id, demand_year, demand_month;

-- GetActiveDemandOverrides returns the overrides in force for the planning date. These are the only mechanism for adjusting demand away from history; there is no growth multiplier.
-- name: GetActiveDemandOverrides :many
SELECT
    do_.id,
    do_.scope_code,
    do_.scope_ref_id,
    do_.period_start_date,
    do_.period_end_date,
    do_.override_type_code,
    do_.value,
    do_.reason_code,
    do_.created_at
FROM demand_override do_
WHERE do_.account_id = sqlc.arg('account_id')
  AND do_.is_active = 1
  AND do_.effective_from <= sqlc.arg('as_of')
  AND (do_.expires_at IS NULL OR do_.expires_at > sqlc.arg('as_of_expiry'))
ORDER BY do_.scope_code, do_.scope_ref_id, do_.period_start_date, do_.created_at;

-- GetItemsForProductLines resolves a product-line-scoped override onto its items so the distribution can be done per item.
-- name: GetItemsForProductLines :many
SELECT
    p.product_line_id,
    p.item_id
FROM product p
JOIN item i ON i.id = p.item_id
WHERE i.account_id = sqlc.arg('account_id')
  AND p.product_line_id IN (sqlc.slice('product_line_ids'))
ORDER BY p.product_line_id, p.item_id;

-- GetAccountProductionScheduleSetting returns the merchant's planning assumptions. Absent means the account has never configured scheduling, and the caller falls back to code defaults rather than refusing to plan.
-- name: GetAccountProductionScheduleSetting :one
SELECT
    s.id,
    s.constraint_department_id,
    s.planning_horizon_weeks,
    s.frozen_weeks,
    s.demand_window_months,
    s.forecast_history_months,
    s.forecast_months,
    s.demand_basis_code,
    s.forecast_z,
    s.changeover_avg_minutes,
    s.changeover_min_minutes,
    s.changeover_max_minutes,
    s.changeover_labor_rate,
    s.holding_rate_pct,
    s.service_level_z,
    s.finish_lead_time_weeks,
    s.default_constraint_lead_time_weeks,
    s.max_weeks_supply,
    s.max_flow_depth,
    s.shifts_per_day,
    s.hours_per_shift,
    s.work_days_per_week,
    s.weeks_per_year,
    s.capacity_headroom_pct,
    s.default_lot_units,
    -- The cadence and display columns are read by the settings API, which shares this query so there is one definition of "the merchant's assumptions".
    s.week_start_day,
    s.is_enabled,
    s.generation_cron,
    s.generation_timezone,
    s.auto_publish,
    s.last_generated_at,
    s.created_at,
    s.updated_at
FROM account_production_schedule_setting s
WHERE s.account_id = sqlc.arg('account_id');

-- GetProductionScheduleItemSettings returns per-item planning overrides.
-- name: GetProductionScheduleItemSettings :many
SELECT
    s.item_id,
    s.is_excluded,
    s.lot_multiple_units
FROM production_schedule_item_setting s
WHERE s.account_id = sqlc.arg('account_id');
