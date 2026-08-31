-- +goose Up

-- The outbound identity an account's merchant-facing mail is sent as. Kept separate from email_inbox
-- rather than flagged on it because sending and receiving are configured independently: an account may
-- send as orders@ without that address receiving anything, or run a bridge inbox it does not want as its
-- outbound identity. Both share email_domain, the DKIM-verified asset.
--
-- One row per account (the unique key). Should a per-document-type sender ever be wanted, that key widens
-- to (account_id, document_type) — a smaller change than walking back a per-type model nobody used.
CREATE TABLE `email_sender` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `email_domain_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `local_part` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `from_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `reply_to` varchar(320) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `email_sender_account_uq` (`account_id`),
  KEY `email_sender_domain_idx` (`email_domain_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- The envelope Return-Path domain SES was told to use for this identity. Without one the Return-Path stays
-- on amazonses.com, which does not align with the From domain, and Gmail appends "via amazonses.com" to the
-- sender line — so a fully DKIM-signed merchant email still does not read as the merchant. Nullable because
-- domains registered before this column exists have no MAIL FROM set in SES either.
ALTER TABLE `email_domain`
  ADD COLUMN `mail_from_domain` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL AFTER `dkim_tokens`;

-- +goose Down

ALTER TABLE `email_domain`
  DROP COLUMN `mail_from_domain`;

DROP TABLE `email_sender`;
