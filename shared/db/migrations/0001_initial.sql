-- +goose Up

DROP TABLE IF EXISTS `InventoryIssueStatus`;
CREATE TABLE `InventoryIssueStatus` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `InventoryIssueStatus_code_key` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `_batch_flow`;
CREATE TABLE `_batch_flow` (
  `A` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `B` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  UNIQUE KEY `_batch_flow_AB_unique` (`A`,`B`),
  KEY `_batch_flow_B_index` (`B`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `_batches_machines`;
CREATE TABLE `_batches_machines` (
  `A` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `B` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  UNIQUE KEY `_batches_machines_AB_unique` (`A`,`B`),
  KEY `_batches_machines_B_index` (`B`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `_departments_picks`;
CREATE TABLE `_departments_picks` (
  `A` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `B` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  UNIQUE KEY `_departments_picks_AB_unique` (`A`,`B`),
  KEY `_departments_picks_B_index` (`B`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `_item_attributes`;
CREATE TABLE `_item_attributes` (
  `A` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `B` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  UNIQUE KEY `_item_attributes_AB_unique` (`A`,`B`),
  KEY `_item_attributes_B_index` (`B`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `_item_categories_properties`;
CREATE TABLE `_item_categories_properties` (
  `A` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `B` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  UNIQUE KEY `_item_categories_properties_AB_unique` (`A`,`B`),
  KEY `_item_categories_properties_B_index` (`B`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `_item_categories_quantity_discounts`;
CREATE TABLE `_item_categories_quantity_discounts` (
  `A` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `B` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  UNIQUE KEY `_item_categories_quantity_discounts_AB_unique` (`A`,`B`),
  KEY `_item_categories_quantity_discounts_B_index` (`B`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `_parent_child_production_steps`;
CREATE TABLE `_parent_child_production_steps` (
  `A` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `B` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  UNIQUE KEY `_parent_child_production_steps_AB_unique` (`A`,`B`),
  KEY `_parent_child_production_steps_B_index` (`B`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `_payment_terms_registration_flows`;
CREATE TABLE `_payment_terms_registration_flows` (
  `A` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `B` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  UNIQUE KEY `_payment_terms_registration_flows_AB_unique` (`A`,`B`),
  KEY `_payment_terms_registration_flows_B_index` (`B`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `_product_lines_quantity_discounts`;
CREATE TABLE `_product_lines_quantity_discounts` (
  `A` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `B` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  UNIQUE KEY `_product_lines_quantity_discounts_AB_unique` (`A`,`B`),
  KEY `_product_lines_quantity_discounts_B_index` (`B`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `_quantity_discounts_attributes`;
CREATE TABLE `_quantity_discounts_attributes` (
  `A` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `B` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  UNIQUE KEY `_quantity_discounts_attributes_AB_unique` (`A`,`B`),
  KEY `_quantity_discounts_attributes_B_index` (`B`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `_quantity_discounts_units`;
CREATE TABLE `_quantity_discounts_units` (
  `A` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `B` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  UNIQUE KEY `_quantity_discounts_units_AB_unique` (`A`,`B`),
  KEY `_quantity_discounts_units_B_index` (`B`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `_registration_flows_shipping_terms`;
CREATE TABLE `_registration_flows_shipping_terms` (
  `A` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `B` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  UNIQUE KEY `_registration_flows_shipping_terms_AB_unique` (`A`,`B`),
  KEY `_registration_flows_shipping_terms_B_index` (`B`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account`;
CREATE TABLE `account` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `onboarding_status_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_billing_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `default_billing_address_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `default_shipping_address_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `account_plan_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `internal_stripe_customer_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `internal_stripe_subscription_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `subscription_status` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `subscription_current_period_end` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_account_billing_id_key` (`account_billing_id`),
  UNIQUE KEY `account_default_billing_address_id_key` (`default_billing_address_id`),
  UNIQUE KEY `account_default_shipping_address_id_key` (`default_shipping_address_id`),
  KEY `account_account_type_code_idx` (`account_type_code`),
  KEY `account_onboarding_status_code_idx` (`onboarding_status_code`),
  FULLTEXT KEY `account_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_address`;
CREATE TABLE `account_address` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `address_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_address_account_id_address_id_key` (`account_id`,`address_id`),
  KEY `account_address_address_id_idx` (`address_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_billing`;
CREATE TABLE `account_billing` (
  `id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_plan_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `internal_stripe_customer_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `internal_stripe_subscription_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `subscription_status` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `subscription_current_period_end` datetime(3) DEFAULT NULL,
  `stripe_billing_profile_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `stripe_billing_cadence_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `stripe_pricing_plan_subscription_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `servicing_status` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `collection_status` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `agent_monthly_spending_cap_cents` bigint DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `account_billing_account_plan_id_idx` (`account_plan_id`),
  KEY `account_billing_internal_stripe_customer_id_idx` (`internal_stripe_customer_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_branding`;
CREATE TABLE `account_branding` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `owner_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `support_email` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `logo_url` varchar(2083) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `favicon_url` varchar(2083) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `phone_number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `facebook_handle` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `instagram_handle` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `linkedin_handle` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `twitter_handle` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `website_url` varchar(2083) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_branding_owner_account_id_key` (`owner_account_id`),
  FULLTEXT KEY `account_branding_support_email_idx` (`support_email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_group`;
CREATE TABLE `account_group` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `owner_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `commission_status_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `freight_status_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_group_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `registration_flow_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `default_lead_time_days` int DEFAULT NULL,
  `fulfillment_policy_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `receive_calendar_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `account_group_registration_flow_id_idx` (`registration_flow_id`),
  KEY `account_group_owner_account_id_idx` (`owner_account_id`),
  KEY `account_group_account_group_type_code_idx` (`account_group_type_code`),
  KEY `account_group_receive_calendar_id_idx` (`receive_calendar_id`),
  FULLTEXT KEY `account_group_name_idx` (`name`),
  FULLTEXT KEY `account_group_description_idx` (`description`),
  FULLTEXT KEY `account_group_name_description_idx` (`name`,`description`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_group_product_line`;
CREATE TABLE `account_group_product_line` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_group_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `product_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_group_product_line_account_group_id_product_line_id_key` (`account_group_id`,`product_line_id`),
  KEY `account_group_product_line_product_line_id_idx` (`product_line_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_group_quantity_discount`;
CREATE TABLE `account_group_quantity_discount` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_group_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `quantity_discount_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_group_quantity_discount_account_group_id_quantity_di_key` (`account_group_id`,`quantity_discount_id`),
  KEY `account_group_quantity_discount_quantity_discount_id_idx` (`quantity_discount_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_integration`;
CREATE TABLE `account_integration` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `integration_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `credentials_v2` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_integration_account_id_integration_code_key` (`account_id`,`integration_code`),
  KEY `account_integration_integration_code_idx` (`integration_code`),
  KEY `account_integration_is_active_idx` (`is_active`),
  FULLTEXT KEY `account_integration_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_inventory_setting`;
CREATE TABLE `account_inventory_setting` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `inventory_policy_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_inventory_setting_account_id_key` (`account_id`),
  KEY `account_inventory_setting_inventory_policy_code_idx` (`inventory_policy_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_plan`;
CREATE TABLE `account_plan` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `type_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `plan_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `version` int NOT NULL DEFAULT '1',
  `price_per_seat` decimal(10,2) NOT NULL,
  `price_per_month` decimal(10,2) DEFAULT NULL,
  `seat_minimum` int DEFAULT NULL,
  `stripe_pricing_plan_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `registration_limit` int DEFAULT NULL,
  `display_features` json NOT NULL,
  `display_order` int NOT NULL DEFAULT '0',
  `is_highlighted` tinyint(1) NOT NULL DEFAULT '0',
  `button_text` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'Get Started',
  `includes_previous_plan` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `is_publicly_visible` tinyint(1) NOT NULL DEFAULT '0',
  `effective_at` datetime(3) NOT NULL,
  `expires_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_plan_type_id_key` (`type_id`),
  KEY `account_plan_plan_type_code_effective_at_idx` (`plan_type_code`,`effective_at`),
  KEY `account_plan_is_publicly_visible_created_at_id_idx` (`is_publicly_visible`,`created_at` DESC,`id` DESC),
  FULLTEXT KEY `account_plan_name_idx` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=133 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_plan_feature`;
CREATE TABLE `account_plan_feature` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `account_plan_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `key` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_plan_feature_account_plan_id_key_key` (`account_plan_id`,`key`)
) ENGINE=InnoDB AUTO_INCREMENT=397 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_plan_limit`;
CREATE TABLE `account_plan_limit` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `account_plan_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `key` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `value` int DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_plan_limit_account_plan_id_key_key` (`account_plan_id`,`key`)
) ENGINE=InnoDB AUTO_INCREMENT=529 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_portal`;
CREATE TABLE `account_portal` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `owner_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `slug` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_portal_owner_account_id_key` (`owner_account_id`),
  UNIQUE KEY `account_portal_slug_key` (`slug`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_price`;
CREATE TABLE `account_price` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `owner_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `unit_value_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `product_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `recipient_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_price_unit_value_id_key` (`unit_value_id`),
  KEY `account_price_recipient_account_id_idx` (`recipient_account_id`),
  KEY `account_price_owner_account_id_idx` (`owner_account_id`),
  KEY `account_price_product_line_id_idx` (`product_line_id`),
  KEY `account_price_owner_recipient_created_idx` (`owner_account_id`,`recipient_account_id`,`created_at` DESC,`id` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_price_attribute`;
CREATE TABLE `account_price_attribute` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_price_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `attribute_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_price_attribute_account_price_id_attribute_id_key` (`account_price_id`,`attribute_id`),
  KEY `account_price_attribute_attribute_id_idx` (`attribute_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_price_item_category`;
CREATE TABLE `account_price_item_category` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_price_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `item_category_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_price_item_category_account_price_id_item_category_i_key` (`account_price_id`,`item_category_id`),
  KEY `account_price_item_category_item_category_id_idx` (`item_category_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_production_schedule_setting`;
CREATE TABLE `account_production_schedule_setting` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `constraint_department_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `planning_horizon_weeks` int NOT NULL DEFAULT '13',
  `frozen_weeks` int NOT NULL DEFAULT '1',
  `week_start_day` int NOT NULL DEFAULT '1',
  `demand_window_months` int NOT NULL DEFAULT '12',
  `forecast_history_months` int NOT NULL DEFAULT '24',
  `forecast_months` int NOT NULL DEFAULT '12',
  `demand_basis_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'trailing_12',
  `forecast_z` decimal(6,4) NOT NULL DEFAULT '1.0000',
  `changeover_avg_minutes` decimal(10,4) NOT NULL DEFAULT '30.0000',
  `changeover_min_minutes` decimal(10,4) NOT NULL DEFAULT '15.0000',
  `changeover_max_minutes` decimal(10,4) NOT NULL DEFAULT '90.0000',
  `changeover_labor_rate` decimal(18,6) NOT NULL DEFAULT '20.000000',
  `holding_rate_pct` decimal(9,6) NOT NULL DEFAULT '0.250000',
  `service_level_z` decimal(6,4) NOT NULL DEFAULT '1.6450',
  `finish_lead_time_weeks` decimal(8,3) NOT NULL DEFAULT '6.000',
  `default_constraint_lead_time_weeks` decimal(8,3) NOT NULL DEFAULT '1.300',
  `max_weeks_supply` decimal(8,3) NOT NULL DEFAULT '12.000',
  `max_flow_depth` int NOT NULL DEFAULT '10',
  `shifts_per_day` int NOT NULL DEFAULT '2',
  `hours_per_shift` decimal(6,3) NOT NULL DEFAULT '7.000',
  `work_days_per_week` int NOT NULL DEFAULT '5',
  `weeks_per_year` int NOT NULL DEFAULT '52',
  `capacity_headroom_pct` decimal(5,4) NOT NULL DEFAULT '0.9000',
  `default_lot_units` decimal(18,6) NOT NULL DEFAULT '60.000000',
  `is_enabled` tinyint(1) NOT NULL DEFAULT '0',
  `generation_cron` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `generation_timezone` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'UTC',
  `auto_publish` tinyint(1) NOT NULL DEFAULT '0',
  `last_generated_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `default_customer_lead_time_days` int NOT NULL DEFAULT '30',
  `default_fulfillment_policy_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'make_to_stock',
  `recommendation_adi_threshold` decimal(8,4) NOT NULL DEFAULT '1.3200',
  `recommendation_concentration_pct` decimal(5,4) NOT NULL DEFAULT '0.8000',
  `recommendation_cv2_threshold` decimal(8,4) NOT NULL DEFAULT '0.4900',
  `recommendation_dormant_months` int NOT NULL DEFAULT '12',
  `recommendation_high_value_unit_cost` decimal(18,6) NOT NULL DEFAULT '50.000000',
  `recommendation_slow_mover_cogs` decimal(18,6) NOT NULL DEFAULT '5000.000000',
  `receive_calendar_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `ship_calendar_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_production_schedule_setting_account_id_key` (`account_id`),
  KEY `acct_prod_sched_setting_enabled_cron_idx` (`is_enabled`,`generation_cron`),
  KEY `account_production_schedule_setting_ship_calendar_id_idx` (`ship_calendar_id`),
  KEY `account_production_schedule_setting_receive_calendar_id_idx` (`receive_calendar_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_relation`;
CREATE TABLE `account_relation` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `owner_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `counterparty_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_relation_role_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `external_number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_edi_enabled` tinyint(1) NOT NULL DEFAULT '0',
  `stripe_customer_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `stripe_email` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `hubspot_company_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `notes` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `parent_account_relation_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `alias` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `default_billing_address_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `default_shipping_address_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `commission_status_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `freight_status_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `default_carrier_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `default_carrier_option_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `default_sales_rep_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `account_status_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `payment_term_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `account_group_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `priority_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `shipping_term_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `carrier_billing_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `carrier_billing_account` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `credit_limit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `default_lead_time_days` int DEFAULT NULL,
  `fulfillment_policy_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `receive_calendar_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_relation_owner_account_id_counterparty_account_id_ac_key` (`owner_account_id`,`counterparty_account_id`,`account_relation_role_code`),
  UNIQUE KEY `account_relation_credit_limit_id_key` (`credit_limit_id`),
  KEY `account_relation_parent_account_relation_id_idx` (`parent_account_relation_id`),
  KEY `account_relation_shipping_term_id_idx` (`shipping_term_id`),
  KEY `account_relation_account_group_id_idx` (`account_group_id`),
  KEY `account_relation_account_status_code_idx` (`account_status_code`),
  KEY `account_relation_default_carrier_option_id_idx` (`default_carrier_option_id`),
  KEY `account_relation_priority_code_idx` (`priority_code`),
  KEY `account_relation_counterparty_account_id_idx` (`counterparty_account_id`),
  KEY `account_relation_default_carrier_id_idx` (`default_carrier_id`),
  KEY `account_relation_default_sales_rep_id_idx` (`default_sales_rep_id`),
  KEY `account_relation_payment_term_id_idx` (`payment_term_id`),
  KEY `account_relation_default_billing_address_id_idx` (`default_billing_address_id`),
  KEY `account_relation_default_shipping_address_id_idx` (`default_shipping_address_id`),
  KEY `account_relation_owner_role_created_idx` (`owner_account_id`,`account_relation_role_code`,`created_at` DESC,`counterparty_account_id` DESC),
  KEY `account_relation_owner_role_status_created_idx` (`owner_account_id`,`account_relation_role_code`(50),`account_status_code`,`created_at` DESC,`counterparty_account_id` DESC),
  KEY `account_relation_owner_role_group_created_idx` (`owner_account_id`,`account_relation_role_code`(50),`account_group_id`,`created_at` DESC,`counterparty_account_id` DESC),
  KEY `account_relation_owner_role_rep_created_idx` (`owner_account_id`,`account_relation_role_code`(50),`default_sales_rep_id`,`created_at` DESC,`counterparty_account_id` DESC),
  KEY `account_relation_receive_calendar_id_idx` (`receive_calendar_id`),
  FULLTEXT KEY `account_relation_alias_idx` (`alias`),
  FULLTEXT KEY `account_relation_notes_idx` (`notes`),
  FULLTEXT KEY `account_relation_alias_notes_idx` (`alias`,`notes`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_relation_notification_preference`;
CREATE TABLE `account_relation_notification_preference` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_relation_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `recipient_account_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `notification_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `account_relation_notification_preference_account_relation_id_idx` (`account_relation_id`),
  KEY `account_relation_notification_preference_notification_type_c_idx` (`notification_type_code`),
  KEY `account_relation_notification_preference_recipient_account_u_idx` (`recipient_account_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_relation_notification_type`;
CREATE TABLE `account_relation_notification_type` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_relation_notification_type_code_key` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_relation_price_group`;
CREATE TABLE `account_relation_price_group` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_relation_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_group_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_relation_price_group_account_relation_id_account_gro_key` (`account_relation_id`,`account_group_id`),
  KEY `account_relation_price_group_account_group_id_idx` (`account_group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_relation_product_line`;
CREATE TABLE `account_relation_product_line` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_relation_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `product_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_relation_product_line_account_relation_id_product_li_key` (`account_relation_id`,`product_line_id`),
  KEY `account_relation_product_line_product_line_id_idx` (`product_line_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_status`;
CREATE TABLE `account_status` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_status_code_key` (`code`),
  FULLTEXT KEY `account_status_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `account_user`;
CREATE TABLE `account_user` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `department_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `last_used_at` datetime(3) DEFAULT NULL,
  `status_code` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `role_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_commission_eligible` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_user_user_id_account_id_key` (`user_id`,`account_id`),
  KEY `account_user_department_id_idx` (`department_id`),
  KEY `account_user_role_id_idx` (`role_id`),
  KEY `account_user_status_code_idx` (`status_code`),
  KEY `account_user_account_id_status_code_created_at_id_idx` (`account_id`,`status_code`,`created_at` DESC,`id` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `action_type`;
CREATE TABLE `action_type` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `action_type_code_key` (`code`),
  FULLTEXT KEY `action_type_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `address`;
CREATE TABLE `address` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `phone` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `email` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `is_drop_ship` tinyint(1) NOT NULL DEFAULT '0',
  `geolocation_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `receive_calendar_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `address_geolocation_id_idx` (`geolocation_id`),
  KEY `address_receive_calendar_id_idx` (`receive_calendar_id`),
  FULLTEXT KEY `address_name_idx` (`name`),
  FULLTEXT KEY `address_phone_idx` (`phone`),
  FULLTEXT KEY `address_email_idx` (`email`),
  FULLTEXT KEY `address_name_phone_idx` (`name`,`phone`),
  FULLTEXT KEY `address_name_email_idx` (`name`,`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `adjustment_type`;
CREATE TABLE `adjustment_type` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `adjustment_type_code_key` (`code`),
  FULLTEXT KEY `adjustment_type_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `agent_token_billing`;
CREATE TABLE `agent_token_billing` (
  `id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `period_start` datetime(3) NOT NULL,
  `period_end` datetime(3) NOT NULL,
  `total_input_tokens` bigint NOT NULL DEFAULT '0',
  `total_output_tokens` bigint NOT NULL DEFAULT '0',
  `total_tokens` bigint NOT NULL DEFAULT '0',
  `tokens_reported_to_stripe` bigint NOT NULL DEFAULT '0',
  `stripe_metered_item_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `run_count` int NOT NULL DEFAULT '0',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `agent_token_billing_account_id_period_start_key` (`account_id`,`period_start`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `announcement`;
CREATE TABLE `announcement` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `scope` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `category` varchar(60) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `template_key` varchar(120) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `template_params` json DEFAULT NULL,
  `title` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `body` varchar(2048) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `link_resource_type` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `link_resource_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `priority` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'normal',
  `audience` json DEFAULT NULL,
  `publish_at` datetime(3) NOT NULL,
  `expires_at` datetime(3) DEFAULT NULL,
  `created_by` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `announcement_scope_window_idx` (`scope`,`account_id`,`publish_at`,`expires_at`),
  KEY `announcement_expires_idx` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `announcement_receipt`;
CREATE TABLE `announcement_receipt` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `announcement_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `seen_at` datetime(3) DEFAULT NULL,
  `read_at` datetime(3) DEFAULT NULL,
  `dismissed_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `anrc_ann_user_uq` (`announcement_id`,`account_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `api_key`;
CREATE TABLE `api_key` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `type_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `key_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `secret_hash` binary(32) NOT NULL,
  `redacted_value` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `owner_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `role_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `last_used_at` datetime(3) DEFAULT NULL,
  `expires_at` datetime(3) DEFAULT NULL,
  `revoked_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `api_key_type_id_key` (`type_id`),
  UNIQUE KEY `api_key_key_id_key` (`key_id`),
  UNIQUE KEY `api_key_secret_hash_key` (`secret_hash`),
  KEY `api_key_owner_account_id_idx` (`owner_account_id`),
  KEY `api_key_role_id_idx` (`role_id`),
  KEY `api_key_owner_account_id_revoked_at_created_at_id_idx` (`owner_account_id`,`revoked_at`,`created_at` DESC,`id` DESC),
  KEY `api_key_owner_account_id_expires_at_created_at_id_idx` (`owner_account_id`,`expires_at`,`created_at` DESC,`id` DESC),
  KEY `api_key_owner_account_id_created_at_id_idx` (`owner_account_id`,`created_at` DESC,`id` DESC),
  FULLTEXT KEY `api_key_name_idx` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=72 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `attribute`;
CREATE TABLE `attribute` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `text` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `order` int NOT NULL DEFAULT '1',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `property_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `color_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_public` tinyint(1) NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  KEY `attribute_property_id_idx` (`property_id`),
  KEY `attribute_color_code_idx` (`color_code`),
  KEY `attribute_account_id_idx` (`account_id`),
  FULLTEXT KEY `attribute_text_idx` (`text`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `audit_event`;
CREATE TABLE `audit_event` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `type_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `actor_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `actor_type` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `identity_type` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `target_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `action` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `resource_type` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `resource_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `changes` json DEFAULT NULL,
  `metadata` json DEFAULT NULL,
  `service_name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `request_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `idempotency_key_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `source_ip` varchar(46) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `occurred_at` datetime(3) NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `root_resource_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `root_resource_type` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `audit_event_type_id_key` (`type_id`),
  KEY `audit_event_resource_idx` (`resource_type`,`resource_id`,`occurred_at`),
  KEY `audit_event_actor_idx` (`actor_id`,`occurred_at`),
  KEY `audit_event_account_idx` (`account_id`,`occurred_at` DESC,`type_id` DESC),
  KEY `audit_event_target_account_idx` (`target_account_id`,`occurred_at` DESC,`type_id` DESC),
  KEY `audit_event_occurred_at_idx` (`occurred_at`),
  KEY `audit_event_action_idx` (`action`),
  KEY `audit_event_account_id_resource_id_occurred_at_type_id_idx` (`account_id`,`resource_id`,`occurred_at` DESC,`type_id` DESC),
  KEY `audit_event_account_id_action_occurred_at_type_id_idx` (`account_id`,`action`,`occurred_at` DESC,`type_id` DESC),
  KEY `audit_event_account_id_resource_type_occurred_at_type_id_idx` (`account_id`,`resource_type`,`occurred_at` DESC,`type_id` DESC),
  KEY `audit_event_account_id_actor_id_occurred_at_type_id_idx` (`account_id`,`actor_id`,`occurred_at` DESC,`type_id` DESC),
  KEY `audit_event_root_idx` (`account_id`,`root_resource_type`,`root_resource_id`,`occurred_at`)
) ENGINE=InnoDB AUTO_INCREMENT=234 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `batch`;
CREATE TABLE `batch` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `closed_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `scanning_station_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `seconds_quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `waste_quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `production_step_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `production_run_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `scanned_at` datetime(3) DEFAULT NULL,
  `location_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `batch_quantity_id_key` (`quantity_id`),
  UNIQUE KEY `batch_seconds_quantity_id_key` (`seconds_quantity_id`),
  UNIQUE KEY `batch_waste_quantity_id_key` (`waste_quantity_id`),
  KEY `batch_location_id_idx` (`location_id`),
  KEY `batch_item_id_idx` (`item_id`),
  KEY `batch_scanning_station_id_idx` (`scanning_station_id`),
  KEY `batch_production_step_id_idx` (`production_step_id`),
  KEY `batch_account_id_scanned_at_idx` (`account_id`,`scanned_at`),
  KEY `batch_created_at_idx` (`created_at`),
  KEY `batch_production_run_id_idx` (`production_run_id`),
  KEY `batch_account_id_scanning_station_id_scanned_at_id_idx` (`account_id`,`scanning_station_id`,`scanned_at` DESC,`id` DESC),
  KEY `batch_account_run_item_idx` (`account_id`,`production_run_id`,`item_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `carrier`;
CREATE TABLE `carrier` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `shippo_carrier_account_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `account_number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `is_portal_enabled` tinyint(1) NOT NULL DEFAULT '0',
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `carrier_account_id_idx` (`account_id`),
  FULLTEXT KEY `carrier_name_idx` (`name`),
  FULLTEXT KEY `carrier_description_idx` (`description`),
  FULLTEXT KEY `carrier_name_description_idx` (`name`,`description`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `carrier_option`;
CREATE TABLE `carrier_option` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `service_level_token` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `is_portal_enabled` tinyint(1) NOT NULL DEFAULT '0',
  `is_default` tinyint(1) NOT NULL DEFAULT '0',
  `carrier_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `default_transit_days` int DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `carrier_option_carrier_id_idx` (`carrier_id`),
  KEY `carrier_option_account_id_idx` (`account_id`),
  FULLTEXT KEY `carrier_option_name_idx` (`name`),
  FULLTEXT KEY `carrier_option_code_idx` (`code`),
  FULLTEXT KEY `carrier_option_name_code_idx` (`name`,`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `carrier_transit_estimate`;
CREATE TABLE `carrier_transit_estimate` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `carrier_option_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `origin_country` varchar(2) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `origin_postal` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `dest_country` varchar(2) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `dest_postal` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `transit_days` int NOT NULL,
  `source_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `refreshed_at` datetime(3) NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `carrier_transit_estimate_lane_uq` (`account_id`,`carrier_option_id`,`origin_country`,`origin_postal`,`dest_country`,`dest_postal`),
  KEY `carrier_transit_estimate_carrier_option_id_idx` (`carrier_option_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `change_log`;
CREATE TABLE `change_log` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `action_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `responsible_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `model_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `record_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  KEY `change_log_account_id_idx` (`account_id`),
  KEY `change_log_account_id_record_id_model_type_idx` (`account_id`,`record_id`,`model_type`),
  KEY `change_log_responsible_user_id_idx` (`responsible_user_id`),
  KEY `change_log_action_type_code_idx` (`action_type_code`),
  FULLTEXT KEY `change_log_description_idx` (`description`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `color`;
CREATE TABLE `color` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `dark_hex` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `light_hex` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `color_code_key` (`code`),
  FULLTEXT KEY `color_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `consumption`;
CREATE TABLE `consumption` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `production_step_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `waste_quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `instructions` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `consumption_quantity_id_key` (`quantity_id`),
  UNIQUE KEY `consumption_waste_quantity_id_key` (`waste_quantity_id`),
  KEY `consumption_production_step_id_idx` (`production_step_id`),
  KEY `consumption_item_id_idx` (`item_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `conversation`;
CREATE TABLE `conversation` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `type` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `audience` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'internal',
  `title` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `group_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `topic_resource_type` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `topic_resource_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_by_participant_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `next_sequence` bigint NOT NULL DEFAULT '1',
  `last_message_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `last_message_at` datetime(3) DEFAULT NULL,
  `is_archived` tinyint(1) NOT NULL DEFAULT '0',
  `legal_hold` tinyint(1) NOT NULL DEFAULT '0',
  `workflow_status` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `assignee_resource_type` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `assignee_resource_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `email_inbox_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `email_external_address` varchar(320) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `metadata` json DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `conversation_account_type_lastmsg_idx` (`account_id`,`type`,`last_message_at` DESC,`id` DESC),
  KEY `conversation_topic_idx` (`account_id`,`topic_resource_type`,`topic_resource_id`,`audience`),
  KEY `conversation_email_inbox_idx` (`email_inbox_id`),
  KEY `conversation_inbox_idx` (`account_id`,`audience`,`is_archived`,`last_message_at` DESC,`id` DESC),
  KEY `conversation_group_idx` (`account_id`,`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `conversation_dm_key`;
CREATE TABLE `conversation_dm_key` (
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `dm_key` varchar(383) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `conversation_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`account_id`,`dm_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `conversation_link`;
CREATE TABLE `conversation_link` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `conversation_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `resource_type` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `resource_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_by_participant_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `cvlk_conv_resource_uq` (`conversation_id`,`resource_type`,`resource_id`),
  KEY `cvlk_resource_idx` (`account_id`,`resource_type`,`resource_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `conversation_participant`;
CREATE TABLE `conversation_participant` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `conversation_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `participant_type` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `agent_config_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `role` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'member',
  `membership` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active',
  `notifications` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'unmuted',
  `muted_until` datetime(3) DEFAULT NULL,
  `last_read_message_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `last_read_sequence` bigint NOT NULL DEFAULT '0',
  `last_read_at` datetime(3) DEFAULT NULL,
  `last_seen_at` datetime(3) DEFAULT NULL,
  `hidden_at` datetime(3) DEFAULT NULL,
  `agent_trigger_policy` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `agent_trigger_keywords` json DEFAULT NULL,
  `relation_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `joined_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `cvpt_conv_user_uq` (`conversation_id`,`account_user_id`),
  UNIQUE KEY `cvpt_conv_agent_uq` (`conversation_id`,`agent_config_id`),
  UNIQUE KEY `cvpt_conv_relation_uq` (`conversation_id`,`relation_account_id`),
  KEY `cvpt_user_membership_idx` (`account_user_id`,`membership`,`conversation_id`),
  KEY `cvpt_conv_membership_idx` (`conversation_id`,`membership`),
  KEY `cvpt_relation_membership_idx` (`relation_account_id`,`membership`,`conversation_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `dc_location`;
CREATE TABLE `dc_location` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `location` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `owner_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  KEY `dc_location_account_id_idx` (`account_id`),
  KEY `dc_location_owner_account_id_idx` (`owner_account_id`),
  FULLTEXT KEY `dc_location_location_idx` (`location`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `deleted_record`;
CREATE TABLE `deleted_record` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `deleted_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `resource_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `resource_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `data` json NOT NULL,
  PRIMARY KEY (`id`),
  KEY `deleted_record_resource_type_resource_id_idx` (`resource_type`,`resource_id`),
  KEY `deleted_record_deleted_at_idx` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `delivery`;
CREATE TABLE `delivery` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_order_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `delivery_status_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `accepted_at` datetime(3) DEFAULT NULL,
  `rejected_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `delivery_sales_order_id_idx` (`sales_order_id`),
  KEY `delivery_account_id_idx` (`account_id`),
  KEY `delivery_delivery_status_code_idx` (`delivery_status_code`),
  KEY `delivery_account_id_delivery_status_code_created_at_id_idx` (`account_id`,`delivery_status_code`,`created_at` DESC,`id` DESC),
  FULLTEXT KEY `delivery_number_idx` (`number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `delivery_line`;
CREATE TABLE `delivery_line` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `delivery_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `receiving_order_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `unit_cost_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `storage_location_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `accepted_at` datetime(3) DEFAULT NULL,
  `rejected_at` datetime(3) DEFAULT NULL,
  `lot_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `delivery_line_quantity_id_key` (`quantity_id`),
  UNIQUE KEY `delivery_line_unit_cost_id_key` (`unit_cost_id`),
  KEY `delivery_line_delivery_id_idx` (`delivery_id`),
  KEY `delivery_line_receiving_order_line_id_idx` (`receiving_order_line_id`),
  KEY `delivery_line_storage_location_id_idx` (`storage_location_id`),
  KEY `delivery_line_lot_id_idx` (`lot_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `delivery_status`;
CREATE TABLE `delivery_status` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `delivery_status_code_key` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `demand_override`;
CREATE TABLE `demand_override` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `scope_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `scope_ref_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `period_start_date` date NOT NULL,
  `period_end_date` date NOT NULL,
  `override_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `value` decimal(20,6) NOT NULL,
  `unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `reason_code` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `note` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `created_by_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `effective_from` datetime(3) NOT NULL,
  `expires_at` datetime(3) DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `demand_override_account_created_idx` (`account_id`,`created_at` DESC,`id` DESC),
  KEY `demand_override_account_scope_period_idx` (`account_id`,`scope_code`,`scope_ref_id`,`period_start_date`),
  KEY `demand_override_account_active_period_idx` (`account_id`,`is_active`,`period_start_date`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `demand_override_type`;
CREATE TABLE `demand_override_type` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `demand_override_type_code_key` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `department`;
CREATE TABLE `department` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `notes` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `location_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `labor_rate_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `department_account_id_name_key` (`account_id`,`name`),
  UNIQUE KEY `department_labor_rate_id_key` (`labor_rate_id`),
  KEY `department_location_id_idx` (`location_id`),
  FULLTEXT KEY `department_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `doc_api_key`;
CREATE TABLE `doc_api_key` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `type_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `api_key_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `owner_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `encrypted_secret` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `doc_api_key_type_id_key` (`type_id`),
  UNIQUE KEY `doc_api_key_api_key_id_key` (`api_key_id`),
  UNIQUE KEY `doc_api_key_owner_account_id_key` (`owner_account_id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `edi_run`;
CREATE TABLE `edi_run` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `completed_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `has_succeeded` tinyint(1) NOT NULL DEFAULT '0',
  `failures` json DEFAULT NULL,
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `edi_run_account_id_idx` (`account_id`),
  KEY `edi_run_account_id_has_succeeded_completed_at_id_idx` (`account_id`,`has_succeeded`,`completed_at` DESC,`id` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `edi_transmission`;
CREATE TABLE `edi_transmission` (
  `id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `direction` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL,
  `document_type` varchar(8) COLLATE utf8mb4_unicode_ci NOT NULL,
  `subject_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `subject_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `counterparty_account_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `attempts` int NOT NULL DEFAULT '0',
  `available_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `claim_owner` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `claim_token` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `claim_expires_at` datetime(3) DEFAULT NULL,
  `last_error` text COLLATE utf8mb4_unicode_ci,
  `transmitted_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `edi_transmission_subject_key` (`account_id`,`document_type`,`subject_id`),
  KEY `edi_transmission_claim_idx` (`account_id`,`status`,`available_at`),
  KEY `edi_transmission_counterparty_idx` (`counterparty_account_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `email_domain`;
CREATE TABLE `email_domain` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `domain` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `dkim_tokens` json DEFAULT NULL,
  `verified_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `email_domain_domain_uq` (`domain`),
  KEY `email_domain_account_idx` (`account_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `email_inbox`;
CREATE TABLE `email_inbox` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `email_domain_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `address` varchar(320) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `from_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active',
  `agent_config_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `agent_trigger_policy` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `agent_trigger_keywords` json DEFAULT NULL,
  `group_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `email_inbox_address_uq` (`address`),
  KEY `email_inbox_account_idx` (`account_id`),
  KEY `email_inbox_domain_idx` (`email_domain_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `email_log`;
CREATE TABLE `email_log` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `has_sent` tinyint(1) NOT NULL DEFAULT '0',
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sent_by_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `subject` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `filename` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `ses_message_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `email_log_account_id_idx` (`account_id`),
  KEY `email_log_sent_by_id_idx` (`sent_by_id`),
  KEY `email_log_ses_message_id_idx` (`ses_message_id`),
  FULLTEXT KEY `email_log_subject_idx` (`subject`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `email_message`;
CREATE TABLE `email_message` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `conversation_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `message_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `email_inbox_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `direction` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `rfc_message_id` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `in_reply_to` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `references` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `from_addr` varchar(320) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `to_addrs` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `cc_addrs` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `subject` varchar(998) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `raw_s3_key` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `ses_message_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `email_message_rfc_uq` (`rfc_message_id`),
  KEY `email_message_conv_idx` (`conversation_id`,`direction`,`created_at` DESC),
  KEY `email_message_in_reply_to_idx` (`in_reply_to`),
  KEY `email_message_message_idx` (`message_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `email_recipient`;
CREATE TABLE `email_recipient` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `email` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `email_log_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `email_recipient_email_log_id_idx` (`email_log_id`),
  FULLTEXT KEY `email_recipient_email_idx` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `error_log`;
CREATE TABLE `error_log` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `affected_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `api_location_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `public_message` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `private_message` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `error_severity_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `location_file` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `location_method` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `error_log_account_id_idx` (`account_id`),
  KEY `error_log_affected_user_id_idx` (`affected_user_id`),
  KEY `error_log_api_location_code_idx` (`api_location_code`),
  KEY `error_log_error_severity_code_idx` (`error_severity_code`),
  FULLTEXT KEY `error_log_public_message_idx` (`public_message`),
  FULLTEXT KEY `error_log_private_message_idx` (`private_message`),
  FULLTEXT KEY `error_log_name_idx` (`name`),
  FULLTEXT KEY `error_log_public_message_private_message_idx` (`public_message`,`private_message`),
  FULLTEXT KEY `error_log_name_public_message_idx` (`name`,`public_message`),
  FULLTEXT KEY `error_log_name_private_message_idx` (`name`,`private_message`),
  FULLTEXT KEY `error_log_name_public_message_private_message_idx` (`name`,`public_message`,`private_message`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `fulfillment_policy`;
CREATE TABLE `fulfillment_policy` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `fulfillment_policy_code_key` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `geolocation`;
CREATE TABLE `geolocation` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `street_line_1` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `street_line_2` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `locality` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `state` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `postal_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `country` varchar(2) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `address_details` json DEFAULT NULL,
  `google_place_id` varchar(767) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `latitude` double DEFAULT NULL,
  `longitude` double DEFAULT NULL,
  `timezone` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `geolocation_google_place_id_key` (`google_place_id`),
  FULLTEXT KEY `geolocation_street_line_1_street_line_2_locality_state_posta_idx` (`street_line_1`,`street_line_2`,`locality`,`state`,`postal_code`,`country`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `hubspot_account_user_link`;
CREATE TABLE `hubspot_account_user_link` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `hubspot_contact_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `owner_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `customer_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `hubspot_account_user_link_owner_account_id_customer_account__key` (`owner_account_id`,`customer_account_id`,`account_user_id`),
  KEY `hubspot_account_user_link_hubspot_contact_id_idx` (`hubspot_contact_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `hubspot_company_review`;
CREATE TABLE `hubspot_company_review` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `job_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `augno_customer_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `customer_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `customer_email` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `customer_url` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `candidate_matches` json DEFAULT NULL,
  `status` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `resolution` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `resolved_hubspot_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `hubspot_company_review_job_id_augno_customer_id_key` (`job_id`,`augno_customer_id`),
  KEY `hubspot_company_review_job_id_status_idx` (`job_id`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `hubspot_sync_job`;
CREATE TABLE `hubspot_sync_job` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'previewing',
  `dry_run` tinyint(1) NOT NULL DEFAULT '1',
  `golive_cutoff_at` datetime(3) DEFAULT NULL,
  `cursors` json DEFAULT NULL,
  `counts` json DEFAULT NULL,
  `last_error` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `started_at` datetime(3) DEFAULT NULL,
  `completed_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `hubspot_sync_job_account_id_status_idx` (`account_id`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `hubspot_sync_record`;
CREATE TABLE `hubspot_sync_record` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `augno_type` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `augno_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `hubspot_type` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `hubspot_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sync_hash` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `last_synced_at` datetime(3) DEFAULT NULL,
  `last_error` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `hubspot_sync_record_account_id_augno_type_augno_id_key` (`account_id`,`augno_type`,`augno_id`),
  KEY `hubspot_sync_record_account_id_hubspot_type_hubspot_id_idx` (`account_id`,`hubspot_type`,`hubspot_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `idempotency_key`;
CREATE TABLE `idempotency_key` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `type_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `idempotency_key` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `actor_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `identity_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `target_account_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `request_method` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `normalized_route` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `request_body_hash` char(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `scope_hash` char(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `request_params` json DEFAULT NULL,
  `response_code` int DEFAULT NULL,
  `response_body` json DEFAULT NULL,
  `response_headers` json DEFAULT NULL,
  `recovery_point` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `locked_at` datetime(3) DEFAULT NULL,
  `lock_owner` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `lock_expires_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `last_run_at` datetime(3) DEFAULT NULL,
  `expires_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idempotency_key_type_id_key` (`type_id`),
  UNIQUE KEY `idempotency_key_scope_hash_key` (`scope_hash`),
  KEY `idempotency_key_idempotency_key_idx` (`idempotency_key`),
  KEY `idempotency_key_target_account_id_idx` (`target_account_id`),
  KEY `idempotency_key_lock_expires_at_idx` (`lock_expires_at`),
  KEY `idempotency_key_expires_at_idx` (`expires_at`)
) ENGINE=InnoDB AUTO_INCREMENT=540 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `inventory_allocation`;
CREATE TABLE `inventory_allocation` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `inventory_receipt_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `inventory_issue_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `unit_cost_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `total_cost_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `inventory_allocation_quantity_id_key` (`quantity_id`),
  UNIQUE KEY `inventory_allocation_unit_cost_id_key` (`unit_cost_id`),
  UNIQUE KEY `inventory_allocation_total_cost_id_key` (`total_cost_id`),
  KEY `inventory_allocation_inventory_receipt_id_idx` (`inventory_receipt_id`),
  KEY `inventory_allocation_inventory_issue_id_idx` (`inventory_issue_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `inventory_change_log`;
CREATE TABLE `inventory_change_log` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `action_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `scanning_station_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `inventory_log_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `responsible_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `inventory_change_log_quantity_id_key` (`quantity_id`),
  UNIQUE KEY `inventory_change_log_inventory_log_id_key` (`inventory_log_id`),
  KEY `inventory_change_log_item_id_idx` (`item_id`),
  KEY `inventory_change_log_action_type_code_idx` (`action_type_code`),
  KEY `inventory_change_log_responsible_user_id_idx` (`responsible_user_id`),
  KEY `inventory_change_log_scanning_station_id_idx` (`scanning_station_id`),
  KEY `inventory_change_log_created_at_idx` (`created_at`),
  KEY `inventory_change_log_account_id_item_id_created_at_id_idx` (`account_id`,`item_id`,`created_at` DESC,`id` DESC),
  KEY `inventory_change_log_acct_action_type_code_created_idx` (`account_id`,`action_type_code`,`created_at` DESC,`id` DESC),
  KEY `inventory_change_log_acct_resp_user_created_idx` (`account_id`,`responsible_user_id`,`created_at` DESC,`id` DESC),
  KEY `inventory_change_log_acct_station_created_idx` (`account_id`,`scanning_station_id`,`created_at` DESC,`id` DESC),
  KEY `inventory_change_log_account_created_idx` (`account_id`,`created_at` DESC,`id` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `inventory_issue`;
CREATE TABLE `inventory_issue` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `storage_location_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `lot_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `issued_at` datetime(3) DEFAULT NULL,
  `order_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `batch_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `inventory_issue_quantity_id_key` (`quantity_id`),
  KEY `inventory_issue_order_id_idx` (`order_id`),
  KEY `inventory_issue_status_code_idx` (`status_code`),
  KEY `inventory_issue_account_id_idx` (`account_id`),
  KEY `inventory_issue_item_id_idx` (`item_id`),
  KEY `inventory_issue_storage_location_id_idx` (`storage_location_id`),
  KEY `inventory_issue_lot_id_idx` (`lot_id`),
  KEY `inventory_issue_batch_id_idx` (`batch_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `inventory_log`;
CREATE TABLE `inventory_log` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `inventory_log_quantity_id_key` (`quantity_id`),
  KEY `inventory_log_item_id_idx` (`item_id`),
  KEY `inventory_log_account_id_idx` (`account_id`),
  KEY `inventory_log_created_at_idx` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `inventory_policy_type`;
CREATE TABLE `inventory_policy_type` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `inventory_policy_type_code_key` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `inventory_receipt`;
CREATE TABLE `inventory_receipt` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `owner_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `holder_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `storage_location_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `lot_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `received_at` datetime(3) NOT NULL,
  `quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `unit_cost_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `order_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `batch_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `inventory_receipt_quantity_id_key` (`quantity_id`),
  UNIQUE KEY `inventory_receipt_unit_cost_id_key` (`unit_cost_id`),
  KEY `inventory_receipt_status_code_idx` (`status_code`),
  KEY `inventory_receipt_owner_account_id_idx` (`owner_account_id`),
  KEY `inventory_receipt_holder_account_id_idx` (`holder_account_id`),
  KEY `inventory_receipt_item_id_idx` (`item_id`),
  KEY `inventory_receipt_item_id_status_code_idx` (`item_id`,`status_code`),
  KEY `inventory_receipt_storage_location_id_idx` (`storage_location_id`),
  KEY `inventory_receipt_lot_id_idx` (`lot_id`),
  KEY `inventory_receipt_received_at_idx` (`received_at`),
  KEY `inventory_receipt_order_id_idx` (`order_id`),
  KEY `inventory_receipt_batch_id_idx` (`batch_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `inventory_receipt_status`;
CREATE TABLE `inventory_receipt_status` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `inventory_receipt_status_code_key` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `invoice`;
CREATE TABLE `invoice` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `note` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `has_been_sent` tinyint(1) NOT NULL DEFAULT '0',
  `is_paid_in_full` tinyint(1) NOT NULL DEFAULT '0',
  `sales_order_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `billing_address_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `is_over_paid` tinyint(1) NOT NULL DEFAULT '0',
  `is_edi_sent` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `invoice_sales_order_id_idx` (`sales_order_id`),
  KEY `invoice_created_at_idx` (`created_at`),
  KEY `invoice_billing_address_id_idx` (`billing_address_id`),
  KEY `invoice_account_created_idx` (`account_id`,`created_at` DESC,`id` DESC),
  FULLTEXT KEY `invoice_number_idx` (`number`),
  FULLTEXT KEY `invoice_note_idx` (`note`),
  FULLTEXT KEY `invoice_number_note_idx` (`number`,`note`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `invoice_line`;
CREATE TABLE `invoice_line` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `invoice_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_order_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `invoice_line_quantity_id_key` (`quantity_id`),
  KEY `invoice_line_invoice_id_idx` (`invoice_id`),
  KEY `invoice_line_sales_order_line_id_idx` (`sales_order_line_id`),
  KEY `invoice_line_invoice_id_sales_order_line_id_idx` (`invoice_id`,`sales_order_line_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `item`;
CREATE TABLE `item` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sku` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `notes` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) DEFAULT NULL,
  `unit_value_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `burn_rate_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `item_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `unit_cost_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `item_category_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_dirty` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `item_unit_value_id_key` (`unit_value_id`),
  UNIQUE KEY `item_burn_rate_id_key` (`burn_rate_id`),
  UNIQUE KEY `item_unit_cost_id_key` (`unit_cost_id`),
  UNIQUE KEY `item_account_id_sku_key` (`account_id`,`sku`),
  KEY `item_item_category_id_idx` (`item_category_id`),
  KEY `item_account_id_idx` (`account_id`),
  KEY `item_item_type_code_idx` (`item_type_code`),
  KEY `item_account_created_idx` (`account_id`,`created_at` DESC,`id` DESC),
  KEY `item_account_type_created_idx` (`account_id`,`item_type_code`,`created_at` DESC,`id` DESC),
  KEY `item_account_category_created_idx` (`account_id`,`item_category_id`,`created_at` DESC,`id` DESC),
  FULLTEXT KEY `item_sku_idx` (`sku`),
  FULLTEXT KEY `item_description_idx` (`description`),
  FULLTEXT KEY `item_sku_description_idx` (`sku`,`description`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `item_category`;
CREATE TABLE `item_category` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `notes` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `item_category_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `unit_group_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `item_category_account_id_name_key` (`account_id`,`name`),
  KEY `item_category_unit_group_id_idx` (`unit_group_id`),
  KEY `item_category_item_category_type_code_idx` (`item_category_type_code`),
  FULLTEXT KEY `item_category_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `item_category_type`;
CREATE TABLE `item_category_type` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `item_category_type_code_key` (`code`),
  FULLTEXT KEY `item_category_type_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `item_type`;
CREATE TABLE `item_type` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `item_type_code_key` (`code`),
  FULLTEXT KEY `item_type_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `job`;
CREATE TABLE `job` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `job_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `type` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `job_items` json NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_by` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `results` json DEFAULT NULL,
  `errors` json DEFAULT NULL,
  `error_summary` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `updated_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `started_at` datetime(6) DEFAULT NULL,
  `cancelled_at` datetime(6) DEFAULT NULL,
  `completed_at` datetime(6) DEFAULT NULL,
  `failed_at` datetime(6) DEFAULT NULL,
  `error` json DEFAULT NULL,
  `resource_type` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `job_job_id_idx` (`job_id`),
  KEY `job_account_id_created_by_idx` (`account_id`,`created_by`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `journal_posting`;
CREATE TABLE `journal_posting` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_transaction_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `posting_side_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `amount_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `journal_posting_amount_id_key` (`amount_id`),
  KEY `journal_posting_account_transaction_id_idx` (`account_transaction_id`),
  KEY `journal_posting_account_id_idx` (`account_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `label_size`;
CREATE TABLE `label_size` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `label_size_code_key` (`code`),
  FULLTEXT KEY `label_size_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `label_type`;
CREATE TABLE `label_type` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `label_type_code_key` (`code`),
  FULLTEXT KEY `label_type_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `lot`;
CREATE TABLE `lot` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `lot_number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `manufactured_at` datetime(3) DEFAULT NULL,
  `expires_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `lot_account_id_item_id_lot_number_key` (`account_id`,`item_id`,`lot_number`),
  KEY `lot_item_id_idx` (`item_id`),
  KEY `lot_expires_at_idx` (`expires_at`),
  FULLTEXT KEY `lot_lot_number_idx` (`lot_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `machine`;
CREATE TABLE `machine` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `notes` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `production_step_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `serial_number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `department_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `machine_account_id_name_key` (`account_id`,`name`),
  KEY `machine_department_id_idx` (`department_id`),
  KEY `machine_production_step_id_idx` (`production_step_id`),
  FULLTEXT KEY `machine_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `machine_downtime_event`;
CREATE TABLE `machine_downtime_event` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `machine_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `department_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `production_step_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `reason_code` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `started_at` datetime(3) NOT NULL,
  `ended_at` datetime(3) DEFAULT NULL,
  `duration_seconds` int DEFAULT NULL,
  `shift_date` date NOT NULL,
  `shift_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `production_run_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `batch_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `schedule_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `note` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `reported_by_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `source_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'manual',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `machine_downtime_account_started_idx` (`account_id`,`started_at` DESC,`id` DESC),
  KEY `machine_downtime_account_machine_started_idx` (`account_id`,`machine_id`,`started_at` DESC,`id` DESC),
  KEY `machine_downtime_account_dept_started_idx` (`account_id`,`department_id`,`started_at` DESC,`id` DESC),
  KEY `machine_downtime_account_reason_started_idx` (`account_id`,`reason_code`,`started_at` DESC,`id` DESC),
  KEY `machine_downtime_open_idx` (`account_id`,`machine_id`,`ended_at`),
  KEY `machine_downtime_account_ended_started_idx` (`account_id`,`ended_at`,`started_at` DESC,`id` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `machine_downtime_reason`;
CREATE TABLE `machine_downtime_reason` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `oee_bucket` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_planned` tinyint(1) NOT NULL DEFAULT '0',
  `sort_order` int NOT NULL DEFAULT '0',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `machine_downtime_reason_code_key` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `material`;
CREATE TABLE `material` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `order_point_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `lead_time_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `material_item_id_key` (`item_id`),
  UNIQUE KEY `material_order_point_id_key` (`order_point_id`),
  UNIQUE KEY `material_lead_time_id_key` (`lead_time_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `message`;
CREATE TABLE `message` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `conversation_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sequence` bigint DEFAULT NULL,
  `kind` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'chat',
  `status` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'sent',
  `visibility` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'internal',
  `sender_participant_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `client_message_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `body` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `subject` varchar(998) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `channel` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `preview` varchar(280) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `event_type` varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `template_key` varchar(120) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `template_params` json DEFAULT NULL,
  `link_resource_type` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `link_resource_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `agent_run_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `reply_to_message_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `source_thread_message_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `approved_by_account_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `streaming_state` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'complete',
  `scheduled_for` datetime(3) DEFAULT NULL,
  `scheduled_attempts` int NOT NULL DEFAULT '0',
  `last_error` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `locked_at` datetime(3) DEFAULT NULL,
  `lock_owner` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `edited_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `metadata` json DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `message_conv_seq_uq` (`conversation_id`,`sequence`),
  UNIQUE KEY `message_client_dedup_uq` (`conversation_id`,`client_message_id`),
  KEY `message_conv_created_idx` (`conversation_id`,`created_at` DESC,`id` DESC),
  KEY `message_reply_idx` (`conversation_id`,`reply_to_message_id`),
  KEY `message_link_idx` (`account_id`,`link_resource_type`,`link_resource_id`),
  KEY `message_event_idx` (`account_id`,`event_type`,`created_at`),
  KEY `message_agent_run_idx` (`account_id`,`agent_run_id`),
  KEY `message_purge_idx` (`deleted_at`),
  KEY `message_status_sched_idx` (`status`,`scheduled_for`),
  KEY `message_conv_status_idx` (`conversation_id`,`status`),
  KEY `message_acct_status_sched_idx` (`account_id`,`status`,`scheduled_for`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `message_attachment`;
CREATE TABLE `message_attachment` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `message_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `kind` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `s3_key` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `url` varchar(1024) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `thumbnail_url` varchar(1024) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `filename` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `content_type` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `size_bytes` bigint DEFAULT NULL,
  `resource_type` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `resource_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `s3_purged_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `mgah_message_idx` (`message_id`),
  KEY `mgah_purge_idx` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `message_inbox`;
CREATE TABLE `message_inbox` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `message_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `service_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `handler` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `message_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `request_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `parent_message_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'received',
  `attempts` int NOT NULL DEFAULT '0',
  `last_error` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `received_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `processed_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `message_inbox_handler_message_id_key` (`handler`,`message_id`),
  KEY `message_inbox_message_type_idx` (`message_type`),
  KEY `message_inbox_processed_at_idx` (`processed_at`),
  KEY `message_inbox_request_id_idx` (`request_id`),
  KEY `message_inbox_parent_message_id_idx` (`parent_message_id`)
) ENGINE=InnoDB AUTO_INCREMENT=151131 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `message_outbox`;
CREATE TABLE `message_outbox` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `message_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `service_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `message_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `destination` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `routing_key` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `headers` json DEFAULT NULL,
  `payload` json NOT NULL,
  `status` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `attempts` int NOT NULL DEFAULT '0',
  `max_attempts` int NOT NULL DEFAULT '25',
  `next_run_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `locked_at` datetime(3) DEFAULT NULL,
  `lock_owner` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `lock_expires_at` datetime(3) DEFAULT NULL,
  `last_error` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `published_at` datetime(3) DEFAULT NULL,
  `request_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `parent_message_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `message_outbox_message_id_key` (`message_id`),
  KEY `message_outbox_status_next_run_at_idx` (`status`,`next_run_at`),
  KEY `message_outbox_status_published_at_idx` (`status`,`published_at`,`id`),
  KEY `message_outbox_lock_expires_at_idx` (`lock_expires_at`),
  KEY `message_outbox_request_id_idx` (`request_id`),
  KEY `message_outbox_parent_message_id_idx` (`parent_message_id`)
) ENGINE=InnoDB AUTO_INCREMENT=151973 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `message_receipt`;
CREATE TABLE `message_receipt` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `message_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `conversation_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `delivered_at` datetime(3) DEFAULT NULL,
  `read_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `mgrc_msg_user_uq` (`message_id`,`account_user_id`),
  KEY `mgrc_user_idx` (`account_user_id`,`read_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `messaging_block`;
CREATE TABLE `messaging_block` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `blocker_account_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `blocked_account_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `msgblock_pair_uq` (`blocker_account_user_id`,`blocked_account_user_id`),
  KEY `msgblock_account_idx` (`account_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `messaging_group`;
CREATE TABLE `messaging_group` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_by_account_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `messaging_group_account_idx` (`account_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `messaging_group_member`;
CREATE TABLE `messaging_group_member` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `group_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `member_type` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `agent_config_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `mggm_group_user_uq` (`group_id`,`account_user_id`),
  UNIQUE KEY `mggm_group_agent_uq` (`group_id`,`agent_config_id`),
  KEY `mggm_group_idx` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `messaging_report`;
CREATE TABLE `messaging_report` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `conversation_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `message_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `reporter_account_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `reason` varchar(2048) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `msgreport_conv_idx` (`conversation_id`),
  KEY `msgreport_account_idx` (`account_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `notification`;
CREATE TABLE `notification` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `recipient_account_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `category` varchar(60) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `source_message_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `conversation_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `title` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `body` varchar(1024) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `template_key` varchar(120) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `template_params` json DEFAULT NULL,
  `link_resource_type` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `link_resource_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `sender_type` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `sender_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `sender_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `priority` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'normal',
  `seen_at` datetime(3) DEFAULT NULL,
  `read_at` datetime(3) DEFAULT NULL,
  `dismissed_at` datetime(3) DEFAULT NULL,
  `metadata` json DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `nf_recipient_feed_idx` (`recipient_account_user_id`,`dismissed_at`,`created_at` DESC,`id` DESC),
  KEY `nf_recipient_unseen_idx` (`recipient_account_user_id`,`seen_at`,`dismissed_at`),
  KEY `nf_source_message_idx` (`source_message_id`),
  KEY `nf_dedup_idx` (`recipient_account_user_id`,`category`,`link_resource_type`,`link_resource_id`),
  KEY `nf_recipient_category_created_idx` (`recipient_account_user_id`,`category`,`created_at` DESC,`id` DESC),
  KEY `nf_recipient_sender_created_idx` (`recipient_account_user_id`,`sender_type`,`sender_id`,`created_at` DESC,`id` DESC),
  KEY `notification_purge_idx` (`created_at`,`read_at`,`dismissed_at`),
  KEY `nf_recipient_conversation_idx` (`recipient_account_user_id`,`conversation_id`,`dismissed_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `notification_preference`;
CREATE TABLE `notification_preference` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `category` varchar(60) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `in_app_enabled` tinyint(1) NOT NULL DEFAULT '1',
  `email_enabled` tinyint(1) NOT NULL DEFAULT '1',
  `push_enabled` tinyint(1) NOT NULL DEFAULT '0',
  `digest` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'instant',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `nfpf_user_category_uq` (`account_user_id`,`category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `onboarding_status`;
CREATE TABLE `onboarding_status` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `onboarding_status_code_key` (`code`),
  FULLTEXT KEY `onboarding_status_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `operating_calendar`;
CREATE TABLE `operating_calendar` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `operating_calendar_kind_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `days_of_week` varchar(7) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '1111100',
  `cutoff_at` varchar(8) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `timezone` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `is_default` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `operating_calendar_account_id_code_key` (`account_id`,`code`),
  KEY `operating_calendar_account_kind_default_idx` (`account_id`,`operating_calendar_kind_code`,`is_default`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `operating_calendar_closure`;
CREATE TABLE `operating_calendar_closure` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `operating_calendar_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `closed_on` date NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `operating_calendar_closure_uq` (`operating_calendar_id`,`closed_on`),
  KEY `operating_calendar_closure_account_date_idx` (`account_id`,`closed_on`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `order_discount`;
CREATE TABLE `order_discount` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `percentage` double NOT NULL DEFAULT '0',
  `value` double NOT NULL DEFAULT '0',
  `discount_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `order_discount_account_id_code_key` (`account_id`,`code`),
  KEY `order_discount_discount_type_code_idx` (`discount_type_code`),
  KEY `order_discount_code_idx` (`code`),
  FULLTEXT KEY `order_discount_name_idx` (`name`),
  FULLTEXT KEY `order_discount_code_fulltext_idx` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `order_email_contact`;
CREATE TABLE `order_email_contact` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_order_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `notification_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `order_email_contact_sales_order_id_account_user_id_notificat_key` (`sales_order_id`,`account_user_id`,`notification_type_code`),
  KEY `order_email_contact_notification_type_code_idx` (`notification_type_code`),
  KEY `order_email_contact_account_user_id_idx` (`account_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `order_payment_intent`;
CREATE TABLE `order_payment_intent` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `payment_intent_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `sales_order_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  KEY `order_payment_intent_sales_order_id_idx` (`sales_order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `part`;
CREATE TABLE `part` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `part_item_id_key` (`item_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `payment_term`;
CREATE TABLE `payment_term` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `payment_term_account_id_idx` (`account_id`),
  FULLTEXT KEY `payment_term_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `permission`;
CREATE TABLE `permission` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `permission_group_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `permission_code_key` (`code`),
  KEY `permission_permission_group_code_idx` (`permission_group_code`),
  FULLTEXT KEY `permission_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `permission_group`;
CREATE TABLE `permission_group` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `permission_group_code_key` (`code`),
  FULLTEXT KEY `permission_group_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `pick`;
CREATE TABLE `pick` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_order_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `finished_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `pick_sales_order_id_key` (`sales_order_id`),
  KEY `pick_account_id_finished_at_idx` (`account_id`,`finished_at`),
  KEY `pick_account_created_idx` (`account_id`,`created_at` DESC,`id` DESC),
  FULLTEXT KEY `pick_number_idx` (`number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `pick_line`;
CREATE TABLE `pick_line` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `pick_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_order_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `packed_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pick_line_quantity_id_key` (`quantity_id`),
  KEY `pick_line_pick_id_idx` (`pick_id`),
  KEY `pick_line_sales_order_line_id_idx` (`sales_order_line_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `portal_domain`;
CREATE TABLE `portal_domain` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `domain` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `dns_records` json DEFAULT NULL,
  `verified_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `portal_domain_domain_uq` (`domain`),
  UNIQUE KEY `portal_domain_account_uq` (`account_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `portal_registration_session`;
CREATE TABLE `portal_registration_session` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `type_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `seller_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `seller_slug` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_existing_customer` tinyint(1) DEFAULT NULL,
  `step` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `customer_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `session_data` json NOT NULL,
  `completed_at` datetime(3) DEFAULT NULL,
  `abandoned_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `portal_registration_session_type_id_key` (`type_id`),
  KEY `portal_registration_session_seller_account_id_idx` (`seller_account_id`),
  KEY `portal_registration_session_user_id_seller_account_id_comple_idx` (`user_id`,`seller_account_id`,`completed_at`,`created_at` DESC,`id` DESC)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `priority`;
CREATE TABLE `priority` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `priority_code_key` (`code`),
  FULLTEXT KEY `priority_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `product`;
CREATE TABLE `product` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `product_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `product_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `is_portal_ready` tinyint(1) NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `product_item_id_key` (`item_id`),
  KEY `product_product_line_id_idx` (`product_line_id`),
  KEY `product_product_type_code_idx` (`product_type_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `product_line`;
CREATE TABLE `product_line` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `notes` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `unit_group_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_commission_exempt` tinyint(1) NOT NULL DEFAULT '0',
  `is_freight_exempt` tinyint(1) NOT NULL DEFAULT '0',
  `default_lot_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `fulfillment_policy_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `product_line_default_lot_id_key` (`default_lot_id`),
  UNIQUE KEY `product_line_account_id_name_key` (`account_id`,`name`),
  KEY `product_line_unit_group_id_idx` (`unit_group_id`),
  KEY `product_line_account_created_idx` (`account_id`,`created_at` DESC,`id` DESC),
  FULLTEXT KEY `product_line_name_idx` (`name`),
  FULLTEXT KEY `product_line_description_idx` (`description`),
  FULLTEXT KEY `product_line_name_description_idx` (`name`,`description`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `product_line_target`;
CREATE TABLE `product_line_target` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `start_date` datetime(3) NOT NULL,
  `end_date` datetime(3) NOT NULL,
  `product_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_rep_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `amount_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `product_line_target_amount_id_key` (`amount_id`),
  KEY `product_line_target_sales_rep_id_idx` (`sales_rep_id`),
  KEY `product_line_target_product_line_id_idx` (`product_line_id`),
  KEY `product_line_target_account_id_idx` (`account_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `product_type`;
CREATE TABLE `product_type` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `product_type_code_key` (`code`),
  FULLTEXT KEY `product_type_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `production`;
CREATE TABLE `production` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `production_step_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `production_quantity_id_key` (`quantity_id`),
  KEY `production_item_id_idx` (`item_id`),
  KEY `production_production_step_id_idx` (`production_step_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `production_run`;
CREATE TABLE `production_run` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `responsible_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `completed_at` datetime(3) DEFAULT NULL,
  `started_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `production_run_account_id_number_key` (`account_id`,`number`),
  KEY `production_run_responsible_user_id_idx` (`responsible_user_id`),
  KEY `production_run_account_id_idx` (`account_id`),
  FULLTEXT KEY `production_run_number_idx` (`number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `production_schedule`;
CREATE TABLE `production_schedule` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `version` int NOT NULL,
  `status_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `planning_as_of` datetime(3) NOT NULL,
  `horizon_start_date` date NOT NULL,
  `horizon_end_date` date NOT NULL,
  `horizon_weeks` int NOT NULL,
  `frozen_weeks` int NOT NULL,
  `frozen_through_date` date DEFAULT NULL,
  `demand_basis_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `generation_source_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `solver_version` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `settings_snapshot` json NOT NULL,
  `diagnostics` json NOT NULL,
  `error_message` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `frozen_line_count` int NOT NULL DEFAULT '0',
  `frozen_planned_quantity` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `generated_by_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `published_by_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `published_at` datetime(3) DEFAULT NULL,
  `superseded_by_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `production_schedule_account_version_key` (`account_id`,`version`),
  KEY `production_schedule_account_created_idx` (`account_id`,`created_at` DESC,`id` DESC),
  KEY `production_schedule_account_status_created_idx` (`account_id`,`status_code`,`created_at` DESC,`id` DESC),
  KEY `production_schedule_account_status_horizon_idx` (`account_id`,`status_code`,`horizon_start_date` DESC,`id` DESC),
  KEY `production_schedule_account_published_idx` (`account_id`,`published_at` DESC,`id` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `production_schedule_derived_line`;
CREATE TABLE `production_schedule_derived_line` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `production_schedule_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `source_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `production_step_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `department_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `week_index` int NOT NULL,
  `week_start_date` date NOT NULL,
  `quantity` decimal(65,30) NOT NULL,
  `planned_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `explosion_depth` int NOT NULL,
  `offset_weeks` int NOT NULL,
  `status_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `prod_sched_derived_sched_dept_week_idx` (`production_schedule_id`,`department_id`,`week_start_date`,`id`),
  KEY `prod_sched_derived_sched_week_step_idx` (`production_schedule_id`,`week_start_date`,`production_step_id`,`id`),
  KEY `prod_sched_derived_account_dept_week_idx` (`account_id`,`department_id`,`week_start_date`,`id`),
  KEY `prod_sched_derived_source_line_idx` (`source_line_id`),
  KEY `prod_sched_derived_sched_week_depth_idx` (`production_schedule_id`,`week_start_date`,`explosion_depth`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `production_schedule_deviation`;
CREATE TABLE `production_schedule_deviation` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `production_schedule_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `production_schedule_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `deviation_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_frozen_week` tinyint(1) NOT NULL DEFAULT '0',
  `week_index` int DEFAULT NULL,
  `machine_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `before_json` json DEFAULT NULL,
  `after_json` json DEFAULT NULL,
  `delta_quantity` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `delta_run_hours` decimal(14,4) NOT NULL DEFAULT '0.0000',
  `reason_code` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `reason_note` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `actor_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `prod_sched_dev_sched_created_idx` (`production_schedule_id`,`created_at` DESC,`id` DESC),
  KEY `prod_sched_dev_account_created_idx` (`account_id`,`created_at` DESC,`id` DESC),
  KEY `prod_sched_dev_line_idx` (`production_schedule_line_id`),
  KEY `prod_sched_dev_sched_frozen_idx` (`production_schedule_id`,`is_frozen_week`,`created_at` DESC,`id` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `production_schedule_finished_policy`;
CREATE TABLE `production_schedule_finished_policy` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `production_schedule_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sku` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `greige_item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `greige_sku` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `product_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `annual_demand` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `weekly_demand` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `sigma_weekly` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `safety_stock` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `reorder_point` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `on_hand` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `weeks_of_cover` decimal(14,4) NOT NULL DEFAULT '0.0000',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `prod_sched_fin_policy_sched_item_key` (`production_schedule_id`,`item_id`),
  KEY `prod_sched_fin_policy_sched_greige_idx` (`production_schedule_id`,`greige_item_id`,`sku`),
  KEY `prod_sched_fin_policy_account_item_idx` (`account_id`,`item_id`,`created_at` DESC,`id` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `production_schedule_finishing_line`;
CREATE TABLE `production_schedule_finishing_line` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `production_schedule_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `week_index` int NOT NULL,
  `week_start_date` date NOT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sku` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `greige_item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `greige_sku` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `department_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `production_step_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `planned_quantity` decimal(65,30) NOT NULL,
  `planned_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `planned_lots` int NOT NULL DEFAULT '0',
  `planned_lot_units` decimal(18,6) NOT NULL DEFAULT '0.000000',
  `planned_run_hours` decimal(14,4) NOT NULL DEFAULT '0.0000',
  `greige_consumed` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `firm_units` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `projected_on_hand_before` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `projected_on_hand_after` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `status_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `source_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_frozen` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `prod_sched_fin_line_sched_week_idx` (`production_schedule_id`,`week_start_date`,`sku`,`id`),
  KEY `prod_sched_fin_line_sched_item_idx` (`production_schedule_id`,`item_id`,`week_start_date`,`id`),
  KEY `prod_sched_fin_line_sched_greige_idx` (`production_schedule_id`,`greige_item_id`,`week_start_date`,`id`),
  KEY `prod_sched_fin_line_account_dept_week_idx` (`account_id`,`department_id`,`week_start_date`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `production_schedule_item_policy`;
CREATE TABLE `production_schedule_item_policy` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `production_schedule_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sku` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `production_step_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `primary_machine_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `annual_demand` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `weekly_demand` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `seconds_per_unit` decimal(14,6) NOT NULL DEFAULT '0.000000',
  `unit_cost` decimal(18,6) NOT NULL DEFAULT '0.000000',
  `setup_cost` decimal(18,6) NOT NULL DEFAULT '0.000000',
  `holding_cost` decimal(18,6) NOT NULL DEFAULT '0.000000',
  `eoq_units` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `constraint_lead_time_weeks` decimal(8,3) NOT NULL DEFAULT '0.000',
  `finish_lead_time_weeks` decimal(8,3) NOT NULL DEFAULT '0.000',
  `sigma_weekly_pooled` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `sigma_downstream_sum` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `safety_stock_primary` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `safety_stock_downstream` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `reorder_point` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `order_up_to` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `on_hand_echelon` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `on_hand_greige` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `average_greige_inventory` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `max_greige_inventory` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `weeks_of_cover` decimal(14,4) NOT NULL DEFAULT '0.0000',
  `projected_on_hand` json DEFAULT NULL,
  `annual_run_hours` decimal(14,4) NOT NULL DEFAULT '0.0000',
  `abc_class` char(1) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `was_eoq_capped` tinyint(1) NOT NULL DEFAULT '0',
  `was_capacity_starved` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `firm_demand_units` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `forecast_demand_units` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `fulfillment_policy_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'make_to_stock',
  `policy_source_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'account_default',
  PRIMARY KEY (`id`),
  UNIQUE KEY `prod_sched_item_policy_sched_item_key` (`production_schedule_id`,`item_id`),
  KEY `prod_sched_item_policy_sched_hours_idx` (`production_schedule_id`,`annual_run_hours` DESC,`id`),
  KEY `prod_sched_item_policy_account_item_idx` (`account_id`,`item_id`,`created_at` DESC,`id` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `production_schedule_item_setting`;
CREATE TABLE `production_schedule_item_setting` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_excluded` tinyint(1) NOT NULL DEFAULT '0',
  `min_lot_units` decimal(18,6) DEFAULT NULL,
  `max_lot_units` decimal(18,6) DEFAULT NULL,
  `lot_multiple_units` decimal(18,6) DEFAULT NULL,
  `fixed_lead_time_weeks` decimal(8,3) DEFAULT NULL,
  `service_level_z` decimal(6,4) DEFAULT NULL,
  `manual_eoq_units` decimal(18,6) DEFAULT NULL,
  `abc_class_override` char(1) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `preferred_machine_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `fulfillment_policy_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `prod_sched_item_setting_account_item_key` (`account_id`,`item_id`),
  KEY `prod_sched_item_setting_excluded_idx` (`account_id`,`is_excluded`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `production_schedule_line`;
CREATE TABLE `production_schedule_line` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `production_schedule_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `week_index` int NOT NULL,
  `week_start_date` date NOT NULL,
  `machine_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `production_step_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `department_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `planned_quantity` decimal(65,30) NOT NULL,
  `planned_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `planned_lots` int NOT NULL DEFAULT '0',
  `planned_run_hours` decimal(14,4) NOT NULL DEFAULT '0.0000',
  `planned_changeover_minutes` decimal(12,4) NOT NULL DEFAULT '0.0000',
  `sequence_index` int NOT NULL DEFAULT '0',
  `projected_on_hand_before` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `projected_on_hand_after` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `status_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `source_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `reason_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `is_frozen` tinyint(1) NOT NULL DEFAULT '0',
  `production_run_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `planned_lot_units` decimal(18,6) NOT NULL DEFAULT '0.000000',
  PRIMARY KEY (`id`),
  KEY `prod_sched_line_sched_week_idx` (`production_schedule_id`,`week_start_date`,`sequence_index`,`id`),
  KEY `prod_sched_line_sched_machine_week_idx` (`production_schedule_id`,`machine_id`,`week_start_date`,`id`),
  KEY `prod_sched_line_sched_item_week_idx` (`production_schedule_id`,`item_id`,`week_start_date`,`id`),
  KEY `prod_sched_line_account_week_machine_idx` (`account_id`,`week_start_date`,`machine_id`,`id`),
  KEY `prod_sched_line_account_dept_week_idx` (`account_id`,`department_id`,`week_start_date`,`id`),
  KEY `prod_sched_line_account_item_week_idx` (`account_id`,`item_id`,`week_start_date`,`id`),
  KEY `prod_sched_line_production_run_idx` (`production_run_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `production_schedule_line_order`;
CREATE TABLE `production_schedule_line_order` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `production_schedule_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `production_schedule_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_order_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_order_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `allocated_quantity` decimal(65,30) NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `prod_sched_line_order_sched_order_idx` (`production_schedule_id`,`sales_order_id`,`id`),
  KEY `prod_sched_line_order_line_idx` (`production_schedule_line_id`,`id`),
  KEY `prod_sched_line_order_account_order_idx` (`account_id`,`sales_order_id`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `production_schedule_line_status`;
CREATE TABLE `production_schedule_line_status` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `production_schedule_line_status_code_key` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `production_schedule_resource_setting`;
CREATE TABLE `production_schedule_resource_setting` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `scope_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `scope_ref_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_excluded` tinyint(1) NOT NULL DEFAULT '0',
  `is_enabled` tinyint(1) NOT NULL DEFAULT '1',
  `shifts_per_day` int DEFAULT NULL,
  `hours_per_shift` decimal(6,3) DEFAULT NULL,
  `work_days_per_week` int DEFAULT NULL,
  `capacity_headroom_pct` decimal(5,4) DEFAULT NULL,
  `efficiency_pct` decimal(5,4) DEFAULT NULL,
  `lead_time_weeks` decimal(8,3) DEFAULT NULL,
  `lead_time_offset_weeks` decimal(8,3) NOT NULL DEFAULT '0.000',
  `changeover_avg_minutes` decimal(10,4) DEFAULT NULL,
  `changeover_min_minutes` decimal(10,4) DEFAULT NULL,
  `changeover_max_minutes` decimal(10,4) DEFAULT NULL,
  `sort_order` int NOT NULL DEFAULT '0',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `prod_sched_resource_scope_key` (`account_id`,`scope_code`,`scope_ref_id`),
  KEY `prod_sched_resource_excluded_idx` (`account_id`,`is_excluded`,`scope_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `production_schedule_status`;
CREATE TABLE `production_schedule_status` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `production_schedule_status_code_key` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `production_shift`;
CREATE TABLE `production_shift` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `department_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `start_time` varchar(8) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `end_time` varchar(8) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `crosses_midnight` tinyint(1) NOT NULL DEFAULT '0',
  `days_of_week` varchar(7) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '1111100',
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `sort_order` int NOT NULL DEFAULT '0',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `production_shift_account_active_code_idx` (`account_id`,`is_active`,`code`),
  KEY `production_shift_account_dept_idx` (`account_id`,`department_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `production_step`;
CREATE TABLE `production_step` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `notes` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `scanning_station_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `allowances` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `labor_rate_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `labor_time_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `leveling_factor` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `overhead_rate_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `department_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `production_step_labor_rate_id_key` (`labor_rate_id`),
  UNIQUE KEY `production_step_labor_time_id_key` (`labor_time_id`),
  UNIQUE KEY `production_step_overhead_rate_id_key` (`overhead_rate_id`),
  UNIQUE KEY `production_step_account_id_name_key` (`account_id`,`name`),
  KEY `production_step_department_id_idx` (`department_id`),
  KEY `production_step_scanning_station_id_idx` (`scanning_station_id`),
  KEY `production_step_account_id_scanning_station_id_created_at_id_idx` (`account_id`,`scanning_station_id`,`created_at` DESC,`id` DESC),
  FULLTEXT KEY `production_step_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `property`;
CREATE TABLE `property` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `is_public` tinyint(1) NOT NULL DEFAULT '1',
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `property_account_id_name_key` (`account_id`,`name`),
  FULLTEXT KEY `property_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `quantity`;
CREATE TABLE `quantity` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `value` decimal(65,30) NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  KEY `quantity_unit_id_idx` (`unit_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `quantity_discount`;
CREATE TABLE `quantity_discount` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  KEY `quantity_discount_account_id_idx` (`account_id`),
  FULLTEXT KEY `quantity_discount_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `quantity_discount_tier`;
CREATE TABLE `quantity_discount_tier` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `discount_percentage` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `threshold` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `parent_tier_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `quantity_discount_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `quantity_discount_tier_parent_tier_id_idx` (`parent_tier_id`),
  KEY `quantity_discount_tier_quantity_discount_id_idx` (`quantity_discount_id`),
  FULLTEXT KEY `quantity_discount_tier_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `rate`;
CREATE TABLE `rate` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `value` decimal(65,30) NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `numerator_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `denominator_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  KEY `rate_numerator_unit_id_idx` (`numerator_unit_id`),
  KEY `rate_denominator_unit_id_idx` (`denominator_unit_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `receiving_order`;
CREATE TABLE `receiving_order` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `order_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `completed_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `receiving_order_order_id_key` (`order_id`),
  KEY `receiving_order_account_id_idx` (`account_id`),
  FULLTEXT KEY `receiving_order_number_idx` (`number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `receiving_order_line`;
CREATE TABLE `receiving_order_line` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `receiving_order_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_order_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `stocked_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `receiving_order_line_quantity_id_key` (`quantity_id`),
  KEY `receiving_order_line_receiving_order_id_idx` (`receiving_order_id`),
  KEY `receiving_order_line_sales_order_line_id_idx` (`sales_order_line_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `refresh_token`;
CREATE TABLE `refresh_token` (
  `token` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `expires_at` datetime(3) NOT NULL,
  `revoked_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`token`),
  KEY `refresh_token_user_id_idx` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `registration_flow`;
CREATE TABLE `registration_flow` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `registration_flow_account_id_idx` (`account_id`),
  FULLTEXT KEY `registration_flow_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `registration_queue`;
CREATE TABLE `registration_queue` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `email` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `plan_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `registration_session_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `contacted` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `registration_queue_session_id_key` (`registration_session_id`),
  KEY `registration_queue_email_idx` (`email`),
  KEY `registration_queue_plan_code_idx` (`plan_code`),
  KEY `registration_queue_contacted_idx` (`contacted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `registration_session`;
CREATE TABLE `registration_session` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `type_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `email` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `plan_code` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `step` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `verification_token` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_email_verified` tinyint(1) NOT NULL DEFAULT '0',
  `is_existing_user` tinyint(1) DEFAULT NULL,
  `user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `stripe_customer_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `stripe_checkout_session_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `stripe_subscription_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `payment_completed` tinyint(1) NOT NULL DEFAULT '0',
  `session_data` json NOT NULL,
  `completed_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `registration_session_type_id_key` (`type_id`),
  UNIQUE KEY `registration_session_verification_token_key` (`verification_token`),
  UNIQUE KEY `registration_session_account_id_key` (`account_id`),
  KEY `registration_session_user_id_idx` (`user_id`),
  KEY `registration_session_email_idx` (`email`),
  KEY `registration_session_stripe_customer_id_idx` (`stripe_customer_id`),
  KEY `registration_session_user_id_completed_at_created_at_id_idx` (`user_id`,`completed_at`,`created_at` DESC,`id` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `request_log`;
CREATE TABLE `request_log` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `method` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `host` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `path` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `normalized_route` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `query_json` json DEFAULT NULL,
  `request_body_json` json DEFAULT NULL,
  `response_body_json` json DEFAULT NULL,
  `status_code` int NOT NULL,
  `latency_us` bigint NOT NULL,
  `target_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `public_endpoint` tinyint(1) NOT NULL DEFAULT '1',
  `api_version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `actor_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `actor_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `identity_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `client_ip` binary(16) DEFAULT NULL,
  `client_ip_string` varchar(46) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `user_agent` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `referrer` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `error_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `error_message` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `occurred_at` datetime(3) NOT NULL,
  `idempotency_key_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `stack_trace` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `internal_error_message` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `trace_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `hidden` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `request_log_status_code_idx` (`status_code`),
  KEY `request_log_error_code_idx` (`error_code`),
  KEY `request_log_account_id_idx` (`account_id`),
  KEY `request_log_actor_id_idx` (`actor_id`),
  KEY `request_log_user_agent_idx` (`user_agent`),
  KEY `request_log_normalized_route_idx` (`normalized_route`),
  KEY `request_log_idempotency_key_id_idx` (`idempotency_key_id`),
  KEY `request_log_occurred_at_idx` (`occurred_at`),
  KEY `request_log_target_account_id_occurred_at_id_idx` (`target_account_id`,`occurred_at` DESC,`id` DESC),
  KEY `request_log_account_id_occurred_at_id_idx` (`account_id`,`occurred_at` DESC,`id` DESC),
  KEY `request_log_target_account_id_actor_id_occurred_at_id_idx` (`target_account_id`,`actor_id`,`occurred_at` DESC,`id` DESC),
  KEY `request_log_target_account_id_identity_type_occurred_at_id_idx` (`target_account_id`,`identity_type`,`occurred_at` DESC,`id` DESC),
  KEY `request_log_client_ip_string_occurred_at_idx` (`client_ip_string`,`occurred_at` DESC),
  KEY `request_log_target_account_id_error_code_occurred_at_id_idx` (`target_account_id`,`error_code`,`occurred_at` DESC,`id` DESC),
  KEY `request_log_target_acct_normalized_route_occurred_at_idx` (`target_account_id`,`normalized_route`(128),`occurred_at` DESC,`id` DESC),
  KEY `request_log_target_account_id_status_code_occurred_at_id_idx` (`target_account_id`,`status_code`,`occurred_at` DESC,`id` DESC),
  KEY `request_log_account_id_actor_id_occurred_at_id_idx` (`account_id`,`actor_id`,`occurred_at` DESC,`id` DESC),
  KEY `request_log_account_id_status_code_occurred_at_id_idx` (`account_id`,`status_code`,`occurred_at` DESC,`id` DESC),
  KEY `request_log_account_id_error_code_occurred_at_id_idx` (`account_id`,`error_code`,`occurred_at` DESC,`id` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `role`;
CREATE TABLE `role` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `role_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `role_account_id_idx` (`account_id`),
  KEY `role_role_type_code_idx` (`role_type_code`),
  FULLTEXT KEY `role_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `role_permission`;
CREATE TABLE `role_permission` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `create` tinyint(1) NOT NULL DEFAULT '0',
  `read` tinyint(1) NOT NULL DEFAULT '0',
  `update` tinyint(1) NOT NULL DEFAULT '0',
  `delete` tinyint(1) NOT NULL DEFAULT '0',
  `role_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `permission_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `role_permission_permission_code_idx` (`permission_code`),
  KEY `role_permission_role_id_idx` (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `role_type`;
CREATE TABLE `role_type` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `role_type_code_key` (`code`),
  FULLTEXT KEY `role_type_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `sales_order`;
CREATE TABLE `sales_order` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `billing_address_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `shipping_address_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `customer_po_number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `note` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_acknowledgment_sent` tinyint(1) NOT NULL DEFAULT '0',
  `carrier_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `carrier_option_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `carrier_billing_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `carrier_billing_account` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `priority_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_rep_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `shipping_term_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `sales_order_status_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_order_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `payment_term_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `production_run_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `order_discount_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `buyer_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `seller_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `owner_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `completed_at` datetime(3) DEFAULT NULL,
  `expired_at` datetime(3) DEFAULT NULL,
  `first_ship_at` datetime(3) DEFAULT NULL,
  `issued_at` datetime(3) DEFAULT NULL,
  `promised_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `lead_time_days` int DEFAULT NULL,
  `lead_time_source_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `ship_by_date` date DEFAULT NULL,
  `transit_days` int DEFAULT NULL,
  `transit_source_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `calendar_adjustment_days` int DEFAULT NULL,
  `lead_time_override_days` int DEFAULT NULL,
  `ship_by_cutoff_at` datetime(3) DEFAULT NULL,
  `ship_by_override_date` date DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `sales_order_owner_account_id_sales_order_type_code_number_key` (`owner_account_id`,`sales_order_type_code`,`number`),
  KEY `sales_order_owner_account_id_idx` (`owner_account_id`),
  KEY `sales_order_owner_account_id_seller_account_id_sales_order_t_idx` (`owner_account_id`,`seller_account_id`,`sales_order_type_code`),
  KEY `sales_order_seller_account_id_idx` (`seller_account_id`),
  KEY `sales_order_buyer_account_id_idx` (`buyer_account_id`),
  KEY `sales_order_billing_address_id_idx` (`billing_address_id`),
  KEY `sales_order_production_run_id_idx` (`production_run_id`),
  KEY `sales_order_shipping_address_id_idx` (`shipping_address_id`),
  KEY `sales_order_sales_order_type_code_idx` (`sales_order_type_code`),
  KEY `sales_order_order_discount_id_idx` (`order_discount_id`),
  KEY `sales_order_carrier_id_idx` (`carrier_id`),
  KEY `sales_order_carrier_option_id_idx` (`carrier_option_id`),
  KEY `sales_order_sales_rep_id_idx` (`sales_rep_id`),
  KEY `sales_order_shipping_term_id_idx` (`shipping_term_id`),
  KEY `idx_sales_order_on_number` (`number`),
  KEY `sales_order_payment_term_id_idx` (`payment_term_id`),
  KEY `sales_order_issued_at_idx` (`issued_at`),
  KEY `sales_order_completed_at_idx` (`completed_at`),
  KEY `sales_order_sales_order_status_code_idx` (`sales_order_status_code`),
  KEY `sales_order_priority_code_idx` (`priority_code`),
  KEY `sales_order_owner_status_created_idx` (`owner_account_id`,`sales_order_status_code`,`created_at` DESC,`id` DESC),
  KEY `sales_order_owner_buyer_created_idx` (`owner_account_id`,`buyer_account_id`,`created_at` DESC,`id` DESC),
  KEY `sales_order_owner_sales_rep_created_idx` (`owner_account_id`,`sales_rep_id`,`created_at` DESC,`id` DESC),
  KEY `sales_order_owner_type_status_created_idx` (`owner_account_id`,`sales_order_type_code`(50),`sales_order_status_code`,`created_at` DESC,`id` DESC),
  KEY `sales_order_owner_type_seller_created_idx` (`owner_account_id`,`sales_order_type_code`(50),`seller_account_id`,`created_at` DESC,`id` DESC),
  KEY `sales_order_owner_created_idx` (`owner_account_id`,`created_at` DESC,`id` DESC),
  KEY `sales_order_owner_customer_po_number_idx` (`owner_account_id`,`customer_po_number`),
  KEY `sales_order_owner_type_issued_idx` (`owner_account_id`,`sales_order_type_code`(50),`issued_at`),
  KEY `sales_order_owner_status_ship_by_idx` (`owner_account_id`,`sales_order_status_code`,`ship_by_date`,`id`),
  FULLTEXT KEY `sales_order_number_idx` (`number`),
  FULLTEXT KEY `sales_order_customer_po_number_idx` (`customer_po_number`),
  FULLTEXT KEY `sales_order_number_customer_po_number_idx` (`number`,`customer_po_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `sales_order_line`;
CREATE TABLE `sales_order_line` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `product_sku` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `product_description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `edi_line_item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `line_item_number` int DEFAULT NULL,
  `product_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `sales_order_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `unit_price_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `unit_cost_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `sales_order_line_quantity_id_key` (`quantity_id`),
  UNIQUE KEY `sales_order_line_unit_price_id_key` (`unit_price_id`),
  UNIQUE KEY `sales_order_line_unit_cost_id_key` (`unit_cost_id`),
  KEY `sales_order_line_product_id_idx` (`product_id`),
  KEY `sales_order_line_item_id_idx` (`item_id`),
  KEY `sales_order_line_sales_order_id_idx` (`sales_order_id`),
  FULLTEXT KEY `sales_order_line_product_sku_idx` (`product_sku`),
  FULLTEXT KEY `sales_order_line_product_description_idx` (`product_description`),
  FULLTEXT KEY `sales_order_line_product_sku_product_description_idx` (`product_sku`,`product_description`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `sales_order_status`;
CREATE TABLE `sales_order_status` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `sales_order_status_code_key` (`code`),
  FULLTEXT KEY `sales_order_status_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `sales_order_type`;
CREATE TABLE `sales_order_type` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `sales_order_type_code_key` (`code`),
  FULLTEXT KEY `sales_order_type_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `sandbox_account`;
CREATE TABLE `sandbox_account` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `type_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `owner_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `sandbox_account_type_id_key` (`type_id`),
  UNIQUE KEY `sandbox_account_account_id_key` (`account_id`),
  KEY `sandbox_account_owner_account_id_idx` (`owner_account_id`)
) ENGINE=InnoDB AUTO_INCREMENT=27 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `scanning_station`;
CREATE TABLE `scanning_station` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `notes` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `department_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `scanning_station_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `label_size_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `label_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `material_check_required` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `scanning_station_account_id_name_key` (`account_id`,`name`),
  KEY `scanning_station_department_id_idx` (`department_id`),
  KEY `scanning_station_scanning_station_type_code_idx` (`scanning_station_type_code`),
  KEY `scanning_station_label_size_code_idx` (`label_size_code`),
  KEY `scanning_station_label_type_code_idx` (`label_type_code`),
  FULLTEXT KEY `scanning_station_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `scanning_station_type`;
CREATE TABLE `scanning_station_type` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `scanning_station_type_code_key` (`code`),
  FULLTEXT KEY `scanning_station_type_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `schedule_deviation_type`;
CREATE TABLE `schedule_deviation_type` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `schedule_deviation_type_code_key` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `service_idempotency_key`;
CREATE TABLE `service_idempotency_key` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `type_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `service_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `handler` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `idempotency_key` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `actor_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `identity_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `scope_hash` char(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `response_code` int DEFAULT NULL,
  `response_body` json DEFAULT NULL,
  `recovery_point` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `locked_at` datetime(3) DEFAULT NULL,
  `lock_owner` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `lock_expires_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `last_run_at` datetime(3) DEFAULT NULL,
  `expires_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `service_idempotency_key_type_id_key` (`type_id`),
  UNIQUE KEY `service_idempotency_key_scope_hash_key` (`scope_hash`),
  KEY `service_idempotency_key_idempotency_key_idx` (`idempotency_key`),
  KEY `service_idempotency_key_lock_expires_at_idx` (`lock_expires_at`),
  KEY `service_idempotency_key_expires_at_idx` (`expires_at`)
) ENGINE=InnoDB AUTO_INCREMENT=204 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `settlement`;
CREATE TABLE `settlement` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `responsible_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `note` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  KEY `settlement_responsible_user_id_idx` (`responsible_user_id`),
  KEY `settlement_account_id_idx` (`account_id`),
  KEY `settlement_account_created_idx` (`account_id`,`created_at` DESC,`id` DESC),
  FULLTEXT KEY `settlement_number_idx` (`number`),
  FULLTEXT KEY `settlement_note_idx` (`note`),
  FULLTEXT KEY `settlement_number_note_idx` (`number`,`note`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `shipment`;
CREATE TABLE `shipment` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `bill_of_lading` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `shipped_at` datetime(3) DEFAULT NULL,
  `note` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `invoice_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `carrier_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `carrier_option_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `shipping_address_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `shipped_by_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `sales_order_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `shipment_status_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `master_tracking_number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `shipment_invoice_id_key` (`invoice_id`),
  KEY `shipment_shipping_address_id_idx` (`shipping_address_id`),
  KEY `shipment_sales_order_id_idx` (`sales_order_id`),
  KEY `shipment_carrier_id_idx` (`carrier_id`),
  KEY `shipment_carrier_option_id_idx` (`carrier_option_id`),
  KEY `shipment_shipped_by_id_idx` (`shipped_by_id`),
  KEY `shipment_shipment_status_code_idx` (`shipment_status_code`),
  KEY `shipment_account_id_idx` (`account_id`),
  KEY `shipment_created_at_idx` (`created_at`),
  KEY `shipment_account_id_shipment_status_code_created_at_id_idx` (`account_id`,`shipment_status_code`,`created_at` DESC,`id` DESC),
  KEY `shipment_account_created_idx` (`account_id`,`created_at` DESC,`id` DESC),
  FULLTEXT KEY `shipment_number_idx` (`number`),
  FULLTEXT KEY `shipment_note_idx` (`note`),
  FULLTEXT KEY `shipment_bill_of_lading_idx` (`bill_of_lading`),
  FULLTEXT KEY `shipment_master_tracking_number_idx` (`master_tracking_number`),
  FULLTEXT KEY `shipment_number_note_idx` (`number`,`note`),
  FULLTEXT KEY `shipment_number_bill_of_lading_idx` (`number`,`bill_of_lading`),
  FULLTEXT KEY `shipment_number_master_tracking_number_idx` (`number`,`master_tracking_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `shipment_line`;
CREATE TABLE `shipment_line` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `shipment_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_order_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `shipment_line_quantity_id_key` (`quantity_id`),
  KEY `shipment_line_shipment_id_idx` (`shipment_id`),
  KEY `shipment_line_sales_order_line_id_idx` (`sales_order_line_id`),
  KEY `shipment_line_created_at_idx` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `shipment_status`;
CREATE TABLE `shipment_status` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `shipment_status_code_key` (`code`),
  FULLTEXT KEY `shipment_status_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `shipping_case`;
CREATE TABLE `shipping_case` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sscc` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tracking_number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `shippo_transaction_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `shipping_label_url` varchar(2083) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `shipped_at` datetime(3) DEFAULT NULL,
  `freight_amount_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `freight_weight_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `shipment_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `carrier_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `shipping_case_freight_amount_id_key` (`freight_amount_id`),
  UNIQUE KEY `shipping_case_freight_weight_id_key` (`freight_weight_id`),
  KEY `shipping_case_shipment_id_idx` (`shipment_id`),
  KEY `shipping_case_carrier_id_idx` (`carrier_id`),
  KEY `shipping_case_account_id_idx` (`account_id`),
  KEY `shipping_case_created_at_idx` (`created_at`),
  FULLTEXT KEY `shipping_case_number_idx` (`number`),
  FULLTEXT KEY `shipping_case_sscc_idx` (`sscc`),
  FULLTEXT KEY `shipping_case_tracking_number_idx` (`tracking_number`),
  FULLTEXT KEY `shipping_case_number_sscc_idx` (`number`,`sscc`),
  FULLTEXT KEY `shipping_case_number_tracking_number_idx` (`number`,`tracking_number`),
  FULLTEXT KEY `shipping_case_sscc_tracking_number_idx` (`sscc`,`tracking_number`),
  FULLTEXT KEY `shipping_case_number_sscc_tracking_number_idx` (`number`,`sscc`,`tracking_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `shipping_term`;
CREATE TABLE `shipping_term` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_freight_exempt` tinyint(1) NOT NULL DEFAULT '0',
  `is_carrier_rate` tinyint(1) NOT NULL DEFAULT '1',
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `flat_rate_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `minimum_order_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `shipping_term_flat_rate_id_key` (`flat_rate_id`),
  UNIQUE KEY `shipping_term_minimum_order_id_key` (`minimum_order_id`),
  KEY `shipping_term_account_id_idx` (`account_id`),
  FULLTEXT KEY `shipping_term_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `shipping_term_free_shipping_rule`;
CREATE TABLE `shipping_term_free_shipping_rule` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `shipping_term_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `carrier_option_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `shipping_term_free_shipping_rule_shipping_term_id_carrier_op_key` (`shipping_term_id`,`carrier_option_id`),
  KEY `shipping_term_free_shipping_rule_carrier_option_id_idx` (`carrier_option_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `storage_location`;
CREATE TABLE `storage_location` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `address_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `storage_location_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `parent_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `storage_location_account_id_name_key` (`account_id`,`name`),
  KEY `storage_location_address_id_idx` (`address_id`),
  KEY `storage_location_storage_location_type_code_idx` (`storage_location_type_code`),
  KEY `storage_location_parent_id_idx` (`parent_id`),
  FULLTEXT KEY `storage_location_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `storage_location_type`;
CREATE TABLE `storage_location_type` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `storage_location_type_code_key` (`code`),
  FULLTEXT KEY `storage_location_type_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `stripe_event_log`;
CREATE TABLE `stripe_event_log` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `event_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `event_type` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `object_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `stripe_event_log_event_id_object_id_key` (`event_id`,`object_id`),
  KEY `stripe_event_log_event_id_idx` (`event_id`),
  KEY `stripe_event_log_event_type_idx` (`event_type`),
  KEY `stripe_event_log_object_id_idx` (`object_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `supplier_material`;
CREATE TABLE `supplier_material` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `material_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `supplier_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `supplier_part_number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `supplier_description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `owner_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `supplier_material_material_id_supplier_account_id_owner_acco_key` (`material_id`,`supplier_account_id`,`owner_account_id`),
  KEY `supplier_material_supplier_account_id_idx` (`supplier_account_id`),
  KEY `supplier_material_owner_account_id_idx` (`owner_account_id`),
  FULLTEXT KEY `supplier_material_supplier_part_number_idx` (`supplier_part_number`),
  FULLTEXT KEY `supplier_material_supplier_description_idx` (`supplier_description`),
  FULLTEXT KEY `supplier_material_supplier_part_number_supplier_description_idx` (`supplier_part_number`,`supplier_description`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `support_route`;
CREATE TABLE `support_route` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `relation_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `group_conversation_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `support_route_scope_uq` (`account_id`,`relation_account_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `sys_property`;
CREATE TABLE `sys_property` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sys_property_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `value` int NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `sys_property_account_id_sys_property_type_code_key` (`account_id`,`sys_property_type_code`),
  KEY `sys_property_account_id_idx` (`account_id`),
  KEY `sys_property_sys_property_type_code_idx` (`sys_property_type_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `sys_property_type`;
CREATE TABLE `sys_property_type` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `sys_property_type_code_key` (`code`),
  FULLTEXT KEY `sys_property_type_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `target`;
CREATE TABLE `target` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `start_date` datetime(3) NOT NULL,
  `end_date` datetime(3) NOT NULL,
  `sales_rep_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `amount_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `target_amount_id_key` (`amount_id`),
  KEY `target_sales_rep_id_idx` (`sales_rep_id`),
  KEY `target_account_id_idx` (`account_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `task_leases`;
CREATE TABLE `task_leases` (
  `name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `holder` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `acquired_at` datetime(6) NOT NULL,
  `expires_at` datetime(6) NOT NULL,
  PRIMARY KEY (`name`),
  KEY `task_leases_expires_at_idx` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `territory`;
CREATE TABLE `territory` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `start_zipcode` int DEFAULT NULL,
  `end_zipcode` int DEFAULT NULL,
  `state` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_rep_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `product_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `territory_sales_rep_id_idx` (`sales_rep_id`),
  KEY `territory_product_line_id_idx` (`product_line_id`),
  KEY `territory_account_id_idx` (`account_id`),
  FULLTEXT KEY `territory_state_idx` (`state`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `token_pack_purchase`;
CREATE TABLE `token_pack_purchase` (
  `id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `pack_id` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `token_count` bigint NOT NULL,
  `price_cents` bigint NOT NULL,
  `stripe_payment_intent_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `stripe_checkout_session_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `token_pack_purchase_account_id_idx` (`account_id`),
  KEY `token_pack_purchase_status_idx` (`status`),
  KEY `token_pack_purchase_account_id_status_idx` (`account_id`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `transaction`;
CREATE TABLE `transaction` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `customer_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `amount_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `note` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `transaction_method_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `transaction_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_fully_allocated` tinyint(1) NOT NULL DEFAULT '0',
  `responsible_user_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `adjustment_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `stripe_payment_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `funds_received_at` datetime(3) DEFAULT NULL,
  `number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `transaction_amount_id_key` (`amount_id`),
  UNIQUE KEY `transaction_stripe_payment_id_key` (`stripe_payment_id`),
  KEY `transaction_transaction_method_code_idx` (`transaction_method_code`),
  KEY `transaction_transaction_type_code_idx` (`transaction_type_code`),
  KEY `transaction_adjustment_type_code_idx` (`adjustment_type_code`),
  KEY `transaction_customer_account_id_idx` (`customer_account_id`),
  KEY `transaction_responsible_user_id_idx` (`responsible_user_id`),
  KEY `transaction_account_id_created_at_idx` (`account_id`,`created_at`),
  KEY `transaction_account_id_transaction_type_code_created_at_id_idx` (`account_id`,`transaction_type_code`,`created_at` DESC,`id` DESC),
  KEY `transaction_account_id_customer_account_id_created_at_id_idx` (`account_id`,`customer_account_id`,`created_at` DESC,`id` DESC),
  KEY `transaction_account_id_transaction_method_code_created_at_id_idx` (`account_id`,`transaction_method_code`,`created_at` DESC,`id` DESC),
  KEY `transaction_account_id_adjustment_type_code_created_at_id_idx` (`account_id`,`adjustment_type_code`,`created_at` DESC,`id` DESC),
  FULLTEXT KEY `transaction_number_idx` (`number`),
  FULLTEXT KEY `transaction_note_idx` (`note`),
  FULLTEXT KEY `transaction_number_note_idx` (`number`,`note`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `transaction_allocation`;
CREATE TABLE `transaction_allocation` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `transaction_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `amount_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `invoice_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `settlement_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `note` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `transaction_allocation_amount_id_key` (`amount_id`),
  KEY `transaction_allocation_settlement_id_idx` (`settlement_id`),
  KEY `transaction_allocation_transaction_id_idx` (`transaction_id`),
  KEY `transaction_allocation_invoice_id_idx` (`invoice_id`),
  KEY `transaction_allocation_created_at_idx` (`created_at`),
  KEY `transaction_allocation_transaction_id_created_at_id_idx` (`transaction_id`,`created_at` DESC,`id` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `transaction_method`;
CREATE TABLE `transaction_method` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `transaction_method_code_key` (`code`),
  FULLTEXT KEY `transaction_method_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `transaction_type`;
CREATE TABLE `transaction_type` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_commission_affected` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `transaction_type_code_key` (`code`),
  FULLTEXT KEY `transaction_type_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `unit`;
CREATE TABLE `unit` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `abbreviation` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `unit_dimension_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `ratio_numerator` decimal(65,30) NOT NULL DEFAULT '1.000000000000000000000000000000',
  `ratio_denominator` decimal(65,30) NOT NULL DEFAULT '1.000000000000000000000000000000',
  `offset_numerator` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `offset_denominator` decimal(65,30) NOT NULL DEFAULT '1.000000000000000000000000000000',
  `is_base_unit` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `unit_account_id_abbreviation_key` (`account_id`,`abbreviation`),
  UNIQUE KEY `unit_account_id_name_key` (`account_id`,`name`),
  KEY `unit_account_id_id_created_at_idx` (`account_id`,`id`,`created_at`),
  KEY `unit_unit_dimension_code_idx` (`unit_dimension_code`),
  FULLTEXT KEY `unit_name_idx` (`name`),
  FULLTEXT KEY `unit_abbreviation_idx` (`abbreviation`),
  FULLTEXT KEY `unit_name_abbreviation_idx` (`name`,`abbreviation`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `unit_group`;
CREATE TABLE `unit_group` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `notes` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `base_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `unit_type_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unit_group_account_id_name_key` (`account_id`,`name`),
  KEY `unit_group_base_unit_id_idx` (`base_unit_id`),
  KEY `unit_group_account_id_idx` (`account_id`),
  KEY `unit_group_unit_type_code_idx` (`unit_type_code`),
  FULLTEXT KEY `unit_group_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `unit_group_unit`;
CREATE TABLE `unit_group_unit` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `unit_group_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `discount_percentage` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `discount_fixed` decimal(65,30) NOT NULL DEFAULT '0.000000000000000000000000000000',
  `is_visible` tinyint(1) NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `unit_group_unit_unit_id_unit_group_id_key` (`unit_id`,`unit_group_id`),
  KEY `unit_group_unit_unit_group_id_idx` (`unit_group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `unit_type`;
CREATE TABLE `unit_type` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unit_type_code_key` (`code`),
  FULLTEXT KEY `unit_type_name_idx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
DROP TABLE IF EXISTS `user`;
CREATE TABLE `user` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `email` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `username` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `hashed_password` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `email_verified` datetime(3) DEFAULT NULL,
  `image_url` varchar(2083) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status_code` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `user_email_key` (`email`),
  UNIQUE KEY `user_username_key` (`username`),
  KEY `user_status_code_idx` (`status_code`),
  FULLTEXT KEY `user_name_idx` (`name`),
  FULLTEXT KEY `user_username_idx` (`username`),
  FULLTEXT KEY `user_email_idx` (`email`),
  FULLTEXT KEY `user_name_username_idx` (`name`,`username`),
  FULLTEXT KEY `user_name_email_idx` (`name`,`email`),
  FULLTEXT KEY `user_username_email_idx` (`username`,`email`),
  FULLTEXT KEY `user_name_username_email_idx` (`name`,`username`,`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


-- +goose Down

-- Disable foreign key checks to allow dropping in any order
SET FOREIGN_KEY_CHECKS=0;
DROP TABLE IF EXISTS `user`;
DROP TABLE IF EXISTS `unit_type`;
DROP TABLE IF EXISTS `unit_group_unit`;
DROP TABLE IF EXISTS `unit_group`;
DROP TABLE IF EXISTS `unit`;
DROP TABLE IF EXISTS `transaction_type`;
DROP TABLE IF EXISTS `transaction_method`;
DROP TABLE IF EXISTS `transaction_allocation`;
DROP TABLE IF EXISTS `transaction`;
DROP TABLE IF EXISTS `token_pack_purchase`;
DROP TABLE IF EXISTS `territory`;
DROP TABLE IF EXISTS `task_leases`;
DROP TABLE IF EXISTS `target`;
DROP TABLE IF EXISTS `sys_property_type`;
DROP TABLE IF EXISTS `sys_property`;
DROP TABLE IF EXISTS `support_route`;
DROP TABLE IF EXISTS `supplier_material`;
DROP TABLE IF EXISTS `stripe_event_log`;
DROP TABLE IF EXISTS `storage_location_type`;
DROP TABLE IF EXISTS `storage_location`;
DROP TABLE IF EXISTS `shipping_term_free_shipping_rule`;
DROP TABLE IF EXISTS `shipping_term`;
DROP TABLE IF EXISTS `shipping_case`;
DROP TABLE IF EXISTS `shipment_status`;
DROP TABLE IF EXISTS `shipment_line`;
DROP TABLE IF EXISTS `shipment`;
DROP TABLE IF EXISTS `settlement`;
DROP TABLE IF EXISTS `service_idempotency_key`;
DROP TABLE IF EXISTS `schedule_deviation_type`;
DROP TABLE IF EXISTS `scanning_station_type`;
DROP TABLE IF EXISTS `scanning_station`;
DROP TABLE IF EXISTS `sandbox_account`;
DROP TABLE IF EXISTS `sales_order_type`;
DROP TABLE IF EXISTS `sales_order_status`;
DROP TABLE IF EXISTS `sales_order_line`;
DROP TABLE IF EXISTS `sales_order`;
DROP TABLE IF EXISTS `role_type`;
DROP TABLE IF EXISTS `role_permission`;
DROP TABLE IF EXISTS `role`;
DROP TABLE IF EXISTS `request_log`;
DROP TABLE IF EXISTS `registration_session`;
DROP TABLE IF EXISTS `registration_queue`;
DROP TABLE IF EXISTS `registration_flow`;
DROP TABLE IF EXISTS `refresh_token`;
DROP TABLE IF EXISTS `receiving_order_line`;
DROP TABLE IF EXISTS `receiving_order`;
DROP TABLE IF EXISTS `rate`;
DROP TABLE IF EXISTS `quantity_discount_tier`;
DROP TABLE IF EXISTS `quantity_discount`;
DROP TABLE IF EXISTS `quantity`;
DROP TABLE IF EXISTS `property`;
DROP TABLE IF EXISTS `production_step`;
DROP TABLE IF EXISTS `production_shift`;
DROP TABLE IF EXISTS `production_schedule_status`;
DROP TABLE IF EXISTS `production_schedule_resource_setting`;
DROP TABLE IF EXISTS `production_schedule_line_status`;
DROP TABLE IF EXISTS `production_schedule_line_order`;
DROP TABLE IF EXISTS `production_schedule_line`;
DROP TABLE IF EXISTS `production_schedule_item_setting`;
DROP TABLE IF EXISTS `production_schedule_item_policy`;
DROP TABLE IF EXISTS `production_schedule_finishing_line`;
DROP TABLE IF EXISTS `production_schedule_finished_policy`;
DROP TABLE IF EXISTS `production_schedule_deviation`;
DROP TABLE IF EXISTS `production_schedule_derived_line`;
DROP TABLE IF EXISTS `production_schedule`;
DROP TABLE IF EXISTS `production_run`;
DROP TABLE IF EXISTS `production`;
DROP TABLE IF EXISTS `product_type`;
DROP TABLE IF EXISTS `product_line_target`;
DROP TABLE IF EXISTS `product_line`;
DROP TABLE IF EXISTS `product`;
DROP TABLE IF EXISTS `priority`;
DROP TABLE IF EXISTS `portal_registration_session`;
DROP TABLE IF EXISTS `portal_domain`;
DROP TABLE IF EXISTS `pick_line`;
DROP TABLE IF EXISTS `pick`;
DROP TABLE IF EXISTS `permission_group`;
DROP TABLE IF EXISTS `permission`;
DROP TABLE IF EXISTS `payment_term`;
DROP TABLE IF EXISTS `part`;
DROP TABLE IF EXISTS `order_payment_intent`;
DROP TABLE IF EXISTS `order_email_contact`;
DROP TABLE IF EXISTS `order_discount`;
DROP TABLE IF EXISTS `operating_calendar_closure`;
DROP TABLE IF EXISTS `operating_calendar`;
DROP TABLE IF EXISTS `onboarding_status`;
DROP TABLE IF EXISTS `notification_preference`;
DROP TABLE IF EXISTS `notification`;
DROP TABLE IF EXISTS `messaging_report`;
DROP TABLE IF EXISTS `messaging_group_member`;
DROP TABLE IF EXISTS `messaging_group`;
DROP TABLE IF EXISTS `messaging_block`;
DROP TABLE IF EXISTS `message_receipt`;
DROP TABLE IF EXISTS `message_outbox`;
DROP TABLE IF EXISTS `message_inbox`;
DROP TABLE IF EXISTS `message_attachment`;
DROP TABLE IF EXISTS `message`;
DROP TABLE IF EXISTS `material`;
DROP TABLE IF EXISTS `machine_downtime_reason`;
DROP TABLE IF EXISTS `machine_downtime_event`;
DROP TABLE IF EXISTS `machine`;
DROP TABLE IF EXISTS `lot`;
DROP TABLE IF EXISTS `label_type`;
DROP TABLE IF EXISTS `label_size`;
DROP TABLE IF EXISTS `journal_posting`;
DROP TABLE IF EXISTS `job`;
DROP TABLE IF EXISTS `item_type`;
DROP TABLE IF EXISTS `item_category_type`;
DROP TABLE IF EXISTS `item_category`;
DROP TABLE IF EXISTS `item`;
DROP TABLE IF EXISTS `invoice_line`;
DROP TABLE IF EXISTS `invoice`;
DROP TABLE IF EXISTS `inventory_receipt_status`;
DROP TABLE IF EXISTS `inventory_receipt`;
DROP TABLE IF EXISTS `inventory_policy_type`;
DROP TABLE IF EXISTS `inventory_log`;
DROP TABLE IF EXISTS `inventory_issue`;
DROP TABLE IF EXISTS `inventory_change_log`;
DROP TABLE IF EXISTS `inventory_allocation`;
DROP TABLE IF EXISTS `idempotency_key`;
DROP TABLE IF EXISTS `hubspot_sync_record`;
DROP TABLE IF EXISTS `hubspot_sync_job`;
DROP TABLE IF EXISTS `hubspot_company_review`;
DROP TABLE IF EXISTS `hubspot_account_user_link`;
DROP TABLE IF EXISTS `geolocation`;
DROP TABLE IF EXISTS `fulfillment_policy`;
DROP TABLE IF EXISTS `error_log`;
DROP TABLE IF EXISTS `email_recipient`;
DROP TABLE IF EXISTS `email_message`;
DROP TABLE IF EXISTS `email_log`;
DROP TABLE IF EXISTS `email_inbox`;
DROP TABLE IF EXISTS `email_domain`;
DROP TABLE IF EXISTS `edi_transmission`;
DROP TABLE IF EXISTS `edi_run`;
DROP TABLE IF EXISTS `doc_api_key`;
DROP TABLE IF EXISTS `department`;
DROP TABLE IF EXISTS `demand_override_type`;
DROP TABLE IF EXISTS `demand_override`;
DROP TABLE IF EXISTS `delivery_status`;
DROP TABLE IF EXISTS `delivery_line`;
DROP TABLE IF EXISTS `delivery`;
DROP TABLE IF EXISTS `deleted_record`;
DROP TABLE IF EXISTS `dc_location`;
DROP TABLE IF EXISTS `conversation_participant`;
DROP TABLE IF EXISTS `conversation_link`;
DROP TABLE IF EXISTS `conversation_dm_key`;
DROP TABLE IF EXISTS `conversation`;
DROP TABLE IF EXISTS `consumption`;
DROP TABLE IF EXISTS `color`;
DROP TABLE IF EXISTS `change_log`;
DROP TABLE IF EXISTS `carrier_transit_estimate`;
DROP TABLE IF EXISTS `carrier_option`;
DROP TABLE IF EXISTS `carrier`;
DROP TABLE IF EXISTS `batch`;
DROP TABLE IF EXISTS `audit_event`;
DROP TABLE IF EXISTS `attribute`;
DROP TABLE IF EXISTS `api_key`;
DROP TABLE IF EXISTS `announcement_receipt`;
DROP TABLE IF EXISTS `announcement`;
DROP TABLE IF EXISTS `agent_token_billing`;
DROP TABLE IF EXISTS `adjustment_type`;
DROP TABLE IF EXISTS `address`;
DROP TABLE IF EXISTS `action_type`;
DROP TABLE IF EXISTS `account_user`;
DROP TABLE IF EXISTS `account_status`;
DROP TABLE IF EXISTS `account_relation_product_line`;
DROP TABLE IF EXISTS `account_relation_price_group`;
DROP TABLE IF EXISTS `account_relation_notification_type`;
DROP TABLE IF EXISTS `account_relation_notification_preference`;
DROP TABLE IF EXISTS `account_relation`;
DROP TABLE IF EXISTS `account_production_schedule_setting`;
DROP TABLE IF EXISTS `account_price_item_category`;
DROP TABLE IF EXISTS `account_price_attribute`;
DROP TABLE IF EXISTS `account_price`;
DROP TABLE IF EXISTS `account_portal`;
DROP TABLE IF EXISTS `account_plan_limit`;
DROP TABLE IF EXISTS `account_plan_feature`;
DROP TABLE IF EXISTS `account_plan`;
DROP TABLE IF EXISTS `account_inventory_setting`;
DROP TABLE IF EXISTS `account_integration`;
DROP TABLE IF EXISTS `account_group_quantity_discount`;
DROP TABLE IF EXISTS `account_group_product_line`;
DROP TABLE IF EXISTS `account_group`;
DROP TABLE IF EXISTS `account_branding`;
DROP TABLE IF EXISTS `account_billing`;
DROP TABLE IF EXISTS `account_address`;
DROP TABLE IF EXISTS `account`;
DROP TABLE IF EXISTS `_registration_flows_shipping_terms`;
DROP TABLE IF EXISTS `_quantity_discounts_units`;
DROP TABLE IF EXISTS `_quantity_discounts_attributes`;
DROP TABLE IF EXISTS `_product_lines_quantity_discounts`;
DROP TABLE IF EXISTS `_payment_terms_registration_flows`;
DROP TABLE IF EXISTS `_parent_child_production_steps`;
DROP TABLE IF EXISTS `_item_categories_quantity_discounts`;
DROP TABLE IF EXISTS `_item_categories_properties`;
DROP TABLE IF EXISTS `_item_attributes`;
DROP TABLE IF EXISTS `_departments_picks`;
DROP TABLE IF EXISTS `_batches_machines`;
DROP TABLE IF EXISTS `_batch_flow`;
DROP TABLE IF EXISTS `InventoryIssueStatus`;
SET FOREIGN_KEY_CHECKS=1;

