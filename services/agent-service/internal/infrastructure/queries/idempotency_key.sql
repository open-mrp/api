-- name: GetIdempotencyKeyByScopeHash :one
SELECT id, type_id, service_name, handler, idempotency_key, actor_id, identity_type,
       scope_hash, response_code, response_body, recovery_point,
       locked_at, lock_owner, lock_expires_at, created_at, updated_at,
       last_run_at, expires_at
FROM service_idempotency_key
WHERE service_name = $1 AND scope_hash = $2
FOR UPDATE;

-- name: CreateIdempotencyKey :one
INSERT INTO service_idempotency_key (
    type_id, service_name, handler, idempotency_key, actor_id, identity_type,
    scope_hash, recovery_point, last_run_at, expires_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), now() + interval '30 days')
RETURNING id;

-- name: AdvanceIdempotencyRecoveryPoint :exec
UPDATE service_idempotency_key
SET recovery_point = $1, last_run_at = now(), updated_at = now()
WHERE type_id = $2;

-- name: GetIdempotencyRecoveryPoint :one
SELECT recovery_point
FROM service_idempotency_key
WHERE type_id = $1;

-- name: SetIdempotencyResponse :exec
UPDATE service_idempotency_key
SET response_code = $1, response_body = $2, recovery_point = $3,
    locked_at = NULL, last_run_at = now(), updated_at = now()
WHERE type_id = $4;
