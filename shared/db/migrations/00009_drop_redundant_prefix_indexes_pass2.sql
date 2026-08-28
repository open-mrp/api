-- +goose Up

-- Second pass of the same cleanup as 00006: each single-column index below is the exact left prefix of
-- a wider composite or unique index on the same table, so the wider index already serves every lookup
-- the narrow one did and keeping both only doubles the write and storage cost.
ALTER TABLE `account_price` DROP KEY `account_price_owner_account_id_idx`;
ALTER TABLE `api_key` DROP KEY `api_key_owner_account_id_idx`;
ALTER TABLE `delivery` DROP KEY `delivery_account_id_idx`;
ALTER TABLE `messaging_group_member` DROP KEY `mggm_group_idx`;
ALTER TABLE `registration_session` DROP KEY `registration_session_user_id_idx`;
ALTER TABLE `sys_property` DROP KEY `sys_property_account_id_idx`;
ALTER TABLE `token_pack_purchase` DROP KEY `token_pack_purchase_account_id_idx`;
ALTER TABLE `unit_group` DROP KEY `unit_group_account_id_idx`;

-- +goose Down

ALTER TABLE `account_price` ADD KEY `account_price_owner_account_id_idx` (`owner_account_id`);
ALTER TABLE `api_key` ADD KEY `api_key_owner_account_id_idx` (`owner_account_id`);
ALTER TABLE `delivery` ADD KEY `delivery_account_id_idx` (`account_id`);
ALTER TABLE `messaging_group_member` ADD KEY `mggm_group_idx` (`group_id`);
ALTER TABLE `registration_session` ADD KEY `registration_session_user_id_idx` (`user_id`);
ALTER TABLE `sys_property` ADD KEY `sys_property_account_id_idx` (`account_id`);
ALTER TABLE `token_pack_purchase` ADD KEY `token_pack_purchase_account_id_idx` (`account_id`);
ALTER TABLE `unit_group` ADD KEY `unit_group_account_id_idx` (`account_id`);
