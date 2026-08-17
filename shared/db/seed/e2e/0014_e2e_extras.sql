-- 0014_e2e_extras.sql
-- Seeds additional data needed for full e2e test coverage.
-- This file runs last to ensure all FK dependencies exist.
-- It adds:
--   1. Second rows for entities that only had 1 (for pagination tests)
--   2. New entity types that had zero rows (suppliers, territories, deliveries, etc.)
--   3. Audit events, email logs, request logs (normally system-generated)

-- ============================================================
-- SECOND CUSTOMER (for pagination on /v1/sales/customers)
-- ============================================================

INSERT IGNORE INTO account (id, name, account_type_code, onboarding_status_code, account_plan_id, created_at, updated_at) VALUES
    ('ac_01seedcustomer2_acct0', 'Pacific Coast Distributors', 'company', 'unclaimed', 'acpl_01seed000free00plan000000', NOW(), NOW());

INSERT IGNORE INTO geolocation (id, street_line_1, locality, state, postal_code, country, created_at, updated_at) VALUES
    ('gl_01seedcust2billto000', '100 Market St', 'Los Angeles', 'CA', '90001', 'US', NOW(), NOW()),
    ('gl_01seedcust2shipto000', '200 Commerce Ave', 'Portland', 'OR', '97201', 'US', NOW(), NOW());

INSERT IGNORE INTO address (id, name, geolocation_id, created_at, updated_at) VALUES
    ('ad_01seedcust2billing000', 'Pacific Coast Distributors', 'gl_01seedcust2billto000', NOW(), NOW()),
    ('ad_01seedcust2shipping00', 'Pacific Coast Distributors', 'gl_01seedcust2shipto000', NOW(), NOW());

INSERT IGNORE INTO account_address (id, account_id, address_id, created_at, updated_at) VALUES
    ('acad_01seedcust2bill0000', 'ac_01seedcustomer2_acct0', 'ad_01seedcust2billing000', NOW(), NOW()),
    ('acad_01seedcust2ship0000', 'ac_01seedcustomer2_acct0', 'ad_01seedcust2shipping00', NOW(), NOW());

UPDATE account SET
    default_billing_address_id = 'ad_01seedcust2billing000',
    default_shipping_address_id = 'ad_01seedcust2shipping00'
WHERE id = 'ac_01seedcustomer2_acct0'
    AND default_billing_address_id IS NULL;

INSERT IGNORE INTO account_relation (id, owner_account_id, counterparty_account_id, account_relation_role_code, external_number, is_edi_enabled, priority_code, account_status_code, commission_status_code, freight_status_code, shipping_term_id, payment_term_id, account_group_id, default_billing_address_id, default_shipping_address_id, default_carrier_id, created_at, updated_at) VALUES
    ('acre_01seedcustomer20000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01seedcustomer2_acct0', 'customer', '99999', 0, 'normal', 'normal', 'commission_applied', 'billed_freight', 'prepaid_billed', 'pytm_01seednet3000000', 'acgp_01k0a413mjeth8pe1g70t0thax', 'ad_01seedcust2billing000', 'ad_01seedcust2shipping00', 'delivery', NOW(), NOW());

-- ============================================================
-- SECOND ACCOUNT GROUP (for pagination)
-- ============================================================

