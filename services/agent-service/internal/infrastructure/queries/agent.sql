-- Agent Definition queries

-- name: GetAgentDefinitionByID :one
SELECT * FROM agent_definition WHERE id = ? AND is_active = TRUE;

-- name: GetAgentDefinitionBySlug :one
SELECT * FROM agent_definition WHERE slug = ?;

-- name: ListAgentDefinitions :many
SELECT * FROM agent_definition WHERE is_active = TRUE ORDER BY name ASC;

-- name: InsertAgentDefinition :exec
INSERT INTO agent_definition (id, account_id, name, slug, description, definition_type, category_code, trigger_type, is_active, config, role_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateAgentDefinition :exec
UPDATE agent_definition SET
    name = COALESCE(sqlc.narg('name'), name),
    slug = COALESCE(sqlc.narg('slug'), slug),
    description = IF(sqlc.arg('clear_description'), NULL, COALESCE(sqlc.narg('description'), description)),
    category_code = COALESCE(sqlc.narg('category_code'), category_code),
    trigger_type = COALESCE(sqlc.narg('trigger_type'), trigger_type),
    config = IF(sqlc.arg('update_config'), CAST(sqlc.arg('new_config') AS JSON), config),
    role_id = IF(sqlc.arg('clear_role_id'), NULL, COALESCE(sqlc.narg('role_id'), role_id)),
    updated_at = NOW(6)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: SoftDeleteAgentDefinition :exec
UPDATE agent_definition
SET is_active = FALSE, updated_at = NOW(6)
WHERE id = ? AND account_id = ?;

-- name: ListAgentDefinitionsByAccount :many
SELECT * FROM agent_definition
WHERE is_active = TRUE AND (account_id IS NULL OR account_id = ?)
ORDER BY definition_type ASC, name ASC;

-- name: GetAgentDefinitionByAccountAndSlug :one
SELECT * FROM agent_definition
WHERE slug = ? AND (account_id IS NULL OR account_id = ?) AND is_active = TRUE;

-- Agent Definition Tool queries
-- Tool definitions (built-in tools) live in the code catalog (agents.BuiltinTools);
-- agent_definition_tool references them by slug. Display metadata is resolved from
-- the catalog, so these queries only touch agent_definition_tool.

-- name: InsertAgentDefinitionTool :exec
INSERT INTO agent_definition_tool (id, agent_definition_id, tool_slug, config, sort_order, require_review)
VALUES (?, ?, ?, ?, ?, ?);

-- name: DeleteAgentDefinitionToolsByAgentID :exec
DELETE FROM agent_definition_tool WHERE agent_definition_id = ?;

-- name: ListToolsByAgentDefinitionID :many
SELECT id, agent_definition_id, tool_slug, config, sort_order, require_review, created_at, updated_at
FROM agent_definition_tool
WHERE agent_definition_id = ?
ORDER BY sort_order ASC;

-- Agent Config queries

-- name: InsertAgentConfig :exec
INSERT INTO agent_config (id, account_id, agent_definition_id, is_enabled, config, schedule)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetAgentConfigByID :one
SELECT * FROM agent_config WHERE id = ?;

-- name: ListAgentConfigsByAccount :many
SELECT * FROM agent_config WHERE account_id = ? ORDER BY created_at DESC;

-- name: UpdateAgentConfigEnabled :exec
UPDATE agent_config SET is_enabled = ?, updated_at = NOW(6) WHERE id = ?;

-- Agent Run queries

-- name: InsertAgentRun :exec
INSERT INTO agent_run (id, account_id, agent_definition_id, agent_config_id, status_code, trigger_type, input, output, triggered_by_actor_id, triggered_by_identity_type, triggered_by_actor_name, conversation_id, trigger_message_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAgentRunByID :one
SELECT * FROM agent_run WHERE id = ?;

-- name: GetAgentRunRetryCount :one
SELECT retry_count FROM agent_run WHERE id = ?;

-- name: ListAgentRunsByAccount :many
SELECT * FROM agent_run WHERE account_id = ? ORDER BY created_at DESC LIMIT ?;

-- name: UpdateAgentRunStatus :exec
UPDATE agent_run
SET status_code = ?, updated_at = NOW(6)
WHERE id = ?;

-- name: MarkAgentRunCancelledByUser :exec
UPDATE agent_run
SET status_code = 'cancelled', completed_at = NULL, duration_ms = NULL, updated_at = NOW(6)
WHERE id = ?;

-- name: MarkAgentRunRetrying :execrows
-- Atomically move a failed run back to running and bump its retry counter. The status guard makes this
-- a no-op (no rows) when the run isn't failed, preventing double-retry races. Clears the prior
-- error so a successful re-attempt doesn't leave a stale error_message behind.
UPDATE agent_run
SET status_code = 'running', retry_count = retry_count + 1, error_message = NULL, updated_at = NOW(6)
WHERE id = ? AND status_code = 'failed';

-- name: MarkAgentRunAutoRetrying :execrows
-- Bump the retry counter for a run the runner is about to transparently re-enqueue after a transient,
-- whole-chain-unavailable failure. Unlike MarkAgentRunRetrying (manual path, guarded on 'failed'), this
-- fires mid-flight before the run is ever marked failed, so it guards on 'running' and leaves the status
-- 'running' — the run is not surfaced as failed, it is simply re-queued. The guard makes this a no-op
-- (no rows) if the run already left the running state, preventing a double re-enqueue.
UPDATE agent_run
SET retry_count = retry_count + 1, error_message = NULL, updated_at = NOW(6)
WHERE id = ? AND status_code = 'running';

-- name: UpdateAgentRunCompleted :exec
UPDATE agent_run
SET status_code = sqlc.arg('status_code'), output = sqlc.arg('output'),
    completed_at = IF(sqlc.arg('is_completed'), NOW(6), NULL),
    duration_ms = IF(sqlc.arg('is_completed'), sqlc.arg('duration_ms'), NULL),
    total_input_tokens = sqlc.arg('total_input_tokens'), total_output_tokens = sqlc.arg('total_output_tokens'),
    updated_at = NOW(6)
WHERE id = sqlc.arg('id') AND status_code != 'cancelled';

-- name: UpdateAgentRunCancelled :exec
UPDATE agent_run
SET status_code = 'cancelled', output = ?, total_input_tokens = ?, total_output_tokens = ?,
    updated_at = NOW(6)
WHERE id = ?;

-- name: UpdateAgentRunFailed :exec
UPDATE agent_run
SET status_code = 'failed', error_message = ?, completed_at = NOW(6),
    duration_ms = ?, updated_at = NOW(6)
WHERE id = ?;

-- name: ListStalledRuns :many
-- Runs orphaned by a process kill (OOM/liveness/SIGTERM) mid-flight: the goroutine that would finalize them is gone, so they sit in 'running' forever.
SELECT id, account_id FROM agent_run
WHERE status_code = 'running' AND started_at IS NOT NULL AND started_at < ?
ORDER BY started_at ASC;

-- name: ReapStalledRun :execrows
-- Guarded on status_code = 'running' so it never disturbs a run that has since paused/completed/cancelled.
UPDATE agent_run
SET status_code = 'failed', error_message = ?, completed_at = NOW(6), updated_at = NOW(6)
WHERE id = ? AND status_code = 'running' AND started_at IS NOT NULL AND started_at < ?;

-- name: MarkAgentRunDivergedFromConversation :exec
UPDATE agent_run
SET diverged_from_conversation = TRUE, updated_at = NOW(6)
WHERE id = ?;

-- Agent Action queries

-- name: InsertAgentAction :exec
INSERT INTO agent_action (id, account_id, agent_run_id, tool_slug, status_code, label, description, input, output, requires_review)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAgentActionByID :one
SELECT * FROM agent_action WHERE id = ?;

-- name: ListAgentActionsByRun :many
SELECT * FROM agent_action WHERE agent_run_id = ? ORDER BY created_at ASC;

-- name: UpdateAgentActionStatus :exec
UPDATE agent_action
SET status_code = ?, output = ?, executed_at = NOW(6), updated_at = NOW(6)
WHERE id = ?;

-- name: MarkAgentActionReviewed :exec
UPDATE agent_action
SET status_code = ?,
    reviewed_at = NOW(6),
    reviewed_by = ?,
    reviewed_by_actor_type = ?,
    reviewed_by_actor_name = ?,
    updated_at = NOW(6)
WHERE id = ?;

-- Agent Artifact queries

-- name: InsertAgentArtifact :exec
INSERT INTO agent_artifact (id, account_id, agent_action_id, artifact_type, name, content, metadata, s3_key, mime_type, size_bytes)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAgentArtifactByID :one
SELECT * FROM agent_artifact WHERE id = ?;

-- name: ListAgentArtifactsByAction :many
SELECT * FROM agent_artifact WHERE agent_action_id = ? ORDER BY created_at ASC;

-- Agent Memory queries

-- name: InsertAgentMemory :exec
INSERT INTO agent_memory (id, account_id, category, content, metadata, entity_type, entity_id, importance, expires_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAgentMemoryByID :one
SELECT * FROM agent_memory WHERE id = ?;

-- name: ListAgentMemoriesByAccount :many
SELECT * FROM agent_memory WHERE account_id = ? ORDER BY created_at DESC LIMIT ?;

-- Agent Token Usage queries

-- name: InsertAgentTokenUsage :exec
INSERT INTO agent_token_usage (id, account_id, `date`, input_tokens, output_tokens, total_cost, run_count)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetAgentTokenUsageByAccountAndDate :one
SELECT * FROM agent_token_usage WHERE account_id = ? AND `date` = ?;

-- name: UpsertAgentTokenUsage :exec
INSERT INTO agent_token_usage (id, account_id, `date`, input_tokens, output_tokens, total_cost, run_count)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    input_tokens = input_tokens + VALUES(input_tokens),
    output_tokens = output_tokens + VALUES(output_tokens),
    total_cost = total_cost + VALUES(total_cost),
    run_count = run_count + VALUES(run_count),
    updated_at = NOW(6);

-- name: ListAgentTokenUsageByAccount :many
SELECT * FROM agent_token_usage
WHERE account_id = sqlc.arg('account_id') AND `date` >= sqlc.arg('since_date')
ORDER BY `date` DESC, id DESC
LIMIT ?;

-- name: ListAgentTokenUsageByAccountAfter :many
SELECT atu.* FROM agent_token_usage atu
JOIN agent_token_usage c ON c.id = sqlc.arg('cursor_id') AND c.account_id = atu.account_id
WHERE atu.account_id = sqlc.arg('account_id')
  AND atu.`date` >= sqlc.arg('since_date')
  AND (atu.`date` < c.`date` OR (atu.`date` = c.`date` AND atu.id < c.id))
ORDER BY atu.`date` DESC, atu.id DESC
LIMIT ?;

-- name: GetMonthlyTokenUsageByAccount :one
SELECT CAST(COALESCE(SUM(input_tokens), 0) AS SIGNED) AS input_tokens,
       CAST(COALESCE(SUM(output_tokens), 0) AS SIGNED) AS output_tokens
FROM agent_token_usage
WHERE account_id = ? AND `date` >= ?;

-- Scheduler queries

-- name: ListEnabledConfigsWithSchedule :many
SELECT ac.id, ac.account_id, ac.agent_definition_id, ac.is_enabled, ac.config, ac.schedule, ac.created_at, ac.updated_at,
       ad.slug as definition_slug
FROM agent_config ac
JOIN agent_definition ad ON ad.id = ac.agent_definition_id
WHERE ac.is_enabled = TRUE AND ac.schedule IS NOT NULL AND ad.is_active = TRUE;

-- name: GetLastRunByConfigID :one
SELECT * FROM agent_run
WHERE agent_config_id = ?
ORDER BY created_at DESC
LIMIT 1;

-- name: GetAgentConfigByAccountAndDefinition :one
SELECT * FROM agent_config
WHERE account_id = ? AND agent_definition_id = ?
LIMIT 1;

-- Memory queries

-- name: ListMemoriesByEntity :many
SELECT * FROM agent_memory
WHERE account_id = ? AND entity_type = ? AND entity_id = ?
  AND (expires_at IS NULL OR expires_at > NOW(6))
ORDER BY importance DESC
LIMIT ?;

-- name: ListAccountMemories :many
SELECT * FROM agent_memory
WHERE account_id = ? AND entity_type = 'account' AND entity_id = ?
  AND (expires_at IS NULL OR expires_at > NOW(6))
ORDER BY importance DESC
LIMIT ?;

-- name: UpdateAgentMemory :exec
UPDATE agent_memory
SET category = COALESCE(sqlc.narg('category'), category),
    content = COALESCE(sqlc.narg('content'), content),
    metadata = COALESCE(CAST(sqlc.narg('metadata') AS JSON), metadata),
    entity_type = IF(sqlc.arg('clear_entity'), NULL, COALESCE(sqlc.narg('entity_type'), entity_type)),
    entity_id = IF(sqlc.arg('clear_entity'), NULL, COALESCE(sqlc.narg('entity_id'), entity_id)),
    importance = COALESCE(sqlc.narg('importance'), importance),
    expires_at = IF(sqlc.arg('clear_expires_at'), NULL, COALESCE(sqlc.narg('expires_at'), expires_at)),
    updated_at = NOW(6)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: DeleteAgentMemory :exec
DELETE FROM agent_memory WHERE id = ? AND account_id = ?;

-- name: GetAgentMemoriesByIDs :many
-- Returns agent memories matching the given IDs that belong to the caller's
-- account. Used by the api-gateway resourcekit resolver.
SELECT * FROM agent_memory
WHERE id IN (sqlc.slice('ids'))
  AND account_id = sqlc.arg('account_id');

-- Agent Account Status queries

-- name: GetAgentAccountStatus :one
SELECT * FROM agent_account_status
WHERE account_id = ? AND agent_definition_id = ?;

-- name: UpsertAgentAccountStatus :exec
INSERT INTO agent_account_status (id, account_id, agent_definition_id, status_code)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    status_code = VALUES(status_code),
    updated_at = NOW(6);

-- name: ListAgentAccountStatusesByAccount :many
SELECT * FROM agent_account_status
WHERE account_id = ?;

-- name: DeleteAgentAccountStatus :exec
DELETE FROM agent_account_status
WHERE account_id = ? AND agent_definition_id = ?;

-- Agent Run Event queries

-- name: InsertAgentRunEvent :exec
INSERT INTO agent_run_event (id, agent_run_id, account_id, step_type, title, content, sequence, duration_ms, agent_action_id, metadata, actor_id, actor_type, actor_name)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListAgentRunEventsByRunID :many
SELECT * FROM agent_run_event WHERE agent_run_id = ? ORDER BY sequence ASC;

-- name: GetMaxAgentRunEventSequence :one
SELECT CAST(COALESCE(MAX(sequence), -1) AS SIGNED) FROM agent_run_event WHERE agent_run_id = ?;

-- Run lifecycle queries

-- name: UpdateAgentRunStarted :execrows
UPDATE agent_run
SET status_code = 'running', started_at = NOW(6), updated_at = NOW(6)
WHERE id = ? AND status_code = 'pending';
