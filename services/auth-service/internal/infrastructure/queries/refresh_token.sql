-- name: FindRefreshToken :one
SELECT user_id, expires_at, revoked_at FROM refresh_token 
WHERE token = ?;

-- name: CreateRefreshToken :exec
INSERT INTO refresh_token (user_id, token, expires_at, created_at, updated_at)
VALUES (?, ?, ?, NOW(), NOW());

-- name: RevokeRefreshToken :exec
UPDATE refresh_token SET revoked_at = NOW(), updated_at = NOW() WHERE token = ?;

-- name: RevokeAllRefreshTokensByUserID :exec
UPDATE refresh_token SET revoked_at = NOW(), updated_at = NOW() WHERE user_id = ?;