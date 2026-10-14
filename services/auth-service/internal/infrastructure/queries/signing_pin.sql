-- name: GetSigningPinByUserID :one
SELECT user_id, pin_hash, is_temporary, failed_attempts, locked_until, set_at, created_at, updated_at
FROM signing_pin
WHERE user_id = ?;

-- name: UpsertSigningPin :exec
INSERT INTO signing_pin (user_id, pin_hash, is_temporary, failed_attempts, locked_until, set_at, created_at, updated_at)
VALUES (sqlc.arg('user_id'), sqlc.arg('pin_hash'), sqlc.arg('is_temporary'), 0, NULL, NOW(3), NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE
    pin_hash = VALUES(pin_hash),
    is_temporary = VALUES(is_temporary),
    failed_attempts = 0,
    locked_until = NULL,
    set_at = VALUES(set_at),
    updated_at = VALUES(updated_at);

-- name: DeleteSigningPin :execrows
DELETE FROM signing_pin WHERE user_id = ?;

-- name: RecordSigningPinFailure :execrows
-- MySQL applies SET assignments left to right, so locked_until reads failed_attempts before it is reset. The pin_hash guard keeps a wrong guess from counting against a PIN set since it was read.
UPDATE signing_pin
SET locked_until = IF(failed_attempts + 1 >= sqlc.arg('max_attempts'), sqlc.arg('lock_until'), locked_until),
    failed_attempts = IF(failed_attempts + 1 >= sqlc.arg('max_attempts'), 0, failed_attempts + 1),
    updated_at = NOW(3)
WHERE user_id = sqlc.arg('user_id') AND pin_hash = sqlc.arg('pin_hash');

-- name: ResetSigningPinFailures :exec
UPDATE signing_pin
SET failed_attempts = 0, locked_until = NULL, updated_at = NOW(3)
WHERE user_id = ? AND (failed_attempts <> 0 OR locked_until IS NOT NULL);
