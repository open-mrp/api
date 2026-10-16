-- +goose Up

-- manufacturing_methods gates manufacturing methods, which replace production steps. Every role that could work with
-- production steps gets the same access to methods. One row per role holding production_steps, read through
-- role_permission_permission_code_idx.
INSERT IGNORE INTO `permission` (`id`, `code`, `name`, `description`, `permission_group_code`, `created_at`, `updated_at`)
VALUES ('pm_mfgm4h7c2kq9', 'manufacturing_methods', 'Manufacturing Methods', 'Read and edit how items are made: methods, their operations and materials.', 'production', NOW(3), NOW(3));

INSERT IGNORE INTO `role_permission` (`id`, `role_id`, `permission_code`, `create`, `read`, `update`, `delete`, `created_at`, `updated_at`)
SELECT CONCAT('rlpm_', LEFT(MD5(CONCAT(rp.`id`, ':manufacturing_methods')), 16)), rp.`role_id`, 'manufacturing_methods', rp.`create`, rp.`read`, rp.`update`, rp.`delete`, NOW(3), NOW(3)
FROM `role_permission` rp
WHERE rp.`permission_code` = 'production_steps'
  AND NOT EXISTS (SELECT 1 FROM `role_permission` x WHERE x.`role_id` = rp.`role_id` AND x.`permission_code` = 'manufacturing_methods');

-- +goose Down

DELETE FROM `role_permission` WHERE `permission_code` = 'manufacturing_methods';
DELETE FROM `permission` WHERE `code` = 'manufacturing_methods';
