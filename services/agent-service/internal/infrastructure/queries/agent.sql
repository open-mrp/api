-- Agent Definition queries

-- name: GetAgentDefinitionByID :one
SELECT * FROM agent_definition WHERE id = $1;

-- name: GetAgentDefinitionBySlug :one
SELECT * FROM agent_definition WHERE slug = $1;

-- name: ListAgentDefinitions :many
SELECT * FROM agent_definition WHERE is_active = true ORDER BY name ASC;

-- name: InsertAgentDefinition :exec
INSERT INTO agent_definition (id, account_id, name, slug, description, definition_type, category_code, trigger_type, is_active, config, role_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: UpdateAgentDefinition :exec
UPDATE agent_definition SET
    name = COALESCE(sqlc.narg('name'), name),
    slug = COALESCE(sqlc.narg('slug'), slug),
    description = COALESCE(sqlc.narg('description'), description),
    category_code = COALESCE(sqlc.narg('category_code'), category_code),
    trigger_type = COALESCE(sqlc.narg('trigger_type'), trigger_type),
    config = CASE WHEN sqlc.arg('update_config')::boolean THEN sqlc.arg('new_config') ELSE config END,
    role_id = COALESCE(sqlc.narg('role_id'), role_id),
    updated_at = now()
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: SoftDeleteAgentDefinition :exec
UPDATE agent_definition
SET is_active = false, updated_at = now()
WHERE id = $1 AND account_id = $2;

-- name: ListAgentDefinitionsByAccount :many
SELECT * FROM agent_definition
WHERE is_active = true AND (account_id IS NULL OR account_id = $1)
ORDER BY definition_type ASC, name ASC;

-- name: ListAgentDefinitionsByAccountFiltered :many
SELECT * FROM agent_definition
WHERE is_active = true
  AND (account_id IS NULL OR account_id = @account_id)
  AND (@filter_definition_type::boolean = false OR definition_type = ANY(@definition_types::text[]))
  AND (@filter_trigger_type::boolean = false OR trigger_type = ANY(@trigger_types::text[]))
ORDER BY definition_type ASC, name ASC;

-- name: ListAgentDefinitionsByAccountCursor :many
SELECT ad.* FROM agent_definition ad
LEFT JOIN agent_account_status aas
  ON aas.agent_definition_id = ad.id AND aas.account_id = @account_id
WHERE ad.is_active = true
  AND (ad.account_id IS NULL OR ad.account_id = @account_id)
  AND (@filter_definition_type::boolean = false OR ad.definition_type = ANY(@definition_types::text[]))
  AND (@filter_trigger_type::boolean = false OR ad.trigger_type = ANY(@trigger_types::text[]))
  AND (@filter_status::boolean = false OR (
    ('active' = ANY(@status_codes::text[]) AND aas.status_code = 'active')
    OR ('inactive' = ANY(@status_codes::text[]) AND (aas.status_code IS NULL OR aas.status_code != 'active'))
  ))
  AND (@filter_query::boolean = false OR (
    ad.id ILIKE '%' || @search || '%'
    OR ad.name ILIKE '%' || @search || '%'
    OR COALESCE(ad.description, '') ILIKE '%' || @search || '%'
    OR ad.slug ILIKE '%' || @search || '%'
  ))
  AND (@has_cursor::boolean = false OR (ad.created_at, ad.id) < (
    (SELECT cr.created_at FROM agent_definition cr WHERE cr.id = @cursor_id),
    @cursor_id
  ))
ORDER BY ad.created_at DESC, ad.id DESC
LIMIT @lim;

-- name: GetAgentDefinitionByAccountAndSlug :one
SELECT * FROM agent_definition
WHERE slug = $1 AND (account_id IS NULL OR account_id = $2) AND is_active = true;

-- Tool Definition queries

-- name: GetToolDefinitionByID :one
SELECT * FROM tool_definition WHERE id = $1;

-- name: ListToolGroups :many
SELECT * FROM tool_group ORDER BY sort_order ASC, name ASC;

-- name: ListToolDefinitions :many
SELECT td.*, tg.name AS group_name, tg.slug AS group_slug
FROM tool_definition td
LEFT JOIN tool_group tg ON tg.id = td.tool_group_id
ORDER BY tg.sort_order ASC, td.display_name ASC;

-- Agent Definition Tool queries

-- name: InsertAgentDefinitionTool :exec
INSERT INTO agent_definition_tool (id, agent_definition_id, tool_definition_id, config, sort_order, require_review)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DeleteAgentDefinitionToolsByAgentID :exec
DELETE FROM agent_definition_tool WHERE agent_definition_id = $1;

-- name: ListToolsByAgentDefinitionID :many
SELECT adt.id, adt.agent_definition_id, adt.tool_definition_id, adt.config, adt.sort_order, adt.require_review, adt.created_at, adt.updated_at,
       td.display_name AS tool_display_name, td.description AS tool_description, td.config_schema AS tool_config_schema, td.category AS tool_category,
       td.slug AS tool_slug, td.input_schema AS tool_input_schema, td.tool_group_id, td.required_permissions,
       tg.name AS tool_group_name, tg.slug AS tool_group_slug
FROM agent_definition_tool adt
JOIN tool_definition td ON td.id = adt.tool_definition_id
LEFT JOIN tool_group tg ON tg.id = td.tool_group_id
WHERE adt.agent_definition_id = $1
ORDER BY adt.sort_order ASC;

-- Agent Config queries

-- name: InsertAgentConfig :exec
INSERT INTO agent_config (id, account_id, agent_definition_id, is_enabled, config, schedule)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetAgentConfigByID :one
SELECT * FROM agent_config WHERE id = $1;

-- name: ListAgentConfigsByAccount :many
SELECT * FROM agent_config WHERE account_id = $1 ORDER BY created_at DESC;

-- name: UpdateAgentConfigEnabled :exec
UPDATE agent_config SET is_enabled = $1, updated_at = now() WHERE id = $2;

-- Agent Run queries

-- name: InsertAgentRun :exec
INSERT INTO agent_run (id, account_id, agent_definition_id, agent_config_id, status_code, trigger_type, input, output, triggered_by_actor_id, triggered_by_identity_type, triggered_by_actor_name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: GetAgentRunByID :one
SELECT * FROM agent_run WHERE id = $1;

-- name: ListAgentRunsByAccount :many
SELECT * FROM agent_run WHERE account_id = $1 ORDER BY created_at DESC LIMIT $2;

-- name: UpdateAgentRunStatus :exec
UPDATE agent_run
SET status_code = $1, updated_at = now()
WHERE id = $2;

-- name: UpdateAgentRunCompleted :exec
UPDATE agent_run
SET status_code = $1, output = $2, completed_at = now(),
    duration_ms = $3, total_input_tokens = $4, total_output_tokens = $5,
    updated_at = now()
WHERE id = $6;

-- name: UpdateAgentRunFailed :exec
UPDATE agent_run
SET status_code = 'failed', error_message = $1, completed_at = now(),
    duration_ms = $2, updated_at = now()
WHERE id = $3;

-- name: UpdateAgentRunAllowedToolSlugs :exec
UPDATE agent_run
SET allowed_tool_slugs = $1, updated_at = now()
WHERE id = $2;

-- Agent Action queries

-- name: InsertAgentAction :exec
INSERT INTO agent_action (id, account_id, agent_run_id, tool_slug, status_code, label, description, input, output, requires_review)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: GetAgentActionByID :one
SELECT * FROM agent_action WHERE id = $1;

-- name: ListAgentActionsByRun :many
SELECT * FROM agent_action WHERE agent_run_id = $1 ORDER BY created_at ASC;

-- name: UpdateAgentActionStatus :exec
UPDATE agent_action
SET status_code = $1, output = $2, executed_at = now(), updated_at = now()
WHERE id = $3;

-- Agent Artifact queries

-- name: InsertAgentArtifact :exec
INSERT INTO agent_artifact (id, account_id, agent_action_id, artifact_type, name, content, metadata, s3_key, mime_type, size_bytes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: GetAgentArtifactByID :one
SELECT * FROM agent_artifact WHERE id = $1;

-- name: ListAgentArtifactsByAction :many
SELECT * FROM agent_artifact WHERE agent_action_id = $1 ORDER BY created_at ASC;

-- Agent Memory queries

-- name: InsertAgentMemory :exec
INSERT INTO agent_memory (id, account_id, category, content, metadata, entity_type, entity_id, importance, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetAgentMemoryByID :one
SELECT * FROM agent_memory WHERE id = $1;

-- name: ListAgentMemoriesByAccount :many
SELECT * FROM agent_memory WHERE account_id = $1 ORDER BY created_at DESC LIMIT $2;

-- Agent Alert queries

-- name: InsertAgentAlert :exec
INSERT INTO agent_alert (id, account_id, agent_run_id, agent_action_id, severity_code, status_code, title, message, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetAgentAlertByID :one
SELECT
    aa.id, aa.account_id, aa.agent_run_id, aa.agent_action_id,
    aa.severity_code, aa.status_code, aa.title, aa.message, aa.metadata,
    aa.acknowledged_at, aa.acknowledged_by_actor_id, aa.acknowledged_by_actor_type,
    aa.acknowledged_by_actor_name, aa.created_at, aa.updated_at,
    ar.status_code AS run_status_code,
    ar.trigger_type AS run_trigger_type,
    ar.created_at AS run_created_at,
    ar.updated_at AS run_updated_at,
    act.tool_slug AS action_tool_slug,
    act.status_code AS action_status_code,
    act.created_at AS action_created_at,
    act.updated_at AS action_updated_at
FROM agent_alert aa
LEFT JOIN agent_run ar ON ar.id = aa.agent_run_id
LEFT JOIN agent_action act ON act.id = aa.agent_action_id
WHERE aa.id = $1;

-- name: ListAgentAlertsByAccount :many
SELECT * FROM agent_alert WHERE account_id = $1 ORDER BY created_at DESC LIMIT $2;

-- name: ListAgentAlertsByAccountCursor :many
SELECT
    aa.id, aa.account_id, aa.agent_run_id, aa.agent_action_id,
    aa.severity_code, aa.status_code, aa.title, aa.message, aa.metadata,
    aa.acknowledged_at, aa.acknowledged_by_actor_id, aa.acknowledged_by_actor_type,
    aa.acknowledged_by_actor_name, aa.created_at, aa.updated_at,
    ar.status_code AS run_status_code,
    ar.trigger_type AS run_trigger_type,
    ar.created_at AS run_created_at,
    ar.updated_at AS run_updated_at,
    act.tool_slug AS action_tool_slug,
    act.status_code AS action_status_code,
    act.created_at AS action_created_at,
    act.updated_at AS action_updated_at
FROM agent_alert aa
LEFT JOIN agent_run ar ON ar.id = aa.agent_run_id
LEFT JOIN agent_action act ON act.id = aa.agent_action_id
WHERE aa.account_id = @account_id
  AND (@filter_severity::boolean = false OR aa.severity_code = @severity_code)
  AND (@filter_status::boolean = false OR aa.status_code = @status_code)
  AND (@filter_query::boolean = false OR (
    aa.id ILIKE '%' || @search || '%'
    OR aa.title ILIKE '%' || @search || '%'
    OR COALESCE(aa.message, '') ILIKE '%' || @search || '%'
  ))
  AND (@has_cursor::boolean = false OR (aa.created_at, aa.id) < (
    (SELECT cr.created_at FROM agent_alert cr WHERE cr.id = @cursor_id),
    @cursor_id
  ))
ORDER BY aa.created_at DESC, aa.id DESC
LIMIT @lim;

-- name: AcknowledgeAgentAlert :exec
UPDATE agent_alert
SET status_code = 'acknowledged', acknowledged_at = now(),
    acknowledged_by_actor_id = $1, acknowledged_by_actor_type = $2, acknowledged_by_actor_name = $3,
    updated_at = now()
WHERE id = $4 AND account_id = $5;

-- Agent Token Usage queries

-- name: InsertAgentTokenUsage :exec
INSERT INTO agent_token_usage (id, account_id, date, input_tokens, output_tokens, total_cost, run_count)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetAgentTokenUsageByAccountAndDate :one
SELECT * FROM agent_token_usage WHERE account_id = $1 AND date = $2;

-- name: UpsertAgentTokenUsage :exec
INSERT INTO agent_token_usage (id, account_id, date, input_tokens, output_tokens, total_cost, run_count)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (account_id, date) DO UPDATE SET
    input_tokens = agent_token_usage.input_tokens + EXCLUDED.input_tokens,
    output_tokens = agent_token_usage.output_tokens + EXCLUDED.output_tokens,
    total_cost = agent_token_usage.total_cost + EXCLUDED.total_cost,
    run_count = agent_token_usage.run_count + EXCLUDED.run_count,
    updated_at = now();

-- name: ListAgentTokenUsageByAccount :many
SELECT atu.* FROM agent_token_usage atu
WHERE atu.account_id = @account_id
  AND atu.date >= @since_date
  AND (@has_cursor::boolean = false OR (atu.date, atu.id) < (
    (SELECT c.date FROM agent_token_usage c WHERE c.id = @cursor_id),
    @cursor_id
  ))
ORDER BY atu.date DESC, atu.id DESC
LIMIT @lim;

-- name: GetMonthlyTokenUsageByAccount :one
SELECT COALESCE(SUM(input_tokens), 0)::bigint AS input_tokens,
       COALESCE(SUM(output_tokens), 0)::bigint AS output_tokens
FROM agent_token_usage
WHERE account_id = $1 AND date >= $2;

-- Scheduler queries

-- name: ListEnabledConfigsWithSchedule :many
SELECT ac.id, ac.account_id, ac.agent_definition_id, ac.is_enabled, ac.config, ac.schedule, ac.created_at, ac.updated_at,
       ad.slug as definition_slug
FROM agent_config ac
JOIN agent_definition ad ON ad.id = ac.agent_definition_id
WHERE ac.is_enabled = true AND ac.schedule IS NOT NULL AND ad.is_active = true;

-- name: GetLastRunByConfigID :one
SELECT * FROM agent_run
WHERE agent_config_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: GetAgentConfigByAccountAndDefinition :one
SELECT * FROM agent_config
WHERE account_id = $1 AND agent_definition_id = $2
LIMIT 1;

-- Memory queries

-- name: ListMemoriesByEntity :many
SELECT * FROM agent_memory
WHERE account_id = $1 AND entity_type = $2 AND entity_id = $3
  AND (expires_at IS NULL OR expires_at > now())
ORDER BY importance DESC
LIMIT $4;

-- name: ListAccountMemories :many
SELECT * FROM agent_memory
WHERE account_id = $1 AND entity_type = 'account' AND entity_id = $2
  AND (expires_at IS NULL OR expires_at > now())
ORDER BY importance DESC
LIMIT $3;

-- name: UpdateAgentMemory :exec
UPDATE agent_memory
SET category = $2, content = $3, metadata = $4, entity_type = $5,
    entity_id = $6, importance = $7, expires_at = $8, updated_at = now()
WHERE id = $1 AND account_id = $9;

-- name: DeleteAgentMemory :exec
DELETE FROM agent_memory WHERE id = $1 AND account_id = $2;

-- name: GetAgentMemoriesByIDs :many
-- Returns agent memories matching the given IDs that belong to the caller's
-- account. Used by the api-gateway resourcekit resolver.
SELECT * FROM agent_memory
WHERE id = ANY(@ids::text[])
  AND account_id = @account_id;

-- name: ListAgentMemoriesByAccountCursor :many
SELECT am.* FROM agent_memory am
WHERE am.account_id = @account_id
  AND (am.expires_at IS NULL OR am.expires_at > now())
  AND (@filter_category::boolean = false OR am.category = @category)
  AND (@filter_entity_type::boolean = false OR am.entity_type = @entity_type)
  AND (@filter_query::boolean = false OR (
    am.id ILIKE '%' || @search || '%'
    OR am.category ILIKE '%' || @search || '%'
    OR am.content ILIKE '%' || @search || '%'
    OR COALESCE(am.entity_id::text, '') ILIKE '%' || @search || '%'
  ))
  AND (@has_cursor::boolean = false OR (am.created_at, am.id) < (
    (SELECT cr.created_at FROM agent_memory cr WHERE cr.id = @cursor_id),
    @cursor_id
  ))
ORDER BY am.created_at DESC, am.id DESC
LIMIT @lim;

-- Agent Account Status queries

-- name: GetAgentAccountStatus :one
SELECT * FROM agent_account_status
WHERE account_id = $1 AND agent_definition_id = $2;

-- name: UpsertAgentAccountStatus :exec
INSERT INTO agent_account_status (id, account_id, agent_definition_id, status_code)
VALUES ($1, $2, $3, $4)
ON CONFLICT (account_id, agent_definition_id) DO UPDATE SET
    status_code = EXCLUDED.status_code,
    updated_at = now();

-- name: ListAgentAccountStatusesByAccount :many
SELECT * FROM agent_account_status
WHERE account_id = $1;

-- name: DeleteAgentAccountStatus :exec
DELETE FROM agent_account_status
WHERE account_id = $1 AND agent_definition_id = $2;

-- name: ListAgentRunsByAccountFiltered :many
SELECT ar.* FROM agent_run ar
WHERE ar.account_id = @account_id
  AND (@filter_status::boolean = false OR ar.status_code = @status_code)
  AND (@filter_definition::boolean = false OR ar.agent_definition_id = @agent_definition_id)
  AND (@filter_query::boolean = false OR (
    ar.id ILIKE '%' || @search || '%'
    OR ar.status_code ILIKE '%' || @search || '%'
    OR ar.agent_definition_id ILIKE '%' || @search || '%'
  ))
  AND (@has_cursor::boolean = false OR (ar.created_at, ar.id) < (
    (SELECT cr.created_at FROM agent_run cr WHERE cr.id = @cursor_id),
    @cursor_id
  ))
ORDER BY ar.created_at DESC, ar.id DESC
LIMIT @lim;

-- Agent Run Event queries

-- name: InsertAgentRunEvent :exec
INSERT INTO agent_run_event (id, agent_run_id, account_id, step_type, title, content, sequence, duration_ms, agent_action_id, metadata, actor_id, actor_type, actor_name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: ListAgentRunEventsByRunID :many
SELECT * FROM agent_run_event WHERE agent_run_id = $1 ORDER BY sequence ASC;

-- name: GetMaxAgentRunEventSequence :one
SELECT COALESCE(MAX(sequence), -1)::int FROM agent_run_event WHERE agent_run_id = $1;

-- Run lifecycle queries

-- name: UpdateAgentRunStarted :exec
UPDATE agent_run
SET status_code = 'running', started_at = now(), updated_at = now()
WHERE id = $1;
