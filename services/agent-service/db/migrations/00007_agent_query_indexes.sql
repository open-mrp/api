-- +goose Up

-- Index pass for hot query paths that currently full-scan (not yet flagged in prod only
-- due to low volume, but they will be as agent_run / agent_memory grow). Each index below
-- is annotated with the query it serves.

-- GetLastRunByConfigID: WHERE agent_config_id = ? ORDER BY created_at DESC LIMIT 1.
-- Runs once per scheduled config every scheduler tick (paired with ListEnabledConfigsWithSchedule).
CREATE INDEX IF NOT EXISTS agent_run_config_created_idx
    ON agent_run (agent_config_id, created_at DESC)
    WHERE agent_config_id IS NOT NULL;

-- ListAgentRunsByAccountFiltered: WHERE account_id = ? ... ORDER BY created_at DESC, id DESC (keyset).
CREATE INDEX IF NOT EXISTS agent_run_account_created_idx
    ON agent_run (account_id, created_at DESC, id DESC);

-- GetAgentConfigByAccountAndDefinition: WHERE account_id = ? AND agent_definition_id = ?.
CREATE INDEX IF NOT EXISTS agent_config_account_definition_idx
    ON agent_config (account_id, agent_definition_id);

-- ListMemoriesByEntity / ListAccountMemories: WHERE account_id = ? AND entity_type = ?
-- AND entity_id = ? ORDER BY importance DESC LIMIT ? (hot memory retrieval during runs).
CREATE INDEX IF NOT EXISTS agent_memory_entity_importance_idx
    ON agent_memory (account_id, entity_type, entity_id, importance DESC);

-- ListAgentMemoriesByAccountCursor: WHERE account_id = ? ... ORDER BY created_at DESC, id DESC (keyset).
CREATE INDEX IF NOT EXISTS agent_memory_account_created_idx
    ON agent_memory (account_id, created_at DESC, id DESC);

-- +goose Down

DROP INDEX IF EXISTS agent_memory_account_created_idx;
DROP INDEX IF EXISTS agent_memory_entity_importance_idx;
DROP INDEX IF EXISTS agent_config_account_definition_idx;
DROP INDEX IF EXISTS agent_run_account_created_idx;
DROP INDEX IF EXISTS agent_run_config_created_idx;
