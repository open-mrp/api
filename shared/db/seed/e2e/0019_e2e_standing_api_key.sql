-- A standing custom role on a standing API key that no test edits or deletes. The key's created_at is in the
-- future so it heads the newest-first API key list, giving the list-api-keys role include sweep a row that
-- outlives the short-lived keys and roles parallel tests create and delete. The secret hash is random: the
-- key cannot authenticate.
INSERT IGNORE INTO role (id, name, role_type_code, account_id, created_at, updated_at) VALUES
    ('rl_01seedstandingkey00', 'E2E Standing Key Role', 'user', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

INSERT IGNORE INTO role_permission (id, role_id, permission_code, `create`, `read`, `update`, `delete`, created_at, updated_at) VALUES
    ('rlpm_01seedstandingkey0', 'rl_01seedstandingkey00', 'items', 0, 1, 0, 0, NOW(), NOW());

INSERT IGNORE INTO api_key (type_id, key_id, name, secret_hash, redacted_value, owner_account_id, role_id, created_at, updated_at) VALUES
    ('apky_01seedstandingkey000000', 'SeedStandingKeyE2e01', 'E2E Standing Key', UNHEX('91f4c32e7ef10bbf51872d6452b24cd8cd0c53c5831b12d1c241c0631d262905'), 'mrp_sk_prod_****none', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'rl_01seedstandingkey00', DATE_ADD(NOW(), INTERVAL 9 YEAR), DATE_ADD(NOW(), INTERVAL 9 YEAR));
