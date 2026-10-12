-- +goose NO TRANSACTION
-- +goose Up

-- A copy of every issued sales order and its lines for the order-book reports, maintained by the sales
-- fact refresher. The reports otherwise look each line's quantity, price and cost up in quantity and rate,
-- whose rows are scattered across tables far larger than memory; here an account's open lines sit on a
-- few adjacent pages. Values are stored raw, as in their source rows, and priced at read time by the same
-- expressions, so the reports return what the live tables would.

-- open_order mirrors the sales_order columns the reports read, under the same names.
CREATE TABLE `open_order` (
  `owner_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_order_type_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_order_status_code` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `buyer_account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_rep_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `order_discount_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `shipping_address_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `customer_po_number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `issued_at` datetime(3) DEFAULT NULL,
  `completed_at` datetime(3) DEFAULT NULL,
  `first_ship_at` datetime(3) DEFAULT NULL,
  `promised_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`owner_account_id`, `id`),
  UNIQUE KEY `open_order_id_key` (`id`),
  KEY `open_order_buyer_account_id_idx` (`buyer_account_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- open_order_line is each line of an open_order with its quantity, unit price and unit cost rows inlined.
-- A source row the line names but that does not exist leaves its columns null, as the reports' left joins do.
CREATE TABLE `open_order_line` (
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_order_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `product_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `quantity_value` decimal(65,30) DEFAULT NULL,
  `quantity_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `unit_price_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `price_value` decimal(65,30) DEFAULT NULL,
  `price_numerator_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `price_denominator_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `unit_cost_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `cost_value` decimal(65,30) DEFAULT NULL,
  `cost_numerator_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `cost_denominator_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`account_id`, `sales_order_id`, `id`),
  UNIQUE KEY `open_order_line_id_key` (`id`),
  KEY `open_order_line_sales_order_id_idx` (`sales_order_id`),
  KEY `open_order_line_quantity_id_idx` (`quantity_id`),
  KEY `open_order_line_unit_price_id_idx` (`unit_price_id`),
  KEY `open_order_line_unit_cost_id_idx` (`unit_cost_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- open_order_invoiced is each invoice line billing an open_order_line, with its quantity row inlined.
-- Invoice lines whose quantity row does not exist are left out, as the reports' inner join does.
CREATE TABLE `open_order_invoiced` (
  `account_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_order_line_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `sales_order_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `invoice_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `quantity_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `quantity_value` decimal(65,30) NOT NULL,
  `quantity_unit_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`account_id`, `sales_order_line_id`, `id`),
  UNIQUE KEY `open_order_invoiced_id_key` (`id`),
  KEY `open_order_invoiced_sales_order_id_idx` (`sales_order_id`),
  KEY `open_order_invoiced_invoice_id_idx` (`invoice_id`),
  KEY `open_order_invoiced_quantity_id_idx` (`quantity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down

DROP TABLE `open_order_invoiced`;
DROP TABLE `open_order_line`;
DROP TABLE `open_order`;
