-- name: FindRefreshToken :one
SELECT user_id, expires_at, revoked_at FROM refresh_token 
WHERE token = ?;

-- name: CreateRefreshToken :exec
INSERT INTO refresh_token (user_id, token, expires_at, created_at, updated_at)
VALUES (?, ?, ?, NOW(3), NOW(3));

-- name: RevokeRefreshToken :exec
UPDATE refresh_token SET revoked_at = NOW(3), updated_at = NOW(3) WHERE token = ?;

-- name: RevokeAllRefreshTokensByUserID :exec
UPDATE refresh_token SET revoked_at = NOW(3), updated_at = NOW(3) WHERE user_id = ?;