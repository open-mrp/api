-- +goose NO TRANSACTION
-- +goose Up

-- invoice_account_number_order_idx has this key as its left prefix and serves every lookup it did.
ALTER TABLE `invoice`
  DROP KEY `invoice_account_number_idx`;

-- +goose Down

ALTER TABLE `invoice`
  ADD KEY `invoice_account_number_idx` (`account_id`, `number`);
