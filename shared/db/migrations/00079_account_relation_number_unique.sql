-- +goose NO TRANSACTION
-- +goose Up

ALTER TABLE `account_relation`
  ADD UNIQUE KEY `account_relation_owner_role_external_number_key` (`owner_account_id`, `account_relation_role_code`, `external_number`);

-- +goose Down

ALTER TABLE `account_relation`
  DROP KEY `account_relation_owner_role_external_number_key`;
