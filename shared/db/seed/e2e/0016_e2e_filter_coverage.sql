-- 0016_e2e_filter_coverage.sql
-- Seeds a second distinct value for list-endpoint array filters that the
-- TestArrayFilters_UnionExclusion e2e test was skipping for lack of seed data.
-- Runs last (after 0014/0015) so all FK dependencies exist.
--
-- Each block below names the array-filter case(s) it unblocks. Cases that the
-- test cannot exercise via seed data alone (the endpoint rejects the include the
-- test sends, the presenter omits the id, or a cross-account loader is missing)
-- are intentionally NOT addressed here and are tracked separately.
--
-- Discovery-critical finance rows (invoices, transactions) use a far-future
-- created_at so they stay on the first page of their list endpoints despite the
-- hundreds of rows other parallel e2e runs generate — the same technique the
-- audit_event / request_log seeds in 0014 use.

-- ============================================================
-- THIRD CUSTOMER — distinct status, sales rep, and price group
-- Unblocks: customers/status_codes, customers/sales_rep_ids,
--           customers/pricing_group_ids
-- (existing customers are all status=normal, rep=acus_s83fjhyfmqen, group=DME)
-- ============================================================

INSERT IGNORE INTO account (id, name, account_type_code, onboarding_status_code, account_plan_id, created_at, updated_at) VALUES
    ('ac_01seedcust3_acct000', 'Mountain West Traders', 'company', 'unclaimed', 'acpl_01seed000free00plan000000', NOW(), NOW());

INSERT IGNORE INTO geolocation (id, street_line_1, locality, state, postal_code, country, created_at, updated_at) VALUES
    ('gl_01seedcust3billto00', '700 Summit Blvd', 'Denver', 'CO', '80201', 'US', NOW(), NOW()),
    ('gl_01seedcust3shipto00', '800 Ridge Rd', 'Salt Lake City', 'UT', '84101', 'US', NOW(), NOW());

INSERT IGNORE INTO address (id, name, geolocation_id, created_at, updated_at) VALUES
    ('ad_01seedcust3billing0', 'Mountain West Traders', 'gl_01seedcust3billto00', NOW(), NOW()),
    ('ad_01seedcust3shipping', 'Mountain West Traders', 'gl_01seedcust3shipto00', NOW(), NOW());

INSERT IGNORE INTO account_address (id, account_id, address_id, created_at, updated_at) VALUES
    ('acad_01seedcust3bill00', 'ac_01seedcust3_acct000', 'ad_01seedcust3billing0', NOW(), NOW()),
    ('acad_01seedcust3ship00', 'ac_01seedcust3_acct000', 'ad_01seedcust3shipping', NOW(), NOW());

