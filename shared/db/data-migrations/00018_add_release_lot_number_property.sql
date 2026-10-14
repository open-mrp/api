-- +goose Up

-- The per-account counter final releases number their lots from.
INSERT IGNORE INTO `sys_property_type` (`id`, `code`, `name`, `created_at`, `updated_at`)
VALUES ('sypptp_1bt4r8zrellot', 'release_lot_number', 'Release Lot Number', NOW(3), NOW(3));

-- +goose Down

DELETE FROM `sys_property_type` WHERE `code` = 'release_lot_number';
