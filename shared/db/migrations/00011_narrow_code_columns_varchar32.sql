-- +goose NO TRANSACTION
-- +goose Up

-- Enum-like code columns were all created as varchar(255), which costs 1020 bytes of key space per column under utf8mb4 and puts any composite index that contains one close to InnoDB's 3072-byte limit. The longest value across every one of these columns is 20 characters (checked against production data and against the enum definitions in packages/static/src/enums.ts), so varchar(32) keeps a wide margin while cutting each column's key contribution to 128 bytes. This is nine tables because deploys go out a few schema changes at a time; the remaining code columns are listed in the follow-up note at the bottom.
-- Six existing indexes carry a 50-character prefix on sales_order_type_code or account_relation_role_code, added to keep those composites under the key limit. MySQL drops a prefix that is no longer shorter than the column as part of the MODIFY, so those indexes come out of this migration indexing the full column and no rebuild is needed; the matching (length: 50) annotations are removed from the Prisma schema in the same change.
-- Nullability is restated on every line: MySQL MODIFY COLUMN replaces the whole definition, so omitting NOT NULL would silently make the column nullable. None of these columns has a default or an EXTRA clause.

ALTER TABLE `sales_order`
  MODIFY `sales_order_type_code` varchar(32) NOT NULL,
  MODIFY `sales_order_status_code` varchar(32) NOT NULL,
  MODIFY `priority_code` varchar(32) NOT NULL,
  MODIFY `carrier_billing_type` varchar(32) NULL;

ALTER TABLE `account_relation`
  MODIFY `account_relation_role_code` varchar(32) NOT NULL,
  MODIFY `priority_code` varchar(32) NOT NULL,
  MODIFY `account_status_code` varchar(32) NULL,
  MODIFY `commission_status_code` varchar(32) NULL,
  MODIFY `freight_status_code` varchar(32) NULL,
  MODIFY `fulfillment_policy_code` varchar(32) NULL,
  MODIFY `carrier_billing_type` varchar(32) NULL;

ALTER TABLE `inventory_change_log`
  MODIFY `action_type_code` varchar(32) NOT NULL;

ALTER TABLE `inventory_issue`
  MODIFY `status_code` varchar(32) NOT NULL;

ALTER TABLE `shipment`
  MODIFY `shipment_status_code` varchar(32) NOT NULL;

ALTER TABLE `change_log`
  MODIFY `action_type_code` varchar(32) NOT NULL;

ALTER TABLE `inventory_receipt`
  MODIFY `status_code` varchar(32) NOT NULL;

ALTER TABLE `transaction`
  MODIFY `transaction_type_code` varchar(32) NOT NULL,
  MODIFY `transaction_method_code` varchar(32) NULL,
  MODIFY `adjustment_type_code` varchar(32) NULL;

ALTER TABLE `item`
  MODIFY `item_type_code` varchar(32) NOT NULL;

-- Covers OrderRepo.searchSalesOrderIDs' buyer branch outright: every predicate it filters on is in the index and InnoDB appends the primary key, so selecting id needs no clustered read. This only fits now that sales_order_type_code is narrow — at varchar(255) the four columns came to 3312 bytes, over InnoDB's 3072-byte key limit. Benchmarked at 123k rows: 135ms with the current index against 54ms index-only.
-- The optimizer does not choose this on its own. The branch matches roughly a third of the table and MySQL costs a covering scan as if it were a table scan, so it picks ALL; a caller has to name the index. Prisma cannot emit hints, so this index does nothing until that query moves to Go and can FORCE INDEX.
ALTER TABLE `sales_order`
  ADD KEY `sales_order_owner_type_seller_buyer_idx` (`owner_account_id`, `sales_order_type_code`, `seller_account_id`, `buyer_account_id`);

-- Still varchar(255), deliberately left for later deploys: request_log (error_code, identity_type, actor_type) is the largest win left but the table is 1M rows / 17GB and deserves its own change; permission.code and role_permission.permission_code hold PermissionDomains values up to 31 characters and need varchar(64), not 32; change_log.model_type holds PascalCase model names that can exceed 32 (ProductionScheduleFinishingLine is 31) and is a discriminator rather than a code.