-- status_code = 'preferred' (vs 'normal'), default_sales_rep_id = acus_ubdx4zebgl6p
-- (a different account_user than the existing customer's acus_s83fjhyfmqen).
INSERT IGNORE INTO account_relation (id, owner_account_id, counterparty_account_id, account_relation_role_code, external_number, is_edi_enabled, priority_code, account_status_code, commission_status_code, freight_status_code, shipping_term_id, payment_term_id, account_group_id, default_sales_rep_id, default_billing_address_id, default_shipping_address_id, default_carrier_id, created_at, updated_at) VALUES
    ('acre_01seedcust3_00000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01seedcust3_acct000', 'customer', '88888', 0, 'normal', 'preferred', 'commission_applied', 'billed_freight', 'prepaid_billed', 'pytm_01seednet3000000', 'acgp_01k0a413mjeth8pe1g70t0thax', 'acus_ubdx4zebgl6p', 'ad_01seedcust3billing0', 'ad_01seedcust3shipping', 'delivery', NOW(), NOW());

-- Price group = National (acgp_01seedsecondgroup0), distinct from the DME group
-- every other seeded customer belongs to.
INSERT IGNORE INTO account_relation_price_group (id, account_relation_id, account_group_id, created_at, updated_at) VALUES
    ('acrepg_01seedcust3pg0', 'acre_01seedcust3_00000', 'acgp_01seedsecondgroup0', NOW(), NOW());

-- Backfill on re-seed if the relation predates these columns.
UPDATE account_relation
   SET account_status_code = 'preferred', default_sales_rep_id = 'acus_ubdx4zebgl6p'
 WHERE id = 'acre_01seedcust3_00000'
   AND (account_status_code <> 'preferred' OR default_sales_rep_id IS NULL);

-- ============================================================
-- PRODUCT + PART in a second item category
-- Unblocks: products/category_ids, parts/category_ids
-- (every listed product/part item is in the Socks category)
-- Items require non-null UNIQUE unit_value / unit_cost / burn_rate rate rows.
-- ============================================================

INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedfcprod_uv00', '12.00', 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedfcprod_uc00', '6.00',  'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedfcprod_br00', '1.00',  'un_01seedpair000000000', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedfcpart_uv00', '3.00',  'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_01seedfcpart_uc00', '1.50',  'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_01seedfcpart_br00', '1.00',  'un_01seedpound00000000', 'un_01seedpound00000000', NOW(), NOW());

-- Product whose item is in the eBad category (a second product_category — products and
-- parts may only use product categories, matching the API validation).
INSERT IGNORE INTO item (id, sku, description, account_id, item_type_code, item_category_id, unit_value_id, unit_cost_id, burn_rate_id, created_at, updated_at) VALUES
    ('it_01seedfcprodyarn00', 'FC-PROD-YARN', 'E2E filter coverage product (eBad category)', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedebad0000000', 'rt_01seedfcprod_uv00', 'rt_01seedfcprod_uc00', 'rt_01seedfcprod_br00', NOW(), NOW());

INSERT IGNORE INTO product (id, item_id, product_type_code, product_line_id, is_portal_ready, created_at, updated_at) VALUES
    ('pd_01seedfcprodyarn00', 'it_01seedfcprodyarn00', 'sale', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', 1, NOW(), NOW());

-- Part whose item is in the eBad category (parts may only use product categories).
INSERT IGNORE INTO item (id, sku, description, account_id, item_type_code, item_category_id, unit_value_id, unit_cost_id, burn_rate_id, created_at, updated_at) VALUES
    ('it_01seedfcpartchem00', 'FC-PART-CHEM', 'E2E filter coverage part (eBad category)', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedebad0000000', 'rt_01seedfcpart_uv00', 'rt_01seedfcpart_uc00', 'rt_01seedfcpart_br00', NOW(), NOW());

INSERT IGNORE INTO part (id, item_id, created_at, updated_at) VALUES
    ('pt_01seedfcpartchem00', 'it_01seedfcpartchem00', NOW(), NOW());

-- ============================================================
-- SALES ORDER + INVOICES + PICK for the SECOND customer
-- Unblocks: sales-orders/customer_ids, invoices/customer_ids,
--           picks/customer_ids
-- (all existing orders/invoices/picks belong to one buyer:
--  Global Manufacturing, ac_01k09wm2fgevdsc344gpbcj30f)
-- Buyer is the existing second customer, Pacific Coast Distributors.
-- ============================================================

INSERT IGNORE INTO sales_order (id, number, sales_order_status_code, sales_order_type_code, priority_code, billing_address_id, shipping_address_id, buyer_account_id, seller_account_id, owner_account_id, payment_term_id, shipping_term_id, issued_at, created_at, updated_at) VALUES
    ('or_01seedfcsocust2_00', 'ORD-FC-CUST2', 'issued', 'sales_order', 'normal', 'ad_01seedcust2billing000', 'ad_01seedcust2shipping00', 'ac_01seedcustomer2_acct0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pytm_01seednet3000000', 'prepaid_billed', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR));

-- Two invoices (far-future) so they are the top two rows of /v1/finance/invoices,
-- which also makes them the values settlements/invoice_ids discovers below.
INSERT IGNORE INTO invoice (id, number, sales_order_id, billing_address_id, account_id, created_at, updated_at) VALUES
    ('iv_01seedfcinv2a0000', 'INV-FC-2A', 'or_01seedfcsocust2_00', 'ad_01seedcust2billing000', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR)),
    ('iv_01seedfcinv2b0000', 'INV-FC-2B', 'or_01seedfcsocust2_00', 'ad_01seedcust2billing000', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR));

INSERT IGNORE INTO pick (id, number, sales_order_id, account_id, created_at, updated_at) VALUES
    ('pk_01seedfcpickcust2', 'PICK-FC-CUST2', 'or_01seedfcsocust2_00', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR));

