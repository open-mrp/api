-- +goose NO TRANSACTION
-- +goose Up

-- Orders keep their own copy of the bill-to and ship-to addresses (the saved address each came from, if any, plus the
-- printed fields), so editing a saved address never rewrites an order. NULL {prefix}_country means no copy yet; reads
-- fall back to the live address until the paced backfill has copied every order.
ALTER TABLE `sales_order`
  ADD COLUMN `bill_to_address_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `bill_to_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `bill_to_line_1` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `bill_to_line_2` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `bill_to_city` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `bill_to_state` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `bill_to_postal_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `bill_to_country` char(2) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `bill_to_phone` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `bill_to_email` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `ship_to_address_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `ship_to_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `ship_to_line_1` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `ship_to_line_2` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `ship_to_city` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `ship_to_state` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `ship_to_postal_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `ship_to_country` char(2) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `ship_to_phone` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD COLUMN `ship_to_email` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  -- The site the order ships from; NULL is the account's default site.
  ADD COLUMN `site_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  -- soft | hard: whether the customer can accept the order late.
  ADD COLUMN `commitment_firmness` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  -- The lead time the ship-by date was promised on, in business days, and where it came from (customer | item | override).
  ADD COLUMN `promised_lead_time_days` int NULL,
  ADD COLUMN `promised_lead_time_source` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL;

-- Probed when a site location is deleted.
ALTER TABLE `sales_order` ADD KEY `sales_order_site_id_idx` (`site_id`);

-- One row per party per two-party document (an order is read by its seller as a sales order and by its buyer as a
-- purchase order): that party's own instructions and metadata, which the other party never sees.
CREATE TABLE `document_party` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `document_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `instructions` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `metadata` json DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `document_party_document_account_key` (`document_id`, `account_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down

DROP TABLE `document_party`;
ALTER TABLE `sales_order` DROP KEY `sales_order_site_id_idx`;
ALTER TABLE `sales_order`
  DROP COLUMN `promised_lead_time_source`,
  DROP COLUMN `promised_lead_time_days`,
  DROP COLUMN `commitment_firmness`,
  DROP COLUMN `site_id`,
  DROP COLUMN `ship_to_email`,
  DROP COLUMN `ship_to_phone`,
  DROP COLUMN `ship_to_country`,
  DROP COLUMN `ship_to_postal_code`,
  DROP COLUMN `ship_to_state`,
  DROP COLUMN `ship_to_city`,
  DROP COLUMN `ship_to_line_2`,
  DROP COLUMN `ship_to_line_1`,
  DROP COLUMN `ship_to_name`,
  DROP COLUMN `ship_to_address_id`,
  DROP COLUMN `bill_to_email`,
  DROP COLUMN `bill_to_phone`,
  DROP COLUMN `bill_to_country`,
  DROP COLUMN `bill_to_postal_code`,
  DROP COLUMN `bill_to_state`,
  DROP COLUMN `bill_to_city`,
  DROP COLUMN `bill_to_line_2`,
  DROP COLUMN `bill_to_line_1`,
  DROP COLUMN `bill_to_name`,
  DROP COLUMN `bill_to_address_id`;
