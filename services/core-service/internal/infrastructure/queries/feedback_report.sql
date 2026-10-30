-- name: InsertFeedbackReport :exec
INSERT INTO feedback_report (
    id,
    account_id,
    user_id,
    message,
    page,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_id'),
    sqlc.arg('user_id'),
    sqlc.arg('message'),
    sqlc.narg('page'),
    NOW(3),
    NOW(3)
);

-- name: GetFeedbackReport :one
SELECT
    fr.id,
    fr.account_id,
    fr.user_id,
    fr.message,
    fr.page,
    fr.created_at,
    fr.updated_at
FROM feedback_report fr
WHERE fr.id = sqlc.arg('id')
AND fr.account_id = sqlc.arg('account_id');

-- name: GetFeedbackReportsByIDs :many
SELECT
    fr.id,
    fr.account_id,
    fr.user_id,
    fr.message,
    fr.page,
    fr.created_at,
    fr.updated_at
FROM feedback_report fr
WHERE fr.id IN (sqlc.slice('ids'))
AND fr.account_id = sqlc.arg('account_id');
