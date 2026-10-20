package repository

import (
	"context"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var defRepoTracer = tracing.GetTracer("agent-service.agent_definition_repository")

type agentDefinitionRepoImpl struct {
	queries *sqlc.Queries
}

func NewAgentDefinitionRepo(queries *sqlc.Queries) *agentDefinitionRepoImpl {
	return &agentDefinitionRepoImpl{queries: queries}
}

func (r *agentDefinitionRepoImpl) GetByID(ctx context.Context, id string) (*sqlc.AgentDefinition, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, defRepoTracer, "repository.agent_definition.get_by_id")
	defer span.End()
	row, err := r.queries.GetAgentDefinitionByID(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &row, nil
}

func (r *agentDefinitionRepoImpl) GetBySlug(ctx context.Context, slug string) (*sqlc.AgentDefinition, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, defRepoTracer, "repository.agent_definition.get_by_slug")
	defer span.End()
	row, err := r.queries.GetAgentDefinitionBySlug(ctx, slug)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &row, nil
}

func (r *agentDefinitionRepoImpl) ListActive(ctx context.Context) ([]sqlc.AgentDefinition, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, defRepoTracer, "repository.agent_definition.list_active")
	defer span.End()
	rows, err := r.queries.ListAgentDefinitions(ctx)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *agentDefinitionRepoImpl) Insert(ctx context.Context, params sqlc.InsertAgentDefinitionParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, defRepoTracer, "repository.agent_definition.insert")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.InsertAgentDefinition(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentDefinitionRepoImpl) Update(ctx context.Context, params sqlc.UpdateAgentDefinitionParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, defRepoTracer, "repository.agent_definition.update")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.UpdateAgentDefinition(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentDefinitionRepoImpl) SoftDelete(ctx context.Context, id, accountID string) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, defRepoTracer, "repository.agent_definition.soft_delete")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.SoftDeleteAgentDefinition(ctx, sqlc.SoftDeleteAgentDefinitionParams{
		ID:        id,
		AccountID: db.NullString(accountID),
	})); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentDefinitionRepoImpl) ListByAccount(ctx context.Context, accountID string) ([]sqlc.AgentDefinition, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, defRepoTracer, "repository.agent_definition.list_by_account")
	defer span.End()
	rows, err := r.queries.ListAgentDefinitionsByAccount(ctx, db.NullString(accountID))
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *agentDefinitionRepoImpl) ListByAccountCursor(ctx context.Context, params domain.AgentDefinitionPageQuery) ([]sqlc.AgentDefinition, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, defRepoTracer, "repository.agent_definition.list_by_account_cursor")
	defer span.End()
	query, args := buildAgentDefinitionListQuery(params)
	rows, err := queryList(ctx, r.queries.DB(), query, args, scanAgentDefinition)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *agentDefinitionRepoImpl) GetByAccountAndSlug(ctx context.Context, slug, accountID string) (*sqlc.AgentDefinition, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, defRepoTracer, "repository.agent_definition.get_by_account_and_slug")
	defer span.End()
	row, err := r.queries.GetAgentDefinitionByAccountAndSlug(ctx, sqlc.GetAgentDefinitionByAccountAndSlugParams{
		Slug:      slug,
		AccountID: db.NullString(accountID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &row, nil
}
