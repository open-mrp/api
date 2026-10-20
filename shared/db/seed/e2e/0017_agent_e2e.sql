-- 0017_agent_e2e.sql
-- Seeds agent-service data needed for e2e test coverage.
-- Adds agent definitions, configs, memories, runs, and alerts (2+ rows each for pagination tests).

-- Agent definitions (system-level, no account_id)
INSERT IGNORE INTO agent_definition (id, name, slug, description, definition_type, category_code, trigger_type, is_active, config)
VALUES
    ('agdf_01seede2e_orderbot0', 'Order Processing Bot', 'order_processing_bot', 'Processes incoming orders automatically', 'system', 'operations', 'event', TRUE, '{"model":"claude-sonnet-4","max_tokens":4096}'),
    ('agdf_01seede2e_csbot0000', 'Customer Service Bot', 'customer_service_bot', 'Handles customer inquiries', 'system', 'customer_service', 'manual', TRUE, '{"model":"claude-sonnet-4","max_tokens":4096}');

-- Custom agent definition (account-scoped, updatable via PATCH).
-- role_id references the seeded admin role so that
-- ?include=role and ?include=role.permissions populate nested data.
INSERT IGNORE INTO agent_definition (id, account_id, name, slug, description, definition_type, category_code, trigger_type, is_active, config, role_id)
VALUES
    ('agdf_01seede2e_custom00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'Custom Test Agent', 'custom_test_agent', 'A custom agent for e2e update tests', 'custom', 'operations', 'manual', TRUE, '{"model":"claude-sonnet-4","max_tokens":4096}', 'rl_mtg88e6u6fbu');

-- Second custom agent definition whose per-account status is 'inactive' (set in
-- agent_account_status below), so the /v1/ai/agents `statuses` array filter has
-- two distinct values (active/inactive) to combine — see
-- TestArrayFilters_UnionExclusion.
INSERT IGNORE INTO agent_definition (id, account_id, name, slug, description, definition_type, category_code, trigger_type, is_active, config)
VALUES
    ('agdf_01seede2e_inact00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'Inactive Test Agent', 'inactive_test_agent', 'A custom agent that is inactive for e2e status-filter coverage', 'custom', 'operations', 'manual', TRUE, '{"model":"claude-sonnet-4","max_tokens":4096}');

-- Agent definition referenced as the actor of the infra-scrub request log
-- (rqlog_01infraagent0 in 0014_e2e_extras.sql). The request-logs presenter
-- resolves an agent actor's name + handle(slug) from agent-service, so this
-- row must exist for ?include=actor to hydrate them.
INSERT IGNORE INTO agent_definition (id, account_id, name, slug, description, definition_type, category_code, trigger_type, is_active, config)
VALUES
    ('agdf_01infraseedagent', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'Infra Scrub Agent', 'infra_scrub_agent', 'Agent that made the internal request used for infra-scrub coverage', 'custom', 'operations', 'event', TRUE, '{"model":"claude-sonnet-4","max_tokens":4096}');

-- Agent definition tools — attaches two system tools to the custom agent so
-- `?include=tools` returns a populated list on the GET/LIST responses. Also
-- attaches one tool to orderbot0 so get-run?include=definition.tools resolves
-- against SeedAgentRunID (run #1, which targets orderbot0).
INSERT IGNORE INTO agent_definition_tool (id, agent_definition_id, tool_slug, config, sort_order, require_review)
VALUES
    ('agdtl_01seede2e_tool001', 'agdf_01seede2e_custom00', 'create_artifact', '{}', 0, FALSE),
    ('agdtl_01seede2e_tool002', 'agdf_01seede2e_custom00', 'send_email', '{}', 1, FALSE),
    ('agdtl_01seede2e_tool003', 'agdf_01seede2e_orderbot0', 'draft_reply', '{}', 0, FALSE);

-- orderbot0 is a system definition but still needs a role_id so
-- `get-run/{id}?include=definition.role` resolves against SeedAgentRunID.
UPDATE agent_definition
   SET role_id = 'rl_mtg88e6u6fbu'
 WHERE id = 'agdf_01seede2e_orderbot0'
   AND role_id IS NULL;

-- Agent configs (account-scoped instances of the definitions)
INSERT IGNORE INTO agent_config (id, account_id, agent_definition_id, is_enabled, config)
VALUES
    ('agcf_01seede2e_ordercfg0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_orderbot0', TRUE, '{}'),
    ('agcf_01seede2e_cscfg0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_csbot0000', TRUE, '{}');

-- Agent memories
INSERT IGNORE INTO agent_memory (id, account_id, category, content, metadata, entity_type, entity_id, importance)
VALUES
    ('agmm_01seede2e_memory01', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'customer_preference', 'Customer prefers ground shipping for all orders', '{}', 'account_relation', 'acre_01seedcustomer00000', 0.8),
    ('agmm_01seede2e_memory02', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ordering_pattern', 'Customer typically orders 50-100 pairs per month', '{}', 'account_relation', 'acre_01seedcustomer00000', 0.6);

-- Agent runs — run #3 is tied to the custom agent (which has role + tools) so
-- list-runs/{definition.config,definition.role,definition.tools,actions} have
-- at least one populated row.
-- Reseeding restores the mutable lifecycle columns: the retry/continue tests flip runfail1/runawti1
-- permanently and the e2e volume persists across local runs, so INSERT IGNORE would leave them mutated.
INSERT INTO agent_run (id, account_id, agent_definition_id, agent_config_id, status_code, trigger_type, input, output, started_at, completed_at, duration_ms, total_input_tokens, total_output_tokens, triggered_by_actor_id, triggered_by_identity_type, triggered_by_actor_name, allowed_tool_slugs)
VALUES
    ('agrn_01seede2e_run00001', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_orderbot0', 'agcf_01seede2e_ordercfg0', 'completed', 'manual', '{"prompt":"Process order ORD-001"}', '{"result":"Order processed successfully"}', NOW(6) - INTERVAL 1 HOUR, NOW(6) - INTERVAL 59 MINUTE, 60000, 500, 200, 'us_1wjfmmbwg8l7', 'internal', 'Admin User', '[]'),
    ('agrn_01seede2e_run00002', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_csbot0000', 'agcf_01seede2e_cscfg0000', 'completed', 'manual', '{"prompt":"Check customer status"}', '{"result":"Customer is in good standing"}', NOW(6) - INTERVAL 30 MINUTE, NOW(6) - INTERVAL 29 MINUTE, 45000, 400, 150, 'us_1wjfmmbwg8l7', 'internal', 'Admin User', '[]'),
    ('agrn_01seede2e_run00003', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_custom00', NULL, 'completed', 'manual', '{"prompt":"Custom agent trigger"}', '{"result":"Custom agent completed"}', NOW(6) - INTERVAL 15 MINUTE, NOW(6) - INTERVAL 14 MINUTE, 30000, 300, 100, 'us_1wjfmmbwg8l7', 'internal', 'Admin User', '["save_memory","create_alert"]'),
    -- runfail1: dedicated terminal 'failed' run for the retry-success happy path (SeedAgentRunFailedID). Do not reuse for other tests — Retry flips it permanently.
    ('agrn_01seede2e_runfail1', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_orderbot0', 'agcf_01seede2e_ordercfg0', 'failed', 'manual', '{"prompt":"Process order that fails"}', '{}', NOW(6) - INTERVAL 10 MINUTE, NOW(6) - INTERVAL 9 MINUTE, 60000, 100, 0, 'us_1wjfmmbwg8l7', 'internal', 'Admin User', '[]'),
    -- runawti1: dedicated 'awaiting_input' run for the continue-success happy path (SeedAgentRunAwaitingInputID). Do not reuse — Continue flips it permanently.
    ('agrn_01seede2e_runawti1', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_orderbot0', 'agcf_01seede2e_ordercfg0', 'awaiting_input', 'manual', '{"prompt":"Need more info"}', '{}', NOW(6) - INTERVAL 5 MINUTE, NULL, NULL, 100, 0, 'us_1wjfmmbwg8l7', 'internal', 'Admin User', '[]')
ON DUPLICATE KEY UPDATE
    status_code         = VALUES(status_code),
    output              = VALUES(output),
    error_message       = NULL,
    started_at          = VALUES(started_at),
    completed_at        = VALUES(completed_at),
    duration_ms         = VALUES(duration_ms),
    total_input_tokens  = VALUES(total_input_tokens),
    total_output_tokens = VALUES(total_output_tokens),
    -- Retrying increments this and the cap is MaxManualRetries, so leaving it alone lets runfail1
    -- survive exactly five reseeds before the happy-path test starts failing on the cap instead.
    retry_count         = 0,
    updated_at          = NOW(6);

-- Agent actions. Run #3 exercises `list-runs/actions` and the alert→action
-- link. Run #1 carries one action + step events so `get-run/actions` and
-- `get-run/steps` resolve against the SeedAgentRunID GET target.
INSERT IGNORE INTO agent_action (id, account_id, agent_run_id, tool_slug, status_code, label, description, input, output, requires_review)
VALUES
    ('agac_01seede2e_action01', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agrn_01seede2e_run00003', 'save_memory', 'completed', 'Remembered preference', 'Saved the customer''s preferred shipping method.', '{"category":"preference"}', '{"ok":true}', FALSE),
    ('agac_01seede2e_action02', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agrn_01seede2e_run00001', 'create_alert', 'completed', 'Order queued', 'Queued ORD-001 for downstream fulfillment.', '{"order_id":"ORD-001"}', '{"ok":true}', FALSE);

-- Agent run events (timeline steps) for run #1 so `get-run/steps` resolves.
INSERT IGNORE INTO agent_run_event (id, agent_run_id, account_id, step_type, title, content, sequence, duration_ms, agent_action_id, metadata, actor_type, actor_name)
VALUES
    ('agev_01seede2e_event001', 'agrn_01seede2e_run00001', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'message',   'User prompt received', 'Process order ORD-001',                    1, 20,    NULL,                     '{}', 'internal', 'Admin User'),
    ('agev_01seede2e_event002', 'agrn_01seede2e_run00001', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'tool_call', 'Queued order for fulfillment', 'create_alert invoked with ORD-001', 2, 30000, 'agac_01seede2e_action02', '{}', 'internal', 'Admin User');

-- Agent account statuses (marks definitions as active for the test account)
INSERT IGNORE INTO agent_account_status (id, account_id, agent_definition_id, status_code)
VALUES
    ('agas_01seede2e_orderstatus', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_orderbot0', 'active'),
    ('agas_01seede2e_csstatus000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_csbot0000', 'active'),
    ('agas_01seede2e_customstat0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_custom00', 'active'),
    ('agas_01seede2e_inactstat0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'agdf_01seede2e_inact00', 'inactive');

-- Agent token usage (for /v1/ai/usage) — unique on (account_id, date)
INSERT IGNORE INTO agent_token_usage (id, account_id, `date`, input_tokens, output_tokens, total_cost, run_count)
VALUES
    ('agtu_01seede2e_usage001', 'ac_01k0a5smf9ekb8rqg12555zjqa', UTC_DATE() - INTERVAL 1 DAY, 900, 350, 0.67, 2),
    ('agtu_01seede2e_usage002', 'ac_01k0a5smf9ekb8rqg12555zjqa', UTC_DATE(), 500, 200, 0.37, 1);
