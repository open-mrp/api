-- 0015_tenant_b_e2e.sql
-- Seeds a second tenant (Tenant B) for e2e tenant isolation testing.
-- This account is completely independent from the primary account (Tenant A)
-- and is used to verify cross-tenant data isolation.

-- Tenant B account
INSERT IGNORE INTO account (id, name, account_type_code, onboarding_status_code, created_at, updated_at) VALUES
    ('ac_tenant2_e2e_isolati', 'Tenant B Isolation Co', 'company', 'active', NOW(), NOW());

-- Tenant B role (admin with full permissions)
INSERT IGNORE INTO role (id, name, role_type_code, account_id, created_at, updated_at) VALUES
    ('rl_tenant2_e2e_admin00', 'Admin', 'admin', 'ac_tenant2_e2e_isolati', NOW(), NOW());

-- Tenant B admin role permissions (full CRUD on all domains)
INSERT IGNORE INTO role_permission (id, role_id, permission_code, `create`, `read`, `update`, `delete`, created_at, updated_at)
SELECT
    CONCAT('rlpm_tn2e2e_', SUBSTRING(p.code, 1, 10)),
    'rl_tenant2_e2e_admin00',
    p.code,
    1, 1, 1, 1,
    NOW(), NOW()
FROM permission p
ON DUPLICATE KEY UPDATE updated_at = NOW();

-- Tenant B account-user (link existing user to tenant B)
INSERT IGNORE INTO account_user (id, user_id, role_id, account_id, last_used_at, created_at, updated_at) VALUES
    ('acus_tenant2_e2e_admin', 'us_1wjfmmbwg8l7', 'rl_tenant2_e2e_admin00', 'ac_tenant2_e2e_isolati', NOW(), NOW(), NOW());

-- Tenant B API key
-- Full key: aug_sk_prod_TenantBKeyForE2eTests1_TenantBSecretForE2eIsolationTestingPurpose12didR71
-- HMAC-SHA256(pepper='pepper', secret='TenantBSecretForE2eIsolationTestingPurpose12')
INSERT IGNORE INTO api_key (type_id, key_id, name, secret_hash, redacted_value, owner_account_id, role_id, created_at, updated_at) VALUES
    ('apky_tenant2_e2e_key00', 'TenantBKeyForE2eTests1', 'Tenant B E2E Key', UNHEX('9928395362577e4c62f4f93f116de080502392523d9de7140ac7b3c5e7d66ba8'), 'aug_sk_prod_****dR71', 'ac_tenant2_e2e_isolati', 'rl_tenant2_e2e_admin00', NOW(), NOW());

-- Tenant B sys properties (required for auto-number generation)
INSERT IGNORE INTO sys_property (id, sys_property_type_code, value, account_id, created_at, updated_at) VALUES
    ('sypp_tn2e2e_txnumber00', 'transaction_number', 1000, 'ac_tenant2_e2e_isolati', NOW(), NOW()),
    ('sypp_tn2e2e_slnumber00', 'settlement_number', 1000, 'ac_tenant2_e2e_isolati', NOW(), NOW()),
    ('sypp_tn2e2e_sonumber00', 'sales_order_number', 1000, 'ac_tenant2_e2e_isolati', NOW(), NOW()),
    ('sypp_tn2e2e_ponumber00', 'purchase_order_number', 1000, 'ac_tenant2_e2e_isolati', NOW(), NOW()),
    ('sypp_tn2e2e_supnumber0', 'supplier_number', 1000, 'ac_tenant2_e2e_isolati', NOW(), NOW()),
    ('sypp_tn2e2e_custnumbr0', 'customer_number', 1000, 'ac_tenant2_e2e_isolati', NOW(), NOW()),
    ('sypp_tn2e2e_sscccount0', 'sscc_count', 0, 'ac_tenant2_e2e_isolati', NOW(), NOW()),
    ('sypp_tn2e2e_prnumber00', 'production_run_number', 1000, 'ac_tenant2_e2e_isolati', NOW(), NOW());
