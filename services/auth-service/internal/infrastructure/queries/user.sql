-- name: FindUserByIdentifier :one
SELECT `user`.id, `user`.email, `user`.name, `user`.username, `user`.hashed_password, `user`.email_verified, `user`.image_url, `user`.status_code, `user`.created_at, `user`.updated_at
FROM `user` WHERE `user`.id = sqlc.arg('id') AND (`user`.status_code = 'active' OR `user`.status_code IS NULL)
UNION ALL
SELECT `user`.id, `user`.email, `user`.name, `user`.username, `user`.hashed_password, `user`.email_verified, `user`.image_url, `user`.status_code, `user`.created_at, `user`.updated_at
FROM `user` WHERE `user`.email = sqlc.arg('email') AND (`user`.status_code = 'active' OR `user`.status_code IS NULL)
UNION ALL
SELECT `user`.id, `user`.email, `user`.name, `user`.username, `user`.hashed_password, `user`.email_verified, `user`.image_url, `user`.status_code, `user`.created_at, `user`.updated_at
FROM `user` WHERE `user`.username = sqlc.arg('username') AND (`user`.status_code = 'active' OR `user`.status_code IS NULL)
LIMIT 1;

-- name: FindLastUsedAccountID :one
SELECT account_id FROM account_user 
WHERE user_id = ? 
    AND (status_code = 'active' OR status_code IS NULL)
ORDER BY last_used_at DESC 
LIMIT 1;

-- name: UpdateUserPassword :exec
UPDATE user SET hashed_password = ? WHERE id = ?;

-- name: CreateUser :exec
INSERT INTO user (id, email, name, hashed_password, status_code, created_at, updated_at)
VALUES (?, ?, ?, ?, 'active', NOW(3), NOW(3));