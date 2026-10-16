-- +goose Up

-- Members with an email were sent a welcome email when they were added, so they read as invited until they sign in.
UPDATE `account_user` au
JOIN `user` u ON u.`id` = au.`user_id`
SET au.`invited_at` = au.`created_at`
WHERE u.`email` IS NOT NULL AND au.`last_used_at` IS NULL AND au.`invited_at` IS NULL;

-- +goose Down

UPDATE `account_user` SET `invited_at` = NULL WHERE `invited_at` IS NOT NULL;
