-- +goose Up

-- Second index pass for hot query paths still relying on sequential scans. Same profile as
-- 00007: not yet a problem at current volume, but agent_action / agent_artifact grow with every
-- tool call across every run, so the unindexed foreign-key fan-outs below degrade with scale.
-- Each index is annotated with the query it serves.

-- ListAgentActionsByRun: WHERE agent_run_id = ? ORDER BY created_at ASC.
-- Per-run action timeline; agent_action has no index other than its PK, so this full-scans + sorts.
CREATE INDEX IF NOT EXISTS agent_action_run_created_idx
    ON agent_action (agent_run_id, created_at);

-- ListAgentArtifactsByAction: WHERE agent_action_id = ? ORDER BY created_at ASC.
-- Per-action artifact list; same unindexed-FK profile as agent_action above.
CREATE INDEX IF NOT EXISTS agent_artifact_action_created_idx
    ON agent_artifact (agent_action_id, created_at);

-- GetAgentDefinitionBySlug / GetAgentDefinitionByAccountAndSlug: WHERE slug = ? (the latter also
-- ANDs the account_id disjunction + is_active). The only btree index is UNIQUE (account_id, slug),
-- which leads with the nullable account_id, so a slug-first lookup can't use its leftmost prefix and
-- falls back to a sequential scan. slug is highly selective, so a standalone index serves both.
CREATE INDEX IF NOT EXISTS agent_definition_slug_idx
    ON agent_definition (slug);

-- PurgePublishedOutboxMessages: WHERE status = 'published' AND published_at < now() - interval LIMIT ?.
-- (status, next_run_at) matches status on its prefix but not the published_at range, so the purge
-- scans every published row. Partial index keeps it to exactly the rows the purge walks.
CREATE INDEX IF NOT EXISTS message_outbox_published_purge_idx
    ON message_outbox (published_at)
    WHERE status = 'published';

-- +goose Down

DROP INDEX IF EXISTS message_outbox_published_purge_idx;
DROP INDEX IF EXISTS agent_definition_slug_idx;
DROP INDEX IF EXISTS agent_artifact_action_created_idx;
DROP INDEX IF EXISTS agent_action_run_created_idx;
