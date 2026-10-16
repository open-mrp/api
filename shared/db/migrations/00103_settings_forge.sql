-- +goose NO TRANSACTION
-- +goose Up

-- Whether the last call to the provider succeeded: connected, invalid_credentials or unknown (never called). The name
-- FULLTEXT index served no query (search uses LIKE on an account's few integrations).
ALTER TABLE `account_integration`
  ADD COLUMN `connection_status` varchar(32) NOT NULL DEFAULT 'unknown';

ALTER TABLE `account_integration`
  DROP INDEX `account_integration_name_idx`;

-- When the background verifier last checked a pending or securing domain, so each waits its turn between checks.
ALTER TABLE `portal_domain`
  ADD COLUMN `checked_at` datetime(3) DEFAULT NULL,
  ADD KEY `portal_domain_status_checked_idx` (`status`, `checked_at`);

-- +goose Down

ALTER TABLE `portal_domain`
  DROP KEY `portal_domain_status_checked_idx`,
  DROP COLUMN `checked_at`;

ALTER TABLE `account_integration`
  ADD FULLTEXT INDEX `account_integration_name_idx` (`name`);

ALTER TABLE `account_integration`
  DROP COLUMN `connection_status`;
