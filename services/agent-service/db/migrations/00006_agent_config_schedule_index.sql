-- +goose Up

-- ListEnabledConfigsWithSchedule (the scheduler poll) full-scanned agent_config to
-- filter (is_enabled = true AND schedule IS NOT NULL). This partial index restricts to
-- the small set of enabled, scheduled configs and keys on agent_definition_id so the
-- join to agent_definition can use it. agent_definition is reached via its primary key,
-- so no additional index is needed on that side.
CREATE INDEX agent_config_enabled_scheduled_idx
    ON agent_config (agent_definition_id)
    WHERE is_enabled = true AND schedule IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS agent_config_enabled_scheduled_idx;
