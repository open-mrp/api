-- name: InsertBatchSignature :exec
INSERT INTO batch_signature (
    id, account_id, batch_id, production_step_id, production_run_id,
    type_code, decision_code, capacity_code,
    signer_account_user_id, signer_name, signer_role_name,
    checklist, note, actor_id, actor_type, lot_id, lot_number, signed_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('batch_id'), sqlc.arg('production_step_id'), sqlc.narg('production_run_id'),
    sqlc.arg('type_code'), sqlc.arg('decision_code'), sqlc.arg('capacity_code'),
    sqlc.arg('signer_account_user_id'), sqlc.arg('signer_name'), sqlc.narg('signer_role_name'),
    sqlc.narg('checklist'), sqlc.narg('note'), sqlc.arg('actor_id'), sqlc.arg('actor_type'), sqlc.narg('lot_id'), sqlc.narg('lot_number'), NOW(3)
);

-- name: GetBatchSignature :one
SELECT * FROM batch_signature
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- ListBatchSignaturesForBatch returns every signature a batch has, newest first. A batch collects a
-- handful at most, so the list is never paged.
-- name: ListBatchSignaturesForBatch :many
SELECT * FROM batch_signature
WHERE account_id = sqlc.arg('account_id') AND batch_id = sqlc.arg('batch_id')
ORDER BY signed_at DESC, id DESC;

-- ListBatchSignaturesForBatches returns the signatures of several batches, newest first, for deciding
-- whether each may move on.
-- name: ListBatchSignaturesForBatches :many
SELECT * FROM batch_signature
WHERE account_id = sqlc.arg('account_id') AND batch_id IN (sqlc.slice('batch_ids'))
ORDER BY signed_at DESC, id DESC;

-- ListRunStepReleaseSignatures returns the release signatures any batch of a production run received
-- at a step, newest first: a first-article sign-off on one of them covers the whole run there.
-- name: ListRunStepReleaseSignatures :many
SELECT * FROM batch_signature
WHERE account_id = sqlc.arg('account_id')
  AND production_run_id = sqlc.arg('production_run_id')
  AND production_step_id = sqlc.arg('production_step_id')
  AND type_code = 'step_release'
ORDER BY signed_at DESC, id DESC;

-- ListEligibleSigners returns the account's active users who may sign in either capacity: admins, and
-- users whose role grants creating batch_releases (supervisor) or quality_releases (quality). The
-- caller narrows supervisors to the step's department.
-- name: ListEligibleSigners :many
SELECT
    au.id AS account_user_id,
    COALESCE(u.name, u.username) AS name,
    au.department_id,
    r.name AS role_name,
    r.role_type_code,
    (sup.id IS NOT NULL) AS can_supervise,
    (qa.id IS NOT NULL) AS can_inspect
FROM account_user au
JOIN `user` u ON u.id = au.user_id
JOIN role r ON r.id = au.role_id
LEFT JOIN role_permission sup ON sup.role_id = r.id AND sup.permission_code = 'batch_releases' AND sup.`create` = 1
LEFT JOIN role_permission qa ON qa.role_id = r.id AND qa.permission_code = 'quality_releases' AND qa.`create` = 1
WHERE au.account_id = sqlc.arg('account_id')
  AND au.status_code = 'active'
  AND (r.role_type_code = 'admin' OR sup.id IS NOT NULL OR qa.id IS NOT NULL)
ORDER BY name ASC, au.id ASC
LIMIT 500;
