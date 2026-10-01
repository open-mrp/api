-- +goose NO TRANSACTION
-- +goose Up
-- The shipment list filters by customer, which lives on the order: through the join, a customer's
-- shipments were found by walking the whole account. buyer_account_id is denormalized from the order
-- (as pick.buyer_account_id is) so the buyer key yields one customer's shipments in list order.
-- The list keys are ascending: InnoDB scans an ascending key both ways and a descending one only
-- forward, so the previous page (read oldest first) sorted the key's whole range.
ALTER TABLE `shipment`
  ADD COLUMN `buyer_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  ADD KEY `shipment_account_buyer_created_idx` (`account_id`, `buyer_account_id`, `created_at`, `id`),
  DROP KEY `shipment_account_created_idx`,
  ADD KEY `shipment_account_created_idx` (`account_id`, `created_at`, `id`),
  DROP KEY `shipment_account_id_shipment_status_code_created_at_id_idx`,
  ADD KEY `shipment_account_id_shipment_status_code_created_at_id_idx` (`account_id`, `shipment_status_code`, `created_at`, `id`);

ALTER TABLE `pick`
  DROP KEY `pick_account_created_idx`,
  ADD KEY `pick_account_created_idx` (`account_id`, `created_at`, `id`),
  DROP KEY `pick_account_finished_created_idx`,
  ADD KEY `pick_account_finished_created_idx` (`account_id`, `finished_at`, `created_at`, `id`),
  DROP KEY `pick_account_buyer_created_idx`,
  ADD KEY `pick_account_buyer_created_idx` (`account_id`, `buyer_account_id`, `created_at`, `id`),
  DROP KEY `pick_account_buyer_finished_created_idx`,
  ADD KEY `pick_account_buyer_finished_created_idx` (`account_id`, `buyer_account_id`, `finished_at`, `created_at`, `id`);

-- The delivery list had only a status-led key, so a list with no status sorted the account.
ALTER TABLE `delivery`
  ADD KEY `delivery_account_created_idx` (`account_id`, `created_at`, `id`),
  DROP KEY `delivery_account_id_delivery_status_code_created_at_id_idx`,
  ADD KEY `delivery_account_id_delivery_status_code_created_at_id_idx` (`account_id`, `delivery_status_code`, `created_at`, `id`);

-- +goose Down
ALTER TABLE `delivery`
  DROP KEY `delivery_account_id_delivery_status_code_created_at_id_idx`,
  ADD KEY `delivery_account_id_delivery_status_code_created_at_id_idx` (`account_id`, `delivery_status_code`, `created_at` DESC, `id` DESC),
  DROP KEY `delivery_account_created_idx`;

ALTER TABLE `pick`
  DROP KEY `pick_account_buyer_finished_created_idx`,
  ADD KEY `pick_account_buyer_finished_created_idx` (`account_id`, `buyer_account_id`, `finished_at`, `created_at` DESC, `id` DESC),
  DROP KEY `pick_account_buyer_created_idx`,
  ADD KEY `pick_account_buyer_created_idx` (`account_id`, `buyer_account_id`, `created_at` DESC, `id` DESC),
  DROP KEY `pick_account_finished_created_idx`,
  ADD KEY `pick_account_finished_created_idx` (`account_id`, `finished_at`, `created_at` DESC, `id` DESC),
  DROP KEY `pick_account_created_idx`,
  ADD KEY `pick_account_created_idx` (`account_id`, `created_at` DESC, `id` DESC);

ALTER TABLE `shipment`
  DROP KEY `shipment_account_id_shipment_status_code_created_at_id_idx`,
  ADD KEY `shipment_account_id_shipment_status_code_created_at_id_idx` (`account_id`, `shipment_status_code`, `created_at` DESC, `id` DESC),
  DROP KEY `shipment_account_created_idx`,
  ADD KEY `shipment_account_created_idx` (`account_id`, `created_at` DESC, `id` DESC),
  DROP KEY `shipment_account_buyer_created_idx`,
  DROP COLUMN `buyer_account_id`;
