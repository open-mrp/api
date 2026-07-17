-- name: InsertHubspotSyncJob :exec
-- dry_run is intentionally omitted: the preview pass is the dry run, so the column is vestigial and left to its default.
INSERT INTO hubspot_sync_job (
    id,
    account_id,
    status,
    golive_cutoff_at,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_id'),
    sqlc.arg('status'),
    sqlc.narg('golive_cutoff_at'),
    NOW(3),
    NOW(3)
);

-- name: GetHubspotSyncJob :one
SELECT
    id,
    account_id,
    status,
    dry_run,
    golive_cutoff_at,
    cursors,
    counts,
    last_error,
    started_at,
    completed_at,
    created_at,
    updated_at
FROM hubspot_sync_job
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: GetLatestHubspotSyncJobForAccount :one
SELECT
    id,
    account_id,
    status,
    dry_run,
    golive_cutoff_at,
    cursors,
    counts,
    last_error,
    started_at,
    completed_at,
    created_at,
    updated_at
FROM hubspot_sync_job
WHERE account_id = sqlc.arg('account_id')
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: UpdateHubspotSyncJob :execresult
UPDATE hubspot_sync_job SET
    status = COALESCE(sqlc.narg('status'), status),
    cursors = COALESCE(sqlc.narg('cursors'), cursors),
    counts = COALESCE(sqlc.narg('counts'), counts),
    last_error = NULLIF(COALESCE(sqlc.narg('last_error'), last_error), ''),
    started_at = COALESCE(sqlc.narg('started_at'), started_at),
    completed_at = COALESCE(sqlc.narg('completed_at'), completed_at),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- Compare-and-swap a job into the executing phase. Matching on the current status makes this the concurrency gate for execute: only one caller can win the transition, so a double-click cannot dispatch two execute commands for the same job.
-- name: ClaimHubspotSyncJobForExecute :execresult
UPDATE hubspot_sync_job SET
    status = 'executing',
    last_error = NULL,
    started_at = NOW(3),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id')
AND status IN ('review_pending', 'failed');
