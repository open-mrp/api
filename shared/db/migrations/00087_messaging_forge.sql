-- +goose NO TRANSACTION
-- +goose Up

-- The conversation a new one picks up from.
ALTER TABLE `conversation`
  ADD COLUMN `continues_conversation_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL;

-- Keyset pages of the small per-account messaging lists walk these keys in list order.
ALTER TABLE `email_domain`
  ADD KEY `email_domain_account_created_idx` (`account_id`, `created_at`, `id`),
  ADD KEY `email_domain_status_updated_idx` (`status`, `updated_at`, `id`);

ALTER TABLE `email_inbox`
  ADD KEY `email_inbox_account_created_idx` (`account_id`, `created_at`, `id`);

ALTER TABLE `messaging_block`
  ADD KEY `msgblock_blocker_created_idx` (`blocker_account_user_id`, `created_at`, `id`);

ALTER TABLE `messaging_group`
  ADD KEY `messaging_group_account_updated_idx` (`account_id`, `updated_at`, `id`);

ALTER TABLE `conversation_link`
  ADD KEY `cvlk_conv_created_idx` (`conversation_id`, `created_at`, `id`);

-- +goose Down

ALTER TABLE `conversation_link`
  DROP KEY `cvlk_conv_created_idx`;

ALTER TABLE `messaging_group`
  DROP KEY `messaging_group_account_updated_idx`;

ALTER TABLE `messaging_block`
  DROP KEY `msgblock_blocker_created_idx`;

ALTER TABLE `email_inbox`
  DROP KEY `email_inbox_account_created_idx`;

ALTER TABLE `email_domain`
  DROP KEY `email_domain_status_updated_idx`,
  DROP KEY `email_domain_account_created_idx`;

ALTER TABLE `conversation`
  DROP COLUMN `continues_conversation_id`;
