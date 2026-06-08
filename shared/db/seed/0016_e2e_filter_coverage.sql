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

-- status_code = 'preferred' (vs 'normal'), default_sales_rep_id = acus_5e77zahotfn0
-- (a different account_user than the existing customer's acus_s83fjhyfmqen).
INSERT IGNORE INTO account_relation (id, owner_account_id, counterparty_account_id, account_relation_role_code, external_number, is_edi_enabled, priority_code, account_status_code, commission_status_code, freight_status_code, shipping_term_id, payment_term_id, account_group_id, default_sales_rep_id, default_billing_address_id, default_shipping_address_id, default_carrier_id, created_at, updated_at) VALUES
    ('acre_01seedcust3_00000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01seedcust3_acct000', 'customer', '88888', 0, 'normal', 'preferred', 'commission_applied', 'billed_freight', 'prepaid_billed', 'pytm_01seednet3000000', 'acgp_01k0a413mjeth8pe1g70t0thax', 'acus_5e77zahotfn0', 'ad_01seedcust3billing0', 'ad_01seedcust3shipping', 'delivery', NOW(), NOW());

-- Price group = National (acgp_01seedsecondgroup0), distinct from the DME group
-- every other seeded customer belongs to.
INSERT IGNORE INTO account_relation_price_group (id, account_relation_id, account_group_id, created_at, updated_at) VALUES
    ('acrepg_01seedcust3pg0', 'acre_01seedcust3_00000', 'acgp_01seedsecondgroup0', NOW(), NOW());

-- Backfill on re-seed if the relation predates these columns.
UPDATE account_relation
   SET account_status_code = 'preferred', default_sales_rep_id = 'acus_5e77zahotfn0'
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

-- Product whose item is in the Yarn category.
INSERT IGNORE INTO item (id, sku, description, account_id, item_type_code, item_category_id, unit_value_id, unit_cost_id, burn_rate_id, created_at, updated_at) VALUES
    ('it_01seedfcprodyarn00', 'FC-PROD-YARN', 'E2E filter coverage product (Yarn category)', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedyarn0000000', 'rt_01seedfcprod_uv00', 'rt_01seedfcprod_uc00', 'rt_01seedfcprod_br00', NOW(), NOW());

INSERT IGNORE INTO product (id, item_id, product_type_code, product_line_id, is_portal_ready, created_at, updated_at) VALUES
    ('pd_01seedfcprodyarn00', 'it_01seedfcprodyarn00', 'sale', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', 1, NOW(), NOW());

-- Part whose item is in the Chemicals category.
INSERT IGNORE INTO item (id, sku, description, account_id, item_type_code, item_category_id, unit_value_id, unit_cost_id, burn_rate_id, created_at, updated_at) VALUES
    ('it_01seedfcpartchem00', 'FC-PART-CHEM', 'E2E filter coverage part (Chemicals category)', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedchemicals00', 'rt_01seedfcpart_uv00', 'rt_01seedfcpart_uc00', 'rt_01seedfcpart_br00', NOW(), NOW());

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
    ('qu_01seedfctx3_amt00', 25,  'dollar', NOW(), NOW());

INSERT IGNORE INTO transaction (id, number, customer_account_id, amount_id, transaction_type_code, transaction_method_code, adjustment_type_code, is_fully_allocated, account_id, created_at, updated_at) VALUES
    ('tx_01seedfctxn1_0000', 'TXN-FC-1', 'ac_01k09wm2fgevdsc344gpbcj30f', 'qu_01seedfctx1_amt00', 'credit_memo', 'ach',   NULL,        0, 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR)),
    ('tx_01seedfctxn2_0000', 'TXN-FC-2', 'ac_01k09wm2fgevdsc344gpbcj30f', 'qu_01seedfctx2_amt00', 'adjustment',  'check', 'fee',       0, 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR)),
    ('tx_01seedfctxn3_0000', 'TXN-FC-3', 'ac_01k09wm2fgevdsc344gpbcj30f', 'qu_01seedfctx3_amt00', 'rebate',      'cash',  'discount',  0, 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR));

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
    ('qu_01seedfcal3_amt00', 25,  'dollar', NOW(), NOW());

-- Allocations cover all three new transactions (so whichever two are discovered
-- as the top of the transactions feed are both linked) and both new invoices
-- (the top of the invoices feed).
INSERT IGNORE INTO transaction_allocation (id, transaction_id, amount_id, invoice_id, settlement_id, created_at, updated_at) VALUES
    ('txal_01seedfcalloc1', 'tx_01seedfctxn1_0000', 'qu_01seedfcal1_amt00', 'iv_01seedfcinv2a0000', 'sl_01seedfcsettl0000', NOW(), NOW()),
    ('txal_01seedfcalloc2', 'tx_01seedfctxn2_0000', 'qu_01seedfcal2_amt00', 'iv_01seedfcinv2b0000', 'sl_01seedfcsettl0000', NOW(), NOW()),
    ('txal_01seedfcalloc3', 'tx_01seedfctxn3_0000', 'qu_01seedfcal3_amt00', 'iv_01seedfcinv2a0000', 'sl_01seedfcsettl0000', NOW(), NOW());
