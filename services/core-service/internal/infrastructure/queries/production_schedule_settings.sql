-- Merchant-editable planning assumptions. Every value the solver uses that was a hardcoded constant in the original script lives here.

-- The read lives in production_schedule_input.sql as GetAccountProductionScheduleSetting, which the solver already uses. One row, one query.

-- UpsertAccountProductionScheduleSetting writes the whole settings row.
--
-- Upsert rather than update: an account that has never opened the settings page has no row, and the API must not make a merchant save twice to change one number.
-- name: UpsertAccountProductionScheduleSetting :exec
INSERT INTO account_production_schedule_setting (
    id, account_id, constraint_department_id,
    planning_horizon_weeks, frozen_weeks, week_start_day,
    demand_window_months, forecast_history_months, forecast_months,
    demand_basis_code, forecast_z,
    changeover_avg_minutes, changeover_min_minutes, changeover_max_minutes, changeover_labor_rate,
    holding_rate_pct, service_level_z, finish_lead_time_weeks,
    default_constraint_lead_time_weeks, max_weeks_supply, max_flow_depth,
    shifts_per_day, hours_per_shift, work_days_per_week, weeks_per_year,
    capacity_headroom_pct, default_lot_units,
    is_enabled, generation_cron, generation_timezone, auto_publish,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.narg('constraint_department_id'),
    sqlc.arg('planning_horizon_weeks'), sqlc.arg('frozen_weeks'), sqlc.arg('week_start_day'),
    sqlc.arg('demand_window_months'), sqlc.arg('forecast_history_months'), sqlc.arg('forecast_months'),
    sqlc.arg('demand_basis_code'), sqlc.arg('forecast_z'),
    sqlc.arg('changeover_avg_minutes'), sqlc.arg('changeover_min_minutes'), sqlc.arg('changeover_max_minutes'), sqlc.arg('changeover_labor_rate'),
    sqlc.arg('holding_rate_pct'), sqlc.arg('service_level_z'), sqlc.arg('finish_lead_time_weeks'),
    sqlc.arg('default_constraint_lead_time_weeks'), sqlc.arg('max_weeks_supply'), sqlc.arg('max_flow_depth'),
    sqlc.arg('shifts_per_day'), sqlc.arg('hours_per_shift'), sqlc.arg('work_days_per_week'), sqlc.arg('weeks_per_year'),
    sqlc.arg('capacity_headroom_pct'), sqlc.arg('default_lot_units'),
    sqlc.arg('is_enabled'), sqlc.narg('generation_cron'), sqlc.arg('generation_timezone'), sqlc.arg('auto_publish'),
    NOW(3), NOW(3)
)
ON DUPLICATE KEY UPDATE
    constraint_department_id = VALUES(constraint_department_id),
    planning_horizon_weeks = VALUES(planning_horizon_weeks),
    frozen_weeks = VALUES(frozen_weeks),
    week_start_day = VALUES(week_start_day),
    demand_window_months = VALUES(demand_window_months),
    forecast_history_months = VALUES(forecast_history_months),
    forecast_months = VALUES(forecast_months),
    demand_basis_code = VALUES(demand_basis_code),
    forecast_z = VALUES(forecast_z),
    changeover_avg_minutes = VALUES(changeover_avg_minutes),
    changeover_min_minutes = VALUES(changeover_min_minutes),
    changeover_max_minutes = VALUES(changeover_max_minutes),
    changeover_labor_rate = VALUES(changeover_labor_rate),
    holding_rate_pct = VALUES(holding_rate_pct),
    service_level_z = VALUES(service_level_z),
    finish_lead_time_weeks = VALUES(finish_lead_time_weeks),
    default_constraint_lead_time_weeks = VALUES(default_constraint_lead_time_weeks),
    max_weeks_supply = VALUES(max_weeks_supply),
    max_flow_depth = VALUES(max_flow_depth),
    shifts_per_day = VALUES(shifts_per_day),
    hours_per_shift = VALUES(hours_per_shift),
    work_days_per_week = VALUES(work_days_per_week),
    weeks_per_year = VALUES(weeks_per_year),
    capacity_headroom_pct = VALUES(capacity_headroom_pct),
    default_lot_units = VALUES(default_lot_units),
    is_enabled = VALUES(is_enabled),
    generation_cron = VALUES(generation_cron),
    generation_timezone = VALUES(generation_timezone),
    auto_publish = VALUES(auto_publish),
    updated_at = NOW(3);

-- name: UpsertProductionScheduleResourceSetting :exec
INSERT INTO production_schedule_resource_setting (
    id, account_id, scope_code, scope_ref_id, is_excluded,
    lead_time_weeks, lead_time_offset_weeks, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('scope_code'), sqlc.arg('scope_ref_id'),
    sqlc.arg('is_excluded'), sqlc.narg('lead_time_weeks'), sqlc.arg('lead_time_offset_weeks'), NOW(3), NOW(3)
)
ON DUPLICATE KEY UPDATE
    is_excluded = VALUES(is_excluded),
    lead_time_weeks = VALUES(lead_time_weeks),
    lead_time_offset_weeks = VALUES(lead_time_offset_weeks),
    updated_at = NOW(3);

-- name: DeleteProductionScheduleResourceSetting :execrows
DELETE FROM production_schedule_resource_setting
WHERE account_id = sqlc.arg('account_id') AND id = sqlc.arg('id');
