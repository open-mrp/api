-- +goose Up

-- Percent discounts become whole-number percents (0.05 -> 5); amount discounts become amount_off in usd.
UPDATE `order_discount`
SET `percent_off` = ROUND(`percentage` * 100, 6)
WHERE `discount_type_code` = 'percentage' AND `percent_off` IS NULL;

UPDATE `order_discount`
SET `amount_off_amount` = ROUND(`value`, 6), `amount_off_currency` = 'usd'
WHERE `discount_type_code` = 'amount' AND `amount_off_amount` IS NULL;

-- The redemption counter starts from the orders that already carry each code.
UPDATE `order_discount` od
SET od.`times_redeemed` = (SELECT COUNT(*) FROM `sales_order` so WHERE so.`order_discount_id` = od.`id`);

-- +goose Down

UPDATE `order_discount` SET `percent_off` = NULL, `amount_off_amount` = NULL, `amount_off_currency` = NULL, `times_redeemed` = 0;
