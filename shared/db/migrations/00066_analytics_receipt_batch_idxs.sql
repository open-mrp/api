-- +goose NO TRANSACTION
-- +goose Up

-- An account's available receipts, as owner or as holder, without reading the ones long since drawn
-- down: the inventory receipt summary and the customer portal read them by account and status. Each
-- single-column account key is the exact left prefix of its new composite, which serves every lookup
-- it did, so it goes.
ALTER TABLE `inventory_receipt`
  ADD KEY `inventory_receipt_owner_status_idx` (`owner_account_id`, `status_code`),
  ADD KEY `inventory_receipt_holder_status_idx` (`holder_account_id`, `status_code`),
  DROP KEY `inventory_receipt_owner_account_id_idx`,
  DROP KEY `inventory_receipt_holder_account_id_idx`;

-- The batches still open on the floor, without reading every batch the account has closed: the
-- open-batches report reads an account's unclosed, scanned batches.
ALTER TABLE `batch`
  ADD KEY `batch_account_closed_scanned_idx` (`account_id`, `closed_at`, `scanned_at`);

-- +goose Down

ALTER TABLE `batch`
  DROP KEY `batch_account_closed_scanned_idx`;

ALTER TABLE `inventory_receipt`
  ADD KEY `inventory_receipt_owner_account_id_idx` (`owner_account_id`),
  ADD KEY `inventory_receipt_holder_account_id_idx` (`holder_account_id`),
  DROP KEY `inventory_receipt_holder_status_idx`,
  DROP KEY `inventory_receipt_owner_status_idx`;
