-- +goose Up

-- Every carrier has a type. ltl1 was only a second LTL label (EDI integrations tell such carriers apart by SCAC
-- now), and freight_collect described how freight is billed rather than who carries it.
UPDATE `carrier` SET `code` = 'ltl' WHERE `code` = 'ltl1';
UPDATE `carrier` SET `code` = 'local_delivery' WHERE `code` = 'delivery';
UPDATE `carrier` SET `code` = 'other' WHERE `code` IS NULL OR `code` = 'freight_collect';

-- The carrier names its default service level; the per-service-level flag is no longer read.
UPDATE `carrier` c
  JOIN `carrier_option` co ON co.`carrier_id` = c.`id` AND co.`is_default` = 1
SET c.`default_carrier_option_id` = co.`id`
WHERE c.`default_carrier_option_id` IS NULL;

-- A service level has one code: the carrier's own token where it was synced from the carrier, which is what rates
-- and labels are keyed on. service_level_token stays set on synced rows to mark them as synced.
UPDATE `carrier_option` SET `code` = `service_level_token`
WHERE `service_level_token` IS NOT NULL AND `service_level_token` <> '' AND `code` <> `service_level_token`;

-- +goose Down

UPDATE `carrier_option` co
  JOIN `carrier` c ON c.`default_carrier_option_id` = co.`id`
SET co.`is_default` = 1;
UPDATE `carrier` SET `code` = NULL WHERE `code` = 'other';
UPDATE `carrier` SET `code` = 'delivery' WHERE `code` = 'local_delivery';
