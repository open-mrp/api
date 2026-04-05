-- +goose Up
-- Seeds agent-service data needed for e2e test coverage.
-- Adds agent definitions, configs, memories, runs, and alerts (2+ rows each for pagination tests).

-- Agent definitions (system-level, no account_id)
INSERT INTO agent_definition (id, name, slug, description, definition_type, category_code, trigger_type, is_active, config)
VALUES
    ('agdf_01seede2e_orderbot0', 'Order Processing Bot', 'order_processing_bot', 'Processes incoming orders automatically', 'system', 'operations', 'event', true, '{"model":"claude-sonnet-4-20250514","max_tokens":4096}'),
    ('agdf_01seede2e_csbot0000', 'Customer Service Bot', 'customer_service_bot', 'Handles customer inquiries', 'system', 'customer_service', 'manual', true, '{"model":"claude-sonnet-4-20250514","max_tokens":4096}')
ON CONFLICT (id) DO NOTHING;

-- Agent configs (account-scoped instances of the definitions)
INSERT INTO agent_config (id, account_id, agent_definition_id, is_enabled, config)
VALUES
    ('agcf_01seede2e_ordercfg0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_orderbot0', true, '{}'),
    ('agcf_01seede2e_cscfg0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_csbot0000', true, '{}')
ON CONFLICT (id) DO NOTHING;

-- Agent memories
INSERT INTO agent_memory (id, account_id, category, content, metadata, entity_type, entity_id, importance)
VALUES
    ('agmm_01seede2e_memory01', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'customer_preference', 'Customer prefers ground shipping for all orders', '{}', 'account_relation', 'acre_01seedcustomer00000', 0.8),
    ('agmm_01seede2e_memory02', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ordering_pattern', 'Customer typically orders 50-100 pairs per month', '{}', 'account_relation', 'acre_01seedcustomer00000', 0.6)
ON CONFLICT (id) DO NOTHING;

-- Agent runs
INSERT INTO agent_run (id, account_id, agent_definition_id, agent_config_id, status_code, trigger_type, input, output, started_at, completed_at, duration_ms, total_input_tokens, total_output_tokens, triggered_by_actor_id, triggered_by_identity_type, triggered_by_actor_name, allowed_tool_slugs)
VALUES
    ('agrn_01seede2e_run00001', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_orderbot0', 'agcf_01seede2e_ordercfg0', 'completed', 'manual', '{"prompt":"Process order ORD-001"}', '{"result":"Order processed successfully"}', now() - interval '1 hour', now() - interval '59 minutes', 60000, 500, 200, 'us_1wjfmmbwg8l7', 'internal', 'Admin User', '[]'),
    ('agrn_01seede2e_run00002', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_csbot0000', 'agcf_01seede2e_cscfg0000', 'completed', 'manual', '{"prompt":"Check customer status"}', '{"result":"Customer is in good standing"}', now() - interval '30 minutes', now() - interval '29 minutes', 45000, 400, 150, 'us_1wjfmmbwg8l7', 'internal', 'Admin User', '[]')
ON CONFLICT (id) DO NOTHING;

-- Agent alerts
INSERT INTO agent_alert (id, account_id, agent_run_id, severity_code, status_code, title, message, metadata)
VALUES
    ('agal_01seede2e_alert001', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agrn_01seede2e_run00001', 'info', 'open', 'Order processed', 'Order ORD-001 was processed automatically', '{}'),
    ('agal_01seede2e_alert002', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agrn_01seede2e_run00002', 'warning', 'open', 'Customer inquiry flagged', 'Customer inquiry requires manual review', '{}')
ON CONFLICT (id) DO NOTHING;

-- Agent account statuses (marks definitions as active for the test account)
INSERT INTO agent_account_status (id, account_id, agent_definition_id, status_code)
VALUES
    ('agas_01seede2e_orderstatus', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_orderbot0', 'active'),
    ('agas_01seede2e_csstatus000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_csbot0000', 'active')
ON CONFLICT (id) DO NOTHING;

-- Agent token usage (for /v1/ai/usage) — unique on (account_id, date)
INSERT INTO agent_token_usage (id, account_id, date, input_tokens, output_tokens, total_cost, run_count)
VALUES
    ('agtu_01seede2e_usage001', 'ac_01k0a5smf9ekb8rqg12555zjqa', CURRENT_DATE - interval '1 day', 900, 350, 0.67, 2),
    ('agtu_01seede2e_usage002', 'ac_01k0a5smf9ekb8rqg12555zjqa', CURRENT_DATE, 500, 200, 0.37, 1)
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM agent_token_usage WHERE id LIKE 'agtu_01seede2e_%';
DELETE FROM agent_alert WHERE id LIKE 'agal_01seede2e_%';
DELETE FROM agent_run WHERE id LIKE 'agrn_01seede2e_%';
DELETE FROM agent_memory WHERE id LIKE 'agmm_01seede2e_%';
DELETE FROM agent_account_status WHERE id LIKE 'agas_01seede2e_%';
DELETE FROM agent_config WHERE id LIKE 'agcf_01seede2e_%';
DELETE FROM agent_definition WHERE id LIKE 'agdf_01seede2e_%';
