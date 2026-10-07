-- +goose NO TRANSACTION
-- +goose Up

-- Client-owned string → string map, returned unchanged and never read by OpenMRP. NULL and {} mean
-- the same thing: nullable so adding the column does not rewrite every row.
ALTER TABLE `sales_order`
  ADD COLUMN `metadata` json DEFAULT NULL;

ALTER TABLE `sales_order_line`
  ADD COLUMN `metadata` json DEFAULT NULL;

ALTER TABLE `invoice`
  ADD COLUMN `metadata` json DEFAULT NULL;

-- +goose Down

ALTER TABLE `invoice`
  DROP COLUMN `metadata`;

ALTER TABLE `sales_order_line`
  DROP COLUMN `metadata`;

ALTER TABLE `sales_order`
  DROP COLUMN `metadata`;
