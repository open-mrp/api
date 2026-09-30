-- +goose NO TRANSACTION
-- +goose Up
-- The state of every pass the sales fact refresher runs, one row per pass: 'reconcile' (the facts),
-- 'rollup' (the rollup sweep) and 'sales_buyer_summary' (the buyer summary sweep). Each pass uses the
-- cursor columns that fit its order. It replaces sales_fact_sync and sales_rollup_sync: the refresher
-- seeds a pass's row from its old table the first time it reads it, and a later release drops them.
CREATE TABLE `sales_sync` (
  `name` varchar(64) NOT NULL,
  -- 'reconcile': the next invoice, in (created_at, id) order.
  `cursor_created_at` datetime(3) NULL,
  `cursor_invoice_id` varchar(191) NULL,
  -- 'rollup': the next (account, day); 'sales_buyer_summary': the last (account, buyer) rebuilt.
  `cursor_account_id` varchar(191) NULL,
  `cursor_day` date NULL,
  `cursor_buyer_account_id` varchar(191) NULL,
  -- 'sales_buyer_summary': when it restarted the fact reconcile to fill the facts' order dates.
  `facts_since` datetime(3) NULL,
  `pass_started_at` datetime(3) NULL,
  `last_completed_at` datetime(3) NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `sales_sync`;
