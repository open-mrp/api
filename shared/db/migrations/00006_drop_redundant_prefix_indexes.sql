-- +goose Up

-- Each of these single-column indexes is the exact left prefix of a wider composite (or unique) index
-- on the same table, so the composite already serves every lookup the single-column index did. Keeping
-- both just doubles the write and storage cost. Ordered largest-first by on-disk size.
ALTER TABLE `request_log` DROP KEY `request_log_account_id_idx`;
ALTER TABLE `inventory_issue` DROP KEY `inventory_issue_account_id_idx`;
ALTER TABLE `shipment` DROP KEY `shipment_account_id_idx`;
ALTER TABLE `change_log` DROP KEY `change_log_account_id_idx`;
ALTER TABLE `sales_order` DROP KEY `sales_order_owner_account_id_idx`;
ALTER TABLE `transaction_allocation` DROP KEY `transaction_allocation_transaction_id_idx`;
ALTER TABLE `inventory_receipt` DROP KEY `inventory_receipt_item_id_idx`;
ALTER TABLE `settlement` DROP KEY `settlement_account_id_idx`;
ALTER TABLE `item` DROP KEY `item_account_id_idx`;
ALTER TABLE `edi_run` DROP KEY `edi_run_account_id_idx`;

-- +goose Down

ALTER TABLE `request_log` ADD KEY `request_log_account_id_idx` (`account_id`);
ALTER TABLE `inventory_issue` ADD KEY `inventory_issue_account_id_idx` (`account_id`);
ALTER TABLE `shipment` ADD KEY `shipment_account_id_idx` (`account_id`);
ALTER TABLE `change_log` ADD KEY `change_log_account_id_idx` (`account_id`);
ALTER TABLE `sales_order` ADD KEY `sales_order_owner_account_id_idx` (`owner_account_id`);
ALTER TABLE `transaction_allocation` ADD KEY `transaction_allocation_transaction_id_idx` (`transaction_id`);
ALTER TABLE `inventory_receipt` ADD KEY `inventory_receipt_item_id_idx` (`item_id`);
ALTER TABLE `settlement` ADD KEY `settlement_account_id_idx` (`account_id`);
ALTER TABLE `item` ADD KEY `item_account_id_idx` (`account_id`);
ALTER TABLE `edi_run` ADD KEY `edi_run_account_id_idx` (`account_id`);