-- A shipment for the second customer's order so the shipments/customer_ids array
-- filter has shipments under >=2 distinct customers (the main seed customer + this one).
INSERT IGNORE INTO shipment (id, number, sales_order_id, carrier_id, shipping_address_id, shipment_status_code, account_id, created_at, updated_at) VALUES
    ('sh_01seedfcshipcust2', 'SHP-FC-CUST2', 'or_01seedfcsocust2_00', 'delivery', 'ad_01seedcust2shipping00', 'packed', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR));

-- ============================================================
-- SALES ORDERS for list-endpoint filter coverage
-- (tests/e2e/api/list_sales_orders_test.go)
-- Far-future created_at keeps both on the first page of /v1/sales/sales-orders
-- regardless of how many orders parallel e2e runs create.
-- ============================================================

-- Internal/self order: buyer == seller == owner == the main seed account.
-- The self-relation acre_01seedhouseacct0000 (owner == counterparty == this
-- account, in 0010_customers.sql) satisfies the list query's inner
-- account_relation join, so this order is returned by the list endpoint.
-- Unblocks: internal-order visibility (created_by include, customer-portal access).
INSERT IGNORE INTO sales_order (id, number, sales_order_status_code, sales_order_type_code, priority_code, billing_address_id, shipping_address_id, buyer_account_id, seller_account_id, owner_account_id, payment_term_id, shipping_term_id, issued_at, created_at, updated_at) VALUES
    ('or_01seedfcsointernal', 'ORD-E2E-INTERNAL', 'issued', 'sales_order', 'normal', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pytm_01seednet3000000', 'prepaid_billed', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR));

-- External order carrying a known customer_po_number so the exact-match search
-- (q=PO) has a positive hit. Buyer is the main seed customer (relation
-- acre_01seedcustomer00000). Unblocks: customer_po_number search branch.
INSERT IGNORE INTO sales_order (id, number, customer_po_number, sales_order_status_code, sales_order_type_code, priority_code, billing_address_id, shipping_address_id, buyer_account_id, seller_account_id, owner_account_id, payment_term_id, shipping_term_id, issued_at, created_at, updated_at) VALUES
    ('or_01seedfcsopo00000', 'ORD-E2E-PO', 'PO-E2E-EXACT-001', 'issued', 'sales_order', 'normal', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pytm_01seednet3000000', 'prepaid_billed', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR));

-- ============================================================
-- TRANSACTIONS with distinct types, methods, and adjustment types
-- Unblocks: transactions/types, transactions/methods,
--           transactions/adjustment_types
-- (all existing transactions are type=payment with NULL method/adjustment)
-- Far-future created_at keeps them on the first page (also makes them the values
-- settlements/transaction_ids discovers below).
-- ============================================================

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedfctx1_amt00', 100, 'dollar', NOW(), NOW()),
    ('qu_01seedfctx2_amt00', 50,  'dollar', NOW(), NOW()),
    ('qu_01seedfctx3_amt00', 25,  'dollar', NOW(), NOW()),
    ('qu_01seedfctx4_amt00', 75,  'dollar', NOW(), NOW());

INSERT IGNORE INTO transaction (id, number, customer_account_id, amount_id, transaction_type_code, transaction_method_code, adjustment_type_code, is_fully_allocated, account_id, created_at, updated_at) VALUES
    ('tx_01seedfctxn1_0000', 'TXN-FC-1', 'ac_01k09wm2fgevdsc344gpbcj30f', 'qu_01seedfctx1_amt00', 'credit_memo', 'ach',   NULL,        0, 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR)),
    ('tx_01seedfctxn2_0000', 'TXN-FC-2', 'ac_01k09wm2fgevdsc344gpbcj30f', 'qu_01seedfctx2_amt00', 'adjustment',  'check', 'fee',       0, 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR)),
    ('tx_01seedfctxn3_0000', 'TXN-FC-3', 'ac_01k09wm2fgevdsc344gpbcj30f', 'qu_01seedfctx3_amt00', 'rebate',      'cash',  'discount',  0, 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR)),
    -- A second customer (Pacific Coast) with an unallocated credit so the
    -- open-credits/customer_ids array filter sees >=2 distinct customers.
    ('tx_01seedfctxn4_0000', 'TXN-FC-4', 'ac_01seedcustomer2_acct0',     'qu_01seedfctx4_amt00', 'credit_memo', 'ach',   NULL,        0, 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR));

-- ============================================================
-- SETTLEMENT + ALLOCATIONS linking it to the discovered top transactions
-- and top invoices.
-- Unblocks: settlements/transaction_ids, settlements/invoice_ids
-- (no settlement was linked to the rows those filters discover from the
--  /transactions and /invoices feeds)
-- ============================================================

