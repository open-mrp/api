-- +goose Up
-- Seeds agent-service data needed for e2e test coverage.
-- Adds agent definitions, configs, memories, runs, and alerts (2+ rows each for pagination tests).

-- Agent definitions (system-level, no account_id)
INSERT INTO agent_definition (id, name, slug, description, definition_type, category_code, trigger_type, is_active, config)
VALUES
    ('agdf_01seede2e_orderbot0', 'Order Processing Bot', 'order_processing_bot', 'Processes incoming orders automatically', 'system', 'operations', 'event', true, '{"model":"claude-sonnet-4","max_tokens":4096}'),
    ('agdf_01seede2e_csbot0000', 'Customer Service Bot', 'customer_service_bot', 'Handles customer inquiries', 'system', 'customer_service', 'manual', true, '{"model":"claude-sonnet-4","max_tokens":4096}')
ON CONFLICT (id) DO NOTHING;

-- Custom agent definition (account-scoped, updatable via PATCH).
-- role_id references the seeded admin role in the core MySQL database so that
-- ?include=role and ?include=role.permissions populate nested data.
INSERT INTO agent_definition (id, account_id, name, slug, description, definition_type, category_code, trigger_type, is_active, config, role_id)
VALUES
    ('agdf_01seede2e_custom00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'Custom Test Agent', 'custom_test_agent', 'A custom agent for e2e update tests', 'custom', 'operations', 'manual', true, '{"model":"claude-sonnet-4","max_tokens":4096}', 'rl_mtg88e6u6fbu')
ON CONFLICT (id) DO NOTHING;

-- Second custom agent definition whose per-account status is 'inactive' (set in
-- agent_account_status below), so the /v1/ai/agents `statuses` array filter has
-- two distinct values (active/inactive) to combine — see
-- TestArrayFilters_UnionExclusion.
INSERT INTO agent_definition (id, account_id, name, slug, description, definition_type, category_code, trigger_type, is_active, config)
VALUES
    ('agdf_01seede2e_inact00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'Inactive Test Agent', 'inactive_test_agent', 'A custom agent that is inactive for e2e status-filter coverage', 'custom', 'operations', 'manual', true, '{"model":"claude-sonnet-4","max_tokens":4096}')
ON CONFLICT (id) DO NOTHING;

-- Agent definition referenced as the actor of the infra-scrub request log
-- (rqlog_01infraagent0 in shared/db/seed/0014_e2e_extras.sql). The request-logs
-- presenter resolves an agent actor's name + handle(slug) from agent-service, so
-- this row must exist for ?include=actor to hydrate them.
INSERT INTO agent_definition (id, account_id, name, slug, description, definition_type, category_code, trigger_type, is_active, config)
VALUES
    ('agdf_01infraseedagent', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'Infra Scrub Agent', 'infra_scrub_agent', 'Agent that made the internal request used for infra-scrub coverage', 'custom', 'operations', 'event', true, '{"model":"claude-sonnet-4","max_tokens":4096}')
ON CONFLICT (id) DO NOTHING;

-- Agent definition tools — attaches two system tools to the custom agent so
-- `?include=tools` returns a populated list on the GET/LIST responses. Also
-- attaches one tool to orderbot0 so get-run?include=definition.tools resolves
-- against SeedAgentRunID (run #1, which targets orderbot0).
INSERT INTO agent_definition_tool (id, agent_definition_id, tool_slug, config, sort_order, require_review)
VALUES
    ('agdtl_01seede2e_tool001', 'agdf_01seede2e_custom00', 'save_memory', '{}', 0, false),
    ('agdtl_01seede2e_tool002', 'agdf_01seede2e_custom00', 'create_alert', '{}', 1, false),
    ('agdtl_01seede2e_tool003', 'agdf_01seede2e_orderbot0', 'create_alert', '{}', 0, false)
ON CONFLICT (id) DO NOTHING;

-- orderbot0 is a system definition but still needs a role_id so
-- `get-run/{id}?include=definition.role` resolves against SeedAgentRunID.
UPDATE agent_definition
   SET role_id = 'rl_mtg88e6u6fbu'
 WHERE id = 'agdf_01seede2e_orderbot0'
   AND role_id IS NULL;

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

-- Agent runs — run #3 is tied to the custom agent (which has role + tools) so
-- list-runs/{definition.config,definition.role,definition.tools,actions} have
-- at least one populated row.
INSERT INTO agent_run (id, account_id, agent_definition_id, agent_config_id, status_code, trigger_type, input, output, started_at, completed_at, duration_ms, total_input_tokens, total_output_tokens, triggered_by_actor_id, triggered_by_identity_type, triggered_by_actor_name, allowed_tool_slugs)
VALUES
    ('agrn_01seede2e_run00001', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_orderbot0', 'agcf_01seede2e_ordercfg0', 'completed', 'manual', '{"prompt":"Process order ORD-001"}', '{"result":"Order processed successfully"}', now() - interval '1 hour', now() - interval '59 minutes', 60000, 500, 200, 'us_1wjfmmbwg8l7', 'internal', 'Admin User', '[]'),
    ('agrn_01seede2e_run00002', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_csbot0000', 'agcf_01seede2e_cscfg0000', 'completed', 'manual', '{"prompt":"Check customer status"}', '{"result":"Customer is in good standing"}', now() - interval '30 minutes', now() - interval '29 minutes', 45000, 400, 150, 'us_1wjfmmbwg8l7', 'internal', 'Admin User', '[]'),
    ('agrn_01seede2e_run00003', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_custom00', NULL,                     'completed', 'manual', '{"prompt":"Custom agent trigger"}', '{"result":"Custom agent completed"}',          now() - interval '15 minutes', now() - interval '14 minutes', 30000, 300, 100, 'us_1wjfmmbwg8l7', 'internal', 'Admin User', '["save_memory","create_alert"]'),
    -- runfail1: dedicated terminal 'failed' run for the retry-success happy path (SeedAgentRunFailedID). Do not reuse for other tests — Retry flips it permanently.
    ('agrn_01seede2e_runfail1', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_orderbot0', 'agcf_01seede2e_ordercfg0', 'failed',         'manual', '{"prompt":"Process order that fails"}', '{}',                                     now() - interval '10 minutes', now() - interval '9 minutes', 60000, 100, 0, 'us_1wjfmmbwg8l7', 'internal', 'Admin User', '[]'),
    -- runawti1: dedicated 'awaiting_input' run for the continue-success happy path (SeedAgentRunAwaitingInputID). Do not reuse — Continue flips it permanently.
    ('agrn_01seede2e_runawti1', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_orderbot0', 'agcf_01seede2e_ordercfg0', 'awaiting_input', 'manual', '{"prompt":"Need more info"}', '{}',                                     now() - interval '5 minutes', NULL, NULL, 100, 0, 'us_1wjfmmbwg8l7', 'internal', 'Admin User', '[]')
ON CONFLICT (id) DO NOTHING;

-- Agent actions. Run #3 exercises `list-runs/actions` and the alert→action
-- link. Run #1 carries one action + step events so `get-run/actions` and
-- `get-run/steps` resolve against the SeedAgentRunID GET target.
INSERT INTO agent_action (id, account_id, agent_run_id, tool_slug, status_code, label, description, input, output, requires_review)
VALUES
    ('agac_01seede2e_action01', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agrn_01seede2e_run00003', 'save_memory', 'completed', 'Remembered preference', 'Saved the customer''s preferred shipping method.', '{"category":"preference"}', '{"ok":true}', false),
    ('agac_01seede2e_action02', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agrn_01seede2e_run00001', 'create_alert', 'completed', 'Order queued', 'Queued ORD-001 for downstream fulfillment.', '{"order_id":"ORD-001"}', '{"ok":true}', false)
ON CONFLICT (id) DO NOTHING;

-- Agent run events (timeline steps) for run #1 so `get-run/steps` resolves.
INSERT INTO agent_run_event (id, agent_run_id, account_id, step_type, title, content, sequence, duration_ms, agent_action_id, metadata, actor_type, actor_name)
VALUES
    ('agev_01seede2e_event001', 'agrn_01seede2e_run00001', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'message',   'User prompt received', 'Process order ORD-001',                    1, 20,    NULL,                     '{}', 'internal', 'Admin User'),
    ('agev_01seede2e_event002', 'agrn_01seede2e_run00001', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'tool_call', 'Queued order for fulfillment', 'create_alert invoked with ORD-001', 2, 30000, 'agac_01seede2e_action02', '{}', 'internal', 'Admin User')
ON CONFLICT (id) DO NOTHING;

-- Agent account statuses (marks definitions as active for the test account)
INSERT INTO agent_account_status (id, account_id, agent_definition_id, status_code)
VALUES
    ('agas_01seede2e_orderstatus', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_orderbot0', 'active'),
    ('agas_01seede2e_csstatus000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_csbot0000', 'active'),
    ('agas_01seede2e_customstat0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_custom00', 'active'),
    ('agas_01seede2e_inactstat0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_inact00', 'inactive')
ON CONFLICT (id) DO NOTHING;

-- Agent token usage (for /v1/ai/usage) — unique on (account_id, date)
INSERT INTO agent_token_usage (id, account_id, date, input_tokens, output_tokens, total_cost, run_count)
VALUES
    ('agtu_01seede2e_usage001', 'ac_01k0a5smf9ekb8rqg12555zjqa', CURRENT_DATE - interval '1 day', 900, 350, 0.67, 2),
    ('agtu_01seede2e_usage002', 'ac_01k0a5smf9ekb8rqg12555zjqa', CURRENT_DATE, 500, 200, 0.37, 1)
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM agent_token_usage WHERE id LIKE 'agtu_01seede2e_%';
DELETE FROM agent_run WHERE id LIKE 'agrn_01seede2e_%';
DELETE FROM agent_memory WHERE id LIKE 'agmm_01seede2e_%';
DELETE FROM agent_account_status WHERE id LIKE 'agas_01seede2e_%';
DELETE FROM agent_config WHERE id LIKE 'agcf_01seede2e_%';
DELETE FROM agent_definition WHERE id LIKE 'agdf_01seede2e_%';
