-- name: FindUserByEmail :one
SELECT id, email, name, username, hashed_password, email_verified, image_url, status_code, created_at, updated_at
FROM `user`
WHERE email = ?;

-- name: FindUserByUsername :one
SELECT id, email, name, username, hashed_password, email_verified, image_url, status_code, created_at, updated_at
FROM `user`
WHERE username = ?;

-- name: InsertUser :exec
INSERT INTO `user` (id, name, email, username, hashed_password, status_code, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, 'active', NOW(3), NOW(3));

-- name: UpdateUserProfile :exec
UPDATE `user`
SET name = COALESCE(sqlc.narg(name), name),
    email = COALESCE(sqlc.narg(email), email),
    username = COALESCE(sqlc.narg(username), username),
    image_url = COALESCE(sqlc.narg(image_url), image_url),
    email_verified = COALESCE(sqlc.narg(email_verified), email_verified),
    updated_at = NOW(3)
WHERE id = ?;

-- name: UpdateUserPassword :exec
UPDATE `user`
SET hashed_password = ?, updated_at = NOW(3)
WHERE id = ?;

-- name: GetUserHashedPassword :one
SELECT hashed_password
FROM `user`
WHERE id = ?;

-- name: FindUserByID :one
SELECT id, email, name, username, hashed_password, email_verified, image_url, status_code, created_at, updated_at
FROM `user`
WHERE id = ?;

-- name: GetUsersByIDs :many
SELECT u.id, u.email, u.name, u.username, u.email_verified, u.image_url, u.status_code, u.created_at, u.updated_at
FROM `user` u
WHERE u.id IN (sqlc.slice('ids'))
AND EXISTS (
    SELECT 1 FROM account_user au
    WHERE au.user_id = u.id AND au.account_id = sqlc.arg('account_id')
);

-- name: UpdateUserImageURL :exec
UPDATE `user`
SET image_url = ?, updated_at = NOW(3)
WHERE id = ?;
