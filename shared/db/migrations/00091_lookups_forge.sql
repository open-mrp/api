-- +goose NO TRANSACTION
-- +goose Up

-- Shipping terms can be archived: they stay on customers and orders but can't be newly assigned.
ALTER TABLE `shipping_term` ADD COLUMN `is_active` tinyint(1) NOT NULL DEFAULT '1';

-- Name search is a tenant-scoped substring LIKE; the word-parser FULLTEXT indexes are unused.
ALTER TABLE `shipping_term` DROP KEY `shipping_term_name_idx`;
ALTER TABLE `payment_term` DROP KEY `payment_term_name_idx`;
ALTER TABLE `unit`
  DROP KEY `unit_name_idx`,
  DROP KEY `unit_abbreviation_idx`,
  DROP KEY `unit_name_abbreviation_idx`;

-- Deleting a unit probes demand overrides recorded in it.
ALTER TABLE `demand_override` ADD KEY `demand_override_unit_id_idx` (`unit_id`);

-- +goose Down

ALTER TABLE `demand_override` DROP KEY `demand_override_unit_id_idx`;
ALTER TABLE `unit`
  ADD FULLTEXT KEY `unit_name_idx` (`name`),
  ADD FULLTEXT KEY `unit_abbreviation_idx` (`abbreviation`),
  ADD FULLTEXT KEY `unit_name_abbreviation_idx` (`name`, `abbreviation`);
ALTER TABLE `payment_term` ADD FULLTEXT KEY `payment_term_name_idx` (`name`);
ALTER TABLE `shipping_term` ADD FULLTEXT KEY `shipping_term_name_idx` (`name`);
ALTER TABLE `shipping_term` DROP COLUMN `is_active`;
