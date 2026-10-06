-- +goose Up

-- costs:read decides whether an internal caller sees cost and margin figures. INSERT IGNORE on the unique code, so a catalog that already carries it is left as it is.
INSERT IGNORE INTO `permission` (`id`, `code`, `name`, `description`, `permission_group_code`, `created_at`, `updated_at`)
VALUES ('pm_16dq9m44rk8e', 'costs', 'Costs', 'See costs and margins: item unit costs, order line costs, costing and margin reports. Only read applies.', 'pricing', NOW(3), NOW(3));

-- +goose Down

-- Role grants of it are kept, so re-applying Up restores what each role had.
DELETE FROM `permission` WHERE `code` = 'costs';
