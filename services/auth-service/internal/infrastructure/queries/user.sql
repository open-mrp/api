-- name: FindUserByIdentifier :one
SELECT * FROM user WHERE username = ? OR email = ? OR id = ?;

-- name: FindLastUsedAccountID :one
SELECT account_id FROM account_user 
WHERE user_id = ? 
ORDER BY last_used_at DESC 
LIMIT 1;

-- name: UpdateUserPassword :exec
UPDATE user SET hashed_password = ? WHERE id = ?;

-- name: CreateUser :exec
INSERT INTO user (id, email, name, hashed_password, created_at, updated_at)
VALUES (?, ?, ?, ?, NOW(3), NOW(3));