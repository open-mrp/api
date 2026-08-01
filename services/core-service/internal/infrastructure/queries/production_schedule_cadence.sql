-- Generation cadence: which accounts have asked for a schedule on a timer.

-- ListAccountsWithGenerationCadence returns every account whose cadence is enabled and has a cron expression. The index on (is_enabled, generation_cron) makes this cheap enough to run on every poll tick regardless of tenant count.
-- name: ListAccountsWithGenerationCadence :many
SELECT
    s.account_id,
    s.generation_cron,
    s.generation_timezone,
    s.auto_publish,
    s.last_generated_at,
    s.created_at
FROM account_production_schedule_setting s
WHERE s.is_enabled = true
AND s.generation_cron IS NOT NULL
AND s.generation_cron != ''
ORDER BY s.account_id;

-- StampGenerationRun records that the cadence fired, so the next tick measures from this moment rather than re-firing the same window.
-- name: StampGenerationRun :exec
UPDATE account_production_schedule_setting
SET last_generated_at = sqlc.arg('last_generated_at'), updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id');

-- ReapStalledGeneratingSchedules fails versions left in `generating` by a process that died mid-solve. Without it they sit there forever, and the account looks like it has a schedule when it has an empty shell.
-- name: ReapStalledGeneratingSchedules :exec
UPDATE production_schedule
SET status_code = 'failed',
    error_message = 'Generation did not complete; the process running it stopped.',
    updated_at = NOW(3)
WHERE status_code = 'generating'
AND created_at < sqlc.arg('stalled_before');

-- name: CreateGeneratingProductionSchedule :exec
INSERT INTO production_schedule (
    id, account_id, version, status_code, name,
    planning_as_of, horizon_start_date, horizon_end_date, horizon_weeks, frozen_weeks,
    demand_basis_code, generation_source_code, solver_version,
    settings_snapshot, diagnostics, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('version'), 'generating', sqlc.narg('name'),
    sqlc.arg('planning_as_of'), sqlc.arg('horizon_start_date'), sqlc.arg('horizon_end_date'),
    sqlc.arg('horizon_weeks'), sqlc.arg('frozen_weeks'),
    sqlc.arg('demand_basis_code'), 'scheduled', '',
    JSON_OBJECT(), JSON_OBJECT(), NOW(3), NOW(3)
);

-- FillGeneratedProductionSchedule completes a placeholder row once its solve finishes. The version number and account are already set; everything the solver produced is written here and the status moves out of `generating`.
-- name: FillGeneratedProductionSchedule :exec
UPDATE production_schedule
SET
    status_code = 'draft',
    name = COALESCE(sqlc.narg('name'), name),
    horizon_start_date = sqlc.arg('horizon_start_date'),
    horizon_end_date = sqlc.arg('horizon_end_date'),
    horizon_weeks = sqlc.arg('horizon_weeks'),
    frozen_weeks = sqlc.arg('frozen_weeks'),
    demand_basis_code = sqlc.arg('demand_basis_code'),
    solver_version = sqlc.arg('solver_version'),
    settings_snapshot = sqlc.arg('settings_snapshot'),
    diagnostics = sqlc.arg('diagnostics'),
    error_message = NULL,
    updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
AND id = sqlc.arg('id')
AND status_code = 'generating';

-- FailProductionScheduleGeneration records why a queued solve could not finish, so the merchant sees that the cadence ran and what went wrong rather than a silent gap.
-- name: FailProductionScheduleGeneration :exec
UPDATE production_schedule
SET status_code = 'failed', error_message = sqlc.arg('error_message'), updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
AND id = sqlc.arg('id')
AND status_code = 'generating';

-- RefreshRegeneratedSchedule re-stamps a draft with a fresh solve's own metadata.
--
-- Separate from FillGeneratedProductionSchedule, which only matches `generating`: that one completes a row the cadence reserved, while this one re-solves a draft that is already complete. Sharing a query would let a regenerate silently resurrect a row that was mid-generation.
-- name: RefreshRegeneratedSchedule :exec
UPDATE production_schedule
SET
    planning_as_of = sqlc.arg('planning_as_of'),
    horizon_start_date = sqlc.arg('horizon_start_date'),
    horizon_end_date = sqlc.arg('horizon_end_date'),
    horizon_weeks = sqlc.arg('horizon_weeks'),
    frozen_weeks = sqlc.arg('frozen_weeks'),
    demand_basis_code = sqlc.arg('demand_basis_code'),
    solver_version = sqlc.arg('solver_version'),
    settings_snapshot = sqlc.arg('settings_snapshot'),
    diagnostics = sqlc.arg('diagnostics'),
    error_message = NULL,
    updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
AND id = sqlc.arg('id')
AND status_code = 'draft';
