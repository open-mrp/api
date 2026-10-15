-- +goose Up

-- A customer's or supplier's note becomes its instructions.
UPDATE `account_relation` SET `instructions` = `notes`
WHERE `notes` IS NOT NULL AND `notes` <> '' AND `instructions` IS NULL;

-- A customer's fulfillment policy becomes its default supply: built to stock is stock, built to order is make.
UPDATE `account_relation`
SET `supply_code` = CASE `fulfillment_policy_code` WHEN 'make_to_stock' THEN 'stock' WHEN 'make_to_order' THEN 'make' END
WHERE `fulfillment_policy_code` IS NOT NULL AND `fulfillment_policy_code` <> '' AND `supply_code` IS NULL;

-- The preferred standing is gone; such a customer has no restrictions.
UPDATE `account_relation` SET `account_status_code` = 'normal' WHERE `account_status_code` = 'preferred';

-- +goose Down

UPDATE `account_relation` SET `supply_code` = NULL WHERE `supply_code` IS NOT NULL;
UPDATE `account_relation` SET `instructions` = NULL WHERE `instructions` IS NOT NULL;
