-- 0011_orders.sql
-- Seeds orders, order lines, picks, shipments, and invoices.

-- ============================================================
-- ORDER QUANTITIES (for order lines)
-- ============================================================

-- Estimate order line quantities and rates
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedest_ln1_qty00', 10, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedest_ln2_qty00', 12, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedest_ln3_qty00', 1, 'each', NOW(), NOW());
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedest_ln1_price', 10, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedest_ln1_cost0', 8, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedest_ln2_price', 8, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedest_ln2_cost0', 8, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedest_ln3_price', 15, 'dollar', 'each', NOW(), NOW()),
    ('rt_01seedest_ln3_cost0', 8, 'dollar', 'each', NOW(), NOW());

-- Issued order line quantities and rates
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seediss_ln1_qty00', 15, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seediss_ln2_qty00', 8, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seediss_ln3_qty00', 1, 'each', NOW(), NOW());
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seediss_ln1_price', 9, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seediss_ln1_cost0', 8, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seediss_ln2_price', 9, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seediss_ln2_cost0', 8, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seediss_ln3_price', 12, 'dollar', 'each', NOW(), NOW()),
    ('rt_01seediss_ln3_cost0', 8, 'dollar', 'each', NOW(), NOW());

-- Packed order line quantities and rates
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedpck_ln1_qty00', 20, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedpck_ln2_qty00', 15, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedpck_ln3_qty00', 1, 'each', NOW(), NOW());
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedpck_ln1_price', 8.5, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedpck_ln1_cost0', 8, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedpck_ln2_price', 8.5, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedpck_ln2_cost0', 8, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedpck_ln3_price', 18, 'dollar', 'each', NOW(), NOW()),
    ('rt_01seedpck_ln3_cost0', 8, 'dollar', 'each', NOW(), NOW());

-- Fulfilled order line quantities and rates
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedful_ln1_qty00', 25, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedful_ln2_qty00', 18, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedful_ln3_qty00', 1, 'each', NOW(), NOW());
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedful_ln1_price', 9.5, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedful_ln1_cost0', 8, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedful_ln2_price', 9.5, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedful_ln2_cost0', 8, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedful_ln3_price', 20, 'dollar', 'each', NOW(), NOW()),
    ('rt_01seedful_ln3_cost0', 8, 'dollar', 'each', NOW(), NOW());

-- ============================================================
-- SALES ORDERS
-- ============================================================

