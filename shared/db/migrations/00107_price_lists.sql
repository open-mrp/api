-- +goose NO TRANSACTION
-- +goose Up

-- A set of pricing rules assigned to customers or account groups. source_type/source_id name the legacy pricing
-- record a list mirrors (a customer's account prices, an account group's volume discounts, or the account's
-- unrestricted volume discounts) while those records are still written; null for lists made as price lists.
CREATE TABLE `price_list` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `currency` char(3) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'usd',
  `source_type` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `source_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `price_list_account_created_idx` (`account_id`, `created_at` DESC, `id` DESC),
  UNIQUE KEY `price_list_account_source_key` (`account_id`, `source_type`, `source_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- One price for a scope of items. price_type is fixed (unit_price_*), percent_of_base, or tiered (tier_unit_id plus
-- price_list_rule_tier rows). source_id names the legacy account price or volume discount a mirrored rule comes from.
CREATE TABLE `price_list_rule` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `price_list_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `price_type` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `unit_price_amount` decimal(20,6) DEFAULT NULL,
  `unit_price_currency` char(3) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `unit_price_per_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `percent_of_base` decimal(20,6) DEFAULT NULL,
  `tier_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `source_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `price_list_rule_list_created_idx` (`price_list_id`, `created_at` DESC, `id` DESC),
  UNIQUE KEY `price_list_rule_list_source_key` (`price_list_id`, `source_id`),
  KEY `price_list_rule_account_source_idx` (`account_id`, `source_id`),
  KEY `price_list_rule_unit_price_per_unit_id_idx` (`unit_price_per_unit_id`),
  KEY `price_list_rule_tier_unit_id_idx` (`tier_unit_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- A tiered rule's breaks, in ascending min_quantity (position 0 first). Each sets percent_of_base or a unit price.
CREATE TABLE `price_list_rule_tier` (
  `price_list_rule_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `position` smallint NOT NULL,
  `min_quantity` decimal(20,6) NOT NULL,
  `percent_of_base` decimal(20,6) DEFAULT NULL,
  `unit_price_amount` decimal(20,6) DEFAULT NULL,
  `unit_price_currency` char(3) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `unit_price_per_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`price_list_rule_id`, `position`),
  KEY `price_list_rule_tier_unit_price_per_unit_id_idx` (`unit_price_per_unit_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- A rule's scope, one table per dimension. An empty dimension matches any item.
CREATE TABLE `price_list_rule_item` (
  `price_list_rule_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`price_list_rule_id`, `item_id`),
  KEY `price_list_rule_item_item_id_idx` (`item_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `price_list_rule_product_line` (
  `price_list_rule_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `product_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`price_list_rule_id`, `product_line_id`),
  KEY `price_list_rule_product_line_product_line_id_idx` (`product_line_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `price_list_rule_item_category` (
  `price_list_rule_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `item_category_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`price_list_rule_id`, `item_category_id`),
  KEY `price_list_rule_item_category_item_category_id_idx` (`item_category_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `price_list_rule_attribute` (
  `price_list_rule_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `attribute_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`price_list_rule_id`, `attribute_id`),
  KEY `price_list_rule_attribute_attribute_id_idx` (`attribute_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Lists assigned directly to a customer (its account relation) or to every customer in an account group.
CREATE TABLE `price_list_customer` (
  `account_relation_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `price_list_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`account_relation_id`, `price_list_id`),
  KEY `price_list_customer_price_list_id_idx` (`price_list_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `price_list_account_group` (
  `account_group_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `price_list_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`account_group_id`, `price_list_id`),
  KEY `price_list_account_group_price_list_id_idx` (`price_list_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Accounts whose legacy account prices and volume discounts have been copied into price lists. Until an account has a
-- row, its orders are still priced from the legacy records.
CREATE TABLE `price_list_legacy_sync` (
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `synced_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`account_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Order discounts: a typed discount in decimals, archivable codes, and a stored redemption counter. The legacy
-- percentage/value doubles stay written alongside until order totals read the new columns.
ALTER TABLE `order_discount`
  ADD COLUMN `percent_off` decimal(9,6) DEFAULT NULL,
  ADD COLUMN `amount_off_amount` decimal(20,6) DEFAULT NULL,
  ADD COLUMN `amount_off_currency` char(3) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  ADD COLUMN `is_active` tinyint(1) NOT NULL DEFAULT '1',
  ADD COLUMN `times_redeemed` int NOT NULL DEFAULT '0',
  ADD KEY `order_discount_account_created_idx` (`account_id`, `created_at` DESC, `id` DESC);

-- Search is a substring match on name or code; the word FULLTEXT indexes are unused.
ALTER TABLE `order_discount` DROP INDEX `order_discount_name_idx`;
ALTER TABLE `order_discount` DROP INDEX `order_discount_code_fulltext_idx`;

-- +goose Down

ALTER TABLE `order_discount` ADD FULLTEXT KEY `order_discount_code_fulltext_idx` (`code`);
ALTER TABLE `order_discount` ADD FULLTEXT KEY `order_discount_name_idx` (`name`);
ALTER TABLE `order_discount`
  DROP KEY `order_discount_account_created_idx`,
  DROP COLUMN `times_redeemed`,
  DROP COLUMN `is_active`,
  DROP COLUMN `amount_off_currency`,
  DROP COLUMN `amount_off_amount`,
  DROP COLUMN `percent_off`;
DROP TABLE `price_list_legacy_sync`;
DROP TABLE `price_list_account_group`;
DROP TABLE `price_list_customer`;
DROP TABLE `price_list_rule_attribute`;
DROP TABLE `price_list_rule_item_category`;
DROP TABLE `price_list_rule_product_line`;
DROP TABLE `price_list_rule_item`;
DROP TABLE `price_list_rule_tier`;
DROP TABLE `price_list_rule`;
DROP TABLE `price_list`;
