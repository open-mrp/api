-- +goose NO TRANSACTION
-- +goose Up

-- Invoice search matches the start of an invoice number and of a customer's name, number or alias,
-- each a range on its own key. The invoice key carries the order so the matches' orders are read off
-- it; it supersedes invoice_account_number_idx, which deployed pods still force, so that one is
-- dropped in a later release. Account names have only an ngram index, and the relation's number and
-- alias keys are not scoped to the owner, so a prefix would read every tenant's customers.
ALTER TABLE `invoice`
  ADD KEY `invoice_account_number_order_idx` (`account_id`, `number`, `sales_order_id`);

ALTER TABLE `account`
  ADD KEY `account_name_idx` (`name`);

ALTER TABLE `account_relation`
  ADD KEY `account_relation_owner_role_external_number_idx` (`owner_account_id`, `account_relation_role_code`, `external_number`, `counterparty_account_id`),
  ADD KEY `account_relation_owner_role_alias_idx` (`owner_account_id`, `account_relation_role_code`, `alias`, `counterparty_account_id`);

-- +goose Down

ALTER TABLE `account_relation`
  DROP KEY `account_relation_owner_role_alias_idx`,
  DROP KEY `account_relation_owner_role_external_number_idx`;

ALTER TABLE `account`
  DROP KEY `account_name_idx`;

ALTER TABLE `invoice`
  DROP KEY `invoice_account_number_order_idx`;
