-- +goose Up

-- A sales order stopped before anything shipped is canceled.
INSERT IGNORE INTO `sales_order_status` (`id`, `code`, `name`, `created_at`, `updated_at`)
VALUES ('orss_01seedcanceled00000', 'canceled', 'Canceled', NOW(3), NOW(3));

-- +goose Down

DELETE FROM `sales_order_status` WHERE `code` = 'canceled';
