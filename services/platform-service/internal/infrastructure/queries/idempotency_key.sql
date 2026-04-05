-- name: SetIdempotencyKeyResponse :exec
UPDATE idempotency_key
SET response_code = ?, response_body = ?, response_headers = ?,
    locked_at = NULL, lock_owner = NULL, lock_expires_at = NULL,
    last_run_at = NOW(3), updated_at = NOW(3), recovery_point = ?
WHERE type_id = ?;

-- name: SetIdempotencyKeyResponseWithTTL :exec
UPDATE idempotency_key
SET response_code = ?, response_body = ?, response_headers = ?,
    expires_at = DATE_ADD(NOW(3), INTERVAL ? SECOND),
    locked_at = NULL, lock_owner = NULL, lock_expires_at = NULL,
    last_run_at = NOW(3), updated_at = NOW(3), recovery_point = ?
WHERE type_id = ?;

-- name: LockIdempotencyKey :execresult
UPDATE idempotency_key
SET locked_at = NOW(3), lock_owner = ?, lock_expires_at = DATE_ADD(NOW(3), INTERVAL 5 MINUTE),
    last_run_at = NOW(3), updated_at = NOW(3)
WHERE type_id = ? AND (lock_expires_at IS NULL OR lock_expires_at < NOW(3));

-- name: DeleteExpiredIdempotencyKeys :execresult
DELETE FROM idempotency_key
WHERE expires_at < NOW(3)
LIMIT ?;

-- name: GetIdempotencyKeyByScopeHashForUpdate :one
SELECT * FROM idempotency_key WHERE scope_hash = ? FOR UPDATE;

-- name: CreateIdempotencyKeyWithScope :execlastid
INSERT INTO idempotency_key (
    type_id, scope_hash, request_body_hash, actor_id, identity_type, target_account_id, request_method,
    normalized_route, idempotency_key, request_params, recovery_point, last_run_at, expires_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), DATE_ADD(NOW(3), INTERVAL 30 DAY)
);

-- name: AdvanceRecoveryPoint :exec
UPDATE idempotency_key
SET recovery_point = ?, request_params = COALESCE(?, request_params), last_run_at = NOW(3), updated_at = NOW(3)
WHERE type_id = ?;

-- name: GetRecoveryPoint :one
SELECT recovery_point, request_params
FROM idempotency_key
WHERE type_id = ?;

-- name: ReleaseIdempotencyKeyLock :exec
UPDATE idempotency_key
SET locked_at = NULL, lock_owner = NULL, lock_expires_at = NULL,
    updated_at = NOW(3)
WHERE type_id = ?;

-- name: DeleteExpiredServiceIdempotencyKeys :execresult
DELETE FROM service_idempotency_key
WHERE expires_at < NOW(3)
LIMIT ?;

-- name: DeleteExpiredDeletedRecords :execresult
DELETE FROM deleted_record
WHERE deleted_at < DATE_SUB(NOW(3), INTERVAL 30 DAY)
LIMIT ?;
