-- +goose NO TRANSACTION
-- +goose Up

ALTER TABLE `item`
  ADD COLUMN `burn_rate_computed_at` datetime(3) NULL,
  ADD KEY `item_burn_rate_computed_idx` (`deleted_at`, `burn_rate_computed_at`);

-- +goose Down

ALTER TABLE `item`
  DROP KEY `item_burn_rate_computed_idx`,
  DROP COLUMN `burn_rate_computed_at`;
