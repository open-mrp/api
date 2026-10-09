-- +goose Up

-- billing gates what the account spends on agents: read sees the estimated spend and the spending cap, update sets the cap. No role is granted it here, so until an admin grants it only admins hold it.
INSERT IGNORE INTO `permission` (`id`, `code`, `name`, `description`, `permission_group_code`, `created_at`, `updated_at`)
VALUES ('pm_1bq7k2xw9hfm', 'billing', 'Billing', 'See agent spend and the monthly agent spending cap; update sets the cap.', 'admin', NOW(3), NOW(3));

-- +goose Down

DELETE FROM `permission` WHERE `code` = 'billing';