-- Estimate order
INSERT IGNORE INTO sales_order (id, number, sales_order_status_code, sales_order_type_code, priority_code, carrier_id, billing_address_id, shipping_address_id, buyer_account_id, seller_account_id, owner_account_id, payment_term_id, shipping_term_id, created_at, updated_at) VALUES
    ('or_01k0a8bs2yfhev5begay245wez', 'EST-001', 'estimate', 'sales_order', 'normal', 'delivery', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pytm_01seednet3000000', 'prepaid_billed', NOW(), NOW());

-- Issued order (SeedSalesOrderID — uses delivery carrier + ground service level so
-- `?include=service_level` returns a populated stub for this seed row).
-- order_discount_id linked post-hoc (the order_discount row is inserted below).
INSERT IGNORE INTO sales_order (id, number, sales_order_status_code, sales_order_type_code, priority_code, carrier_id, carrier_option_id, billing_address_id, shipping_address_id, buyer_account_id, seller_account_id, owner_account_id, payment_term_id, shipping_term_id, issued_at, created_at, updated_at) VALUES
    ('or_01k0a8bs2yejxbsvqhrx4drkq1', 'ORD-001', 'issued', 'sales_order', 'normal', 'delivery', 'crop_01seedground000000', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pytm_01seednet3000000', 'prepaid_billed', DATE_SUB(NOW(), INTERVAL 2 DAY), NOW(), NOW());

-- Packed order
INSERT IGNORE INTO sales_order (id, number, sales_order_status_code, sales_order_type_code, priority_code, carrier_id, billing_address_id, shipping_address_id, buyer_account_id, seller_account_id, owner_account_id, payment_term_id, shipping_term_id, issued_at, created_at, updated_at) VALUES
    ('or_01k0a8bs2ye3f9p8sj0m4dfmwe', 'ORD-002', 'issued', 'sales_order', 'normal', 'delivery', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pytm_01seednet3000000', 'prepaid_billed', DATE_SUB(NOW(), INTERVAL 5 DAY), DATE_SUB(NOW(), INTERVAL 3 DAY), DATE_SUB(NOW(), INTERVAL 3 DAY));

-- Fulfilled order
INSERT IGNORE INTO sales_order (id, number, sales_order_status_code, sales_order_type_code, priority_code, carrier_id, billing_address_id, shipping_address_id, buyer_account_id, seller_account_id, owner_account_id, payment_term_id, shipping_term_id, issued_at, first_ship_at, completed_at, created_at, updated_at) VALUES
    ('or_01k0a8bs2yf909wjkd7ecd6x4z', 'ORD-003', 'fulfilled', 'sales_order', 'normal', 'will_call', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pytm_01seednet3000000', 'prepaid_billed', DATE_SUB(NOW(), INTERVAL 5 DAY), DATE_SUB(NOW(), INTERVAL 5 DAY) + INTERVAL 90 MINUTE, DATE_SUB(NOW(), INTERVAL 5 DAY) + INTERVAL 90 MINUTE, DATE_SUB(NOW(), INTERVAL 3 DAY), DATE_SUB(NOW(), INTERVAL 3 DAY));

-- ============================================================
-- SALES ORDER LINES
-- ============================================================

-- Estimate lines
INSERT IGNORE INTO sales_order_line (id, product_sku, product_description, product_id, item_id, sales_order_id, quantity_id, unit_price_id, unit_cost_id, created_at, updated_at) VALUES
    ('orln_01seedest_ln1_0000', 'SCK-006', 'Large beige sock', 'pd_01k0a65nx5e67rd1rahv4tdnrp', 'it_01k0a7100ae85v16mmxx5gx2w3', 'or_01k0a8bs2yfhev5begay245wez', 'qu_01seedest_ln1_qty00', 'rt_01seedest_ln1_price', 'rt_01seedest_ln1_cost0', NOW(), NOW()),
    ('orln_01seedest_ln2_0000', 'SCK-004', 'Large black sock', 'pd_01k0a65nx5eeavcs322b06pgr8', 'it_01k0a7100af709nn7sgg8tbxte', 'or_01k0a8bs2yfhev5begay245wez', 'qu_01seedest_ln2_qty00', 'rt_01seedest_ln2_price', 'rt_01seedest_ln2_cost0', NOW(), NOW()),
    ('orln_01seedest_ln3_0000', 'Freight', 'Charges for shipping', 'pd_01k0a65nx5fj1bxedew2jvjpwz', 'it_01k0a71009fc5szdjy8mn2nzq5', 'or_01k0a8bs2yfhev5begay245wez', 'qu_01seedest_ln3_qty00', 'rt_01seedest_ln3_price', 'rt_01seedest_ln3_cost0', NOW(), NOW());

-- Issued order lines
INSERT IGNORE INTO sales_order_line (id, product_sku, product_description, product_id, item_id, sales_order_id, quantity_id, unit_price_id, unit_cost_id, created_at, updated_at) VALUES
    ('orln_01seediss_ln1_0000', 'SCK-001', 'Small white sock', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'it_01k0a7100aeysrs9vxpeq14yxj', 'or_01k0a8bs2yejxbsvqhrx4drkq1', 'qu_01seediss_ln1_qty00', 'rt_01seediss_ln1_price', 'rt_01seediss_ln1_cost0', NOW(), NOW()),
    ('orln_01seediss_ln2_0000', 'SCK-002', 'Large white sock', 'pd_01k0a65nx5e3haz2fgfm34hmcz', 'it_01k0a7100aedgv8416p4p2v9ks', 'or_01k0a8bs2yejxbsvqhrx4drkq1', 'qu_01seediss_ln2_qty00', 'rt_01seediss_ln2_price', 'rt_01seediss_ln2_cost0', NOW(), NOW()),
    ('orln_01seediss_ln3_0000', 'Freight', 'Charges for shipping', 'pd_01k0a65nx5fj1bxedew2jvjpwz', 'it_01k0a71009fc5szdjy8mn2nzq5', 'or_01k0a8bs2yejxbsvqhrx4drkq1', 'qu_01seediss_ln3_qty00', 'rt_01seediss_ln3_price', 'rt_01seediss_ln3_cost0', NOW(), NOW());

-- Email contacts for ORD-001 (SeedSalesOrderID): one invoice recipient (John Doe /
-- dane@augno.com) and one acknowledgement recipient (Sarah Martinez /
-- smartinez@augno.com), exercising the ?include=contacts expansion.
INSERT IGNORE INTO order_email_contact (id, sales_order_id, account_user_id, notification_type_code, created_at, updated_at) VALUES
    ('oec_01seediss_invoice00', 'or_01k0a8bs2yejxbsvqhrx4drkq1', 'acus_s83fjhyfmqen', 'invoice', NOW(), NOW()),
    ('oec_01seediss_acknowled', 'or_01k0a8bs2yejxbsvqhrx4drkq1', 'acus_ubdx4zebgl6p', 'order_acknowledgement', NOW(), NOW());

-- Packed order lines
INSERT IGNORE INTO sales_order_line (id, product_sku, product_description, product_id, item_id, sales_order_id, quantity_id, unit_price_id, unit_cost_id, created_at, updated_at) VALUES
    ('orln_01seedpck_ln1_0000', 'SCK-003', 'Small black sock', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'it_01k0a7100afdnr1b41917qs27k', 'or_01k0a8bs2ye3f9p8sj0m4dfmwe', 'qu_01seedpck_ln1_qty00', 'rt_01seedpck_ln1_price', 'rt_01seedpck_ln1_cost0', NOW(), NOW()),
    ('orln_01seedpck_ln2_0000', 'SCK-005', 'Small beige sock', 'pd_01k0a65nx5fwmt17sqp317ekyr', 'it_01k0a7100aef2997gw0t7nxd9d', 'or_01k0a8bs2ye3f9p8sj0m4dfmwe', 'qu_01seedpck_ln2_qty00', 'rt_01seedpck_ln2_price', 'rt_01seedpck_ln2_cost0', NOW(), NOW()),
    ('orln_01seedpck_ln3_0000', 'Freight', 'Charges for shipping', 'pd_01k0a65nx5fj1bxedew2jvjpwz', 'it_01k0a71009fc5szdjy8mn2nzq5', 'or_01k0a8bs2ye3f9p8sj0m4dfmwe', 'qu_01seedpck_ln3_qty00', 'rt_01seedpck_ln3_price', 'rt_01seedpck_ln3_cost0', NOW(), NOW());

-- Fulfilled order lines
INSERT IGNORE INTO sales_order_line (id, product_sku, product_description, product_id, item_id, sales_order_id, quantity_id, unit_price_id, unit_cost_id, created_at, updated_at) VALUES
    ('orln_01seedful_ln1_0000', 'SCK-005', 'Small beige sock', 'pd_01k0a65nx5fwmt17sqp317ekyr', 'it_01k0a7100aef2997gw0t7nxd9d', 'or_01k0a8bs2yf909wjkd7ecd6x4z', 'qu_01seedful_ln1_qty00', 'rt_01seedful_ln1_price', 'rt_01seedful_ln1_cost0', NOW(), NOW()),
    ('orln_01seedful_ln2_0000', 'SCK-006', 'Large beige sock', 'pd_01k0a65nx5e67rd1rahv4tdnrp', 'it_01k0a7100ae85v16mmxx5gx2w3', 'or_01k0a8bs2yf909wjkd7ecd6x4z', 'qu_01seedful_ln2_qty00', 'rt_01seedful_ln2_price', 'rt_01seedful_ln2_cost0', NOW(), NOW()),
    ('orln_01seedful_ln3_0000', 'Freight', 'Charges for shipping', 'pd_01k0a65nx5fj1bxedew2jvjpwz', 'it_01k0a71009fc5szdjy8mn2nzq5', 'or_01k0a8bs2yf909wjkd7ecd6x4z', 'qu_01seedful_ln3_qty00', 'rt_01seedful_ln3_price', 'rt_01seedful_ln3_cost0', NOW(), NOW());

-- ============================================================
-- PICKS
-- ============================================================

INSERT IGNORE INTO pick (id, number, sales_order_id, account_id, created_at, updated_at) VALUES
    ('pk_01k0a5tsn7f7psgagr1732fxqa', 'PICK-001', 'or_01k0a8bs2yejxbsvqhrx4drkq1', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('pk_01k0a5tsn7ejfrwg5dnshzfwsx', 'PICK-002', 'or_01k0a8bs2ye3f9p8sj0m4dfmwe', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(), INTERVAL 2 DAY), NOW()),
    ('pk_01k0a5tsn7eeht162chb2jcknc', 'PICK-003', 'or_01k0a8bs2yf909wjkd7ecd6x4z', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(), INTERVAL 4 DAY), NOW());

-- Update pick finished_at
UPDATE pick SET finished_at = DATE_SUB(NOW(), INTERVAL 2 DAY) WHERE id = 'pk_01k0a5tsn7ejfrwg5dnshzfwsx' AND finished_at IS NULL;
UPDATE pick SET finished_at = DATE_SUB(NOW(), INTERVAL 4 DAY) WHERE id = 'pk_01k0a5tsn7eeht162chb2jcknc' AND finished_at IS NULL;

-- Pick lines (quantities for pick lines)
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedpkln_iss_ln100', 15, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedpkln_iss_ln200', 8, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedpkln_pck_ln100', 20, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedpkln_pck_ln200', 15, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedpkln_ful_ln100', 25, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedpkln_ful_ln200', 18, 'un_01seedpair000000000', NOW(), NOW());

INSERT IGNORE INTO pick_line (id, pick_id, quantity_id, sales_order_line_id, created_at, updated_at) VALUES
    ('pkln_01seediss_ln1_0000', 'pk_01k0a5tsn7f7psgagr1732fxqa', 'qu_01seedpkln_iss_ln100', 'orln_01seediss_ln1_0000', NOW(), NOW()),
    ('pkln_01seediss_ln2_0000', 'pk_01k0a5tsn7f7psgagr1732fxqa', 'qu_01seedpkln_iss_ln200', 'orln_01seediss_ln2_0000', NOW(), NOW()),
    ('pkln_01seedpck_ln1_0000', 'pk_01k0a5tsn7ejfrwg5dnshzfwsx', 'qu_01seedpkln_pck_ln100', 'orln_01seedpck_ln1_0000', NOW(), NOW()),
    ('pkln_01seedpck_ln2_0000', 'pk_01k0a5tsn7ejfrwg5dnshzfwsx', 'qu_01seedpkln_pck_ln200', 'orln_01seedpck_ln2_0000', NOW(), NOW()),
    ('pkln_01seedful_ln1_0000', 'pk_01k0a5tsn7eeht162chb2jcknc', 'qu_01seedpkln_ful_ln100', 'orln_01seedful_ln1_0000', NOW(), NOW()),
    ('pkln_01seedful_ln2_0000', 'pk_01k0a5tsn7eeht162chb2jcknc', 'qu_01seedpkln_ful_ln200', 'orln_01seedful_ln2_0000', NOW(), NOW());

-- Mark packed pick lines as packed
UPDATE pick_line SET packed_at = DATE_SUB(NOW(), INTERVAL 2 DAY) WHERE id IN ('pkln_01seedpck_ln1_0000', 'pkln_01seedpck_ln2_0000') AND packed_at IS NULL;
UPDATE pick_line SET packed_at = DATE_SUB(NOW(), INTERVAL 4 DAY) WHERE id IN ('pkln_01seedful_ln1_0000', 'pkln_01seedful_ln2_0000') AND packed_at IS NULL;

INSERT IGNORE INTO `_departments_picks` (`A`, `B`) VALUES
    ('dp_01k0a5r01yfx3sj1vy9qgv3dc0', 'pk_01k0a5tsn7f7psgagr1732fxqa'),
    ('dp_01k0a5r01yfx3sj1vy9qgv3dc0', 'pk_01k0a5tsn7ejfrwg5dnshzfwsx'),
    ('dp_01k0a5r01yfx3sj1vy9qgv3dc0', 'pk_01k0a5tsn7eeht162chb2jcknc'),
    -- A second distinct department on a pick so the picks/department_ids array
    -- filter has >=2 distinct values to exercise union/exclusion (Washing).
    ('dp_01k0a5r01yf5csvz0jqfznf13d', 'pk_01k0a5tsn7ejfrwg5dnshzfwsx');

-- ============================================================
-- SHIPMENTS
-- ============================================================

-- Packed shipment (SeedShipmentID) — links carrier_option_id + shipped_by_id so
-- `?include=service_level` and `?include=shipped_by` both return populated stubs
-- on the seeded row. Invoice is linked post-hoc in 0013_finance.sql (INV-003).
INSERT IGNORE INTO shipment (id, number, sales_order_id, carrier_id, carrier_option_id, shipping_address_id, shipment_status_code, shipped_by_id, account_id, created_at, updated_at) VALUES
    ('sh_01k0a87w33emw8pmkz1mf86cg1', 'SHP-001', 'or_01k0a8bs2ye3f9p8sj0m4dfmwe', 'delivery', 'crop_01seedground000000', 'ad_01k09wnpvrea0awz7vem2j8j7g', 'packed', 'acus_s83fjhyfmqen', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- Fulfilled shipment (status: shipped)
INSERT IGNORE INTO shipment (id, number, sales_order_id, carrier_id, shipping_address_id, shipment_status_code, shipped_at, shipped_by_id, master_tracking_number, account_id, created_at, updated_at) VALUES
    ('sh_01k0a87w33fw0shhsahaa0yq6r', 'SHP-002', 'or_01k0a8bs2yf909wjkd7ecd6x4z', 'will_call', 'ad_01k09wnpvrea0awz7vem2j8j7g', 'shipped', DATE_SUB(NOW(), INTERVAL 2 DAY), 'acus_s83fjhyfmqen', '1234567890', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- Packed shipment for the shared seed order (SeedSalesOrderID / ORD-001) so that
-- ?include=related.shipments populates on its detail (ORD-001 has PICK-001).
INSERT IGNORE INTO shipment (id, number, sales_order_id, carrier_id, shipping_address_id, shipment_status_code, account_id, created_at, updated_at) VALUES
    ('sh_01k0a87w33emw8pmkz1mf86cg2', 'SHP-003', 'or_01k0a8bs2yejxbsvqhrx4drkq1', 'delivery', 'ad_01k09wnpvrea0awz7vem2j8j7g', 'packed', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- Shipping cases — quantities for freight weight/amount
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedshcase1_fwt000', 0, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedshcase1_fam000', 0, 'dollar', NOW(), NOW()),
    ('qu_01seedshcase2_fwt000', 0, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedshcase2_fam000', 0, 'dollar', NOW(), NOW()),
    ('qu_01seedshcase3_fwt000', 8, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedshcase3_fam000', 12, 'dollar', NOW(), NOW());

-- Packed shipment cases (2 cases)
INSERT IGNORE INTO shipping_case (id, number, freight_amount_id, freight_weight_id, shipment_id, carrier_id, account_id, created_at, updated_at) VALUES
    ('shcs_01seedshcase1_00000', 'SHP-001-1', 'qu_01seedshcase1_fam000', 'qu_01seedshcase1_fwt000', 'sh_01k0a87w33emw8pmkz1mf86cg1', 'delivery', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('shcs_01seedshcase2_00000', 'SHP-001-2', 'qu_01seedshcase2_fam000', 'qu_01seedshcase2_fwt000', 'sh_01k0a87w33emw8pmkz1mf86cg1', 'delivery', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- Fulfilled shipment case (1 case with tracking)
INSERT IGNORE INTO shipping_case (id, number, sscc, tracking_number, shipping_label_url, shipped_at, freight_amount_id, freight_weight_id, shipment_id, carrier_id, account_id, created_at, updated_at) VALUES
    ('shcs_01seedshcase3_00000', 'SHP-002-1', '1234567890', '1234567890', 'https://www.google.com', DATE_SUB(NOW(), INTERVAL 2 DAY), 'qu_01seedshcase3_fam000', 'qu_01seedshcase3_fwt000', 'sh_01k0a87w33fw0shhsahaa0yq6r', 'will_call', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- ============================================================
-- INVOICE (for fulfilled order)
-- ============================================================

INSERT IGNORE INTO invoice (id, number, sales_order_id, billing_address_id, account_id, created_at, updated_at) VALUES
    ('iv_01k09wnac0e1ar211e0sy0ba4g', 'INV-001', 'or_01k0a8bs2yf909wjkd7ecd6x4z', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(), INTERVAL 3 DAY), DATE_SUB(NOW(), INTERVAL 3 DAY));

-- Link shipment to invoice
UPDATE shipment SET invoice_id = 'iv_01k09wnac0e1ar211e0sy0ba4g' WHERE id = 'sh_01k0a87w33fw0shhsahaa0yq6r' AND invoice_id IS NULL;

-- Invoice line quantities
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedivln_ful_ln100', 25, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedivln_ful_ln200', 18, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedivln_ful_ln300', 1, 'each', NOW(), NOW());

INSERT IGNORE INTO invoice_line (id, invoice_id, quantity_id, sales_order_line_id, created_at, updated_at) VALUES
    ('ivln_01seedful_ln1_0000', 'iv_01k09wnac0e1ar211e0sy0ba4g', 'qu_01seedivln_ful_ln100', 'orln_01seedful_ln1_0000', NOW(), NOW()),
    ('ivln_01seedful_ln2_0000', 'iv_01k09wnac0e1ar211e0sy0ba4g', 'qu_01seedivln_ful_ln200', 'orln_01seedful_ln2_0000', NOW(), NOW()),
    ('ivln_01seedful_ln3_0000', 'iv_01k09wnac0e1ar211e0sy0ba4g', 'qu_01seedivln_ful_ln300', 'orln_01seedful_ln3_0000', NOW(), NOW());

-- ============================================================
-- ORDER DISCOUNTS
-- ============================================================

INSERT IGNORE INTO order_discount (id, name, code, percentage, value, discount_type_code, account_id, created_at, updated_at) VALUES
    ('ords_01seedpct10discount', '10% Off', 'PCT10', 10, 0, 'percentage', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- Link SeedSalesOrderID (ORD-001) to the seeded discount so
-- `?include=order_discount` resolves.
UPDATE sales_order SET order_discount_id = 'ords_01seedpct10discount'
    WHERE id = 'or_01k0a8bs2yejxbsvqhrx4drkq1' AND order_discount_id IS NULL;

-- Backfill include-critical references for SeedSalesOrderID when the row already
-- exists from an older seed run (INSERT IGNORE keeps stale values).
UPDATE sales_order
SET
    payment_term_id = COALESCE(payment_term_id, 'pytm_01seednet3000000'),
    shipping_term_id = COALESCE(shipping_term_id, 'prepaid_billed'),
    carrier_option_id = COALESCE(carrier_option_id, 'crop_01seedground000000')
WHERE id = 'or_01k0a8bs2yejxbsvqhrx4drkq1';

-- Production run linked to SeedSalesOrderID so `?include=related.production_run`
-- resolves with real data.
INSERT IGNORE INTO production_run (id, responsible_user_id, number, account_id, started_at, created_at, updated_at) VALUES
    ('pr_01seedsalesorder0001', 'us_1wjfmmbwg8l7', 'PR-SEED-001', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW(), NOW());

-- Backfill sales_rep + production_run on SeedSalesOrderID so `?include=sales_rep`
-- and `?include=related.production_run` resolve with real data.
UPDATE sales_order
SET
    sales_rep_id = COALESCE(sales_rep_id, 'acus_s83fjhyfmqen'),
    production_run_id = COALESCE(production_run_id, 'pr_01seedsalesorder0001')
WHERE id = 'or_01k0a8bs2yejxbsvqhrx4drkq1';

-- Assign a second, distinct sales rep to ORD-002 so the sales-orders/customers
-- sales_rep_ids array filters have >=2 distinct values to exercise union/exclusion.
UPDATE sales_order
SET sales_rep_id = COALESCE(sales_rep_id, 'acus_ubdx4zebgl6p')
WHERE id = 'or_01k0a8bs2ye3f9p8sj0m4dfmwe';