INSERT IGNORE INTO settlement (id, number, account_id, responsible_user_id, created_at, updated_at) VALUES
    ('sl_01seedfcsettl0000', 'STL-FC', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', NOW(), NOW());

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedfcal1_amt00', 100, 'dollar', NOW(), NOW()),
    ('qu_01seedfcal2_amt00', 50,  'dollar', NOW(), NOW()),
    ('qu_01seedfcal3_amt00', 25,  'dollar', NOW(), NOW()),
    ('qu_01seedfcal4_amt00', 25,  'dollar', NOW(), NOW());

-- Allocations cover all three new transactions (so whichever two are discovered
-- as the top of the transactions feed are both linked) and both new invoices
-- (the top of the invoices feed).
INSERT IGNORE INTO transaction_allocation (id, transaction_id, amount_id, invoice_id, settlement_id, created_at, updated_at) VALUES
    ('txal_01seedfcalloc1', 'tx_01seedfctxn1_0000', 'qu_01seedfcal1_amt00', 'iv_01seedfcinv2a0000', 'sl_01seedfcsettl0000', NOW(), NOW()),
    ('txal_01seedfcalloc2', 'tx_01seedfctxn2_0000', 'qu_01seedfcal2_amt00', 'iv_01seedfcinv2b0000', 'sl_01seedfcsettl0000', NOW(), NOW()),
    ('txal_01seedfcalloc3', 'tx_01seedfctxn3_0000', 'qu_01seedfcal3_amt00', 'iv_01seedfcinv2a0000', 'sl_01seedfcsettl0000', NOW(), NOW()),
    -- Partially allocate the second-customer credit (tx4) to the settlement so it
    -- stays top-of-feed for open-credits AND is linked for settlements/transaction_ids.
    ('txal_01seedfcalloc4', 'tx_01seedfctxn4_0000', 'qu_01seedfcal4_amt00', 'iv_01seedfcinv2a0000', 'sl_01seedfcsettl0000', NOW(), NOW());

-- Production run + batches producing the first two filter-coverage catalog items
-- (FC-PROD-YARN, FC-PART-CHEM) so the production-runs/item_ids array filter has
-- runs linked to the top-of-feed item ids (runs match via batch.item_id).
INSERT IGNORE INTO production_run (id, responsible_user_id, number, account_id, started_at, created_at, updated_at) VALUES
    ('pnrn_01seedfcrun00000', 'us_1wjfmmbwg8l7', 'PR-FC-001', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW(), NOW());

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedfcbatch1_qty', 10, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedfcbatch2_qty', 10, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedfcbatch3_qty', 10, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedfcbatch4_qty', 10, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedfcbatch5_qty', 10, 'un_01seedpound00000000', NOW(), NOW());

INSERT IGNORE INTO batch (id, account_id, item_id, quantity_id, scanning_station_id, production_step_id, production_run_id, created_at, updated_at) VALUES
    ('bt_01seedfcbatch1_000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedfcprodyarn00', 'qu_01seedfcbatch1_qty', 'sgsn_01k0a8201zegarjfsjaw5n7yfv', 'prs_01k0a51qxceydax5036pegvzzy', 'pnrn_01seedfcrun00000', NOW(), NOW()),
    ('bt_01seedfcbatch2_000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedfcpartchem00', 'qu_01seedfcbatch2_qty', 'sgsn_01k0a8201zegarjfsjaw5n7yfv', 'prs_01k0a575j3fqr97khk36v114nj', 'pnrn_01seedfcrun00000', NOW(), NOW()),
    -- The first rows of /catalog/items (created_at DESC, id DESC) must all have batches
    -- because production-runs/item_ids samples that feed: the far-future eBad item is
    -- row 1, and the e2evol items sort next among the NOW()-cohort by id. Any new seed
    -- item that outranks these must get a batch here too.
    ('bt_01seedfcbatch3_000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedfcebad000000', 'qu_01seedfcbatch3_qty', 'sgsn_01k0a8201zegarjfsjaw5n7yfv', 'prs_01k0a51qxceydax5036pegvzzy', 'pnrn_01seedfcrun00000', NOW(), NOW()),
    ('bt_01seedfcbatch4_000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_e2evol2000000000', 'qu_01seedfcbatch4_qty', 'sgsn_01k0a8201zegarjfsjaw5n7yfv', 'prs_01k0a51qxceydax5036pegvzzy', 'pnrn_01seedfcrun00000', NOW(), NOW()),
    ('bt_01seedfcbatch5_000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_e2evol1000000000', 'qu_01seedfcbatch5_qty', 'sgsn_01k0a8201zegarjfsjaw5n7yfv', 'prs_01k0a51qxceydax5036pegvzzy', 'pnrn_01seedfcrun00000', NOW(), NOW());
-- Production run + batches linked to catalog item ids that
-- production-runs/item_ids discovers from /v1/catalog/items (runs match via batch.item_id).
-- FC-PROD-YARN / FC-PART-CHEM are filter-coverage rows; FC-EBAD-PROD is far-future in
-- the items feed; it_e2evol* volume-discount products (0014_e2e_extras.sql) are also
-- sampled from the top of that feed.

-- A product (and its item) in a SECOND product line (eBad) with a far-future
-- created_at so the catalog products/items feeds always carry >=2 distinct
-- product lines on the first page despite parallel-test churn.
-- Unblocks: products/product_line_ids, items/product_line_ids
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedfcebad_uv00', '20.00', 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedfcebad_uc00', '10.00', 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedfcebad_br00', '1.00',  'un_01seedpair000000000', 'un_01seedpair000000000', NOW(), NOW());

INSERT IGNORE INTO item (id, sku, description, account_id, item_type_code, item_category_id, unit_value_id, unit_cost_id, burn_rate_id, created_at, updated_at) VALUES
    ('it_01seedfcebad000000', 'FC-EBAD-PROD', 'E2E filter coverage product (eBad product line)', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedebad0000000', 'rt_01seedfcebad_uv00', 'rt_01seedfcebad_uc00', 'rt_01seedfcebad_br00', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR));

INSERT IGNORE INTO product (id, item_id, product_type_code, product_line_id, is_portal_ready, created_at, updated_at) VALUES
    ('pd_01seedfcebad000000', 'it_01seedfcebad000000', 'sale', 'pdln_01k0a735ypfjva933tg57wfx0t', 1, DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR));

INSERT IGNORE INTO production_run (id, responsible_user_id, number, account_id, started_at, created_at, updated_at) VALUES
    ('pnrn_01seedfcrun00000', 'us_1wjfmmbwg8l7', 'PR-FC-001', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW(), NOW());

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedfcbatch1_qty', 10, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedfcbatch2_qty', 10, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedfcbatch3_qty', 10, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedfcbatch4_qty', 10, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedfcbatch5_qty', 10, 'un_01seedpair000000000', NOW(), NOW());

INSERT IGNORE INTO batch (id, account_id, item_id, quantity_id, scanning_station_id, production_step_id, production_run_id, created_at, updated_at) VALUES
    ('bt_01seedfcbatch1_000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedfcprodyarn00', 'qu_01seedfcbatch1_qty', 'sgsn_01k0a8201zegarjfsjaw5n7yfv', 'prs_01k0a51qxceydax5036pegvzzy', 'pnrn_01seedfcrun00000', NOW(), NOW()),
    ('bt_01seedfcbatch2_000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedfcpartchem00', 'qu_01seedfcbatch2_qty', 'sgsn_01k0a8201zegarjfsjaw5n7yfv', 'prs_01k0a575j3fqr97khk36v114nj', 'pnrn_01seedfcrun00000', NOW(), NOW()),
    ('bt_01seedfcbatch3_000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedfcebad000000', 'qu_01seedfcbatch3_qty', 'sgsn_01k0a8201zegarjfsjaw5n7yfv', 'prs_01k0a51qxceydax5036pegvzzy', 'pnrn_01seedfcrun00000', NOW(), NOW()),
    ('bt_01seedfcbatch4_000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_e2evol1000000000', 'qu_01seedfcbatch4_qty', 'sgsn_01k0a8201zegarjfsjaw5n7yfv', 'prs_01k0a51qxceydax5036pegvzzy', 'pnrn_01seedfcrun00000', NOW(), NOW()),
    ('bt_01seedfcbatch5_000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_e2evol2000000000', 'qu_01seedfcbatch5_qty', 'sgsn_01k0a8201zegarjfsjaw5n7yfv', 'prs_01k0a575j3fqr97khk36v114nj', 'pnrn_01seedfcrun00000', NOW(), NOW());
