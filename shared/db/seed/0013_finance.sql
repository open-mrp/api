-- 0013_finance.sql
-- Seeds dc_locations, settlements, transactions, and transaction allocations.
-- Runs after 0012 so all FK dependencies (accounts, invoices, users) exist.

-- ============================================================
-- DC LOCATIONS
-- ============================================================

INSERT IGNORE INTO dc_location (id, location, account_id, owner_account_id, created_at, updated_at) VALUES
    ('dclc_01seeddc_location0', 'Distribution Center East', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- ============================================================
-- SETTLEMENTS
-- ============================================================

INSERT IGNORE INTO settlement (id, number, account_id, responsible_user_id, created_at, updated_at) VALUES
    ('sl_01seedsettlement000', 'STL-001', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'us_1wjfmmbwg8l7', NOW(), NOW());

-- ============================================================
-- TRANSACTIONS
-- ============================================================

-- Transaction amounts (UNIQUE FK quantities)
-- INV-001 total = 25×9.50 + 18×9.50 + 1×20 = 428.50
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedtx_amount0000', 428.50, 'dollar', NOW(), NOW()),
    ('qu_01seedtxal_amount00', 428.50, 'dollar', NOW(), NOW());

INSERT IGNORE INTO transaction (id, number, customer_account_id, amount_id, transaction_type_code, is_fully_allocated, account_id, created_at, updated_at) VALUES
    ('tx_01seedtransaction00', 'TXN-001', 'ac_01k09wm2fgevdsc344gpbcj30f', 'qu_01seedtx_amount0000', 'payment', 1, 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- ============================================================
-- TRANSACTION ALLOCATIONS
-- ============================================================

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedtxal2_amt000', 50.00, 'dollar', NOW(), NOW());

INSERT IGNORE INTO transaction_allocation (id, transaction_id, amount_id, invoice_id, settlement_id, created_at, updated_at) VALUES
    ('txal_01seedtxalloc0000', 'tx_01seedtransaction00', 'qu_01seedtxal_amount00', 'iv_01k09wnac0e1ar211e0sy0ba4g', 'sl_01seedsettlement000', NOW(), NOW()),
    ('txal_01seedtxalloc0002', 'tx_01seedtransaction02', 'qu_01seedtxal2_amt000', 'iv_01seedinvoice002000', 'sl_01seedsettlement000', DATE_SUB(NOW(), INTERVAL 12 HOUR), DATE_SUB(NOW(), INTERVAL 12 HOUR));

-- Mark INV-001 as paid (transaction fully covers invoice total of 428.50)
UPDATE invoice SET is_paid_in_full = 1 WHERE id = 'iv_01k09wnac0e1ar211e0sy0ba4g' AND is_paid_in_full = 0;

-- ============================================================
-- ADDITIONAL UNPAID INVOICES (for receivables / customer-invoices pagination)
-- ============================================================

-- Invoice for issued order (ORD-001)
INSERT IGNORE INTO invoice (id, number, sales_order_id, billing_address_id, account_id, created_at, updated_at) VALUES
    ('iv_01seedinvoice002000', 'INV-002', 'or_01k0a8bs2yejxbsvqhrx4drkq1', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(), INTERVAL 2 DAY), DATE_SUB(NOW(), INTERVAL 2 DAY));

-- Invoice for packed order (ORD-002)
INSERT IGNORE INTO invoice (id, number, sales_order_id, billing_address_id, account_id, created_at, updated_at) VALUES
    ('iv_01seedinvoice003000', 'INV-003', 'or_01k0a8bs2ye3f9p8sj0m4dfmwe', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(), INTERVAL 1 DAY), DATE_SUB(NOW(), INTERVAL 1 DAY));

-- Invoice line quantities for INV-002
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedivln_iss_ln100', 15, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedivln_iss_ln200', 8, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedivln_iss_ln300', 1, 'each', NOW(), NOW());

INSERT IGNORE INTO invoice_line (id, invoice_id, quantity_id, sales_order_line_id, created_at, updated_at) VALUES
    ('ivln_01seediss_ln1_0000', 'iv_01seedinvoice002000', 'qu_01seedivln_iss_ln100', 'orln_01seediss_ln1_0000', NOW(), NOW()),
    ('ivln_01seediss_ln2_0000', 'iv_01seedinvoice002000', 'qu_01seedivln_iss_ln200', 'orln_01seediss_ln2_0000', NOW(), NOW()),
    ('ivln_01seediss_ln3_0000', 'iv_01seedinvoice002000', 'qu_01seedivln_iss_ln300', 'orln_01seediss_ln3_0000', NOW(), NOW());

-- Invoice line quantities for INV-003
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedivln_pck_ln100', 20, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedivln_pck_ln200', 15, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedivln_pck_ln300', 1, 'each', NOW(), NOW());

INSERT IGNORE INTO invoice_line (id, invoice_id, quantity_id, sales_order_line_id, created_at, updated_at) VALUES
    ('ivln_01seedpck_ln1_0000', 'iv_01seedinvoice003000', 'qu_01seedivln_pck_ln100', 'orln_01seedpck_ln1_0000', NOW(), NOW()),
    ('ivln_01seedpck_ln2_0000', 'iv_01seedinvoice003000', 'qu_01seedivln_pck_ln200', 'orln_01seedpck_ln2_0000', NOW(), NOW()),
    ('ivln_01seedpck_ln3_0000', 'iv_01seedinvoice003000', 'qu_01seedivln_pck_ln300', 'orln_01seedpck_ln3_0000', NOW(), NOW());

-- ============================================================
-- ADDITIONAL UNALLOCATED TRANSACTIONS (for open-credits pagination)
-- ============================================================

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedtx2_amount000', 150.00, 'dollar', NOW(), NOW()),
    ('qu_01seedtx3_amount000', 200.00, 'dollar', NOW(), NOW());

INSERT IGNORE INTO transaction (id, number, customer_account_id, amount_id, transaction_type_code, is_fully_allocated, account_id, created_at, updated_at) VALUES
    ('tx_01seedtransaction02', 'TXN-002', 'ac_01k09wm2fgevdsc344gpbcj30f', 'qu_01seedtx2_amount000', 'payment', 0, 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(), INTERVAL 1 DAY), DATE_SUB(NOW(), INTERVAL 1 DAY)),
    ('tx_01seedtransaction03', 'TXN-003', 'ac_01k09wm2fgevdsc344gpbcj30f', 'qu_01seedtx3_amount000', 'payment', 0, 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());
