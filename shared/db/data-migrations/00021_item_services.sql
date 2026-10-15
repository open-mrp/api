-- +goose Up

-- A service is its own item type rather than a product whose product type says so. The product row stays: it holds
-- the service's sales details.
UPDATE `item` i
  JOIN `product` p ON p.`item_id` = i.`id`
SET i.`item_type_code` = 'service'
WHERE p.`product_type_code` = 'service' AND i.`item_type_code` <> 'service';

-- +goose Down

UPDATE `item` i
  JOIN `product` p ON p.`item_id` = i.`id`
SET i.`item_type_code` = 'product'
WHERE p.`product_type_code` = 'service' AND i.`item_type_code` = 'service';
