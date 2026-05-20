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
    ('spml_01seedsupmat2_0000', 'ml_01seedyrn2mat000000', 'ac_01seedsupplier_acct0', 'YRN-EXT-002', 'Premium Yarn Type 2', 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

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
-- DELIVERIES (2 rows for pagination)
-- ============================================================

INSERT IGNORE INTO delivery (id, number, sales_order_id, account_id, delivery_status_code, created_at, updated_at) VALUES
    ('dv_01seeddelivery1_0000', 'DLV-001', 'or_01k0a8bs2yf909wjkd7ecd6x4z', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'accepted', NOW(), NOW()),
    ('dv_01seeddelivery2_0000', 'DLV-002', 'or_01k0a8bs2ye3f9p8sj0m4dfmwe', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'accepted', NOW(), NOW());

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
    ('acin_01seedintegration1', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'shippo', 'Shippo Integration', 'seed-placeholder-credentials', 1, NOW(), NOW()),
    ('acin_01seedintegration2', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'quickbooks', 'QuickBooks Integration', 'seed-placeholder-credentials', 1, NOW(), NOW());

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
INSERT IGNORE INTO audit_event (type_id, actor_id, actor_type, identity_type, account_id, action, resource_type, resource_id, changes, metadata, service_name, request_id, occurred_at, created_at) VALUES
    ('adev_01seedauditevent01', 'us_1wjfmmbwg8l7', 'user', 'internal', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'create', 'unit', 'un_01seedpair000000000', '[{"field":"name","old_value":null,"new_value":"Pair"}]', NULL, 'core-service', NULL, DATE_SUB(NOW(), INTERVAL 1 HOUR), NOW()),
    ('adev_01seedauditevent02', 'us_1wjfmmbwg8l7', 'user', 'internal', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'update', 'property', 'pp_01k0a7ntn1ez6aw8x850femxeh', '[{"field":"name","old_value":"Colour","new_value":"Color"}]', '{"seed":true,"note":"manual e2e seed"}', 'core-service', 'rqlog_01seedreqlog1_000', DATE_ADD(NOW(), INTERVAL 10 YEAR), DATE_ADD(NOW(), INTERVAL 10 YEAR));

-- Backfill request_id on re-seed when INSERT IGNORE skips existing rows.
UPDATE audit_event SET request_id = 'rqlog_01seedreqlog1_000' WHERE type_id = 'adev_01seedauditevent02' AND (request_id IS NULL OR request_id = '');

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
-- ============================================================

INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, occurred_at, created_at) VALUES
    ('rqlog_01seedreqlog1_000', 'GET', 'api.augno.com', '/v1/catalog/items', '/v1/catalog/items', 200, 15000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', DATE_SUB(NOW(), INTERVAL 1 HOUR), NOW()),
    ('rqlog_01seedreqlog2_000', 'POST', 'api.augno.com', '/v1/catalog/units', '/v1/catalog/units', 201, 25000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', NOW(), NOW()),
    ('rqlog_01seedreqlog3_000', 'GET', 'api.augno.com', '/v1/catalog/items', '/v1/catalog/items', 200, 12000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'apky_pajbskcck3cabxajdh8h8', 'api_key', 'api_key', DATE_SUB(NOW(), INTERVAL 30 MINUTE), NOW());

INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, error_code, error_message, occurred_at, created_at) VALUES
    ('rqlog_01seedreqlog4_000', 'POST', 'api.augno.com', '/v1/catalog/units', '/v1/catalog/units', 422, 9000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', 'validation_failed', 'Name is required.', DATE_SUB(NOW(), INTERVAL 2 HOUR), NOW());

INSERT IGNORE INTO request_log (id, method, host, path, normalized_route, query_json, status_code, latency_us, public_endpoint, account_id, target_account_id, actor_id, actor_type, identity_type, occurred_at, created_at) VALUES
    ('rqlog_01seedreqlog5_000', 'GET', 'api.augno.com', '/v1/catalog/items', '/v1/catalog/items', '{"limit":10,"status_codes":["200"]}', 200, 8000, 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', 'user', 'user', DATE_SUB(NOW(), INTERVAL 3 HOUR), NOW());

INSERT IGNORE INTO idempotency_key (type_id, idempotency_key, identity_type, request_method, normalized_route, request_body_hash, scope_hash, recovery_point, target_account_id, actor_id, created_at, updated_at) VALUES
    ('idk_01seedreqlogik001', 'e2e-seed-idempotency-key-01', 'user', 'POST', '/v1/catalog/units', 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb', 'finished', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', NOW(3), NOW(3));

UPDATE request_log SET idempotency_key_id = 'idk_01seedreqlogik001' WHERE id = 'rqlog_01seedreqlog2_000';

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

UPDATE inventory_change_log
   SET scanning_station_id = 'sgsn_01k0a8201zegarjfsjaw5n7yfv',
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
