-- +goose Up

-- Deleting a supplier now takes suppliers:delete rather than suppliers:update. Every role that could delete a supplier
-- before keeps that ability. One row per role holding suppliers, read through role_permission_permission_code_idx.
UPDATE `role_permission`
SET `delete` = 1, `updated_at` = NOW(3)
WHERE `permission_code` = 'suppliers'
  AND `update` = 1
  AND `delete` = 0;

-- +goose Down

-- The roles widened above can't be told apart from roles granted suppliers:delete since, so nothing is narrowed.
SELECT 1;
