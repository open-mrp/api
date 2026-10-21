package repository

import (
	"context"
	gosql "database/sql"
	"slices"
	"strings"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
)

const agentDefinitionColumns = "ad.id, ad.account_id, ad.name, ad.slug, ad.description, ad.definition_type, ad.category_code, ad.trigger_type, ad.is_active, ad.config, ad.role_id, ad.created_at, ad.updated_at"

const agentRunColumns = "ar.id, ar.account_id, ar.agent_definition_id, ar.agent_config_id, ar.status_code, ar.trigger_type, ar.input, ar.output, ar.error_message, ar.started_at, ar.completed_at, ar.duration_ms, ar.total_input_tokens, ar.total_output_tokens, ar.triggered_by_actor_id, ar.triggered_by_identity_type, ar.triggered_by_actor_name, ar.allowed_tool_slugs, ar.conversation_id, ar.trigger_message_id, ar.diverged_from_conversation, ar.retry_count, ar.created_at, ar.updated_at"

const agentMemoryColumns = "am.id, am.account_id, am.category, am.content, am.metadata, am.entity_type, am.entity_id, am.importance, am.expires_at, am.created_at, am.updated_at"

type listQuery struct {
	from  strings.Builder
	where []string
	args  []any
}

func (q *listQuery) and(predicate string, args ...any) {
	q.where = append(q.where, predicate)
	q.args = append(q.args, args...)
}

func (q *listQuery) search(query string, columns ...string) {
	pattern := db.NullStringLikePtr(&query)
	if !pattern.Valid {
		return
	}
	arms := make([]string, len(columns))
	for i, column := range columns {
		arms[i] = column + " LIKE ?"
		q.args = append(q.args, pattern.String)
	}
	q.where = append(q.where, "("+strings.Join(arms, " OR ")+")")
}

func (q *listQuery) sql(columns, alias string, limit int32) (string, []any) {
	return "SELECT " + columns + " FROM " + q.from.String() +
		" WHERE " + strings.Join(q.where, " AND ") +
		" ORDER BY " + alias + ".created_at DESC, " + alias + ".id DESC LIMIT ?", append(q.args, limit)
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}

func anyArgs(values []string) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

func buildAgentDefinitionListQuery(params domain.AgentDefinitionPageQuery) (string, []any) {
	var q listQuery
	q.from.WriteString("agent_definition ad")
	includeActive := slices.Contains(params.Statuses, "active")
	includeInactive := slices.Contains(params.Statuses, "inactive")
	filterStatus := len(params.Statuses) > 0 && includeActive != includeInactive
	if filterStatus {
		q.from.WriteString(" LEFT JOIN agent_account_status aas ON aas.agent_definition_id = ad.id AND aas.account_id = ?")
		q.args = append(q.args, params.AccountID)
	}
	if params.CursorID != "" {
		q.from.WriteString(" JOIN agent_definition cr ON cr.id = ?")
		q.args = append(q.args, params.CursorID)
	}

	q.and("ad.is_active = TRUE")
	q.and("(ad.account_id IS NULL OR ad.account_id = ?)", params.AccountID)
	if len(params.DefinitionTypes) > 0 {
		q.and("ad.definition_type IN ("+placeholders(len(params.DefinitionTypes))+")", anyArgs(params.DefinitionTypes)...)
	}
	if len(params.TriggerTypes) > 0 {
		q.and("ad.trigger_type IN ("+placeholders(len(params.TriggerTypes))+")", anyArgs(params.TriggerTypes)...)
	}
	switch {
	case filterStatus && includeActive:
		q.and("aas.status_code = 'active'")
	case filterStatus && includeInactive:
		q.and("(aas.status_code IS NULL OR aas.status_code != 'active')")
	case len(params.Statuses) > 0 && !includeActive && !includeInactive:
		q.and("FALSE")
	}
	q.search(params.Query, "ad.id", "ad.name", "ad.description", "ad.slug")
	if params.CursorID != "" {
		q.and("(ad.created_at < cr.created_at OR (ad.created_at = cr.created_at AND ad.id < cr.id))")
	}
	return q.sql(agentDefinitionColumns, "ad", params.Limit)
}

