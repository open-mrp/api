-- +goose Up

-- notes gates the note resource, which replaces agent memories. Every role that could work with memories gets the same
-- access to notes. One row per role holding agent_memories, read through role_permission_permission_code_idx.
INSERT IGNORE INTO `permission` (`id`, `code`, `name`, `description`, `permission_group_code`, `created_at`, `updated_at`)
VALUES ('pm_nt7k3wq9xd2f', 'notes', 'Notes', 'Read and write notes on records and the account. Update and delete apply to your own notes; admins can delete any.', 'teams', NOW(3), NOW(3));

INSERT IGNORE INTO `role_permission` (`id`, `role_id`, `permission_code`, `create`, `read`, `update`, `delete`, `created_at`, `updated_at`)
SELECT CONCAT('rlpm_', LEFT(MD5(CONCAT(rp.`id`, ':notes')), 16)), rp.`role_id`, 'notes', rp.`create`, rp.`read`, rp.`update`, rp.`delete`, NOW(3), NOW(3)
FROM `role_permission` rp
WHERE rp.`permission_code` = 'agent_memories'
  AND NOT EXISTS (SELECT 1 FROM `role_permission` x WHERE x.`role_id` = rp.`role_id` AND x.`permission_code` = 'notes');

-- +goose Down

DELETE FROM `role_permission` WHERE `permission_code` = 'notes';
DELETE FROM `permission` WHERE `code` = 'notes';