INSERT IGNORE INTO account_group (id, owner_account_id, name, account_group_type_code, commission_status_code, freight_status_code, created_at, updated_at) VALUES
    ('acgp_01seedsecondgroup0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'National', 'type_group', 'commission_applied', 'billed_freight', NOW(), NOW());

-- Second account group product line access (for pagination on /v1/sales/product-line-access/account-groups)
INSERT IGNORE INTO account_group_product_line (id, account_group_id, product_line_id, created_at, updated_at) VALUES
    ('acgrpl_01seedsecondgrpl', 'acgp_01seedsecondgroup0', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', NOW(), NOW());

-- ============================================================
-- SECOND CUSTOMER PRODUCT LINE ACCESS (for pagination)
-- ============================================================

INSERT IGNORE INTO account_relation_product_line (id, account_relation_id, product_line_id, created_at, updated_at) VALUES
    ('acrepdln_01seedcust2pl', 'acre_01seedcustomer20000', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', NOW(), NOW());

-- ============================================================
-- SECOND REGISTRATION FLOW (for pagination)
-- ============================================================

INSERT IGNORE INTO registration_flow (id, name, account_id, created_at, updated_at) VALUES
    ('rgfw_01seedsecondflow00', 'Portal Registration Flow', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- ============================================================
-- SUPPLIERS (2 account_relation rows with role='supplier', for pagination)
-- ============================================================

INSERT IGNORE INTO account (id, name, account_type_code, onboarding_status_code, account_plan_id, created_at, updated_at) VALUES
    ('ac_01seedsupplier_acct0', 'Yarn Supply Co', 'company', 'unclaimed', 'acpl_01seed000free00plan000000', NOW(), NOW()),
    ('ac_01seedsupplier_acct1', 'Dye Supply Co', 'company', 'unclaimed', 'acpl_01seed000free00plan000000', NOW(), NOW());

INSERT IGNORE INTO geolocation (id, street_line_1, locality, state, postal_code, country, created_at, updated_at) VALUES
    ('gl_01seedsuppliergeo000', '500 Industrial Blvd', 'Charlotte', 'NC', '28201', 'US', NOW(), NOW()),
    ('gl_01seedsupplier2geo00', '600 Factory Rd', 'Atlanta', 'GA', '30301', 'US', NOW(), NOW());

INSERT IGNORE INTO address (id, name, geolocation_id, created_at, updated_at) VALUES
    ('ad_01seedsupplieraddr00', 'Yarn Supply Co', 'gl_01seedsuppliergeo000', NOW(), NOW()),
    ('ad_01seedsupplier2addr0', 'Dye Supply Co', 'gl_01seedsupplier2geo00', NOW(), NOW());

INSERT IGNORE INTO account_address (id, account_id, address_id, created_at, updated_at) VALUES
    ('acad_01seedsupplier1addr', 'ac_01seedsupplier_acct0', 'ad_01seedsupplieraddr00', NOW(), NOW()),
    ('acad_01seedsupplier2addr', 'ac_01seedsupplier_acct1', 'ad_01seedsupplier2addr0', NOW(), NOW());

INSERT IGNORE INTO account_relation (id, owner_account_id, counterparty_account_id, account_relation_role_code, external_number, is_edi_enabled, priority_code, default_billing_address_id, default_shipping_address_id, created_at, updated_at) VALUES
    ('acre_01seedsupplier0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01seedsupplier_acct0', 'supplier', 'SUP-001', 0, 'normal', 'ad_01seedsupplieraddr00', 'ad_01seedsupplieraddr00', NOW(), NOW()),
    ('acre_01seedsupplier0001', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01seedsupplier_acct1', 'supplier', 'SUP-002', 0, 'normal', 'ad_01seedsupplier2addr0', 'ad_01seedsupplier2addr0', NOW(), NOW());

-- Set default billing/shipping addresses on supplier accounts themselves so the
-- Supplier adapter (which falls back to account defaults when the relation lacks them)
-- always resolves to a concrete address.
UPDATE account SET
    default_billing_address_id  = 'ad_01seedsupplieraddr00',
    default_shipping_address_id = 'ad_01seedsupplieraddr00'
WHERE id = 'ac_01seedsupplier_acct0'
    AND default_billing_address_id IS NULL;

UPDATE account SET
    default_billing_address_id  = 'ad_01seedsupplier2addr0',
    default_shipping_address_id = 'ad_01seedsupplier2addr0'
WHERE id = 'ac_01seedsupplier_acct1'
    AND default_billing_address_id IS NULL;

-- Backfill supplier account_relation defaults for DBs that were seeded before the
-- INSERTs above included these columns (INSERT IGNORE would skip the pre-existing rows).
UPDATE account_relation
   SET default_billing_address_id  = 'ad_01seedsupplieraddr00',
       default_shipping_address_id = 'ad_01seedsupplieraddr00'
 WHERE id = 'acre_01seedsupplier0000'
   AND (default_billing_address_id IS NULL OR default_shipping_address_id IS NULL);

UPDATE account_relation
   SET default_billing_address_id  = 'ad_01seedsupplier2addr0',
       default_shipping_address_id = 'ad_01seedsupplier2addr0'
 WHERE id = 'acre_01seedsupplier0001'
   AND (default_billing_address_id IS NULL OR default_shipping_address_id IS NULL);

-- Supplier materials (supplier_account_id is the counterparty account, not the relation ID)
INSERT IGNORE INTO supplier_material (id, material_id, supplier_account_id, supplier_part_number, supplier_description, is_active, owner_account_id, created_at, updated_at) VALUES
    ('spml_01seedsupmat1_0000', 'ml_01seedyrn1mat000000', 'ac_01seedsupplier_acct0', 'YRN-EXT-001', 'Premium Yarn Type 1', 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('spml_01seedsupmat2_0000', 'ml_01seedyrn2mat000000', 'ac_01seedsupplier_acct0', 'YRN-EXT-002', 'Premium Yarn Type 2', 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    -- Link a third yarn to a second supplier so the suppliers/item_ids array
    -- filter sees the top-of-feed material items linked to >=2 distinct suppliers.
    ('spml_01seedsupmat3_0000', 'ml_01seedyrn3mat000000', 'ac_01seedsupplier_acct1', 'YRN-EXT-003', 'Premium Yarn Type 3', 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- ============================================================
-- TERRITORIES (2 rows for pagination)
-- ============================================================

-- Note: the list-territories path is /v1/sales/accounts/{account_id}/territories.
-- The territory service overrides account_id with the authenticated user's account
-- (identity.Target.AccountID), so territories must belong to SeedAccountID.
INSERT IGNORE INTO territory (id, start_zipcode, end_zipcode, state, sales_rep_id, account_id, product_line_id, created_at, updated_at) VALUES
    ('tr_01seedterritory1_000', 10000, 19999, 'NY', 'acus_s83fjhyfmqen', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', NOW(), NOW()),
    ('tr_01seedterritory2_000', 90000, 99999, 'CA', 'acus_s83fjhyfmqen', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', NOW(), NOW());

-- ============================================================
-- SECOND INVOICE (for pagination on /v1/finance/invoices)
-- ============================================================

INSERT IGNORE INTO invoice (id, number, sales_order_id, billing_address_id, account_id, created_at, updated_at) VALUES
    ('iv_01seedsecondinvoice0', 'INV-002', 'or_01k0a8bs2ye3f9p8sj0m4dfmwe', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(), INTERVAL 3 DAY), DATE_SUB(NOW(), INTERVAL 3 DAY));

-- Invoice line quantities for INV-002 (packed order)
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedivln_pck_ln100', 20, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedivln_pck_ln200', 15, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedivln_pck_ln300', 1, 'each', NOW(), NOW());

INSERT IGNORE INTO invoice_line (id, invoice_id, quantity_id, sales_order_line_id, created_at, updated_at) VALUES
    ('ivln_01seedpck_ln1_0000', 'iv_01seedsecondinvoice0', 'qu_01seedivln_pck_ln100', 'orln_01seedpck_ln1_0000', NOW(), NOW()),
    ('ivln_01seedpck_ln2_0000', 'iv_01seedsecondinvoice0', 'qu_01seedivln_pck_ln200', 'orln_01seedpck_ln2_0000', NOW(), NOW()),
    ('ivln_01seedpck_ln3_0000', 'iv_01seedsecondinvoice0', 'qu_01seedivln_pck_ln300', 'orln_01seedpck_ln3_0000', NOW(), NOW());

-- ============================================================
-- INVOICE-SYNC ORDER + INVOICE (order-line → invoice-line quantity sync e2e)
-- Dedicated rows for tests that MUTATE the order line's quantity and assert the
-- invoice line follows. No other test may assert on these values.
-- ============================================================

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedsync_ln1_qty0', 25, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedsync_ivln1_q0', 25, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedsync_ivln2_q0', 10, 'un_01seedpair000000000', NOW(), NOW());

INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedsync_ln1_prc0', 9, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedsync_ln1_cst0', 8, 'dollar', 'un_01seedpair000000000', NOW(), NOW());

INSERT IGNORE INTO sales_order (id, number, sales_order_status_code, sales_order_type_code, priority_code, carrier_id, billing_address_id, shipping_address_id, buyer_account_id, seller_account_id, owner_account_id, payment_term_id, shipping_term_id, issued_at, created_at, updated_at) VALUES
    ('or_01seedsyncorder0000', 'ORD-SYNC-001', 'issued', 'sales_order', 'normal', 'delivery', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pytm_01seednet3000000', 'prepaid_billed', DATE_SUB(NOW(), INTERVAL 1 DAY), NOW(), NOW());

INSERT IGNORE INTO sales_order_line (id, product_sku, product_description, product_id, item_id, sales_order_id, quantity_id, unit_price_id, unit_cost_id, line_item_number, created_at, updated_at) VALUES
    ('orln_01seedsync_ln1_00', 'SCK-001', 'Small white sock', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'it_01k0a7100aeysrs9vxpeq14yxj', 'or_01seedsyncorder0000', 'qu_01seedsync_ln1_qty0', 'rt_01seedsync_ln1_prc0', 'rt_01seedsync_ln1_cst0', 1, NOW(), NOW());

INSERT IGNORE INTO invoice (id, number, sales_order_id, billing_address_id, account_id, created_at, updated_at) VALUES
    ('iv_01seedsyncinvoice00', 'INV-SYNC-001', 'or_01seedsyncorder0000', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('iv_01seedsyncinvoice02', 'INV-SYNC-002', 'or_01seedsyncorder0000', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- INV-SYNC-001's line mirrors the order line's full ordered quantity (25) and must
-- follow order-line edits; INV-SYNC-002's line is a partial (shipped-snapshot style)
-- quantity (10) and must NOT be touched by the sync.
INSERT IGNORE INTO invoice_line (id, invoice_id, quantity_id, sales_order_line_id, created_at, updated_at) VALUES
    ('ivln_01seedsync_ln1_00', 'iv_01seedsyncinvoice00', 'qu_01seedsync_ivln1_q0', 'orln_01seedsync_ln1_00', NOW(), NOW()),
    ('ivln_01seedsync_ln2_00', 'iv_01seedsyncinvoice02', 'qu_01seedsync_ivln2_q0', 'orln_01seedsync_ln1_00', NOW(), NOW());

-- Fully-picked pick and shipped shipment for ORD-SYNC-001. The packed pick line's
-- quantity (25 pair picked) is picking progress: only its UNIT follows order-line
-- edits. The shipment line (25 pair) mirrors the full ordered quantity and follows
-- edits the same way as the mirror invoice line.
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedsync_pkln1_q0', 25, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedsync_shln1_q0', 25, 'un_01seedpair000000000', NOW(), NOW());

INSERT IGNORE INTO pick (id, number, sales_order_id, account_id, created_at, updated_at) VALUES
    ('pk_01seedsyncpick00000', 'PICK-SYNC-001', 'or_01seedsyncorder0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

UPDATE pick SET finished_at = NOW() WHERE id = 'pk_01seedsyncpick00000' AND finished_at IS NULL;

INSERT IGNORE INTO pick_line (id, pick_id, quantity_id, sales_order_line_id, packed_at, created_at, updated_at) VALUES
    ('pkln_01seedsync_ln1_00', 'pk_01seedsyncpick00000', 'qu_01seedsync_pkln1_q0', 'orln_01seedsync_ln1_00', NOW(), NOW(), NOW());

INSERT IGNORE INTO shipment (id, number, sales_order_id, carrier_id, shipping_address_id, shipment_status_code, shipped_at, shipped_by_id, account_id, created_at, updated_at) VALUES
    ('sh_01seedsyncship00000', 'SHP-SYNC-001', 'or_01seedsyncorder0000', 'delivery', 'ad_01k09wnpvrea0awz7vem2j8j7g', 'shipped', NOW(), 'acus_s83fjhyfmqen', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

INSERT IGNORE INTO shipment_line (id, shipment_id, sales_order_line_id, quantity_id, created_at, updated_at) VALUES
    ('shln_01seedsync_ln1_00', 'sh_01seedsyncship00000', 'orln_01seedsync_ln1_00', 'qu_01seedsync_shln1_q0', NOW(), NOW());

-- ============================================================
-- SHIPMENT LINES (2 rows for the packed shipment)
-- ============================================================

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedshln1_qty00000', 20, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedshln2_qty00000', 15, 'un_01seedpair000000000', NOW(), NOW());

INSERT IGNORE INTO shipment_line (id, shipment_id, sales_order_line_id, quantity_id, created_at, updated_at) VALUES
    ('shln_01seedshpln1_00000', 'sh_01k0a87w33emw8pmkz1mf86cg1', 'orln_01seedpck_ln1_0000', 'qu_01seedshln1_qty00000', NOW(), NOW()),
    ('shln_01seedshpln2_00000', 'sh_01k0a87w33emw8pmkz1mf86cg1', 'orln_01seedpck_ln2_0000', 'qu_01seedshln2_qty00000', NOW(), NOW());

-- ============================================================
-- CARRIER OPTIONS (2 rows for carrier 'delivery')
-- ============================================================

INSERT IGNORE INTO carrier_option (id, code, name, service_level_token, carrier_id, account_id, created_at, updated_at) VALUES
    ('crop_01seedground000000', 'ground', 'Ground Shipping', 'fedex_ground', 'delivery', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('crop_01seedexpress00000', 'express', 'Express Shipping', 'fedex_express', 'delivery', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- System-owned (account_id = NULL) carrier + service level for write-guard e2e coverage.
INSERT IGNORE INTO carrier (id, code, name, account_id, created_at, updated_at) VALUES
    ('syscar_01seedsysdefault', 'delivery', 'System Default Delivery', NULL, NOW(), NOW());

INSERT IGNORE INTO carrier_option (id, code, name, carrier_id, account_id, created_at, updated_at) VALUES
    ('crop_01seedsysground000', 'ground', 'System Ground', 'syscar_01seedsysdefault', NULL, NOW(), NOW());

-- ============================================================
-- CARRIER TRANSIT FIXTURES
-- ============================================================
-- A carrier that can actually be rated. The seed 'delivery' carrier has no
-- shippo_carrier_account_id, so the rate path short-circuits before reaching the
-- stub and no lane is ever warmed -- which is why every pre-existing order test
-- sees a ship-by date equal to its promised date. Transit coverage needs a
-- carrier that gets as far as the carrier client.
INSERT IGNORE INTO carrier (id, code, name, shippo_carrier_account_id, account_id, created_at, updated_at) VALUES
    ('cr_01e2etransitcarrier', 'fedex', 'E2E Transit Carrier', 'shippoacct_e2e_transit', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- The tokens match the ones the Shippo stub quotes, so a warm files its estimates
-- against these rows. The last two carry no token on purpose: nothing the carrier
-- returns can match them, which is how a service the carrier will not rate behaves
-- (freight, will-call) and is the only way the service-level fallback is reachable.
INSERT IGNORE INTO carrier_option (id, code, name, service_level_token, default_transit_days, carrier_id, account_id, created_at, updated_at) VALUES
    ('crop_01e2etransitgrnd00', 'e2e_transit_ground', 'E2E Transit Ground', 'fedex_ground', NULL, 'cr_01e2etransitcarrier', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('crop_01e2etransit2day0', 'e2e_transit_2day', 'E2E Transit 2 Day', 'fedex_2_day', NULL, 'cr_01e2etransitcarrier', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('crop_01e2etransitovrn0', 'e2e_transit_overnight', 'E2E Transit Overnight', 'fedex_priority_overnight', NULL, 'cr_01e2etransitcarrier', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('crop_01e2etransitdflt0', 'e2e_transit_default', 'E2E Transit Unratable With Default', NULL, 5, 'cr_01e2etransitcarrier', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('crop_01e2etransitnone0', 'e2e_transit_none', 'E2E Transit Unratable No Default', NULL, NULL, 'cr_01e2etransitcarrier', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- DELIVERIES are seeded after the PURCHASE ORDERS block below, because a
-- delivery's sales_order_id references a purchase_order-type order (which is
-- defined further down in this file).

-- ============================================================
-- BATCHES (2 rows for scanning station batches)
-- ============================================================

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedbatch1_qty0000', 10, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedbatch2_qty0000', 8, 'un_01seedpair000000000', NOW(), NOW());

INSERT IGNORE INTO batch (id, account_id, item_id, quantity_id, scanning_station_id, production_step_id, production_run_id, scanned_at, created_at, updated_at) VALUES
    ('bt_01seedbatch1_0000000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedlknitem000000', 'qu_01seedbatch1_qty0000', 'sgsn_01k0a8201zegarjfsjaw5n7yfv', 'prs_01k0a51qxceydax5036pegvzzy', 'pnrn_01seedprod_run0000', NOW(), NOW(), NOW()),
    ('bt_01seedbatch2_0000000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedsknitem000000', 'qu_01seedbatch2_qty0000', 'sgsn_01k0a8201zegarjfsjaw5n7yfv', 'prs_01k0a575j3fqr97khk36v114nj', 'pnrn_01seedprod_run0000', NOW(), NOW(), NOW());

-- ============================================================
-- EDI RUNS (2 rows for pagination)
-- ============================================================

INSERT IGNORE INTO edi_run (id, account_id, completed_at, has_succeeded, created_at, updated_at) VALUES
    ('edir_01seededirun1_0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), 1, NOW(), NOW()),
    ('edir_01seededirun2_0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), 1, NOW(), NOW());

-- ============================================================
-- SALES TARGETS (2 rows for account-user sales targets)
-- ============================================================

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedtarget1_amt000', 50000, 'dollar', NOW(), NOW()),
    ('qu_01seedtarget2_amt000', 75000, 'dollar', NOW(), NOW());

INSERT IGNORE INTO target (id, start_date, end_date, sales_rep_id, account_id, amount_id, created_at, updated_at) VALUES
    ('tg_01seedtarget1_000000', DATE_SUB(NOW(), INTERVAL 30 DAY), DATE_ADD(NOW(), INTERVAL 60 DAY), 'acus_s83fjhyfmqen', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'qu_01seedtarget1_amt000', NOW(), NOW()),
    ('tg_01seedtarget2_000000', DATE_ADD(NOW(), INTERVAL 61 DAY), DATE_ADD(NOW(), INTERVAL 150 DAY), 'acus_s83fjhyfmqen', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'qu_01seedtarget2_amt000', NOW(), NOW());

-- ============================================================
-- SECOND PRODUCTION RUN (for pagination)
-- ============================================================

INSERT IGNORE INTO production_run (id, responsible_user_id, number, account_id, created_at, updated_at) VALUES
    ('pnrn_01seedprod_run0001', 'acus_s83fjhyfmqen', 'E2E-SEED-PR-002', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- ============================================================
-- SECOND DC LOCATION (for pagination)
-- ============================================================

INSERT IGNORE INTO dc_location (id, location, account_id, owner_account_id, created_at, updated_at) VALUES
    ('dclc_01seeddc_location1', 'Distribution Center West', 'ac_01seedcustomer2_acct0', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- ============================================================
-- SECOND SANDBOX (for pagination on /v1/core/sandboxes)
-- ============================================================

INSERT IGNORE INTO account (id, name, account_type_code, onboarding_status_code, created_at, updated_at) VALUES
    ('ac_sandbox_01seedsecond0', 'Sandbox 2', 'sandbox', 'active', NOW(), NOW());

INSERT IGNORE INTO sandbox_account (type_id, owner_account_id, account_id, created_at, updated_at) VALUES
    ('sbac_01seedsandbox2_0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_sandbox_01seedsecond0', NOW(), NOW());

-- ============================================================
-- SECOND ORDER DISCOUNT (for pagination)
-- ============================================================

INSERT IGNORE INTO order_discount (id, name, code, percentage, value, discount_type_code, account_id, created_at, updated_at) VALUES
    ('ords_01seedfixed5discount', '$5 Off', 'FIXED5', 0, 5, 'fixed', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- ============================================================
-- SECOND ACCOUNT PRICE (for pagination)
-- ============================================================

INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedacctprice2val0', 7.5, 'dollar', 'un_01seedpair000000000', NOW(), NOW());

INSERT IGNORE INTO account_price (id, owner_account_id, unit_value_id, product_line_id, recipient_account_id, created_at, updated_at) VALUES
    ('acpr_01seedaccprice0001', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'rt_01seedacctprice2val0', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', 'ac_01seedcustomer2_acct0', NOW(), NOW());

-- ============================================================
-- SECOND VOLUME DISCOUNT (for pagination)
-- ============================================================

INSERT IGNORE INTO quantity_discount (id, name, account_id, created_at, updated_at) VALUES
    ('quds_01seedvoldiscoun01', 'Bulk Volume Discount', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- ============================================================
-- SECOND SETTLEMENT, TRANSACTION, ALLOCATION (for pagination)
-- ============================================================

INSERT IGNORE INTO settlement (id, number, account_id, responsible_user_id, created_at, updated_at) VALUES
    ('sl_01seedsettlement001', 'STL-002', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', NOW(), NOW());

-- INV-002 total = 20×8.50 + 15×8.50 + 1×18 = 315.50
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedtx2_amount000', 315.50, 'dollar', NOW(), NOW()),
    ('qu_01seedtxal2_amount0', 315.50, 'dollar', NOW(), NOW());

INSERT IGNORE INTO transaction (id, number, customer_account_id, amount_id, transaction_type_code, is_fully_allocated, account_id, created_at, updated_at) VALUES
    ('tx_01seedtransaction01', 'TXN-002', 'ac_01k09wm2fgevdsc344gpbcj30f', 'qu_01seedtx2_amount000', 'credit_memo', 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

INSERT IGNORE INTO transaction_allocation (id, transaction_id, amount_id, invoice_id, settlement_id, created_at, updated_at) VALUES
    ('txal_01seedtxalloc0001', 'tx_01seedtransaction01', 'qu_01seedtxal2_amount0', 'iv_01seedsecondinvoice0', 'sl_01seedsettlement001', NOW(), NOW());

-- Mark INV-002 as paid (TXN-002 fully allocated to it)
UPDATE invoice SET is_paid_in_full = 1 WHERE id = 'iv_01seedsecondinvoice0' AND is_paid_in_full = 0;

-- ============================================================
-- ACCOUNT INTEGRATIONS (2 rows for pagination)
-- ============================================================

INSERT IGNORE INTO account_integration (id, account_id, integration_code, name, credentials_v2, is_active, created_at, updated_at) VALUES
    -- A real AES-GCM envelope rather than a placeholder, sealed with the e2e
    -- INTEGRATION_ENCRYPTION_KEY and the account ID as additional authenticated
    -- data. Anything that reaches a carrier decrypts these credentials first, so a
    -- placeholder fails the whole rate path with "invalid envelope format". That
    -- stayed invisible until a seeded carrier had a shippo_carrier_account_id, since
    -- without one the rate path returns before it ever looks at credentials.
    -- Plaintext: {"api_key":"shippo_test_e2e_stub_key"} — the stub ignores the value.
    ('acin_01seedintegration1', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'shippo', 'Shippo Integration', 'enc_v1_kdev-key-1_1nnZlpr3AyXho51Hb4NX-Mx-eGSDG1qe091TxAKVl8-3RjsbB9Ts-MsFreA5mysTOx0FSVFE_zgyIZA7nIpWWiNK', 1, NOW(), NOW()),
    ('acin_01seedintegration2', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'quickbooks', 'QuickBooks Integration', 'seed-placeholder-credentials', 1, NOW(), NOW());

-- ============================================================
-- HUBSPOT SYNC (1 job + 1 pending company review so the
-- hubspot-sync read endpoints resolve {id}/{review_id} and the
-- company-reviews list returns a real item)
-- ============================================================

INSERT IGNORE INTO hubspot_sync_job (id, account_id, status, dry_run, counts, created_at, updated_at) VALUES
    ('igjb_01seedhubspotjob1', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'review_pending', 1,
     '{"customers_total":120,"companies_confident":80,"companies_ambiguous":12,"companies_to_create":28,"contacts_with_email":95}',
     NOW(), NOW());

INSERT IGNORE INTO hubspot_company_review (id, job_id, account_id, augno_customer_id, customer_name, candidate_matches, status, created_at, updated_at) VALUES
    ('igrv_01seedhubspotrev1', 'igjb_01seedhubspotjob1', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k09wm2fgevdsc344gpbcj30f', 'E2E Review Customer',
     '[{"hubspot_id":"hs_company_1001","name":"Acme Co","domain":"acme.example"}]', 'pending', NOW(), NOW());

-- Two synced mappings so the sync-records list endpoint returns real rows and the pagination sweep (limit=1, then follow next_page_url) has a second page to walk. Both augno_ids point at real customers so the name join resolves; the list keysets on augno_id ASC, so ac_01k09... pages before ac_01seed....
INSERT IGNORE INTO hubspot_sync_record (id, account_id, augno_type, augno_id, hubspot_type, hubspot_id, last_synced_at, created_at, updated_at) VALUES
    ('igrd_01seedhubspotrec1', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'customer', 'ac_01k09wm2fgevdsc344gpbcj30f', 'companies', 'hs_company_1001', NOW(), NOW(), NOW()),
    ('igrd_01seedhubspotrec2', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'customer', 'ac_01seedcust3_acct000', 'companies', 'hs_company_1002', NOW(), NOW(), NOW());

-- ============================================================
-- EMAIL BRIDGE (1 verified domain + 1 active inbox so the
-- email-domains/email-inboxes list endpoints return a real item
-- and the update/{id} endpoints resolve their path param)
-- ============================================================

INSERT IGNORE INTO email_domain (id, account_id, domain, status, dkim_tokens, verified_at, created_at, updated_at) VALUES
    ('emdom_01seeddomain1_00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'mail.e2e.augno.com', 'verified',
     '["sel1._domainkey.mail.e2e.augno.com","sel2._domainkey.mail.e2e.augno.com"]', NOW(), NOW(), NOW());

INSERT IGNORE INTO email_inbox (id, account_id, email_domain_id, address, from_name, status, agent_config_id, created_at, updated_at) VALUES
    ('eminb_01seedinbox1_000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'emdom_01seeddomain1_00', 'support@mail.e2e.augno.com', 'E2E Support', 'active', 'agdf_01seede2e_orderbot0', NOW(), NOW());

-- ============================================================
-- AUDIT EVENTS (2 rows so audit event tests don't skip)
-- ============================================================

-- `changes` is stored as a JSON array of FieldChange records (see
-- shared/audit/types.go). Using the list form so the presenter's
-- AuditFieldChangesPresenter populates the nested list include.
-- The metadata-carrying event uses a far-future occurred_at so it stays at the
-- top of list-audit-events ordering (events are sorted by occurred_at DESC
-- and other e2e runs generate hundreds of newer events that would otherwise
-- push this one off the default page).
-- target_account_id is the account the mutation was performed against — here the
-- e2e account itself, which exists in `account` so the `account` sub-resource
-- include + the account_ids filter resolve a real account name.
INSERT IGNORE INTO audit_event (type_id, actor_id, actor_type, identity_type, account_id, target_account_id, action, resource_type, resource_id, changes, metadata, service_name, request_id, occurred_at, created_at) VALUES
    ('adev_01seedauditevent01', 'us_1wjfmmbwg8l7', 'user', 'user', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'create', 'unit', 'un_01seedpair000000000', '[{"field":"name","old_value":null,"new_value":"Pair"}]', NULL, 'core-service', NULL, DATE_SUB(NOW(), INTERVAL 1 HOUR), NOW()),
    ('adev_01seedauditevent02', 'us_1wjfmmbwg8l7', 'user', 'user', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'update', 'property', 'pp_01k0a7ntn1ez6aw8x850femxeh', '[{"field":"name","old_value":"Colour","new_value":"Color"}]', '{"seed":true,"note":"manual e2e seed"}', 'core-service', 'rqlog_01seedreqlog1_000', DATE_ADD(NOW(), INTERVAL 10 YEAR), DATE_ADD(NOW(), INTERVAL 10 YEAR));

-- Create event for the seed sales order so `?include=created_by` on that order
-- resolves a real internal creator (relation=internal + actor). actor_type holds
-- the relation (internal/customer/supplier); identity_type the kind (user/api_key).
-- SeedInternalSalesOrderID is intentionally left without a create event so the
-- created_by system-fallback (relation=system, actor=null) is also testable.
INSERT IGNORE INTO audit_event (type_id, actor_id, actor_type, identity_type, account_id, target_account_id, action, resource_type, resource_id, changes, metadata, service_name, request_id, occurred_at, created_at) VALUES
    ('adev_01seedsocreatedby0', 'us_1wjfmmbwg8l7', 'internal', 'user', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'create', 'sales_order', 'or_01k0a8bs2yejxbsvqhrx4drkq1', NULL, NULL, 'core-service', NULL, DATE_SUB(NOW(), INTERVAL 1 HOUR), NOW());

-- Backfill request_id + target_account_id on re-seed when INSERT IGNORE skips existing rows.
UPDATE audit_event SET request_id = 'rqlog_01seedreqlog1_000' WHERE type_id = 'adev_01seedauditevent02' AND (request_id IS NULL OR request_id = '');
UPDATE audit_event SET target_account_id = 'ac_01k0a5smf9ekb8rqg12555zjqa' WHERE type_id IN ('adev_01seedauditevent01', 'adev_01seedauditevent02') AND target_account_id IS NULL;

-- Audit event that populates source_ip + idempotency_key_id (both columns exist
-- but are NULL on every other seed row). source_ip is a stable test value
-- (SeedAuditEventSourceIP); idempotency_key_id reuses idk_01seedreqlogik001 so the
-- joined idempotency_key surfaces as 'e2e-seed-idempotency-key-01'
-- (SeedAuditEventIdempotencyKey). Far-future occurred_at keeps it a stable
-- GET-by-id target (SeedAuditEventWithSourceIPID).
INSERT IGNORE INTO audit_event (type_id, actor_id, actor_type, identity_type, account_id, target_account_id, action, resource_type, resource_id, changes, metadata, service_name, request_id, idempotency_key_id, source_ip, occurred_at, created_at) VALUES
    ('adev_01seedsrcipkey0', 'us_1wjfmmbwg8l7', 'user', 'user', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'update', 'unit', 'un_01seedpair000000000', '[{"field":"name","old_value":"Pair","new_value":"Pair"}]', NULL, 'core-service', NULL, 'idk_01seedreqlogik001', '198.51.100.42', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR));
-- Re-seed-safe backfill in case the row pre-exists without these columns set.
UPDATE audit_event SET idempotency_key_id = 'idk_01seedreqlogik001', source_ip = '198.51.100.42' WHERE type_id = 'adev_01seedsrcipkey0' AND (source_ip IS NULL OR source_ip = '');

-- Search + multi-actor fixtures.
--  * adev_01seedsearchtgt01 carries a distinctive resource_id and request_id so
--    the free-text search ('q') tests can paste an id and find the event. Audit
--    search now matches resource_id and request_id in addition to resource_type
--    and action (see queries/audit_event.sql). Far-future occurred_at keeps it on
--    the first page despite hundreds of newer events other e2e runs generate.
--  * adev_01seedactor2event0 is authored by the seed API key (a second, distinct
--    actor) so the multi-actor union filter test has two real actors to combine.
INSERT IGNORE INTO audit_event (type_id, actor_id, actor_type, identity_type, account_id, action, resource_type, resource_id, changes, metadata, service_name, request_id, occurred_at, created_at) VALUES
    ('adev_01seedsearchtgt01', 'us_1wjfmmbwg8l7', 'user', 'user', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'create', 'item', 'it_01seedauditsrchtgt', '[{"field":"name","old_value":null,"new_value":"Audit Search Target"}]', NULL, 'core-service', 'rqlog_01seedauditsrchrq', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR)),
    ('adev_01seedactor2event0', 'apky_pajbskcck3cabxajdh8h8', 'api_key', 'api_key', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'create', 'item', 'it_01seedauditactor2', '[{"field":"name","old_value":null,"new_value":"Audit Actor Two"}]', NULL, 'core-service', NULL, DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR));

-- actor-or-target scope cohort — the caller is the seed account; the list scope
-- returns events where the seed account is EITHER the acting account (account_id)
-- OR the target account (target_account_id). Each row has a distinct resource_id
-- so the tests scope to this cohort via the resource_ids filter, then assert which
-- rows the scope + the actor_account_ids / target_account_ids filters return:
--   scope-actor   account_id=seed,     target=customer  -> visible via actor
--   scope-target  account_id=customer, target=seed      -> visible via target
--   scope-both    account_id=seed,     target=seed       -> visible via both
--   scope-neither account_id=child,    target=customer  -> NEVER visible (out of scope)
INSERT IGNORE INTO audit_event (type_id, actor_id, actor_type, identity_type, account_id, target_account_id, action, resource_type, resource_id, changes, metadata, service_name, request_id, occurred_at, created_at) VALUES
    ('adev_01seedscopeactor', 'us_1wjfmmbwg8l7', 'user', 'user', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k09wm2fgevdsc344gpbcj30f', 'create', 'item', 'it_01seedauditscopeac', NULL, NULL, 'core-service', NULL, DATE_ADD(NOW(), INTERVAL 8 YEAR), DATE_ADD(NOW(), INTERVAL 8 YEAR)),
    ('adev_01seedscopetarget', 'us_1wjfmmbwg8l7', 'user', 'user', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'create', 'item', 'it_01seedauditscopetg', NULL, NULL, 'core-service', NULL, DATE_ADD(NOW(), INTERVAL 8 YEAR), DATE_ADD(NOW(), INTERVAL 8 YEAR)),
    ('adev_01seedscopeboth00', 'us_1wjfmmbwg8l7', 'user', 'user', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'create', 'item', 'it_01seedauditscopebt', NULL, NULL, 'core-service', NULL, DATE_ADD(NOW(), INTERVAL 8 YEAR), DATE_ADD(NOW(), INTERVAL 8 YEAR)),
    ('adev_01seedscopeneither', 'us_1wjfmmbwg8l7', 'user', 'user', 'ac_01seedchild_acct0001', 'ac_01k09wm2fgevdsc344gpbcj30f', 'create', 'item', 'it_01seedauditscopenn', NULL, NULL, 'core-service', NULL, DATE_ADD(NOW(), INTERVAL 8 YEAR), DATE_ADD(NOW(), INTERVAL 8 YEAR));

-- ============================================================
-- CATALOG ATTRIBUTE LINKS (give materials and parts a 2nd distinct
-- attribute so the array-filter union tests for attribute_ids have two
-- values to combine — see TestArrayFilters_UnionExclusion). `_item_attributes`
-- is the Prisma m2m join table (A = attribute id, B = item id).
-- ============================================================

INSERT IGNORE INTO _item_attributes (A, B) VALUES
    ('at_01seedblack00000000', 'it_01seedyrn1item00000'),  -- material (yarn) gets Black, alongside existing Beige material tags
    ('at_01seedsmall00000000', 'it_01seedlknitem000000');  -- part (large knitted sock) gets Small, alongside existing Large part tags

-- ============================================================
-- EMAIL LOGS (2 rows for pagination)
-- ============================================================

INSERT IGNORE INTO email_log (id, has_sent, account_id, sent_by_id, subject, created_at, updated_at) VALUES
    ('emlog_01seedemaillog1_0', 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'Order Confirmation - ORD-001', NOW(), NOW()),
    ('emlog_01seedemaillog2_0', 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'Shipment Notification - SHP-001', NOW(), NOW());

INSERT IGNORE INTO email_recipient (id, email, email_log_id, created_at, updated_at) VALUES
    ('emrp_01seedemailrcpt1_0', 'customer@example.com', 'emlog_01seedemaillog1_0', NOW(), NOW()),
    ('emrp_01seedemailrcpt2_0', 'warehouse@example.com', 'emlog_01seedemaillog2_0', NOW(), NOW());

-- ============================================================
-- REQUEST LOGS (5 rows — 3 for pagination, 1 with an error_code, 1 with query_json for include tests)
-- actor_id stores the raw actor id the API exposes: user_id (us_…) for user
-- actors, api_key type_id (apky_…) for api keys.
-- ============================================================

INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, occurred_at, created_at) VALUES
    ('rqlog_01seedreqlog1_000', 'GET', 'api.augno.com', '/v1/catalog/items', '/v1/catalog/items', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', DATE_SUB(NOW(), INTERVAL 1 HOUR), NOW()),
    ('rqlog_01seedreqlog2_000', 'POST', 'api.augno.com', '/v1/catalog/units', '/v1/catalog/units', 201, 25000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', NOW(), NOW()),
    ('rqlog_01seedreqlog3_000', 'GET', 'api.augno.com', '/v1/catalog/items', '/v1/catalog/items', 200, 12000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'apky_pajbskcck3cabxajdh8h8', 'api_key', 'api_key', DATE_SUB(NOW(), INTERVAL 30 MINUTE), NOW());

INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, error_code, error_message, occurred_at, created_at) VALUES
    ('rqlog_01seedreqlog4_000', 'POST', 'api.augno.com', '/v1/catalog/units', '/v1/catalog/units', 422, 9000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', 'validation_failed', 'Name is required.', DATE_SUB(NOW(), INTERVAL 2 HOUR), NOW());

INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, query_json, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, occurred_at, created_at) VALUES
    ('rqlog_01seedreqlog5_000', 'GET', 'api.augno.com', '/v1/catalog/items', '/v1/catalog/items', '{"limit":10,"status_codes":["200"]}', 200, 8000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', DATE_SUB(NOW(), INTERVAL 3 HOUR), NOW());

-- Search target: a distinctive resource id embedded in the request path so the
-- free-text search ('q') test can paste that id and find the log. Request log
-- search matches the literal path (and normalized_route) — see
-- repository/request_log_list_query.go.
INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, occurred_at, created_at) VALUES
    ('rqlog_01seedsearchtgt0', 'GET', 'api.augno.com', '/v1/catalog/items/it_01seedreqlogsrchtgt', '/v1/catalog/items/{id}', 200, 11000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', DATE_SUB(NOW(), INTERVAL 20 MINUTE), NOW());

-- Multi-actor union fixtures. One user-authored and one api_key-authored log,
-- both with a far-future occurred_at so they stay at the top of any
-- actor-filtered page. Without this the harness's own continuous api_key traffic
-- buries the (older) user-authored seed rows past the first page, so a union
-- filter test could not observe a result for the user actor even though the
-- filter returns it. See TestRequestLogs_ListFilterByMultipleActorsUnion.
INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, occurred_at, created_at) VALUES
    ('rqlog_01sedunionuser0', 'GET', 'api.augno.com', '/v1/catalog/items', '/v1/catalog/items', 200, 10000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR)),
    ('rqlog_01sedunionapik0', 'GET', 'api.augno.com', '/v1/catalog/units', '/v1/catalog/units', 200, 10000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'apky_pajbskcck3cabxajdh8h8', 'api_key', 'api_key', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR)),
    -- A far-future POST/201 row so the methods and status_codes filter discoveries
    -- always see >=2 distinct values on the first page (the harness's own traffic is
    -- all GET/200, and the seeded POST/422 rows are old enough to be buried).
    ('rqlog_01sedunionpost0', 'POST', 'api.augno.com', '/v1/catalog/units', '/v1/catalog/units', 201, 10000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR));

INSERT IGNORE INTO idempotency_key (type_id, idempotency_key, identity_type, request_method, normalized_route, request_body_hash, scope_hash, recovery_point, target_account_id, actor_id, created_at, updated_at) VALUES
    ('idk_01seedreqlogik001', 'e2e-seed-idempotency-key-01', 'user', 'POST', '/v1/catalog/units', 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb', 'finished', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', NOW(3), NOW(3));

UPDATE request_log SET idempotency_key_id = 'idk_01seedreqlogik001' WHERE id = 'rqlog_01seedreqlog2_000';

-- ============================================================
-- REQUEST LOG FILTER COHORTS (robust per-filter e2e coverage)
-- ============================================================
-- The e2e harness emits a continuous stream of its own request_log rows against
-- the seed account, so a filter test cannot prove exclusion by counting rows or
-- by inspecting the "most recent" page — the harness noise drowns the seed rows.
--
-- These cohorts give every filter dimension a private, deterministic universe:
--   * Each cohort shares a distinctive scope value the harness NEVER produces — a
--     synthetic normalized_route (e.g. /filtertest/methods) for most cohorts, or a
--     synthetic host (rqlog-route-e2e.test) for the route-filter cohort whose
--     route is itself under test.
--   * A test ANDs that scope value with the filter under test, so the result set
--     is exactly the cohort's rows — harness traffic is excluded because it never
--     carries the synthetic scope value (filters AND across dimensions).
--   * Each cohort has THREE rows with three distinct values of the dimension; the
--     test filters by two of them and asserts both are returned and the third is
--     excluded entirely.
--
-- occurred_at is pinned in the past so these rows never surface in the harness-
-- dominated "recent" window that the pre-existing discovery/shape tests sample;
-- the filter tests below locate them by scope value, not by recency.
-- See tests/e2e/api/crud_request_logs_test.go.

-- methods cohort — scope normalized_route=/filtertest/methods, vary method.
INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, occurred_at, created_at) VALUES
    ('rqlog_01fltmethget00', 'GET',  'rqlog-filter-e2e.test', '/filtertest/methods', '/filtertest/methods', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01fltmethpost0', 'POST', 'rqlog-filter-e2e.test', '/filtertest/methods', '/filtertest/methods', 201, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01fltmethput00', 'PUT',  'rqlog-filter-e2e.test', '/filtertest/methods', '/filtertest/methods', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW());

-- status_codes cohort — scope normalized_route=/filtertest/status, vary status_code.
INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, occurred_at, created_at) VALUES
    ('rqlog_01fltstat20000', 'GET', 'rqlog-filter-e2e.test', '/filtertest/status', '/filtertest/status', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01fltstat40400', 'GET', 'rqlog-filter-e2e.test', '/filtertest/status', '/filtertest/status', 404, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01fltstat50000', 'GET', 'rqlog-filter-e2e.test', '/filtertest/status', '/filtertest/status', 500, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW());

-- account_ids cohort — scope normalized_route=/filtertest/accounts, vary acting
-- account_id while keeping target_account_id = the seed (caller) account so all
-- three rows are visible. account_id is NOT surfaced in the response (the API
-- exposes target_account_id as `account`), so the test verifies the filter by
-- which seed-row IDs come back, not by reading a field.
INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, occurred_at, created_at) VALUES
    ('rqlog_01fltacct1000', 'GET', 'rqlog-filter-e2e.test', '/filtertest/accounts', '/filtertest/accounts', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01fltacct2000', 'GET', 'rqlog-filter-e2e.test', '/filtertest/accounts', '/filtertest/accounts', 200, 15000, 1, 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01fltacct3000', 'GET', 'rqlog-filter-e2e.test', '/filtertest/accounts', '/filtertest/accounts', 200, 15000, 1, 'ac_01seedchild_acct0001', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW());

-- actor-or-target scope cohort — scope normalized_route=/filtertest/scope. The
-- caller is the seed account; the list scope returns rows where the seed account
-- is EITHER the acting account (account_id) OR the target account
-- (target_account_id). The four rows cover every quadrant so the scope + the
-- explicit actor_account_ids / target_account_ids filters can be verified:
--   scope-actor   account_id=seed,     target=customer  -> visible via actor
--   scope-target  account_id=customer, target=seed      -> visible via target
--   scope-both    account_id=seed,     target=seed       -> visible via both
--   scope-neither account_id=child,    target=customer  -> NEVER visible (out of scope)
INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, occurred_at, created_at) VALUES
    ('rqlog_01fltscopeactr', 'GET', 'rqlog-filter-e2e.test', '/filtertest/scope', '/filtertest/scope', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k09wm2fgevdsc344gpbcj30f', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01fltscopetgt0', 'GET', 'rqlog-filter-e2e.test', '/filtertest/scope', '/filtertest/scope', 200, 15000, 1, 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01fltscopeboth', 'GET', 'rqlog-filter-e2e.test', '/filtertest/scope', '/filtertest/scope', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01fltscopenone', 'GET', 'rqlog-filter-e2e.test', '/filtertest/scope', '/filtertest/scope', 200, 15000, 1, 'ac_01seedchild_acct0001', 'ac_01k09wm2fgevdsc344gpbcj30f', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW());

-- actor_ids cohort — scope normalized_route=/filtertest/actorids, vary the user
-- actor (User1 / User2 / User3). Filtering by User1+User2 must include both and
-- exclude User3 (us_fltactor3, seeded in 0004_auth.sql).
INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, occurred_at, created_at) VALUES
    ('rqlog_01fltactid100', 'GET', 'rqlog-filter-e2e.test', '/filtertest/actorids', '/filtertest/actorids', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01fltactid200', 'GET', 'rqlog-filter-e2e.test', '/filtertest/actorids', '/filtertest/actorids', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_6p7460uuwibz', 'user', 'user', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01fltactid300', 'GET', 'rqlog-filter-e2e.test', '/filtertest/actorids', '/filtertest/actorids', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_fltactor3',    'user', 'user', '2022-01-01 00:00:00', NOW());

-- actor_types cohort — scope normalized_route=/filtertest/actortypes, vary
-- identity_type (user / api_key / internal). The actor_types filter matches the
-- identity_type column. Filtering by user+api_key must include those two and
-- exclude the internal-actor row.
INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, occurred_at, created_at) VALUES
    ('rqlog_01flttypeusr0', 'GET', 'rqlog-filter-e2e.test', '/filtertest/actortypes', '/filtertest/actortypes', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7',            'user',    'user',     '2022-01-01 00:00:00', NOW()),
    ('rqlog_01flttypekey0', 'GET', 'rqlog-filter-e2e.test', '/filtertest/actortypes', '/filtertest/actortypes', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'apky_pajbskcck3cabxajdh8h8', 'api_key', 'api_key',  '2022-01-01 00:00:00', NOW()),
    ('rqlog_01flttypeint0', 'GET', 'rqlog-filter-e2e.test', '/filtertest/actortypes', '/filtertest/actortypes', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', NULL,                        'internal','internal', '2022-01-01 00:00:00', NOW());

-- infra-scrub cohort — agent requests come through the gateway's internal listener,
-- so their stored host (internal k8s service name:port) and client_ip (pod IP) are
-- internal infrastructure that must NEVER reach customers. The customer-facing
-- presenter rewrites host -> the public API host and drops client_ip for
-- identity_type='agent' (apiresource.RequestLog.ScrubInternalInfra), while user/
-- api_key logs keep theirs. The agent's actor_id is the agent definition
-- agdf_01infraseedagent (seeded in agent-service), whose name + slug the request-logs
-- presenter hydrates onto the actor. api_version is set so the customer-facing log
-- shows the version the agent's internal call carried (the agent client sends
-- Augno-Version). The agent row's id is also referenced by an audit_event
-- (request_id) below so the audit-event ?include=request path is covered too. These
-- rows are fetched by the normalized_route scope filter and by id (both
-- date-independent), so they are dated in the past (like the other filter cohorts) to
-- stay OUT of the default recent listing. See crud_request_logs_test.go /
-- crud_audit_events_test.go.
INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, api_version, client_ip_string, user_agent, occurred_at, created_at) VALUES
    ('rqlog_01infraagent0', 'GET', 'api-gateway-internal:8091', '/v1/sales/customers', '/filtertest/infra-scrub', 200, 295867, 0, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01infraseedagent', 'internal', 'agent', '1.0.forge-preview.2', '10.244.0.18',  'Go-http-client/1.1', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01infrauser00', 'GET', 'api.augno.com',             '/v1/sales/customers', '/filtertest/infra-scrub', 200, 15000,  1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7',       'user',     'user',  '1.0.forge-preview.2', '198.51.100.7', 'Mozilla/5.0',        '2022-01-01 00:00:00', NOW());

-- referrer is the only optional request_log field no seed row sets; put it on the
-- stable infra-scrub user row (SeedReqLogInfraUserID) so the allFields test can
-- assert a non-null referrer (SeedRequestLogReferrerValue). Re-seed-safe UPDATE
-- because the INSERT above omits the column.
UPDATE request_log SET referrer = 'https://dashboard.augno.com/inbox' WHERE id = 'rqlog_01infrauser00' AND (referrer IS NULL OR referrer = '');

-- Audit event whose request_id points at the agent request_log above, so the
-- audit-event ?include=request expansion is exercised against an internal/agent log.
-- The embedded request_log must be scrubbed there too (it never carries host/ip).
-- Fetched by id, so the past date keeps it out of default audit-event listings.
INSERT IGNORE INTO audit_event (type_id, actor_id, actor_type, identity_type, account_id, target_account_id, action, resource_type, resource_id, changes, metadata, service_name, request_id, occurred_at, created_at) VALUES
    ('adev_01infraauditreq0', 'us_1wjfmmbwg8l7', 'user', 'user', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'update', 'customer', 'ac_01seedcustomer2_acct0', NULL, NULL, 'api-gateway', 'rqlog_01infraagent0', '2022-01-01 00:00:00', NOW());
-- Backfill request_id on re-seed when INSERT IGNORE skips the existing audit row.
UPDATE audit_event SET request_id = 'rqlog_01infraagent0' WHERE type_id = 'adev_01infraauditreq0' AND (request_id IS NULL OR request_id = '');

-- normalized_routes cohort — scope host=rqlog-route-e2e.test (the route is the
-- dimension under test, so the host is the scope), vary normalized_route.
INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, occurred_at, created_at) VALUES
    ('rqlog_01fltroutea00', 'GET', 'rqlog-route-e2e.test', '/filtertest/route-a', '/filtertest/route-a', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01fltrouteb00', 'GET', 'rqlog-route-e2e.test', '/filtertest/route-b', '/filtertest/route-b', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01fltroutec00', 'GET', 'rqlog-route-e2e.test', '/filtertest/route-c', '/filtertest/route-c', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW());

-- normalized_route param-name drift cohort — scope host=rqlog-drift-e2e.test.
-- The stored normalized_route uses the Go router's snake_case param name
-- ({unit_group_id}); the dashboard endpoint filter derives its templates from
-- the Stainless public spec, which camelCases multi-word path params
-- ({unitGroupId}). The filter compares on route shape (param names collapsed to
-- {}), so the camelCase template must still match this snake_case row. Regression
-- guard for the endpoint filter silently returning zero results on multi-word
-- path params — see normalizeRouteParams in the platform-service query builder.
INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, occurred_at, created_at) VALUES
    ('rqlog_01fltdrift000', 'GET', 'rqlog-drift-e2e.test', '/v1/catalog/unit-groups/ug_seeddrift00/units', '/v1/catalog/unit-groups/{unit_group_id}/units', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW());

-- hosts cohort — scope normalized_route=/filtertest/hosts (the host is the
-- dimension under test), vary host.
INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, occurred_at, created_at) VALUES
    ('rqlog_01flthosta000', 'GET', 'rqlog-hosta-e2e.test', '/filtertest/hosts', '/filtertest/hosts', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01flthostb000', 'GET', 'rqlog-hostb-e2e.test', '/filtertest/hosts', '/filtertest/hosts', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01flthostc000', 'GET', 'rqlog-hostc-e2e.test', '/filtertest/hosts', '/filtertest/hosts', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW());

-- min_latency_us cohort — scope normalized_route=/filtertest/latency, vary latency.
INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, occurred_at, created_at) VALUES
    ('rqlog_01fltlatlo000', 'GET', 'rqlog-filter-e2e.test', '/filtertest/latency', '/filtertest/latency', 200, 1000,   1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01fltlatmid00', 'GET', 'rqlog-filter-e2e.test', '/filtertest/latency', '/filtertest/latency', 200, 50000,  1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01fltlathi000', 'GET', 'rqlog-filter-e2e.test', '/filtertest/latency', '/filtertest/latency', 200, 100000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2022-01-01 00:00:00', NOW());

-- date-range cohort — scope normalized_route=/filtertest/dates, vary occurred_at
-- with fixed absolute timestamps so the start_date / end_date boundary tests are
-- deterministic. old=2023-01-01, mid=2023-06-01, new=2023-12-01.
INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, occurred_at, created_at) VALUES
    ('rqlog_01fltdateold0', 'GET', 'rqlog-filter-e2e.test', '/filtertest/dates', '/filtertest/dates', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2023-01-01 00:00:00', NOW()),
    ('rqlog_01fltdatemid0', 'GET', 'rqlog-filter-e2e.test', '/filtertest/dates', '/filtertest/dates', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2023-06-01 00:00:00', NOW()),
    ('rqlog_01fltdatenew0', 'GET', 'rqlog-filter-e2e.test', '/filtertest/dates', '/filtertest/dates', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', '2023-12-01 00:00:00', NOW());

-- error_codes cohort — scope normalized_route=/filtertest/errors, vary error_code.
INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, error_code, error_message, occurred_at, created_at) VALUES
    ('rqlog_01flterrnf000', 'GET',  'rqlog-filter-e2e.test', '/filtertest/errors', '/filtertest/errors', 404, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', 'resource_not_found', 'Resource not found.', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01flterrvf000', 'POST', 'rqlog-filter-e2e.test', '/filtertest/errors', '/filtertest/errors', 422, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', 'validation_failed',  'Validation failed.', '2022-01-01 00:00:00', NOW()),
    ('rqlog_01flterrua000', 'GET',  'rqlog-filter-e2e.test', '/filtertest/errors', '/filtertest/errors', 401, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', 'invalid_credentials', 'Unauthorized.',      '2022-01-01 00:00:00', NOW());

-- ============================================================
-- ADDRESSES linked to OWNER ACCOUNT (for /v1/sales/addresses pagination)
-- The endpoint queries account_address.account_id = owner_account_id.
-- ============================================================

INSERT IGNORE INTO geolocation (id, street_line_1, locality, state, postal_code, country, created_at, updated_at) VALUES
    ('gl_01seedownerbilling00', '123 Warehouse Ave', 'Columbus', 'OH', '43201', 'US', NOW(), NOW()),
    ('gl_01seedownershipping0', '456 Factory Ln', 'Columbus', 'OH', '43202', 'US', NOW(), NOW()),
    ('gl_01seedownerreturn000', '789 Return Center Dr', 'Cleveland', 'OH', '44101', 'US', NOW(), NOW()),
    ('gl_01seedownerdropship0', '321 Drop Ship Way', 'Cincinnati', 'OH', '45201', 'US', NOW(), NOW());

INSERT IGNORE INTO address (id, name, geolocation_id, created_at, updated_at) VALUES
    ('ad_01seedowneraddress01', 'Acme Inc HQ', 'gl_01seedownerbilling00', NOW(), NOW()),
    ('ad_01seedowneraddress02', 'Acme Inc Warehouse', 'gl_01seedownershipping0', NOW(), NOW()),
    ('ad_01seedowneraddress03', 'Acme Inc Returns', 'gl_01seedownerreturn000', NOW(), NOW()),
    ('ad_01seedowneraddress04', 'Acme Inc Drop Ship', 'gl_01seedownerdropship0', NOW(), NOW());

INSERT IGNORE INTO account_address (id, account_id, address_id, created_at, updated_at) VALUES
    ('acad_01seedownerbill000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ad_01seedowneraddress01', NOW(), NOW()),
    ('acad_01seedownership000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ad_01seedowneraddress02', NOW(), NOW()),
    ('acad_01seedownerreturn0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ad_01seedowneraddress03', NOW(), NOW()),
    ('acad_01seedownerdrop000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ad_01seedowneraddress04', NOW(), NOW()),
    ('acad_01seedcustbillmain', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ad_01k09wnac0e1ar211e0sy0ba4g', NOW(), NOW());

-- ============================================================
-- CATALOG PRODUCTS: set is_portal_ready = 1 (required by catalog endpoint)
-- ============================================================

UPDATE product SET is_portal_ready = 1
    WHERE id IN (
        'pd_01k0a65nx2e2crfxrvryyxnmdh',
        'pd_01k0a65nx5e3haz2fgfm34hmcz',
        'pd_01k0a65nx5fjz8m1s3ytayfdby',
        'pd_01k0a65nx5eeavcs322b06pgr8',
        'pd_01k0a65nx5fwmt17sqp317ekyr',
        'pd_01k0a65nx5e67rd1rahv4tdnrp'
    );

-- ============================================================
-- THIRD TRANSACTION (not fully allocated — for /v1/finance/open-credits pagination)
-- ============================================================

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedtx3_amount000', 150, 'dollar', NOW(), NOW());

INSERT IGNORE INTO transaction (id, number, customer_account_id, amount_id, transaction_type_code, is_fully_allocated, account_id, created_at, updated_at) VALUES
    ('tx_01seedtransaction02', 'TXN-003', 'ac_01k09wm2fgevdsc344gpbcj30f', 'qu_01seedtx3_amount000', 'credit_memo', 0, 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- ============================================================
-- CHILD ACCOUNTS (parent-child account setup for /v1/identity/child-accounts)
-- The endpoint uses ActorAccountID (API key owner) as the relation owner and
-- TargetAccountID (Augno-Account header) as the parent. Since the e2e test
-- client sends Augno-Account: SeedAccountID, we create a "house account"
-- self-relation so FindRelationByOwnerAndCounterparty(SeedAccountID, SeedAccountID)
-- resolves, then create child relations pointing to it.
-- ============================================================

-- House account self-relation (SeedAccountID as its own customer)
INSERT IGNORE INTO account_relation (id, owner_account_id, counterparty_account_id, account_relation_role_code, external_number, is_edi_enabled, priority_code, account_status_code, commission_status_code, freight_status_code, shipping_term_id, payment_term_id, account_group_id, created_at, updated_at) VALUES
    ('acre_01seedhouseacct0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'customer', 'HOUSE-001', 0, 'normal', 'normal', 'commission_exempt', 'billed_freight', 'prepaid_billed', 'pytm_01seednet3000000', 'acgp_01k0a413mjeth8pe1g70t0thax', NOW(), NOW());

-- Child account 1
INSERT IGNORE INTO account (id, name, account_type_code, onboarding_status_code, account_plan_id, created_at, updated_at) VALUES
    ('ac_01seedchild_acct0001', 'East Division', 'company', 'unclaimed', 'acpl_01seed000free00plan000000', NOW(), NOW());

INSERT IGNORE INTO account_relation (id, owner_account_id, counterparty_account_id, account_relation_role_code, external_number, parent_account_relation_id, is_edi_enabled, priority_code, account_status_code, commission_status_code, freight_status_code, shipping_term_id, payment_term_id, account_group_id, created_at, updated_at) VALUES
    ('acre_01seedchild_rel001', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01seedchild_acct0001', 'customer', 'CHILD-001', 'acre_01seedhouseacct0000', 0, 'normal', 'normal', 'commission_applied', 'billed_freight', 'prepaid_billed', 'pytm_01seednet3000000', 'acgp_01k0a413mjeth8pe1g70t0thax', NOW(), NOW());

-- Child account 2 (for pagination)
INSERT IGNORE INTO account (id, name, account_type_code, onboarding_status_code, account_plan_id, created_at, updated_at) VALUES
    ('ac_01seedchild_acct0002', 'West Division', 'company', 'unclaimed', 'acpl_01seed000free00plan000000', NOW(), NOW());

INSERT IGNORE INTO account_relation (id, owner_account_id, counterparty_account_id, account_relation_role_code, external_number, parent_account_relation_id, is_edi_enabled, priority_code, account_status_code, commission_status_code, freight_status_code, shipping_term_id, payment_term_id, account_group_id, created_at, updated_at) VALUES
    ('acre_01seedchild_rel002', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01seedchild_acct0002', 'customer', 'CHILD-002', 'acre_01seedhouseacct0000', 0, 'normal', 'normal', 'commission_applied', 'billed_freight', 'prepaid_billed', 'pytm_01seednet3000000', 'acgp_01k0a413mjeth8pe1g70t0thax', NOW(), NOW());

-- A user + account_user scoped to child account 1, distinct from the seed
-- account. Used as the cross-account block target (SeedChildAccountUserID): the
-- block-resolution path resolves account_user -> user_id without an account
-- filter, so this row lets the messaging_blocks cross-account test pin down
-- actual behavior. Named user so ?include=blocked_user resolves a real name.
INSERT IGNORE INTO user (id, email, name, username, status_code) VALUES
    ('us_childblktgt00', 'child-block-target@e2e.augno.com', 'Blocked Child User', 'childblocktgt', 'active');
INSERT IGNORE INTO account_user (id, user_id, account_id, status_code) VALUES
    ('acus_childblktgt', 'us_childblktgt00', 'ac_01seedchild_acct0001', 'active');

-- ============================================================
-- PURCHASE ORDERS (e2e include coverage)
-- ============================================================
-- A purchase_order is a sales_order row with sales_order_type_code = 'purchase_order',
-- where seller_account_id is the supplier account. Seeded supplier relations:
--   acre_01seedsupplier0000 → ac_01seedsupplier_acct0, acre_01seedsupplier0001 → ac_01seedsupplier_acct1.

INSERT IGNORE INTO sales_order (id, number, sales_order_status_code, sales_order_type_code, priority_code, carrier_id, billing_address_id, shipping_address_id, buyer_account_id, seller_account_id, owner_account_id, payment_term_id, shipping_term_id, issued_at, created_at, updated_at) VALUES
    ('or_01seedpurchord1_000', 'PO-001', 'issued', 'purchase_order', 'normal', 'delivery', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01seedsupplier_acct0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pytm_01seednet3000000', 'prepaid_billed', DATE_SUB(NOW(), INTERVAL 2 DAY), NOW(), NOW());

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedpoln1_qty00000', 50, 'un_01seedpound00000000', NOW(), NOW());

INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedpoln1_price000', '5.00', 'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_01seedpoln1_cost0000', '4.00', 'dollar', 'un_01seedpound00000000', NOW(), NOW());

INSERT IGNORE INTO sales_order_line (id, product_sku, product_description, product_id, item_id, sales_order_id, quantity_id, unit_price_id, unit_cost_id, created_at, updated_at) VALUES
    ('orln_01seedpoln1_000000', 'YRN-001', 'Small white yarn for PO', NULL, 'it_01seedyrn1item00000', 'or_01seedpurchord1_000', 'qu_01seedpoln1_qty00000', 'rt_01seedpoln1_price000', 'rt_01seedpoln1_cost0000', NOW(), NOW());

INSERT IGNORE INTO sales_order (id, number, sales_order_status_code, sales_order_type_code, priority_code, carrier_id, billing_address_id, shipping_address_id, buyer_account_id, seller_account_id, owner_account_id, payment_term_id, shipping_term_id, issued_at, created_at, updated_at) VALUES
    ('or_01seedpurchord2_000', 'PO-002', 'issued', 'purchase_order', 'normal', 'delivery', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01seedsupplier_acct1', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pytm_01seednet3000000', 'prepaid_billed', DATE_SUB(NOW(), INTERVAL 1 DAY), NOW(), NOW());

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedpoln2_qty00000', 30, 'un_01seedpound00000000', NOW(), NOW());

INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedpoln2_price000', '5.50', 'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_01seedpoln2_cost0000', '4.25', 'dollar', 'un_01seedpound00000000', NOW(), NOW());

INSERT IGNORE INTO sales_order_line (id, product_sku, product_description, product_id, item_id, sales_order_id, quantity_id, unit_price_id, unit_cost_id, created_at, updated_at) VALUES
    ('orln_01seedpoln2_000000', 'YRN-002', 'Yarn type 2 for PO', NULL, 'it_01seedyrn2item00000', 'or_01seedpurchord1_000', 'qu_01seedpoln2_qty00000', 'rt_01seedpoln2_price000', 'rt_01seedpoln2_cost0000', NOW(), NOW());

-- ============================================================
-- RECEIVING ORDERS (2 rows for pagination)
-- ============================================================
-- Receiving summaries resolve the supplier via purchase_order.seller_account_id and
-- account_relation (owner → counterparty, role supplier). Sales orders use the seed
-- account as seller, so they must not be linked here.

INSERT IGNORE INTO receiving_order (id, number, order_id, account_id, created_at, updated_at) VALUES
    ('rcor_01seedrecvorder1_0', 'RCV-001', 'or_01seedpurchord1_000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('rcor_01seedrecvorder2_0', 'RCV-002', 'or_01seedpurchord2_000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedrcln1_qty00000', 20, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedrcln2_qty00000', 15, 'un_01seedpound00000000', NOW(), NOW());

INSERT IGNORE INTO receiving_order_line (id, receiving_order_id, quantity_id, sales_order_line_id, created_at, updated_at) VALUES
    ('rcln_01seedrecvln1_0000', 'rcor_01seedrecvorder1_0', 'qu_01seedrcln1_qty00000', 'orln_01seedpoln1_000000', NOW(), NOW()),
    ('rcln_01seedrecvln2_0000', 'rcor_01seedrecvorder1_0', 'qu_01seedrcln2_qty00000', 'orln_01seedpoln2_000000', NOW(), NOW());

UPDATE receiving_order SET order_id = 'or_01seedpurchord1_000' WHERE id = 'rcor_01seedrecvorder1_0' AND order_id = 'or_01k0a8bs2ye3f9p8sj0m4dfmwe';
UPDATE receiving_order SET order_id = 'or_01seedpurchord2_000' WHERE id = 'rcor_01seedrecvorder2_0' AND order_id = 'or_01k0a8bs2yf909wjkd7ecd6x4z';
UPDATE receiving_order_line SET sales_order_line_id = 'orln_01seedpoln1_000000' WHERE id = 'rcln_01seedrecvln1_0000' AND sales_order_line_id = 'orln_01seedpck_ln1_0000';
UPDATE receiving_order_line SET sales_order_line_id = 'orln_01seedpoln2_000000' WHERE id = 'rcln_01seedrecvln2_0000' AND sales_order_line_id = 'orln_01seedpck_ln2_0000';

-- Yarn PO/receiving lines must use pound (item category unit group) so client stocking
-- validation can sum allocations with the same dimension as unitGroup.baseUnit.
UPDATE quantity SET unit_id = 'un_01seedpound00000000'
 WHERE id IN ('qu_01seedpoln1_qty00000', 'qu_01seedpoln2_qty00000', 'qu_01seedrcln1_qty00000', 'qu_01seedrcln2_qty00000')
   AND unit_id = 'un_01seedpair000000000';
UPDATE rate SET denominator_unit_id = 'un_01seedpound00000000'
 WHERE id IN ('rt_01seedpoln1_price000', 'rt_01seedpoln1_cost0000', 'rt_01seedpoln2_price000', 'rt_01seedpoln2_cost0000')
   AND denominator_unit_id = 'un_01seedpair000000000';

-- ============================================================
-- DELIVERIES (2 rows for pagination)
-- ============================================================
-- A delivery is an inbound receipt against a purchase order, so its
-- sales_order_id references a purchase_order-type order (defined above). Seeded
-- after the purchase orders and receiving-order lines so the FK targets exist.

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seeddlvln1_qty000', 20, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seeddlvln2_qty000', 10, 'un_01seedpound00000000', NOW(), NOW());

INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seeddlvln1_cost00', '4.00', 'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_01seeddlvln2_cost00', '4.25', 'dollar', 'un_01seedpound00000000', NOW(), NOW());

INSERT IGNORE INTO delivery (id, number, sales_order_id, account_id, delivery_status_code, created_at, updated_at) VALUES
    ('dv_01seeddelivery1_0000', 'DLV-001', 'or_01seedpurchord1_000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'accepted', NOW(), NOW()),
    ('dv_01seeddelivery2_0000', 'DLV-002', 'or_01seedpurchord2_000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'accepted', NOW(), NOW());

INSERT IGNORE INTO delivery_line (id, delivery_id, receiving_order_line_id, quantity_id, unit_cost_id, created_at, updated_at) VALUES
    ('dvln_01seeddlvln1_0000', 'dv_01seeddelivery1_0000', 'rcln_01seedrecvln1_0000', 'qu_01seeddlvln1_qty000', 'rt_01seeddlvln1_cost00', NOW(), NOW()),
    ('dvln_01seeddlvln2_0000', 'dv_01seeddelivery1_0000', 'rcln_01seedrecvln2_0000', 'qu_01seeddlvln2_qty000', 'rt_01seeddlvln2_cost00', NOW(), NOW());

-- ============================================================
-- CUSTOMER RICH LINKS (seed-gap fill for `?include=` coverage)
-- ============================================================
-- Populates freight preferences service level, credit limit, price-group
-- assignment, parent account, and child accounts on the SeedCustomer
-- account_relation (acre_01seedcustomer00000) so its GET/LIST responses
-- expose every declared include with real data.

-- Credit-limit quantity (dollar amount).
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcustcredit000', 5000, 'dollar', NOW(), NOW());

UPDATE account_relation
   SET default_carrier_option_id = 'crop_01seedground000000',
       credit_limit_id            = 'qu_01seedcustcredit000',
       parent_account_relation_id = 'acre_01seedhouseacct0000'
 WHERE id = 'acre_01seedcustomer00000';

-- Price-group assignment (reuses the seeded DME account_group).
INSERT IGNORE INTO account_relation_price_group (id, account_relation_id, account_group_id, created_at, updated_at) VALUES
    ('acrepg_01seedcustomer00', 'acre_01seedcustomer00000', 'acgp_01k0a413mjeth8pe1g70t0thax', NOW(), NOW());

-- Child customer accounts rooted at SeedCustomer (distinct from the HOUSE
-- children in the block above; gives `?include=child_accounts` a non-empty
-- list when the caller targets SeedCustomerAccountID).
INSERT IGNORE INTO account (id, name, account_type_code, onboarding_status_code, account_plan_id, created_at, updated_at) VALUES
    ('ac_01seedcustchild00001', 'GMS East Division', 'company', 'unclaimed', 'acpl_01seed000free00plan000000', NOW(), NOW()),
    ('ac_01seedcustchild00002', 'GMS West Division', 'company', 'unclaimed', 'acpl_01seed000free00plan000000', NOW(), NOW());

INSERT IGNORE INTO account_relation (id, owner_account_id, counterparty_account_id, account_relation_role_code, external_number, parent_account_relation_id, is_edi_enabled, priority_code, account_status_code, commission_status_code, freight_status_code, shipping_term_id, payment_term_id, account_group_id, created_at, updated_at) VALUES
    ('acre_01seedcustchildr01', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01seedcustchild00001', 'customer', 'GMS-CHILD-001', 'acre_01seedcustomer00000', 0, 'normal', 'normal', 'commission_applied', 'billed_freight', 'prepaid_billed', 'pytm_01seednet3000000', 'acgp_01k0a413mjeth8pe1g70t0thax', NOW(), NOW()),
    ('acre_01seedcustchildr02', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01seedcustchild00002', 'customer', 'GMS-CHILD-002', 'acre_01seedcustomer00000', 0, 'normal', 'normal', 'commission_applied', 'billed_freight', 'prepaid_billed', 'pytm_01seednet3000000', 'acgp_01k0a413mjeth8pe1g70t0thax', NOW(), NOW());

-- ============================================================
-- LOCATION PARENT
-- ============================================================
-- Seeds a campus that owns SeedLocationID (the Main Building) so
-- `?include=parent` on the seeded location resolves to a populated stub.

INSERT IGNORE INTO storage_location (id, account_id, storage_location_type_code, name, created_at, updated_at) VALUES
    ('sglc_01seedcampus00000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'building', 'Augno Campus', NOW(), NOW());

UPDATE storage_location SET parent_id = 'sglc_01seedcampus00000'
    WHERE id = 'sglc_01seedbuilding0000' AND parent_id IS NULL;

-- ============================================================
-- ACCOUNT-OWNED SHIPPING TERM (for list-shipping-terms include coverage)
-- ============================================================
-- 'prepaid_billed' is system-owned and has no flat_rate / minimum_order /
-- free_shipping_rule. Seed an account-owned term with all of them so the
-- list includes at least one row where every declared include resolves.

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedshipflatrate0', 12.00, 'dollar', NOW(), NOW()),
    ('qu_01seedshipminorder', 100.00, 'dollar', NOW(), NOW()),
    ('qu_01seedshipflatpaid', 5.00, 'dollar', NOW(), NOW()),
    ('qu_01seedshipminpaid0', 50.00, 'dollar', NOW(), NOW());

INSERT IGNORE INTO shipping_term (id, name, is_freight_exempt, is_carrier_rate, account_id, flat_rate_id, minimum_order_id, created_at, updated_at) VALUES
    ('shtm_01seedcustflat000', 'Custom Flat Rate', 0, 0, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'qu_01seedshipflatrate0', 'qu_01seedshipminorder', NOW(), NOW());

INSERT IGNORE INTO shipping_term_free_shipping_rule (id, shipping_term_id, carrier_option_id, created_at) VALUES
    ('shtmfsr_01seedcustom00', 'shtm_01seedcustflat000', 'crop_01seedground000000', NOW()),
    ('shtmfsr_01seedprepaid0', 'prepaid_billed', 'crop_01seedground000000', NOW());

-- Back-fill SeedShippingTermID ('prepaid_billed') with flat_rate and
-- minimum_order so `get-shipping-term/{flat_rate.unit,minimum_order_value.unit}`
-- resolve on the GET path too.
UPDATE shipping_term
   SET flat_rate_id     = 'qu_01seedshipflatpaid',
       minimum_order_id = 'qu_01seedshipminpaid0'
 WHERE id = 'prepaid_billed' AND flat_rate_id IS NULL;

-- ============================================================
-- ACCOUNT PRICE ATTRIBUTES & CATEGORIES
-- ============================================================
-- Seeds association rows for SeedAccountPriceID so `?include=attributes`
-- and `?include=categories` both resolve to populated lists.

INSERT IGNORE INTO account_price_attribute (id, account_price_id, attribute_id, created_at, updated_at) VALUES
    ('acprattr_01seedaccpric', 'acpr_01seedaccprice0000', 'at_01seedbeige00000000', NOW(), NOW());

INSERT IGNORE INTO account_price_item_category (id, account_price_id, item_category_id, created_at, updated_at) VALUES
    ('acprcat_01seedaccprice', 'acpr_01seedaccprice0000', 'itcg_01seedsocks000000', NOW(), NOW());

-- ============================================================
-- VOLUME / QUANTITY DISCOUNT ASSOCIATIONS
-- ============================================================
-- Seeds associations for SeedVolumeDiscountID (quds_01seedvoldiscount0):
-- attributes, categories, customer_groups, product_lines, acceptable_units.

INSERT IGNORE INTO _item_categories_quantity_discounts (A, B) VALUES
    ('itcg_01seedsocks000000', 'quds_01seedvoldiscount0');

INSERT IGNORE INTO _product_lines_quantity_discounts (A, B) VALUES
    ('pdln_01k0a735ype5e8nrhv1n5dhq1q', 'quds_01seedvoldiscount0');

-- A = attribute_id, B = quantity_discount_id (see GetVolumeDiscountAttributes query).
INSERT IGNORE INTO _quantity_discounts_attributes (A, B) VALUES
    ('at_01seedbeige00000000', 'quds_01seedvoldiscount0');

INSERT IGNORE INTO _quantity_discounts_units (A, B) VALUES
    ('quds_01seedvoldiscount0', 'un_01seedpair000000000');

INSERT IGNORE INTO account_group_quantity_discount (id, account_group_id, quantity_discount_id, created_at, updated_at) VALUES
    ('acgpqds_01seedcustgrp0', 'acgp_01k0a413mjeth8pe1g70t0thax', 'quds_01seedvoldiscount0', NOW(), NOW());

-- ============================================================
-- INVENTORY CHANGE LOG: responsible_user + scanning_station
-- ============================================================
-- Back-fill two seeded inventory_change_log rows with responsible_user_id /
-- scanning_station_id so `?include=responsible_user` and
-- `?include=responsible_scanning_station` resolve for at least one list item.

UPDATE inventory_change_log
   SET responsible_user_id = 'us_1wjfmmbwg8l7',
       scanning_station_id = 'sgsn_01k0a8201zegarjfsjaw5n7yfv',
       created_at = '2099-12-31 23:59:59.000',
       updated_at = '2099-12-31 23:59:59.000'
 WHERE id = 'ivcl_01seedwss000000000';

-- A second distinct action_type_code + responsible_user so the
-- inventory-change-logs action_type_codes and changed_by_user_ids array filters
-- have >=2 distinct values to exercise union/exclusion.
UPDATE inventory_change_log
   SET scanning_station_id = 'sgsn_01k0a8201zegarjfsjaw5n7yfv',
       action_type_code = 'system_action',
       responsible_user_id = 'us_6p7460uuwibz',
       created_at = '2099-12-31 23:59:58.000',
       updated_at = '2099-12-31 23:59:58.000'
 WHERE id = 'ivcl_01seedwls000000000';

-- ============================================================
-- CATALOG SEARCH RANK FIXTURES (e2e GET ...?q=621 SKU tier order)
-- Tier: exact < token < prefix < loose substring (see shared/db/catalog_search.go)
-- created_at: loose oldest ... exact newest so ties sort predictably with DESC recency.
-- ============================================================

INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_e2e621rank_pt_lo_uv', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2e621rank_pt_lo_uc', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2e621rank_pt_lo_br', 0, 'un_01seedpair000000000', 'day', NOW(), NOW()),
    ('rt_e2e621rank_pt_pf_uv', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2e621rank_pt_pf_uc', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2e621rank_pt_pf_br', 0, 'un_01seedpair000000000', 'day', NOW(), NOW()),
    ('rt_e2e621rank_pt_tk_uv', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2e621rank_pt_tk_uc', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2e621rank_pt_tk_br', 0, 'un_01seedpair000000000', 'day', NOW(), NOW()),
    ('rt_e2e621rank_pt_ex_uv', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2e621rank_pt_ex_uc', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2e621rank_pt_ex_br', 0, 'un_01seedpair000000000', 'day', NOW(), NOW()),
    ('rt_e2e621rank_ml_lo_uv', 0, 'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_e2e621rank_ml_lo_uc', 6, 'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_e2e621rank_ml_lo_br', 0, 'un_01seedpound00000000', 'day', NOW(), NOW()),
    ('rt_e2e621rank_ml_pf_uv', 0, 'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_e2e621rank_ml_pf_uc', 6, 'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_e2e621rank_ml_pf_br', 0, 'un_01seedpound00000000', 'day', NOW(), NOW()),
    ('rt_e2e621rank_ml_tk_uv', 0, 'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_e2e621rank_ml_tk_uc', 6, 'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_e2e621rank_ml_tk_br', 0, 'un_01seedpound00000000', 'day', NOW(), NOW()),
    ('rt_e2e621rank_pd_lo_uv', 10, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2e621rank_pd_lo_uc', 7, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2e621rank_pd_lo_br', 1, 'un_01seedpair000000000', 'day', NOW(), NOW()),
    ('rt_e2e621rank_pd_pf_uv', 10, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2e621rank_pd_pf_uc', 7, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2e621rank_pd_pf_br', 1, 'un_01seedpair000000000', 'day', NOW(), NOW()),
    ('rt_e2e621rank_pd_tk_uv', 10, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2e621rank_pd_tk_uc', 7, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2e621rank_pd_tk_br', 1, 'un_01seedpair000000000', 'day', NOW(), NOW());

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_e2e621rank_ml_op1', 100, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_e2e621rank_ml_lt1', 14, 'day', NOW(), NOW()),
    ('qu_e2e621rank_ml_op2', 100, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_e2e621rank_ml_lt2', 14, 'day', NOW(), NOW()),
    ('qu_e2e621rank_ml_op3', 100, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_e2e621rank_ml_lt3', 14, 'day', NOW(), NOW());

INSERT IGNORE INTO item (id, sku, description, unit_value_id, unit_cost_id, burn_rate_id, account_id, item_type_code, item_category_id, created_at, updated_at) VALUES
    ('it_e2e621rank_pt_lo0', 'rkpt47562183', 'e2e catalog search rank fixture', 'rt_e2e621rank_pt_lo_uv', 'rt_e2e621rank_pt_lo_uc', 'rt_e2e621rank_pt_lo_br', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', '2020-01-01 00:00:00.000', '2020-01-01 00:00:00.000'),
    ('it_e2e621rank_pt_pf0', '621rkpt8f3a', 'e2e catalog search rank fixture', 'rt_e2e621rank_pt_pf_uv', 'rt_e2e621rank_pt_pf_uc', 'rt_e2e621rank_pt_pf_br', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', '2020-01-02 00:00:00.000', '2020-01-02 00:00:00.000'),
    ('it_e2e621rank_pt_tk0', 'rkpt7f3a 621', 'e2e catalog search rank fixture', 'rt_e2e621rank_pt_tk_uv', 'rt_e2e621rank_pt_tk_uc', 'rt_e2e621rank_pt_tk_br', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', '2020-01-03 00:00:00.000', '2020-01-03 00:00:00.000'),
    ('it_e2e621rank_pt_ex0', '621', 'e2e catalog search rank fixture', 'rt_e2e621rank_pt_ex_uv', 'rt_e2e621rank_pt_ex_uc', 'rt_e2e621rank_pt_ex_br', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', '2020-01-04 00:00:00.000', '2020-01-04 00:00:00.000'),
    ('it_e2e621rank_ml_lo0', 'rkmt47562183', 'e2e catalog search rank fixture', 'rt_e2e621rank_ml_lo_uv', 'rt_e2e621rank_ml_lo_uc', 'rt_e2e621rank_ml_lo_br', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'material', 'itcg_01seedyarn0000000', '2020-01-01 00:00:00.000', '2020-01-01 00:00:00.000'),
    ('it_e2e621rank_ml_pf0', '621rkmt8f3a', 'e2e catalog search rank fixture', 'rt_e2e621rank_ml_pf_uv', 'rt_e2e621rank_ml_pf_uc', 'rt_e2e621rank_ml_pf_br', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'material', 'itcg_01seedyarn0000000', '2020-01-02 00:00:00.000', '2020-01-02 00:00:00.000'),
    ('it_e2e621rank_ml_tk0', 'rkmt9b2c 621', 'e2e catalog search rank fixture', 'rt_e2e621rank_ml_tk_uv', 'rt_e2e621rank_ml_tk_uc', 'rt_e2e621rank_ml_tk_br', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'material', 'itcg_01seedyarn0000000', '2020-01-03 00:00:00.000', '2020-01-03 00:00:00.000'),
    ('it_e2e621rank_pd_lo0', 'rkrpd56214z', 'e2e catalog search rank fixture', 'rt_e2e621rank_pd_lo_uv', 'rt_e2e621rank_pd_lo_uc', 'rt_e2e621rank_pd_lo_br', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedsocks000000', '2020-01-01 00:00:00.000', '2020-01-01 00:00:00.000'),
    ('it_e2e621rank_pd_pf0', '621rkrp9pfx', 'e2e catalog search rank fixture', 'rt_e2e621rank_pd_pf_uv', 'rt_e2e621rank_pd_pf_uc', 'rt_e2e621rank_pd_pf_br', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedsocks000000', '2020-01-02 00:00:00.000', '2020-01-02 00:00:00.000'),
    ('it_e2e621rank_pd_tk0', 'rkrpd4e1f 621', 'e2e catalog search rank fixture', 'rt_e2e621rank_pd_tk_uv', 'rt_e2e621rank_pd_tk_uc', 'rt_e2e621rank_pd_tk_br', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedsocks000000', '2020-01-03 00:00:00.000', '2020-01-03 00:00:00.000');

INSERT IGNORE INTO part (id, item_id, created_at, updated_at) VALUES
    ('pt_e2e621rank_pt_lo0', 'it_e2e621rank_pt_lo0', '2020-01-01 00:00:00.000', '2020-01-01 00:00:00.000'),
    ('pt_e2e621rank_pt_pf0', 'it_e2e621rank_pt_pf0', '2020-01-02 00:00:00.000', '2020-01-02 00:00:00.000'),
    ('pt_e2e621rank_pt_tk0', 'it_e2e621rank_pt_tk0', '2020-01-03 00:00:00.000', '2020-01-03 00:00:00.000'),
    ('pt_e2e621rank_pt_ex0', 'it_e2e621rank_pt_ex0', '2020-01-04 00:00:00.000', '2020-01-04 00:00:00.000');

INSERT IGNORE INTO material (id, item_id, order_point_id, lead_time_id, created_at, updated_at) VALUES
    ('ml_e2e621rank_ml_lo0', 'it_e2e621rank_ml_lo0', 'qu_e2e621rank_ml_op1', 'qu_e2e621rank_ml_lt1', '2020-01-01 00:00:00.000', '2020-01-01 00:00:00.000'),
    ('ml_e2e621rank_ml_pf0', 'it_e2e621rank_ml_pf0', 'qu_e2e621rank_ml_op2', 'qu_e2e621rank_ml_lt2', '2020-01-02 00:00:00.000', '2020-01-02 00:00:00.000'),
    ('ml_e2e621rank_ml_tk0', 'it_e2e621rank_ml_tk0', 'qu_e2e621rank_ml_op3', 'qu_e2e621rank_ml_lt3', '2020-01-03 00:00:00.000', '2020-01-03 00:00:00.000');

INSERT IGNORE INTO product (id, item_id, product_type_code, product_line_id, created_at, updated_at) VALUES
    ('pd_e2e621rank_pd_lo0', 'it_e2e621rank_pd_lo0', 'sale', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', '2020-01-01 00:00:00.000', '2020-01-01 00:00:00.000'),
    ('pd_e2e621rank_pd_pf0', 'it_e2e621rank_pd_pf0', 'sale', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', '2020-01-02 00:00:00.000', '2020-01-02 00:00:00.000'),
    ('pd_e2e621rank_pd_tk0', 'it_e2e621rank_pd_tk0', 'sale', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', '2020-01-03 00:00:00.000', '2020-01-03 00:00:00.000');

UPDATE product SET is_portal_ready = 1
 WHERE id IN ('pd_e2e621rank_pd_lo0', 'pd_e2e621rank_pd_pf0', 'pd_e2e621rank_pd_tk0');

-- ============================================================
-- PUT ?include[]= meta walker fixtures (estimate SO/PO + disposable catalog rows)
-- ============================================================

UPDATE sales_order
SET
    carrier_option_id = COALESCE(carrier_option_id, 'crop_01seedground000000'),
    order_discount_id = COALESCE(order_discount_id, 'ords_01seedpct10discount')
WHERE id = 'or_01k0a8bs2yfhev5begay245wez';

INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seed_putinc_chg_uv00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seed_putinc_chg_uc00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seed_putinc_chg_br00', 0, 'un_01seedpair000000000', 'day', NOW(), NOW()),
    ('rt_01seed_putinc_att_uv00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seed_putinc_att_uc00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seed_putinc_att_br00', 0, 'un_01seedpair000000000', 'day', NOW(), NOW()),
    ('rt_01seed_putinc_pln_uv00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seed_putinc_pln_uc00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seed_putinc_pln_br00', 0, 'un_01seedpair000000000', 'day', NOW(), NOW());

INSERT IGNORE INTO item (id, sku, description, unit_value_id, unit_cost_id, burn_rate_id, account_id, item_type_code, item_category_id, created_at, updated_at) VALUES
    ('it_01seed_putinc_chgcat00', 'PUTINC-CATCHG', 'e2e put-include change category walker', 'rt_01seed_putinc_chg_uv00', 'rt_01seed_putinc_chg_uc00', 'rt_01seed_putinc_chg_br00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01seed_putinc_attradd0', 'PUTINC-ATTR', 'e2e put-include add-attribute walker', 'rt_01seed_putinc_att_uv00', 'rt_01seed_putinc_att_uc00', 'rt_01seed_putinc_att_br00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01seed_putinc_chprdln_it', 'PUTINC-PRDPL', 'e2e put-include change product line walker', 'rt_01seed_putinc_pln_uv00', 'rt_01seed_putinc_pln_uc00', 'rt_01seed_putinc_pln_br00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedsocks000000', NOW(), NOW());

INSERT IGNORE INTO product (id, item_id, product_type_code, product_line_id, created_at, updated_at) VALUES
    ('pd_01seed_putinc_chprdln0', 'it_01seed_putinc_chprdln_it', 'sale', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', NOW(), NOW());

INSERT IGNORE INTO _item_attributes (A, B) VALUES
    ('at_01seedbeige00000000', 'it_01seed_putinc_chgcat00'),
    ('at_01seedbeige00000000', 'it_01seed_putinc_chprdln_it');

INSERT IGNORE INTO _item_categories_properties (A, B) VALUES
    ('itcg_01seedshipping000', 'pp_01k0a7ntn1ez6aw8x850femxeh');

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedputinc_polqty00', 6, 'un_01seedpound00000000', NOW(), NOW());

INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedputinc_pol_pri', '5.75', 'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_01seedputinc_pol_cst', '4.05', 'dollar', 'un_01seedpound00000000', NOW(), NOW());

INSERT IGNORE INTO sales_order (id, number, sales_order_status_code, sales_order_type_code, priority_code, carrier_id, carrier_option_id, billing_address_id, shipping_address_id, buyer_account_id, seller_account_id, owner_account_id, payment_term_id, shipping_term_id, created_at, updated_at) VALUES
    ('or_01seed_putinc_po_es00', 'PO-E2EPUTINC', 'estimate', 'purchase_order', 'normal', 'delivery', 'crop_01seedground000000', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01seedsupplier_acct0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pytm_01seednet3000000', 'prepaid_billed', NOW(), NOW());

INSERT IGNORE INTO sales_order_line (id, product_sku, product_description, product_id, item_id, sales_order_id, quantity_id, unit_price_id, unit_cost_id, created_at, updated_at) VALUES
    ('orln_01seedputinc_pol1_000', 'YRN-PINC', 'Yarn PO for put-include walker', NULL, 'it_01seedyrn1item00000', 'or_01seed_putinc_po_es00', 'qu_01seedputinc_polqty00', 'rt_01seedputinc_pol_pri', 'rt_01seedputinc_pol_cst', NOW(), NOW());

INSERT IGNORE INTO order_email_contact (id, sales_order_id, account_user_id, notification_type_code, created_at, updated_at) VALUES
    ('oec_01seed_putinc_po_subm', 'or_01seed_putinc_po_es00', 'acus_s83fjhyfmqen', 'purchaseOrderSubmission', NOW(), NOW());

-- ============================================================
-- SECTION: VOLUME-DISCOUNT + PER-PAIR ACCOUNT-PRICE PRICING CASES
-- Replicates production pricing scenarios for the price-quote engine:
--   - A "Carton (12 pr)" unit (= 12 pairs = 24 each) added to the Socks unit group.
--   - A new "E2E Volume LTD" product line (NO account price → volume discount fires) with
--     two products listed at 29.95/pair (= 359.40 per carton).
--   - A volume discount with the production tier ladder (0→5.8096828%, then 4% at 4/7/10),
--     scoped to the line + Socks category + beige attribute + carton unit, NO customer
--     group (applies to all). Expected per-carton prices by summed carton quantity:
--       1ct → 338.52, 4ct → 324.98, 8ct → 311.98, 11ct → 299.50 (multiplicative).
--   - A per-pair account price (18.45/pair, beige-gated) for customer2 on the same line,
--     which must beat the volume discount (account price wins).
-- ============================================================

INSERT IGNORE INTO unit (id, name, abbreviation, unit_dimension_code, ratio_numerator, ratio_denominator, offset_numerator, offset_denominator, is_base_unit, account_id, created_at, updated_at) VALUES
    ('un_e2ecarton12pr000', 'Carton (12 pr)', 'ct12pr', 'quantity', 24, 1, 0, 1, 0, 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

INSERT IGNORE INTO unit_group_unit (id, unit_group_id, unit_id, created_at, updated_at) VALUES
    ('ungpun_e2ecart12pr0', 'ungp_01k0a5ecy9edg9za40dnccw53n', 'un_e2ecarton12pr000', NOW(), NOW());

INSERT IGNORE INTO product_line (id, name, account_id, unit_group_id, is_commission_exempt, is_freight_exempt, created_at, updated_at) VALUES
    ('pdln_e2evolumeline0', 'E2E Volume LTD', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ungp_01k0a5ecy9edg9za40dnccw53n', 0, 0, NOW(), NOW());

INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_e2evol1list00000', 29.95, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2evol2list00000', 29.95, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2evol1cost00000', 10.00, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2evol2cost00000', 10.00, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2evol1burn00000', 0.50, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2evol2burn00000', 0.50, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_e2evolacctprice0', 18.45, 'dollar', 'un_01seedpair000000000', NOW(), NOW());

-- item.burn_rate_id / unit_value_id / unit_cost_id are each UNIQUE, so every item needs
-- its own rate rows.
INSERT IGNORE INTO item (id, sku, description, unit_value_id, unit_cost_id, burn_rate_id, account_id, item_type_code, item_category_id, created_at, updated_at) VALUES
    ('it_e2evol1000000000', 'E2E-LTD-A', 'E2E LTD product A', 'rt_e2evol1list00000', 'rt_e2evol1cost00000', 'rt_e2evol1burn00000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_e2evol2000000000', 'E2E-LTD-B', 'E2E LTD product B', 'rt_e2evol2list00000', 'rt_e2evol2cost00000', 'rt_e2evol2burn00000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedsocks000000', NOW(), NOW());

INSERT IGNORE INTO product (id, item_id, product_type_code, product_line_id, created_at, updated_at) VALUES
    ('pd_e2evol1000000000', 'it_e2evol1000000000', 'sale', 'pdln_e2evolumeline0', NOW(), NOW()),
    ('pd_e2evol2000000000', 'it_e2evol2000000000', 'sale', 'pdln_e2evolumeline0', NOW(), NOW());

-- Both products carry the beige attribute (A = attribute id, B = item id).
INSERT IGNORE INTO _item_attributes (A, B) VALUES
    ('at_01seedbeige00000000', 'it_e2evol1000000000'),
    ('at_01seedbeige00000000', 'it_e2evol2000000000');

INSERT IGNORE INTO quantity_discount (id, name, account_id, created_at, updated_at) VALUES
    ('quds_e2evolume00000', 'E2E Volume Discount', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

INSERT IGNORE INTO quantity_discount_tier (id, discount_percentage, name, threshold, parent_tier_id, quantity_discount_id, created_at, updated_at) VALUES
    ('qudt_e2evoltier000', 0.05809682805, 'Tier 0',  0,  NULL, 'quds_e2evolume00000', NOW(), NOW()),
    ('qudt_e2evoltier004', 0.04,          'Tier 4',  4,  NULL, 'quds_e2evolume00000', NOW(), NOW()),
    ('qudt_e2evoltier007', 0.04,          'Tier 7',  7,  NULL, 'quds_e2evolume00000', NOW(), NOW()),
    ('qudt_e2evoltier010', 0.04,          'Tier 10', 10, NULL, 'quds_e2evolume00000', NOW(), NOW());

-- Discount scoping (A = entity id, B = quantity_discount id for these Prisma M2M tables).
INSERT IGNORE INTO _product_lines_quantity_discounts (A, B) VALUES ('pdln_e2evolumeline0', 'quds_e2evolume00000');
INSERT IGNORE INTO _item_categories_quantity_discounts (A, B) VALUES ('itcg_01seedsocks000000', 'quds_e2evolume00000');
INSERT IGNORE INTO _quantity_discounts_attributes (A, B) VALUES ('at_01seedbeige00000000', 'quds_e2evolume00000');
INSERT IGNORE INTO _quantity_discounts_units (A, B) VALUES ('un_e2ecarton12pr000', 'quds_e2evolume00000');

-- Per-pair account price for customer2 on the volume line (beats the volume discount).
INSERT IGNORE INTO account_price (id, owner_account_id, unit_value_id, product_line_id, recipient_account_id, created_at, updated_at) VALUES
    ('acpr_e2evolacctpr0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'rt_e2evolacctprice0', 'pdln_e2evolumeline0', 'ac_01seedcustomer2_acct0', NOW(), NOW());

INSERT IGNORE INTO account_price_attribute (id, account_price_id, attribute_id, created_at, updated_at) VALUES
    ('acprattr_e2evol000', 'acpr_e2evolacctpr0', 'at_01seedbeige00000000', NOW(), NOW());

-- ============================================================
-- PAYMENT-STATUS REGRESSION FIXTURE (sales-order "paid" parity)
-- ============================================================
-- Reproduces the reported bug: a sales order shows "unpaid" under the customer
-- list even though its invoice is marked paid. A fulfilled order whose only
-- invoice is paid in full (is_paid_in_full = 1) but which has NO settlement
-- transaction_allocation rows. The legacy dashboard rule marks this order PAID
-- (fulfilled AND every invoice paid in full); the earlier allocation-vs-invoiced
-- derivation marked it UNPAID because there were no allocations to reconcile.
-- This fixture pins the legacy-parity behavior. Intentionally no
-- transaction_allocation / invoice_line rows: the "paid" rule reads
-- invoice.is_paid_in_full directly off invoice.sales_order_id.

INSERT IGNORE INTO sales_order (id, number, sales_order_status_code, sales_order_type_code, priority_code, carrier_id, billing_address_id, shipping_address_id, buyer_account_id, seller_account_id, owner_account_id, payment_term_id, shipping_term_id, issued_at, first_ship_at, completed_at, created_at, updated_at) VALUES
    ('or_01seedpaidnoalloc00', 'ORD-PAID-NOALLOC', 'fulfilled', 'sales_order', 'normal', 'will_call', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pytm_01seednet3000000', 'prepaid_billed', DATE_SUB(NOW(), INTERVAL 6 DAY), DATE_SUB(NOW(), INTERVAL 5 DAY), DATE_SUB(NOW(), INTERVAL 4 DAY), DATE_SUB(NOW(), INTERVAL 6 DAY), DATE_SUB(NOW(), INTERVAL 4 DAY));

INSERT IGNORE INTO invoice (id, number, is_paid_in_full, sales_order_id, billing_address_id, account_id, created_at, updated_at) VALUES
    ('iv_01seedpaidnoalloc00', 'INV-PAIDNOALLOC', 1, 'or_01seedpaidnoalloc00', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(), INTERVAL 4 DAY), DATE_SUB(NOW(), INTERVAL 4 DAY));

-- ============================================================
-- MACHINE DOWNTIME EVENTS (for /v1/operations/machine-downtime-events)
-- ============================================================
-- Three CLOSED events so the generic list/pagination/include conformance suites have
-- fixtures. They are dated ~60 days back on purpose: the OEE analytics e2e tests
-- measure short recent windows, and seeding anything recent would move their numbers.
-- All are closed, so they never occupy the one-open-event-per-machine slot the
-- downtime tests rely on.
INSERT IGNORE INTO machine_downtime_event (
    id, account_id, machine_id, department_id, production_step_id, reason_code,
    started_at, ended_at, duration_seconds, shift_date, shift_code,
    item_id, production_run_id, batch_id, schedule_line_id, note,
    reported_by_id, source_code, created_at, updated_at
) VALUES
    ('mcdt_01seede2edowntime01', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh',
     'dp_01k0a5r01yfx3sj1vy9qgv3dc0', NULL, 'breakdown',
     DATE_SUB(NOW(3), INTERVAL 60 DAY), DATE_SUB(NOW(3), INTERVAL 60 DAY) + INTERVAL 45 MINUTE, 2700,
     DATE(DATE_SUB(NOW(3), INTERVAL 60 DAY)), NULL,
     'it_01k0a7100aeysrs9vxpeq14yxj', NULL, NULL, NULL, 'Seeded: needle bar jam',
     'acus_s83fjhyfmqen', 'manual', NOW(3), NOW(3)),
    ('mcdt_01seede2edowntime02', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh',
     'dp_01k0a5r01yfx3sj1vy9qgv3dc0', NULL, 'changeover',
     DATE_SUB(NOW(3), INTERVAL 59 DAY), DATE_SUB(NOW(3), INTERVAL 59 DAY) + INTERVAL 30 MINUTE, 1800,
     DATE(DATE_SUB(NOW(3), INTERVAL 59 DAY)), NULL,
     'it_01k0a7100aeysrs9vxpeq14yxj', NULL, NULL, NULL, 'Seeded: yarn changeover',
     'acus_s83fjhyfmqen', 'manual', NOW(3), NOW(3)),
    ('mcdt_01seede2edowntime03', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh',
     'dp_01k0a5r01yfx3sj1vy9qgv3dc0', NULL, 'material_shortage',
     DATE_SUB(NOW(3), INTERVAL 58 DAY), DATE_SUB(NOW(3), INTERVAL 58 DAY) + INTERVAL 90 MINUTE, 5400,
     DATE(DATE_SUB(NOW(3), INTERVAL 58 DAY)), NULL,
     'it_01k0a7100aeysrs9vxpeq14yxj', NULL, NULL, NULL, 'Seeded: waiting on yarn delivery',
     'acus_s83fjhyfmqen', 'manual', NOW(3), NOW(3));

-- ============================================================
-- PRODUCTION SCHEDULE CONSTRAINT (for /v1/operations/production-schedules)
-- ============================================================
-- Names the knitting department as the planning constraint so the solver has something
-- to schedule. Without one the preview endpoint correctly refuses to plan, which would
-- leave the solve path untested. Selection is by department, so every machine in it is
-- planned and no per-machine row is needed.
INSERT INTO account_production_schedule_setting (
    id, account_id, constraint_department_id, created_at, updated_at
) VALUES
    ('acpnscst_01seede2esetting', 'ac_01k0a5smf9ekb8rqg12555zjqa',
     'dp_01k0a5r01yfx3sj1vy9qgv3dc0', NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE constraint_department_id = VALUES(constraint_department_id);

-- ============================================================
-- BATCH -> MACHINE LINKS (for the production scheduling solver)
-- ============================================================
-- The solver measures run rates by joining batch -> _batches_machines -> machine, so
-- without these links it finds no items and every generated plan is empty. That made
-- the schedule e2e tests pass vacuously.
-- A = batch, B = machine.
INSERT IGNORE INTO _batches_machines (A, B) VALUES
    ('bt_01seedbatch1_0000000', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh'),
    ('bt_01seedbatch2_0000000', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh');

-- ============================================================
-- BATCH GENEALOGY (so knit demand can be pooled from finished goods)
-- ============================================================
-- The solver pools a constraint item's demand from the finished goods it becomes,
-- walking batch -> _batch_flow -> child batch. Without a descendant that carries order
-- demand, every knit item plans to zero and no campaigns are generated — which made
-- the schedule line assertions pass vacuously.
--
-- This links the seeded knit batch to a batch of SCK-001, which does have order lines.
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedschedfg_qty00', 10, 'un_01seedpair000000000', NOW(), NOW());

INSERT IGNORE INTO batch (id, account_id, item_id, quantity_id, scanning_station_id, production_step_id, production_run_id, scanned_at, created_at, updated_at) VALUES
    ('bt_01seedschedfg000000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qu_01seedschedfg_qty00', 'sgsn_01k0a8201zegarjfsjaw5n7yfv', 'prs_01k0a56yc1e8wag6wexn4pp8t9', 'pnrn_01seedprod_run0000', NOW(), NOW(), NOW());

-- A = downstream (target), B = upstream (source), per the Prisma orientation of _batch_flow.
INSERT IGNORE INTO _batch_flow (A, B) VALUES
    ('bt_01seedschedfg000000', 'bt_01seedbatch1_0000000');

-- Backdated demand for the scheduling solver.
--
-- Trailing-12 demand deliberately excludes the current partial month, so an order
-- issued today contributes nothing. Without demand in a COMPLETED month every knit
-- item plans to zero and no campaigns are generated, which left the schedule-line
-- write path untested. This order is issued three months back for the same SCK-001
-- product the knit batch feeds, on a separate order so no existing fixture moves.
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedscheddem_qty0', 400, 'un_01seedpair000000000', NOW(), NOW());

INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at)
SELECT 'rt_01seedscheddem_prc', value, numerator_unit_id, denominator_unit_id, NOW(), NOW()
FROM rate WHERE id = 'rt_01seediss_ln1_price';

INSERT IGNORE INTO sales_order (
    id, number, sales_order_status_code, sales_order_type_code, priority_code,
    carrier_id, carrier_option_id, billing_address_id, shipping_address_id,
    buyer_account_id, seller_account_id, owner_account_id,
    payment_term_id, shipping_term_id, issued_at, created_at, updated_at
) VALUES (
    'or_01seedscheddemand00', 'E2E-SCHED-DEMAND', 'issued', 'sales_order', 'normal',
    'delivery', 'crop_01seedground000000', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g',
    'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa',
    'pytm_01seednet3000000', 'prepaid_billed',
    DATE_SUB(NOW(3), INTERVAL 3 MONTH), DATE_SUB(NOW(3), INTERVAL 3 MONTH), NOW(3)
);

INSERT IGNORE INTO sales_order_line (
    id, product_sku, product_id, item_id, sales_order_id, quantity_id, unit_price_id, created_at, updated_at
) VALUES (
    'orln_01seedscheddemand', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'it_01k0a7100aeysrs9vxpeq14yxj',
    'or_01seedscheddemand00', 'qu_01seedscheddem_qty0', 'rt_01seedscheddem_prc', NOW(3), NOW(3)
);

-- ============================================================
-- PRODUCTION SCHEDULE (for /v1/operations/production-schedules)
-- ============================================================
-- Schedules are normally *generated*, never seeded, but the generic spec-driven
-- conformance suites need a stable {id} to resolve nested paths ({id}/lines,
-- {id}/item-policies) against. This is version 1 so generation still allocates
-- upward from it, and it stays 'draft' so it can never be picked as the current
-- published schedule.
INSERT IGNORE INTO production_schedule (
    id, account_id, version, status_code, name,
    planning_as_of, horizon_start_date, horizon_end_date, horizon_weeks, frozen_weeks,
    demand_basis_code, generation_source_code, solver_version,
    settings_snapshot, diagnostics, generated_by_id, created_at, updated_at
) VALUES (
    'pnsc_01seede2eschedule', 'ac_01k0a5smf9ekb8rqg12555zjqa', 1, 'draft', 'Seeded e2e schedule',
    NOW(3), DATE(NOW(3) - INTERVAL WEEKDAY(NOW(3)) DAY), DATE(NOW(3) - INTERVAL WEEKDAY(NOW(3)) DAY) + INTERVAL 13 WEEK, 13, 1,
    'trailing_12', 'manual', 'v1',
    JSON_OBJECT('horizon_weeks', 13, 'frozen_weeks', 1, 'shifts_per_day', 2, 'hours_per_shift', 7,
                'work_days_per_week', 5, 'capacity_headroom_pct', 0.9, 'default_lot_units', 60),
    -- Carries one at-risk order so the spec-driven list suites have a row to validate
    -- against; a solved version writes this same shape from its allocation walk.
    JSON_OBJECT('seeded', true,
                'at_risk_orders', JSON_ARRAY(
                    JSON_OBJECT('sales_order_id', 'or_01seedscheddemand00',
                                'sales_order_number', 'E2E-SCHED-DEMAND',
                                'item_id', 'it_01k0a7100aeysrs9vxpeq14yxj',
                                'sku', 'SCK-001',
                                'units', 80,
                                'due_week', 0,
                                'reason', 'past_due'))),
    NULL, NOW(3), NOW(3)
);

INSERT IGNORE INTO production_schedule_line (
    id, account_id, production_schedule_id,
    week_index, week_start_date, machine_id, production_step_id, department_id, item_id,
    planned_quantity, planned_lots, planned_run_hours, planned_changeover_minutes,
    sequence_index, projected_on_hand_before, projected_on_hand_after,
    status_code, source_code, is_frozen, created_at, updated_at
) VALUES (
    'pnscln_01seede2eline01', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pnsc_01seede2eschedule',
    1, DATE(NOW(3) - INTERVAL WEEKDAY(NOW(3)) DAY) + INTERVAL 1 WEEK, 'mc_01k0a52fb6eqhtbx9hdxj3vvnh', NULL, 'dp_01k0a5r01yfx3sj1vy9qgv3dc0',
    'it_01k0a7100aeysrs9vxpeq14yxj',
    120.000000000000000000000000000000, 2, 4.5000, 30.0000,
    0, 40.000000000000000000000000000000, 160.000000000000000000000000000000,
    'planned', 'solver', 0, NOW(3), NOW(3)
);

-- One campaign earmarked for that order, so `covering_lines` is populated rather than
-- an empty list — a partly-covered order is what the report exists to distinguish.
INSERT IGNORE INTO production_schedule_line_order (
    id, account_id, production_schedule_id,
    production_schedule_line_id, sales_order_id, sales_order_line_id,
    allocated_quantity, created_at, updated_at
)
SELECT
    'pnsclnor_01seede2elink', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pnsc_01seede2eschedule',
    'pnscln_01seede2eline01', 'or_01seedscheddemand00', sol.id,
    40.000000000000000000000000000000, NOW(3), NOW(3)
FROM sales_order_line sol
WHERE sol.sales_order_id = 'or_01seedscheddemand00'
LIMIT 1;

INSERT IGNORE INTO production_schedule_item_policy (
    id, account_id, production_schedule_id, item_id, sku, primary_machine_id,
    annual_demand, weekly_demand, seconds_per_unit, unit_cost,
    setup_cost, holding_cost, eoq_units,
    constraint_lead_time_weeks, finish_lead_time_weeks,
    sigma_weekly_pooled, sigma_downstream_sum,
    safety_stock_primary, safety_stock_downstream,
    reorder_point, order_up_to, on_hand_echelon,
    on_hand_greige, average_greige_inventory, max_greige_inventory,
    weeks_of_cover, annual_run_hours,
    abc_class, created_at, updated_at
) VALUES (
    'pnscpl_01seede2epolicy', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pnsc_01seede2eschedule',
    'it_01k0a7100aeysrs9vxpeq14yxj', 'E2E-SEED-SKU', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh',
    5200.000000000000000000000000000000, 100.000000000000000000000000000000, 135.000000, 4.250000,
    10.000000, 1.062500, 240.000000000000000000000000000000,
    1.300, 6.000,
    18.000000000000000000000000000000, 12.000000000000000000000000000000,
    34.000000000000000000000000000000, 20.000000000000000000000000000000,
    164.000000000000000000000000000000, 404.000000000000000000000000000000,
    200.000000000000000000000000000000,
    80.000000000000000000000000000000, 154.000000000000000000000000000000, 274.000000000000000000000000000000,
    2.0000, 195.0000,
    'A', NOW(3), NOW(3)
);

-- The finished-goods decomposition of that pooled buffer. Seeded so the endpoint has
-- rows to validate against; without it the shape test passes vacuously.
INSERT IGNORE INTO production_schedule_finished_policy (
    id, account_id, production_schedule_id,
    item_id, sku, greige_item_id, greige_sku,
    annual_demand, weekly_demand, sigma_weekly,
    safety_stock, reorder_point, on_hand, weeks_of_cover,
    created_at, updated_at
) VALUES (
    'pnscfipc_01seede2efinished', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pnsc_01seede2eschedule',
    'it_01k0a7100aeysrs9vxpeq14yxj', 'E2E-SEED-FG-SKU',
    'it_01k0a7100aeysrs9vxpeq14yxj', 'E2E-SEED-SKU',
    2600.000000000000000000000000000000, 50.000000000000000000000000000000, 12.000000000000000000000000000000,
    20.000000000000000000000000000000, 320.000000000000000000000000000000, 120.000000000000000000000000000000, 2.4000,
    NOW(3), NOW(3)
);

-- A per-resource planning override. Machines are selected by department now, so this is
-- a lead-time offset on a downstream step rather than a constraint opt-in.
INSERT IGNORE INTO production_schedule_resource_setting (
    id, account_id, scope_code, scope_ref_id, is_excluded, is_enabled,
    lead_time_offset_weeks, sort_order, created_at, updated_at
) VALUES
    ('pnscrrsd_01seede2eoffset', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'production_step',
     'prs_01k0a56yc1e8wag6wexn4pp8t9', 0, 1, 1, 0, NOW(3), NOW(3));

-- A per-item planning override, so the list endpoint has a row to serve. The
-- spec-driven list suites assert every list endpoint returns at least one item,
-- and only items that have been given an override ever appear here.
--
-- Deliberately make_to_stock, and deliberately on a yarn item no test mutates: this is
-- a fixture the conformance suites read, so a test that set a policy on the same item
-- would delete this row on cleanup and leave the list endpoint with nothing to serve.
INSERT IGNORE INTO production_schedule_item_setting (
    id, account_id, item_id, is_excluded, lot_multiple_units, fulfillment_policy_code,
    created_at, updated_at
) VALUES
    ('pnscitsd_01seede2eitem0', 'ac_01k0a5smf9ekb8rqg12555zjqa',
     'it_01seedyrn3item00000', 0, 60, 'make_to_stock', NOW(3), NOW(3));

-- ============================================================
-- DEMAND OVERRIDES (for /v1/operations/demand-overrides)
-- ============================================================
-- Three rows so the generic list/pagination/include conformance suites have fixtures
-- and both scopes are represented. They are dated in the past on purpose: an override
-- overlapping the current planning window would shift the numbers every production
-- schedule e2e test asserts on.
INSERT IGNORE INTO demand_override (
    id, account_id, scope_code, scope_ref_id,
    period_start_date, period_end_date, override_type_code, value, unit_id,
    reason_code, note, created_by_id, effective_from, expires_at, is_active,
    created_at, updated_at
) VALUES
    ('deov_01seede2eoverride1', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'item', 'it_01k0a7100aeysrs9vxpeq14yxj',
     DATE(NOW(3)) - INTERVAL 18 MONTH, DATE(NOW(3)) - INTERVAL 15 MONTH, 'delta_units', 5000.000000, NULL,
     'new_customer', 'Seeded: Northwind onboarding', 'acus_s83fjhyfmqen',
     DATE_SUB(NOW(3), INTERVAL 18 MONTH), NULL, 1, NOW(3), NOW(3)),
    ('deov_01seede2eoverride2', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product_line', 'pdln_01k0a735ype5e8nrhv1n5dhq1q',
     DATE(NOW(3)) - INTERVAL 17 MONTH, DATE(NOW(3)) - INTERVAL 14 MONTH, 'delta_percent', 10.000000, NULL,
     'promotion', 'Seeded: spring promotion lift', 'acus_s83fjhyfmqen',
     DATE_SUB(NOW(3), INTERVAL 17 MONTH), NULL, 1, NOW(3), NOW(3)),
    ('deov_01seede2eoverride3', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'item', 'it_01k0a7100aeysrs9vxpeq14yxj',
     DATE(NOW(3)) - INTERVAL 16 MONTH, DATE(NOW(3)) - INTERVAL 13 MONTH, 'absolute', 12000.000000, NULL,
     'discontinued', 'Seeded: inactive historical override', 'acus_s83fjhyfmqen',
     DATE_SUB(NOW(3), INTERVAL 16 MONTH), NULL, 0, NOW(3), NOW(3));

-- ============================================================
-- PRODUCTION SCHEDULE DEVIATIONS (for {id}/deviations)
-- ============================================================
-- Two rows so the generic list AND pagination suites have fixtures, one frozen and one
-- not so the `frozen` filter has both sides to choose between. Both reference the
-- seeded schedule line; the removed-line row deliberately has no after snapshot, which
-- is what a NULL JSON column has to survive being read back as.
INSERT IGNORE INTO production_schedule_deviation (
    id, account_id, production_schedule_id, production_schedule_line_id,
    deviation_type_code, is_frozen_week, week_index, machine_id, item_id,
    before_json, after_json, delta_quantity, delta_run_hours,
    reason_code, reason_note, actor_id, created_at
) VALUES
    ('pnscdw_01seede2edev01', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pnsc_01seede2eschedule', 'pnscln_01seede2eline01',
     'quantity_changed', 0, 1, 'mc_01k0a52fb6eqhtbx9hdxj3vvnh', 'it_01k0a7100aeysrs9vxpeq14yxj',
     JSON_OBJECT('id', 'pnscln_01seede2eline01', 'planned_quantity', 100, 'planned_run_hours', 4.0),
     JSON_OBJECT('id', 'pnscln_01seede2eline01', 'planned_quantity', 120, 'planned_run_hours', 4.5),
     20.000000000000000000000000000000, 0.5000,
     'demand_change', 'Seeded: pulled forward', 'acus_s83fjhyfmqen', NOW(3) - INTERVAL 2 HOUR),
    ('pnscdw_01seede2edev02', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pnsc_01seede2eschedule', NULL,
     'line_removed', 1, 0, 'mc_01k0a52fb6eqhtbx9hdxj3vvnh', 'it_01k0a7100aeysrs9vxpeq14yxj',
     JSON_OBJECT('id', 'pnscln_01seedremoved0', 'planned_quantity', 60, 'planned_run_hours', 2.0),
     NULL,
     -60.000000000000000000000000000000, -2.0000,
     'material_shortage', 'Seeded: yarn shortage', 'acus_s83fjhyfmqen', NOW(3) - INTERVAL 1 HOUR);

-- ============================================================
-- DERIVED DEPARTMENT WORK (for {id}/derived-lines)
-- ============================================================
-- Two rows at different depths so the generic list suite has fixtures and the
-- department filter has more than one answer. Derived work is normally regenerated with
-- its schedule; these exist only because the seeded schedule is inserted directly.
INSERT IGNORE INTO production_schedule_derived_line (
    id, account_id, production_schedule_id, source_line_id,
    production_step_id, department_id, item_id,
    week_index, week_start_date, quantity, planned_unit_id,
    explosion_depth, offset_weeks, status_code, created_at, updated_at
) VALUES
    ('pnscdl_01seede2ederiv1', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pnsc_01seede2eschedule', 'pnscln_01seede2eline01',
     'prs_01k0a56yc1e8wag6wexn4pp8t9', 'dp_01k0a5r01yfx3sj1vy9qgv3dc0', 'it_01k0a7100aeysrs9vxpeq14yxj',
     2, DATE(NOW(3) - INTERVAL WEEKDAY(NOW(3)) DAY) + INTERVAL 2 WEEK, 120.000000000000000000000000000000, NULL,
     1, 1, 'planned', NOW(3), NOW(3)),
    ('pnscdl_01seede2ederiv2', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'pnsc_01seede2eschedule', 'pnscln_01seede2eline01',
     'prs_01k0a57f3dfsmtzc8txbq43eth', 'dp_01k0a5r01yfx3sj1vy9qgv3dc0', 'it_01k0a7100aeysrs9vxpeq14yxj',
     3, DATE(NOW(3) - INTERVAL WEEKDAY(NOW(3)) DAY) + INTERVAL 3 WEEK, 120.000000000000000000000000000000, NULL,
     2, 2, 'planned', NOW(3), NOW(3));

-- ============================================================
-- JOB (for GET /v1/core/jobs/{id} include coverage)
-- ============================================================
-- A completed bulk upsert attributed to the seeded admin account user so retrieve-job's created_by / created_by.role includes have a fixture.
INSERT IGNORE INTO job (
    job_id, type, resource_type, account_id, created_by, job_items, results,
    started_at, completed_at, created_at, updated_at
) VALUES (
    'jb_01seedincludejob0',
    'bulk_upsert',
    'unit',
    'ac_01k0a5smf9ekb8rqg12555zjqa',
    'acus_s83fjhyfmqen',
    '[]',
    '{"rows":[{"index":0,"status":"created","resource_type":"unit","id":"un_01seedpair000000000"}],"truncated":false}',
    NOW(3) - INTERVAL 1 MINUTE,
    NOW(3),
    NOW(3) - INTERVAL 1 MINUTE,
    NOW(3)
);
