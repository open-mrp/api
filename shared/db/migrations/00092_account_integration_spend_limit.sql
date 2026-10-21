-- +goose NO TRANSACTION
-- +goose Up

-- A monthly spend cap for integrations that spend on the account's behalf (OpenRouter, which serves agent runs). NULL means no cap.
ALTER TABLE `account_integration`
  ADD COLUMN `monthly_spend_limit_amount` decimal(19,4) NULL,
  ADD COLUMN `monthly_spend_limit_currency` varchar(3) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL;

-- +goose Down
ALTER TABLE `account_integration`
  DROP COLUMN `monthly_spend_limit_amount`,
  DROP COLUMN `monthly_spend_limit_currency`;
