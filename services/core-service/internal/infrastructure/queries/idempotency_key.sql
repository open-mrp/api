-- name: GetIdempotencyKeyByScopeHash :one
SELECT id, type_id, service_name, handler, idempotency_key, actor_id, identity_type,
       scope_hash, response_code, response_body, recovery_point,
       locked_at, lock_owner, lock_expires_at, created_at, updated_at,
       last_run_at, expires_at, response_body_key
FROM service_idempotency_key
WHERE service_name = ? AND scope_hash = ?
FOR UPDATE;

-- name: CreateIdempotencyKey :execlastid
INSERT INTO service_idempotency_key (
    type_id, service_name, handler, idempotency_key, actor_id, identity_type,
    scope_hash, recovery_point, last_run_at, expires_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(3), DATE_ADD(NOW(3), INTERVAL 30 DAY));

-- name: AdvanceIdempotencyRecoveryPoint :exec
UPDATE service_idempotency_key
SET recovery_point = ?, last_run_at = NOW(3), updated_at = NOW(3)
WHERE type_id = ?;

-- name: GetIdempotencyRecoveryPoint :one
SELECT recovery_point
FROM service_idempotency_key
WHERE type_id = ?;

-- name: SetIdempotencyResponse :exec
UPDATE service_idempotency_key
SET response_code = ?, response_body = ?, recovery_point = ?,
    locked_at = NULL, last_run_at = NOW(3), updated_at = NOW(3)
WHERE type_id = ?;

-- name: MoveIdempotencyResponseToObjectStorage :exec
-- Runs once the response's transaction has committed and its body is in the bucket. The guard
-- makes a repeated move a no-op.
UPDATE service_idempotency_key
SET response_body = NULL, response_body_key = sqlc.arg('response_body_key')
WHERE type_id = sqlc.arg('type_id') AND response_body_key IS NULL;

-- name: GetIdempotencyKeyByTypeID :one
SELECT id, type_id, service_name, handler, idempotency_key, actor_id, identity_type,
       scope_hash, response_code, response_body, recovery_point,
       locked_at, lock_owner, lock_expires_at, created_at, updated_at,
       last_run_at, expires_at, response_body_key
FROM service_idempotency_key
WHERE type_id = ?;

-- name: DeleteExpiredIdempotencyKeys :execresult
DELETE FROM service_idempotency_key
WHERE expires_at < NOW(3)
LIMIT ?;
