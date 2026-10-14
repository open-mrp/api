-- name: GetDepartmentReleasePolicy :one
SELECT id, account_id, name, release_requirement_code, release_enforcement_code, release_operator_signing_code, release_rationale, release_checklist, updated_at
FROM department
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- ListDepartmentReleasePolicies returns the release policies of several departments, for checking the
-- batches a scan consumes.
-- name: ListDepartmentReleasePolicies :many
SELECT id, account_id, name, release_requirement_code, release_enforcement_code, release_operator_signing_code, release_rationale, release_checklist, updated_at
FROM department
WHERE account_id = sqlc.arg('account_id') AND id IN (sqlc.slice('ids'));

-- name: UpdateDepartmentReleasePolicy :exec
UPDATE department SET
    release_requirement_code = sqlc.arg('release_requirement_code'),
    release_enforcement_code = sqlc.arg('release_enforcement_code'),
    release_operator_signing_code = sqlc.arg('release_operator_signing_code'),
    release_rationale = sqlc.narg('release_rationale'),
    release_checklist = sqlc.narg('release_checklist'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');
