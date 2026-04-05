-- 0010_customers.sql
-- Seeds customer groups, customers (account relations), customer addresses, and customer users.

-- Customer account (the customer is a separate account)
INSERT IGNORE INTO account (id, name, account_type_code, onboarding_status_code, account_plan_id, created_at, updated_at) VALUES
    ('ac_01k09wm2fgevdsc344gpbcj30f', 'Global Manufacturing Solutions', 'company', 'unclaimed', 'acpl_01seed000free00plan000000', NOW(), NOW());

-- Customer geolocations
INSERT IGNORE INTO geolocation (id, street_line_1, locality, state, postal_code, country, created_at, updated_at) VALUES
    ('gl_01seedcustbillto00000', '789 Mission St', 'San Francisco', 'CA', '94103', 'US', NOW(), NOW()),
    ('gl_01seedcustshipto00000', '2278 W 19th St', 'Cleveland', 'OH', '44113', 'US', NOW(), NOW());

-- Customer addresses
INSERT IGNORE INTO address (id, name, geolocation_id, created_at, updated_at) VALUES
    ('ad_01k09wnac0e1ar211e0sy0ba4g', 'Global Manufacturing Solutions', 'gl_01seedcustbillto00000', NOW(), NOW()),
    ('ad_01k09wnpvrea0awz7vem2j8j7g', 'Global Manufacturing Solutions', 'gl_01seedcustshipto00000', NOW(), NOW());

-- Customer group
INSERT IGNORE INTO account_group (id, owner_account_id, name, account_group_type_code, commission_status_code, freight_status_code, created_at, updated_at) VALUES
    ('acgp_01k0a413mjeth8pe1g70t0thax', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'DME', 'type_group', 'commission_applied', 'billed_freight', NOW(), NOW());

-- Account relation (customer relationship)
INSERT IGNORE INTO account_relation (id, owner_account_id, counterparty_account_id, account_relation_role_code, external_number, is_edi_enabled, priority_code, account_status_code, commission_status_code, freight_status_code, shipping_term_id, payment_term_id, account_group_id, default_billing_address_id, default_shipping_address_id, default_carrier_id, created_at, updated_at) VALUES
    ('acre_01seedcustomer00000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k09wm2fgevdsc344gpbcj30f', 'customer', '45678', 0, 'normal', 'normal', 'commission_applied', 'billed_freight', 'prepaid_billed', 'pytm_01seednet3000000', 'acgp_01k0a413mjeth8pe1g70t0thax', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 'delivery', NOW(), NOW());

-- Account-address associations (link addresses to customer account)
INSERT IGNORE INTO account_address (id, account_id, address_id, created_at, updated_at) VALUES
    ('acad_01seedcustbillto000', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ad_01k09wnac0e1ar211e0sy0ba4g', NOW(), NOW()),
    ('acad_01seedcustshipto000', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ad_01k09wnpvrea0awz7vem2j8j7g', NOW(), NOW());

-- Set default billing/shipping addresses on customer account
UPDATE account SET
    default_billing_address_id = 'ad_01k09wnac0e1ar211e0sy0ba4g',
    default_shipping_address_id = 'ad_01k09wnpvrea0awz7vem2j8j7g'
WHERE id = 'ac_01k09wm2fgevdsc344gpbcj30f'
    AND default_billing_address_id IS NULL;

-- Customer user
INSERT IGNORE INTO user (id, name, username, email, hashed_password, email_verified, created_at, updated_at) VALUES
    ('us_01seedcustuser000000', 'Jane Doe', 'dev@augno.com', 'dev@augno.com', '$2a$10$w68CrxLdi9fdVttqNZMAZesPa2dJlsUrGNy39boKJXz81hyvX0m6y', NOW(), NOW(), NOW());

INSERT IGNORE INTO account_user (id, user_id, account_id, created_at, updated_at) VALUES
    ('acus_01seedcustuser00000', 'us_01seedcustuser000000', 'ac_01k09wm2fgevdsc344gpbcj30f', NOW(), NOW());

-- Registration flow
INSERT IGNORE INTO registration_flow (id, name, account_id, created_at, updated_at) VALUES
    ('mock-registration-flow', 'Mock Registration Flow', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- Account group product line (relevant products)
INSERT IGNORE INTO account_group_product_line (id, account_group_id, product_line_id, created_at, updated_at) VALUES
    ('mock-relevant-products', 'acgp_01k0a413mjeth8pe1g70t0thax', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', NOW(), NOW());

-- ============================================================
-- ACCOUNT PRICES
-- ============================================================

INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedacctprice_val0', 8.5, 'dollar', 'un_01seedpair000000000', NOW(), NOW());

INSERT IGNORE INTO account_price (id, owner_account_id, unit_value_id, product_line_id, recipient_account_id, created_at, updated_at) VALUES
    ('acpr_01seedaccprice0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'rt_01seedacctprice_val0', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', 'ac_01k09wm2fgevdsc344gpbcj30f', NOW(), NOW());

-- ============================================================
-- VOLUME / QUANTITY DISCOUNTS
-- ============================================================

INSERT IGNORE INTO quantity_discount (id, name, account_id, created_at, updated_at) VALUES
    ('quds_01seedvoldiscount0', 'Standard Volume Discount', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- ============================================================
-- PRODUCT LINE ACCESS (customer → product line)
-- ============================================================

INSERT IGNORE INTO account_relation_product_line (id, account_relation_id, product_line_id, created_at, updated_at) VALUES
    ('acrepdln_01seedcustpl0', 'acre_01seedcustomer00000', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', NOW(), NOW());
