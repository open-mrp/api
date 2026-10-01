-- +goose NO TRANSACTION
-- +goose Up
-- InnoDB scans an ascending key in both directions and a descending one only forward, so a previous
-- page (read oldest first) sorted the filter's whole range. Rebuilt ascending under the same names,
-- which the item and customer lists' FORCE INDEX name.
ALTER TABLE `item`
  DROP KEY `item_account_created_idx`,
  ADD KEY `item_account_created_idx` (`account_id`, `created_at`, `id`),
  DROP KEY `item_account_type_created_idx`,
  ADD KEY `item_account_type_created_idx` (`account_id`, `item_type_code`, `created_at`, `id`),
  DROP KEY `item_account_category_created_idx`,
  ADD KEY `item_account_category_created_idx` (`account_id`, `item_category_id`, `created_at`, `id`);

ALTER TABLE `account_relation`
  DROP KEY `account_relation_owner_role_created_idx`,
  ADD KEY `account_relation_owner_role_created_idx` (`owner_account_id`, `account_relation_role_code`, `created_at`, `counterparty_account_id`),
  DROP KEY `account_relation_owner_role_group_created_idx`,
  ADD KEY `account_relation_owner_role_group_created_idx` (`owner_account_id`, `account_relation_role_code`, `account_group_id`, `created_at`, `counterparty_account_id`),
  DROP KEY `account_relation_owner_role_rep_created_idx`,
  ADD KEY `account_relation_owner_role_rep_created_idx` (`owner_account_id`, `account_relation_role_code`, `default_sales_rep_id`, `created_at`, `counterparty_account_id`),
  DROP KEY `account_relation_owner_role_status_created_idx`,
  ADD KEY `account_relation_owner_role_status_created_idx` (`owner_account_id`, `account_relation_role_code`, `account_status_code`, `created_at`, `counterparty_account_id`);

-- The customer list's carrier, service level, and payment and shipping term filters take a few values
-- across thousands of customers; walking list order for a less common one read the account.
ALTER TABLE `account_relation`
  ADD KEY `account_relation_owner_role_carrier_created_idx` (`owner_account_id`, `account_relation_role_code`, `default_carrier_id`, `created_at`, `counterparty_account_id`),
  ADD KEY `account_relation_owner_role_carrier_option_created_idx` (`owner_account_id`, `account_relation_role_code`, `default_carrier_option_id`, `created_at`, `counterparty_account_id`),
  ADD KEY `account_relation_owner_role_payment_term_created_idx` (`owner_account_id`, `account_relation_role_code`, `payment_term_id`, `created_at`, `counterparty_account_id`),
  ADD KEY `account_relation_owner_role_shipping_term_created_idx` (`owner_account_id`, `account_relation_role_code`, `shipping_term_id`, `created_at`, `counterparty_account_id`);

-- The product list drives from a product line (with portal readiness) or from portal readiness when
-- that is narrower than the account's product items; product has no account to scope by.
ALTER TABLE `product`
  ADD KEY `product_line_portal_created_idx` (`product_line_id`, `is_portal_ready`, `created_at`, `id`),
  ADD KEY `product_portal_created_idx` (`is_portal_ready`, `created_at`, `id`);

-- +goose Down
ALTER TABLE `product`
  DROP KEY `product_line_portal_created_idx`,
  DROP KEY `product_portal_created_idx`;

ALTER TABLE `account_relation`
  DROP KEY `account_relation_owner_role_carrier_created_idx`,
  DROP KEY `account_relation_owner_role_carrier_option_created_idx`,
  DROP KEY `account_relation_owner_role_payment_term_created_idx`,
  DROP KEY `account_relation_owner_role_shipping_term_created_idx`;

ALTER TABLE `account_relation`
  DROP KEY `account_relation_owner_role_created_idx`,
  ADD KEY `account_relation_owner_role_created_idx` (`owner_account_id`, `account_relation_role_code`, `created_at` DESC, `counterparty_account_id` DESC),
  DROP KEY `account_relation_owner_role_group_created_idx`,
  ADD KEY `account_relation_owner_role_group_created_idx` (`owner_account_id`, `account_relation_role_code`, `account_group_id`, `created_at` DESC, `counterparty_account_id` DESC),
  DROP KEY `account_relation_owner_role_rep_created_idx`,
  ADD KEY `account_relation_owner_role_rep_created_idx` (`owner_account_id`, `account_relation_role_code`, `default_sales_rep_id`, `created_at` DESC, `counterparty_account_id` DESC),
  DROP KEY `account_relation_owner_role_status_created_idx`,
  ADD KEY `account_relation_owner_role_status_created_idx` (`owner_account_id`, `account_relation_role_code`, `account_status_code`, `created_at` DESC, `counterparty_account_id` DESC);

ALTER TABLE `item`
  DROP KEY `item_account_created_idx`,
  ADD KEY `item_account_created_idx` (`account_id`, `created_at` DESC, `id` DESC),
  DROP KEY `item_account_type_created_idx`,
  ADD KEY `item_account_type_created_idx` (`account_id`, `item_type_code`, `created_at` DESC, `id` DESC),
  DROP KEY `item_account_category_created_idx`,
  ADD KEY `item_account_category_created_idx` (`account_id`, `item_category_id`, `created_at` DESC, `id` DESC);
