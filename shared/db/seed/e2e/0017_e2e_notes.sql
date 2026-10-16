-- 0017_e2e_notes.sql
-- A note on the seed customer, written by OpenMRP as copied notes are. The preview.9 agent memory tests read it as a
-- memory, so it is a preference.
INSERT IGNORE INTO note (id, account_id, record_type, record_id, body, type, importance, actor_type, actor_id, created_at, updated_at) VALUES
    ('nt_01seede2e_note0001', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'customer', 'ac_01k09wm2fgevdsc344gpbcj30f', 'Customer prefers ground shipping for all orders', 'preference', 3, 'system', NULL, NOW(3), NOW(3));