-- +goose Down

-- The Up widened six prefixed indexes to full-column (MySQL drops a prefix once it is no longer shorter
-- than the column). Widening the columns back to varchar(255) with those indexes still full-column
-- exceeds InnoDB's 3072-byte key limit, so each one is dropped, the columns are widened, and it is then
-- recreated with the 50-character prefix it originally had.
ALTER TABLE `sales_order`
  DROP KEY `sales_order_owner_type_seller_buyer_idx`,
  DROP KEY `sales_order_owner_type_issued_idx`,
  DROP KEY `sales_order_owner_type_seller_created_idx`,
  DROP KEY `sales_order_owner_type_status_created_idx`;

ALTER TABLE `account_relation`
  DROP KEY `account_relation_owner_role_group_created_idx`,
  DROP KEY `account_relation_owner_role_rep_created_idx`,
  DROP KEY `account_relation_owner_role_status_created_idx`;

ALTER TABLE `sales_order`
  MODIFY `sales_order_type_code` varchar(255) NOT NULL,
  MODIFY `sales_order_status_code` varchar(255) NOT NULL,
  MODIFY `priority_code` varchar(255) NOT NULL,
  MODIFY `carrier_billing_type` varchar(255) NULL;

ALTER TABLE `account_relation`
  MODIFY `account_relation_role_code` varchar(255) NOT NULL,
  MODIFY `priority_code` varchar(255) NOT NULL,
  MODIFY `account_status_code` varchar(255) NULL,
  MODIFY `commission_status_code` varchar(255) NULL,
  MODIFY `freight_status_code` varchar(255) NULL,
  MODIFY `fulfillment_policy_code` varchar(255) NULL,
  MODIFY `carrier_billing_type` varchar(255) NULL;

ALTER TABLE `sales_order`
  ADD KEY `sales_order_owner_type_issued_idx` (`owner_account_id`, `sales_order_type_code`(50), `issued_at`),
  ADD KEY `sales_order_owner_type_seller_created_idx` (`owner_account_id`, `sales_order_type_code`(50), `seller_account_id`, `created_at` DESC, `id` DESC),
  ADD KEY `sales_order_owner_type_status_created_idx` (`owner_account_id`, `sales_order_type_code`(50), `sales_order_status_code`, `created_at` DESC, `id` DESC);

ALTER TABLE `account_relation`
  ADD KEY `account_relation_owner_role_group_created_idx` (`owner_account_id`, `account_relation_role_code`(50), `account_group_id`, `created_at` DESC, `counterparty_account_id` DESC),
  ADD KEY `account_relation_owner_role_rep_created_idx` (`owner_account_id`, `account_relation_role_code`(50), `default_sales_rep_id`, `created_at` DESC, `counterparty_account_id` DESC),
  ADD KEY `account_relation_owner_role_status_created_idx` (`owner_account_id`, `account_relation_role_code`(50), `account_status_code`, `created_at` DESC, `counterparty_account_id` DESC);

ALTER TABLE `inventory_change_log`
  MODIFY `action_type_code` varchar(255) NOT NULL;

ALTER TABLE `inventory_issue`
  MODIFY `status_code` varchar(255) NOT NULL;

ALTER TABLE `shipment`
  MODIFY `shipment_status_code` varchar(255) NOT NULL;

ALTER TABLE `change_log`
  MODIFY `action_type_code` varchar(255) NOT NULL;

ALTER TABLE `inventory_receipt`
  MODIFY `status_code` varchar(255) NOT NULL;

ALTER TABLE `transaction`
  MODIFY `transaction_type_code` varchar(255) NOT NULL,
  MODIFY `transaction_method_code` varchar(255) NULL,
  MODIFY `adjustment_type_code` varchar(255) NULL;

ALTER TABLE `item`
  MODIFY `item_type_code` varchar(255) NOT NULL;
