-- name: ListJobsForward :many
SELECT
    j.job_id AS id,
    j.type,
    j.account_id,
    j.created_by,
    j.results,
    j.errors,
    j.error_summary,
    j.started_at,
    j.completed_at,
    j.failed_at, 
    j.cancelled_at,
    j.created_at,
    j.updated_at
FROM job j
WHERE j.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR j.created_at < sqlc.narg('cursor_created_at')
    OR (j.created_at = sqlc.narg('cursor_created_at') AND j.job_id < sqlc.narg('cursor_id'))
)
ORDER BY j.created_at DESC, j.job_id DESC
LIMIT ?;

-- name: ListJobsBackward :many
SELECT
    j.job_id AS id,
    j.type,
    j.account_id,
    j.created_by,
    j.results,
    j.errors,
    j.error_summary,
    j.started_at,
    j.completed_at,
    j.failed_at, 
    j.cancelled_at,
    j.updated_at,
    j.created_at
FROM job j
WHERE j.account_id = sqlc.arg('account_id')
AND (
    j.created_at > sqlc.arg('cursor_created_at')
    OR (j.created_at = sqlc.arg('cursor_created_at') AND j.job_id > sqlc.arg('cursor_id'))
)
ORDER BY j.created_at ASC, j.job_id ASC
LIMIT ?;

-- name: GetJob :one
SELECT
    j.job_id AS id,
    j.type,
    j.account_id,
    j.created_by,
    u.name AS created_by_name,
    u.username AS created_by_username,
    u.email AS created_by_email,
    j.job_items,
    j.results,
    j.errors,
    j.error_summary,
    j.started_at,
    j.completed_at,
    j.failed_at, 
    j.cancelled_at,
    j.created_at,
    j.updated_at
FROM job j
LEFT JOIN account_user au
    ON au.id = j.created_by
    AND au.account_id = j.account_id
LEFT JOIN `user` u
    ON u.id = au.user_id
WHERE j.job_id = sqlc.arg('id')
AND j.account_id = sqlc.arg('account_id');

-- name: InsertJob :exec
INSERT INTO job (
    job_id,
    type,
    account_id,
    created_by,
    job_items,
    results,
    created_at,
    updated_at
) Values (
    sqlc.arg('id'),
    sqlc.arg('type'),
    sqlc.arg('account_id'),
    sqlc.arg('created_by'),
    sqlc.arg('job_items'),
    sqlc.narg('results'),
    NOW(3),
    NOW(3)
);

-- name: UpdateJob :execrows
-- The terminal timestamps guard the update so a job that already settled matches no
-- row: that is what serializes a client's cancel against the worker's completion.
UPDATE job SET
    results = COALESCE(sqlc.narg('results'), results),
    errors = COALESCE(sqlc.narg('errors'), errors),
    error_summary = COALESCE(sqlc.narg('error_summary'), error_summary),
    started_at = COALESCE(sqlc.narg('started_at'), started_at),
    completed_at = COALESCE(sqlc.narg('completed_at'), completed_at),
    failed_at = COALESCE(sqlc.narg('failed_at'), failed_at),
    cancelled_at = COALESCE(sqlc.narg('cancelled_at'), cancelled_at),
    updated_at = NOW(3)
WHERE job_id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id')
AND completed_at IS NULL
AND cancelled_at IS NULL;