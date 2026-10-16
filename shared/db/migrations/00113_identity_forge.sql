-- +goose NO TRANSACTION
-- +goose Up

-- When the member was last sent a sign-in link. With last_used_at it gives portal_access: none, invited or active.
-- The created key serves the account-user list when no status filter is given (every status is listed by default).
ALTER TABLE `account_user`
  ADD COLUMN `invited_at` datetime(3) DEFAULT NULL,
  ADD KEY `account_user_account_created_idx` (`account_id`, `created_at` DESC, `id` DESC);

-- The client an API key was made for (for example Claude Code), shown on the key and on what it does.
ALTER TABLE `api_key`
  ADD COLUMN `client_name` varchar(100) DEFAULT NULL,
  ADD COLUMN `client_source` varchar(100) DEFAULT NULL;

-- A role's access level per area ({"catalog":"read",...}) and the template it started from. role_permission keeps the
-- effective permissions every check reads; a role saved before access levels has NULL access and is described from
-- those rows.
ALTER TABLE `role`
  ADD COLUMN `access` json DEFAULT NULL,
  ADD COLUMN `template_code` varchar(32) DEFAULT NULL;

-- +goose Down

ALTER TABLE `role`
  DROP COLUMN `template_code`,
  DROP COLUMN `access`;

ALTER TABLE `api_key`
  DROP COLUMN `client_source`,
  DROP COLUMN `client_name`;

ALTER TABLE `account_user`
  DROP KEY `account_user_account_created_idx`,
  DROP COLUMN `invited_at`;
