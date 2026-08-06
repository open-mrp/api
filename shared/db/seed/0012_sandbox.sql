-- 0012_sandbox.sql
-- Seeds sandbox account with duplicated infrastructure.

-- Sandbox geolocation
INSERT IGNORE INTO geolocation (id, street_line_1, locality, state, postal_code, country, created_at, updated_at) VALUES
    ('gl_01seedsandboxgeo00000', '123 Main St', 'New York', 'NY', '10001', 'US', NOW(), NOW());

-- Sandbox address
INSERT IGNORE INTO address (id, name, geolocation_id, created_at, updated_at) VALUES
    ('ad_sandbox_01k0a5smf9enr81a4zvyht3zw1', 'Acme Inc. (Sandbox)', 'gl_01seedsandboxgeo00000', NOW(), NOW());

-- Sandbox account
INSERT IGNORE INTO account (id, name, account_type_code, onboarding_status_code, default_billing_address_id, default_shipping_address_id, created_at, updated_at) VALUES
    ('ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'Acme Inc. (Sandbox)', 'sandbox', 'active', 'ad_sandbox_01k0a5smf9enr81a4zvyht3zw1', 'ad_sandbox_01k0a5smf9enr81a4zvyht3zw1', NOW(), NOW());

-- Sandbox account branding
INSERT IGNORE INTO account_branding (id, owner_account_id, support_email, logo_url, created_at, updated_at) VALUES
    ('acbr_01seedsandboxbrand0', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'support@acme.com', 'https://augno-public-images.s3.us-east-2.amazonaws.com/acme-logo.webp', NOW(), NOW());

-- Sandbox account portal
INSERT IGNORE INTO account_portal (id, owner_account_id, slug, created_at, updated_at) VALUES
    ('acpo_01seedsandboxportal', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'acme-inc-sandbox', NOW(), NOW());

-- Sandbox account record
INSERT IGNORE INTO sandbox_account (type_id, owner_account_id, account_id, created_at, updated_at) VALUES
    ('sbac_01seedsandbox000000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW());

-- Sandbox carriers
INSERT IGNORE INTO carrier (id, code, name, account_id, created_at, updated_at) VALUES
    ('cr_01seedsb_delivery000', 'delivery', 'Delivery', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('cr_01seedsb_willcall00', 'will_call', 'Will Call', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW());

-- Sandbox departments (duplicate of main account)
INSERT IGNORE INTO department (id, name, account_id, created_at, updated_at) VALUES
    ('dp_01seedsb_knitting000', 'Knitting', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('dp_01seedsb_washing0000', 'Washing', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('dp_01seedsb_dyeing00000', 'Dyeing', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('dp_01seedsb_sewing00000', 'Sewing', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('dp_01seedsb_boarding000', 'Boarding', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('dp_01seedsb_packing0000', 'Packing', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('dp_01seedsb_inspection0', 'Inspection', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW());

-- Sandbox roles
INSERT IGNORE INTO role (id, name, role_type_code, account_id, created_at, updated_at) VALUES
    ('rl_01seedsb_admin000000', 'Admin', 'admin', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW());

-- Sandbox admin role permissions (full CRUD on all domains)
INSERT IGNORE INTO role_permission (id, role_id, permission_code, `create`, `read`, `update`, `delete`, created_at, updated_at)
SELECT
    CONCAT('rlpm_01seedsb_', SUBSTRING(p.code, 1, 10)),
    'rl_01seedsb_admin000000',
    p.code,
    1, 1, 1, 1,
    NOW(), NOW()
FROM permission p
ON DUPLICATE KEY UPDATE updated_at = NOW();

-- Sandbox account-user (link main admin to sandbox)
INSERT IGNORE INTO account_user (id, user_id, role_id, account_id, last_used_at, created_at, updated_at) VALUES
    ('acus_01seedsb_admin00000', 'us_1wjfmmbwg8l7', 'rl_01seedsb_admin000000', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW(), NOW());

-- Sandbox sys properties
INSERT IGNORE INTO sys_property (id, sys_property_type_code, value, account_id, created_at, updated_at) VALUES
    ('sypp_01seedsb_txnumber0', 'transaction_number', 1000, 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('sypp_01seedsb_slnumber0', 'settlement_number', 1000, 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('sypp_01seedsb_sonumber0', 'sales_order_number', 1000, 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('sypp_01seedsb_ponumber0', 'purchase_order_number', 1000, 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('sypp_01seedsb_supnumbr0', 'supplier_number', 1000, 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('sypp_01seedsb_custnmbr0', 'customer_number', 1000, 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('sypp_01seedsb_sscccount', 'sscc_count', 0, 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('sypp_01seedsb_prnumber0', 'production_run_number', 1000, 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW());

-- Sandbox payment terms
INSERT IGNORE INTO payment_term (id, name, account_id, created_at, updated_at) VALUES
    ('pytm_01seedsb_net30000', 'Net 30', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('pytm_01seedsb_prepaid0', 'Prepaid', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW());

-- ============================================================
-- Sandbox units (account-scoped, same as main but with _sandbox suffix)
-- ============================================================
INSERT IGNORE INTO unit (id, name, abbreviation, unit_dimension_code, account_id, ratio_numerator, ratio_denominator, offset_numerator, offset_denominator, is_base_unit, created_at, updated_at) VALUES
    ('un_01seedsb_dozen00000', 'Dozen Acc',  'dz',   'quantity', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 12, 1, 0, 1, 0, NOW(), NOW()),
    ('un_01seedsb_pair000000', 'Pair Acc',   'pr',   'quantity', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 2, 1, 0, 1, 0, NOW(), NOW()),
    ('un_01seedsb_day0000000', 'Day Acc',    'd',    'time',     'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 24, 1, 0, 1, 0, NOW(), NOW()),
    ('un_01seedsb_pound00000', 'Pound Acc',  'lbs',  'mass',     'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 45359237, 100000, 0, 1, 0, NOW(), NOW()),
    ('un_01seedsb_minute0000', 'Minute Acc', 'min',  'time',     'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 1, 60, 0, 1, 0, NOW(), NOW()),
    ('un_01seedsb_second0000', 'Second Acc', 's',    'time',     'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 1, 3600, 0, 1, 0, NOW(), NOW()),
    ('un_01seedsb_grain00000', 'Grain Acc',  'gr',   'mass',     'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 6479891, 100000000, 0, 1, 0, NOW(), NOW());

-- ============================================================
-- Sandbox unit groups (account-scoped only: socks, sellableSocks, yarn, chemicals)
-- Shared groups (each_group, time_group, currency_group) are reused
-- ============================================================
INSERT IGNORE INTO unit_group (id, name, base_unit_id, account_id, unit_type_code, created_at, updated_at) VALUES
    ('ungp_01seedsb_socks0000', 'Socks', 'un_01seedsb_pair000000', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'quantity', NOW(), NOW()),
    ('ungp_01seedsb_sellsocks', 'Sellable Socks', 'un_01seedsb_pair000000', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'quantity', NOW(), NOW()),
    ('ungp_01seedsb_yarn00000', 'Yarn', 'un_01seedsb_pound00000', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'mass', NOW(), NOW()),
    ('ungp_01seedsb_chemicals', 'Chemicals', 'gram', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'mass', NOW(), NOW());

INSERT IGNORE INTO unit_group_unit (id, unit_group_id, unit_id, created_at, updated_at) VALUES
    -- Socks: each, pair, dozen
    ('ungpun_01seedsb_socksea', 'ungp_01seedsb_socks0000', 'each', NOW(), NOW()),
    ('ungpun_01seedsb_sockspr', 'ungp_01seedsb_socks0000', 'un_01seedsb_pair000000', NOW(), NOW()),
    ('ungpun_01seedsb_socksdz', 'ungp_01seedsb_socks0000', 'un_01seedsb_dozen00000', NOW(), NOW()),
    -- Sellable socks: pair, dozen
    ('ungpun_01seedsb_ssockpr', 'ungp_01seedsb_sellsocks', 'un_01seedsb_pair000000', NOW(), NOW()),
    ('ungpun_01seedsb_ssockdz', 'ungp_01seedsb_sellsocks', 'un_01seedsb_dozen00000', NOW(), NOW()),
    -- Yarn: pound, grain
    ('ungpun_01seedsb_yarnlbs', 'ungp_01seedsb_yarn00000', 'un_01seedsb_pound00000', NOW(), NOW()),
    ('ungpun_01seedsb_yarngr0', 'ungp_01seedsb_yarn00000', 'un_01seedsb_grain00000', NOW(), NOW()),
    -- Chemicals: gram
    ('ungpun_01seedsb_chemg00', 'ungp_01seedsb_chemicals', 'gram', NOW(), NOW());

-- ============================================================
-- Sandbox properties
-- ============================================================
INSERT IGNORE INTO property (id, name, account_id, created_at, updated_at) VALUES
    ('pp_01seedsb_color000000', 'Color', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('pp_01seedsb_size0000000', 'Size', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('pp_01seedsb_twist000000', 'Twist', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('pp_01seedsb_denier00000', 'Denier', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('pp_01seedsb_material000', 'Material', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW());

-- ============================================================
-- Sandbox attributes
-- ============================================================
INSERT IGNORE INTO attribute (id, text, `order`, property_id, color_code, account_id, created_at, updated_at) VALUES
    ('at_01seedsb_beige000000', 'Beige', 1, 'pp_01seedsb_color000000', 'brown', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('at_01seedsb_black000000', 'Black', 2, 'pp_01seedsb_color000000', 'gray', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('at_01seedsb_small000000', 'Small', 1, 'pp_01seedsb_size0000000', 'blue', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('at_01seedsb_medium00000', 'Medium', 2, 'pp_01seedsb_size0000000', 'green', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('at_01seedsb_large000000', 'Large', 3, 'pp_01seedsb_size0000000', 'red', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('at_01seedsb_ztwist00000', 'Z-Twist', 1, 'pp_01seedsb_twist000000', 'red', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('at_01seedsb_stwist00000', 'S-Twist', 2, 'pp_01seedsb_twist000000', 'orange', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('at_01seedsb_denier70000', 'Denier 70', 1, 'pp_01seedsb_denier00000', 'red', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW());

-- ============================================================
-- Sandbox categories
-- Uses sandbox unit groups for socks/yarn/dye/chemicals/packaging,
-- shared unit groups (each_group) for shipping/credit/ebad/label
-- ============================================================
INSERT IGNORE INTO item_category (id, name, item_category_type_code, unit_group_id, account_id, created_at, updated_at) VALUES
    ('itcg_01seedsb_socks0000', 'Socks', 'product_category', 'ungp_01seedsb_socks0000', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('itcg_01seedsb_yarn00000', 'Yarn', 'material_category', 'ungp_01seedsb_yarn00000', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('itcg_01seedsb_dye000000', 'Dye', 'material_category', 'ungp_01seedsb_chemicals', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('itcg_01seedsb_chemicals', 'Chemicals', 'material_category', 'ungp_01seedsb_chemicals', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('itcg_01seedsb_packaging', 'Packaging', 'material_category', 'each_group', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('itcg_01seedsb_shipping0', 'Shipping', 'product_category', 'each_group', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('itcg_01seedsb_credit000', 'Credit', 'product_category', 'each_group', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('itcg_01seedsb_ebad00000', 'eBad', 'product_category', 'each_group', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('itcg_01seedsb_label0000', 'Label', 'material_category', 'each_group', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW());

-- Sandbox category-property associations
INSERT IGNORE INTO _item_categories_properties (A, B) VALUES
    ('itcg_01seedsb_socks0000', 'pp_01seedsb_color000000'),
    ('itcg_01seedsb_socks0000', 'pp_01seedsb_size0000000');

-- ============================================================
-- Sandbox product lines
-- ============================================================
INSERT IGNORE INTO product_line (id, name, unit_group_id, account_id, created_at, updated_at) VALUES
    ('pdln_01seedsb_socks0000', 'Socks', 'ungp_01seedsb_sellsocks', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('pdln_01seedsb_shipping0', 'Shipping', 'each_group', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('pdln_01seedsb_credit000', 'Credit', 'each_group', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('pdln_01seedsb_ebad00000', 'eBad', 'each_group', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('pdln_01seedsb_pace00000', 'Pace', 'ungp_01seedsb_sellsocks', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW());

-- ============================================================
-- Sandbox storage locations (1 building + 6 dept sections + Internal 1 + Customer Held 1)
-- ============================================================
INSERT IGNORE INTO storage_location (id, account_id, storage_location_type_code, name, created_at, updated_at) VALUES
    ('sglc_01seedsb_building0', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'building', 'Main Building', NOW(), NOW()),
    ('sglc_01seedsb_knitting0', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'section', 'Knitting Section', NOW(), NOW()),
    ('sglc_01seedsb_washing00', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'section', 'Washing Section', NOW(), NOW()),
    ('sglc_01seedsb_dyeing000', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'section', 'Dyeing Section', NOW(), NOW()),
    ('sglc_01seedsb_sewing000', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'section', 'Sewing Section', NOW(), NOW()),
    ('sglc_01seedsb_boarding0', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'section', 'Boarding Section', NOW(), NOW()),
    ('sglc_01seedsb_packing00', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'section', 'Packing Section', NOW(), NOW()),
    ('sglc_01seedsb_internal0', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'section', 'Internal 1', NOW(), NOW()),
    ('sglc_01seedsb_custheld0', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'section', 'Customer Held 1', NOW(), NOW());

UPDATE storage_location SET parent_id = 'sglc_01seedsb_building0'
    WHERE id IN (
        'sglc_01seedsb_knitting0', 'sglc_01seedsb_washing00', 'sglc_01seedsb_dyeing000',
        'sglc_01seedsb_sewing000', 'sglc_01seedsb_boarding0', 'sglc_01seedsb_packing00',
        'sglc_01seedsb_internal0', 'sglc_01seedsb_custheld0'
    )
    AND parent_id IS NULL;

-- Sandbox department location_id updates
UPDATE department SET location_id = 'sglc_01seedsb_knitting0' WHERE id = 'dp_01seedsb_knitting000' AND location_id IS NULL;
UPDATE department SET location_id = 'sglc_01seedsb_washing00' WHERE id = 'dp_01seedsb_washing0000' AND location_id IS NULL;
UPDATE department SET location_id = 'sglc_01seedsb_dyeing000' WHERE id = 'dp_01seedsb_dyeing00000' AND location_id IS NULL;
UPDATE department SET location_id = 'sglc_01seedsb_sewing000' WHERE id = 'dp_01seedsb_sewing00000' AND location_id IS NULL;
UPDATE department SET location_id = 'sglc_01seedsb_boarding0' WHERE id = 'dp_01seedsb_boarding000' AND location_id IS NULL;
UPDATE department SET location_id = 'sglc_01seedsb_packing00' WHERE id = 'dp_01seedsb_packing0000' AND location_id IS NULL;

-- ============================================================
-- Sandbox scanning stations (with label_type_code and label_size_code)
-- ============================================================
INSERT IGNORE INTO scanning_station (id, name, scanning_station_type_code, material_check_required, label_type_code, label_size_code, department_id, account_id, created_at, updated_at) VALUES
    ('sgsn_01seedsb_knitting0', 'Knitting Station', 'init_batch', 0, 'tag', '1x4', 'dp_01seedsb_knitting000', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('sgsn_01seedsb_washing00', 'Washing Station', 'move_batch', 1, 'tag', '1x4', 'dp_01seedsb_washing0000', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('sgsn_01seedsb_dyeing000', 'Dyeing Station', 'move_batch', 1, 'tag', '1x4', 'dp_01seedsb_dyeing00000', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('sgsn_01seedsb_sewing000', 'Sewing Station', 'move_batch', 0, 'tag', '1x4', 'dp_01seedsb_sewing00000', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('sgsn_01seedsb_boarding0', 'Boarding Station', 'split_batch', 0, 'tag', '1x4', 'dp_01seedsb_boarding000', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('sgsn_01seedsb_packing00', 'Packing Station', 'move_batch', 0, 'tag', '1x4', 'dp_01seedsb_packing0000', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW()),
    ('sgsn_01seedsb_inspect00', 'Inspection Station', 'split_batch', 0, 'tag', '1x4', 'dp_01seedsb_inspection0', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', NOW(), NOW());

-- ============================================================
-- Sandbox machines (all in knitting department)
-- ============================================================
INSERT IGNORE INTO machine (id, account_id, name, serial_number, department_id, created_at, updated_at) VALUES
    ('mc_01seedsb_machine1000', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'Knitting Machine 1', 'J24-001', 'dp_01seedsb_knitting000', NOW(), NOW()),
    ('mc_01seedsb_machine2000', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'Knitting Machine 2', 'J24-002', 'dp_01seedsb_knitting000', NOW(), NOW()),
    ('mc_01seedsb_machine3000', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'Knitting Machine 3', 'J24-003', 'dp_01seedsb_knitting000', NOW(), NOW());
