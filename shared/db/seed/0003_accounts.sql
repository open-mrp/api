-- 0003_accounts.sql
-- Seeds main account, billing, branding, portal, addresses, and carriers.
-- Uses @plan_id variable (substituted by seed script).

-- Account billing record
INSERT IGNORE INTO account_billing (id, account_plan_id, subscription_status, created_at, updated_at) VALUES
    ('acbl_01seedacmebilling0000', @plan_id, 'active', NOW(), NOW());

-- Geolocation for bill-to address
INSERT IGNORE INTO geolocation (id, street_line_1, locality, state, postal_code, country, google_place_id, created_at, updated_at) VALUES
    ('gl_01seedacmegeo000000000', '123 Main St', 'New York', 'NY', '10001', 'US', 'ChIJN1gggt_t2Z44AR4PVM_67p73Y', NOW(), NOW());

-- Bill-to/Ship-to address (same for seed)
INSERT IGNORE INTO address (id, name, geolocation_id, created_at, updated_at) VALUES
    ('ad_01k0a5smf9enr81a4zvyht3zw0', 'Acme Inc.', 'gl_01seedacmegeo000000000', NOW(), NOW());

-- Main account
INSERT IGNORE INTO account (id, name, account_type_code, onboarding_status_code, account_billing_id, default_billing_address_id, default_shipping_address_id, created_at, updated_at) VALUES
    ('ac_01k0a5smf9ekb8rqg12555zjqa', 'Acme Inc.', 'company', 'active', 'acbl_01seedacmebilling0000', 'ad_01k0a5smf9enr81a4zvyht3zw0', 'ad_01k0a5smf9enr81a4zvyht3zw0', NOW(), NOW());

-- Account branding
INSERT IGNORE INTO account_branding (id, owner_account_id, support_email, logo_url, created_at, updated_at) VALUES
    ('acbr_01seedacmebranding00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'support@acme.com', 'https://augno-public-images.s3.us-east-2.amazonaws.com/acme-logo.webp', NOW(), NOW());

-- Account portal
INSERT IGNORE INTO account_portal (id, owner_account_id, slug, created_at, updated_at) VALUES
    ('acpo_01seedacmeportal000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'acme-inc', NOW(), NOW());

-- Custom portal domain (verified) for the Acme account
INSERT IGNORE INTO portal_domain (id, account_id, domain, status, dns_records, verified_at, created_at, updated_at) VALUES
    ('podn_01seedacmeportaldm', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'shop.acme.com', 'verified',
     '[{"type":"CNAME","name":"shop.acme.com","value":"cname.vercel-dns.com","reason":"routing"}]', NOW(), NOW(), NOW());

-- Carriers (from mockCarriers — delivery and will_call)
INSERT IGNORE INTO carrier (id, code, name, account_id, created_at, updated_at) VALUES
    ('delivery', 'delivery', 'Delivery', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('will_call', 'will_call', 'Will Call', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- Service levels for the delivery carrier. Base fixture both dev and e2e need: rate-shop and
-- the sales-order shipping-rate cascade emit one option per service level, so without these the
-- account returns no shipping options at all. The tokens map to Shippo's service levels for live rating.
INSERT IGNORE INTO carrier_option (id, code, name, service_level_token, carrier_id, account_id, created_at, updated_at) VALUES
    ('crop_01seedground000000', 'ground', 'Ground Shipping', 'fedex_ground', 'delivery', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('crop_01seedexpress00000', 'express', 'Express Shipping', 'fedex_express', 'delivery', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- Default payment term (system-level, no account)
INSERT IGNORE INTO payment_term (id, name, account_id, created_at, updated_at) VALUES
    ('pytm_01seeddefault00000', 'Due on Receipt', NULL, NOW(), NOW());

-- Payment terms (account-scoped, no code column)
INSERT IGNORE INTO payment_term (id, name, account_id, created_at, updated_at) VALUES
    ('pytm_01seedcod00000000', 'COD', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('pytm_01seedcia00000000', 'CIA', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('pytm_01seedccd00000000', 'CCD', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('pytm_01seednet3000000', 'Net 30', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('pytm_01seednet4500000', 'Net 45', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('pytm_01seednet6000000', 'Net 60', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('pytm_01seednet9000000', 'Net 90', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('pytm_01seedprepaid0000', 'Prepaid', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- Sys properties (counters for number generation)
INSERT IGNORE INTO sys_property (id, sys_property_type_code, value, account_id, created_at, updated_at) VALUES
    ('sypp_01seedtxnumber000', 'transaction_number', 1000, 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('sypp_01seedslnumber000', 'settlement_number', 1000, 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('sypp_01seedsonumber000', 'sales_order_number', 1000, 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('sypp_01seedponumber000', 'purchase_order_number', 1000, 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('sypp_01seedsupnumber00', 'supplier_number', 1000, 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('sypp_01seedcustnumber0', 'customer_number', 1000, 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('sypp_01seedsscccount00', 'sscc_count', 0, 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('sypp_01seedprnumber000', 'production_run_number', 1000, 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());
