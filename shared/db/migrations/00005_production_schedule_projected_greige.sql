-- +goose NO TRANSACTION
-- +goose Up

-- The week-by-week physical greige store, alongside the echelon projected_on_hand it complements.
-- Nullable JSON so a schedule generated before the greige buffer existed reads as absent rather
-- than as an empty curve. DDL, so it reaches prod through a deploy request under safe migrations.
ALTER TABLE `production_schedule_item_policy`
  ADD COLUMN `projected_greige_on_hand` json DEFAULT NULL AFTER `projected_on_hand`;

-- +goose Down

ALTER TABLE `production_schedule_item_policy`
  DROP COLUMN `projected_greige_on_hand`;