func buildAgentRunListQuery(params domain.AgentRunPageQuery) (string, []any) {
	var q listQuery
	q.from.WriteString("agent_run ar")
	if params.CursorID != "" {
		q.from.WriteString(" JOIN agent_run cr ON cr.id = ?")
		q.args = append(q.args, params.CursorID)
	}

	q.and("ar.account_id = ?", params.AccountID)
	if params.StatusCode != "" {
		q.and("ar.status_code = ?", params.StatusCode)
	}
	if params.AgentDefinitionID != "" {
		q.and("ar.agent_definition_id = ?", params.AgentDefinitionID)
	}
	q.search(params.Query, "ar.id", "ar.status_code", "ar.agent_definition_id")
	if params.CursorID != "" {
		q.and("(ar.created_at < cr.created_at OR (ar.created_at = cr.created_at AND ar.id < cr.id))")
	}
	return q.sql(agentRunColumns, "ar", params.Limit)
}

func buildAgentMemoryListQuery(params domain.AgentMemoryPageQuery) (string, []any) {
	var q listQuery
	q.from.WriteString("agent_memory am")
	if params.CursorID != "" {
		q.from.WriteString(" JOIN agent_memory cr ON cr.id = ?")
		q.args = append(q.args, params.CursorID)
	}

	q.and("am.account_id = ?", params.AccountID)
	q.and("(am.expires_at IS NULL OR am.expires_at > NOW(6))")
	if params.Category != "" {
		q.and("am.category = ?", params.Category)
	}
	if params.EntityType != "" {
		q.and("am.entity_type = ?", params.EntityType)
	}
	q.search(params.Query, "am.id", "am.category", "am.content", "am.entity_id")
	if params.CursorID != "" {
		q.and("(am.created_at < cr.created_at OR (am.created_at = cr.created_at AND am.id < cr.id))")
	}
	return q.sql(agentMemoryColumns, "am", params.Limit)
}

func queryList[T any](ctx context.Context, dbtx sqlc.DBTX, query string, args []any, scan func(*gosql.Rows, *T) error) ([]T, error) {
	rows, err := dbtx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []T
	for rows.Next() {
		var item T
		if err := scan(rows, &item); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func scanAgentDefinition(rows *gosql.Rows, d *sqlc.AgentDefinition) error {
	return rows.Scan(&d.ID, &d.AccountID, &d.Name, &d.Slug, &d.Description, &d.DefinitionType, &d.CategoryCode,
		&d.TriggerType, &d.IsActive, &d.Config, &d.RoleID, &d.CreatedAt, &d.UpdatedAt)
}

func scanAgentRun(rows *gosql.Rows, r *sqlc.AgentRun) error {
	return rows.Scan(&r.ID, &r.AccountID, &r.AgentDefinitionID, &r.AgentConfigID, &r.StatusCode, &r.TriggerType,
		&r.Input, &r.Output, &r.ErrorMessage, &r.StartedAt, &r.CompletedAt, &r.DurationMs, &r.TotalInputTokens,
		&r.TotalOutputTokens, &r.TriggeredByActorID, &r.TriggeredByIdentityType, &r.TriggeredByActorName,
		&r.AllowedToolSlugs, &r.ConversationID, &r.TriggerMessageID, &r.DivergedFromConversation, &r.RetryCount,
		&r.CreatedAt, &r.UpdatedAt)
}

func scanAgentMemory(rows *gosql.Rows, m *sqlc.AgentMemory) error {
	return rows.Scan(&m.ID, &m.AccountID, &m.Category, &m.Content, &m.Metadata, &m.EntityType, &m.EntityID,
		&m.Importance, &m.ExpiresAt, &m.CreatedAt, &m.UpdatedAt)
}
