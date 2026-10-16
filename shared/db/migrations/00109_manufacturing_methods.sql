-- +goose NO TRANSACTION
-- +goose Up

-- A manufacturing method is how an item is made, versioned: a draft is edited, then activated. Its operations are
-- production_step rows and its materials are consumption rows, so batches, runs and schedules that name a step keep
-- naming the same operation. Only active methods carry production rows and step links, which execution reads.
CREATE TABLE `manufacturing_method` (
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `item_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `version` int NOT NULL,
  `status_code` varchar(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `output_quantity_value` decimal(65,30) NOT NULL,
  `output_quantity_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_order_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `activated_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `manufacturing_method_item_id_version_key` (`item_id`, `version`),
  KEY `manufacturing_method_account_id_created_at_id_idx` (`account_id`, `created_at`, `id`),
  KEY `manufacturing_method_item_id_created_at_id_idx` (`item_id`, `created_at`, `id`),
  KEY `manufacturing_method_output_quantity_unit_id_idx` (`output_quantity_unit_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Operation fields on the step row. setup_time is a quantity row and machine_time a rate row, like labor_time.
ALTER TABLE `production_step`
  ADD COLUMN `manufacturing_method_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  ADD COLUMN `position` int NOT NULL DEFAULT 1,
  ADD COLUMN `sequence_code` varchar(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT 'after_previous',
  ADD COLUMN `type_code` varchar(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT 'inside',
  ADD COLUMN `setup_time_quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  ADD COLUMN `machine_time_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  ADD COLUMN `work_instructions` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  ADD KEY `production_step_manufacturing_method_id_position_idx` (`manufacturing_method_id`, `position`);

-- Material fields on the consumption row. manufacturing_method_id lists a method's materials without the step join.
ALTER TABLE `consumption`
  ADD COLUMN `manufacturing_method_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  ADD COLUMN `supply_code` varchar(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT 'stock',
  ADD KEY `consumption_manufacturing_method_id_created_at_id_idx` (`manufacturing_method_id`, `created_at`, `id`);

-- An operation's machines. A machine can serve an operation in every version of a method, so this replaces
-- machine.production_step_id, which keeps naming the machine's operation in the active method for execution.
CREATE TABLE `method_operation_machine` (
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `operation_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `machine_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`operation_id`, `machine_id`),
  KEY `method_operation_machine_machine_id_idx` (`machine_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down

DROP TABLE `method_operation_machine`;

ALTER TABLE `consumption`
  DROP KEY `consumption_manufacturing_method_id_created_at_id_idx`,
  DROP COLUMN `supply_code`,
  DROP COLUMN `manufacturing_method_id`;

ALTER TABLE `production_step`
  DROP KEY `production_step_manufacturing_method_id_position_idx`,
  DROP COLUMN `work_instructions`,
  DROP COLUMN `machine_time_id`,
  DROP COLUMN `setup_time_quantity_id`,
  DROP COLUMN `type_code`,
  DROP COLUMN `sequence_code`,
  DROP COLUMN `position`,
  DROP COLUMN `manufacturing_method_id`;

DROP TABLE `manufacturing_method`;
