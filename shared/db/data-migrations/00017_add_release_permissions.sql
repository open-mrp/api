-- +goose Up

-- batch_releases lets a supervisor sign a batch off at a step that requires a supervisor release; quality_releases lets a quality inspector sign quality and final releases. Only create applies to either. No role is granted them here, so until an admin grants them only admins hold them.
INSERT IGNORE INTO `permission` (`id`, `code`, `name`, `description`, `permission_group_code`, `created_at`, `updated_at`)
VALUES
    ('pm_1bt4r8zq2kcm', 'batch_releases', 'Batch Releases', 'Sign batches off at production steps that require a supervisor release, in your own department. Only create applies.', 'production', NOW(3), NOW(3)),
    ('pm_1bt4r8zq2kcn', 'quality_releases', 'Quality Releases', 'Sign quality and final releases of batches at any production step. Only create applies.', 'production', NOW(3), NOW(3));

-- +goose Down

DELETE FROM `permission` WHERE `code` IN ('batch_releases', 'quality_releases');
