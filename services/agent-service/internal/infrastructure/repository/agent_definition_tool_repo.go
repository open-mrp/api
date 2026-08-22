package repository

import (
	"context"

	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var adtRepoTracer = tracing.GetTracer("agent-service.agent_definition_tool_repository")

type agentDefinitionToolRepoImpl struct {
	queries *sqlc.Queries
}

func NewAgentDefinitionToolRepo(queries *sqlc.Queries) *agentDefinitionToolRepoImpl {
	return &agentDefinitionToolRepoImpl{queries: queries}
}

func (r *agentDefinitionToolRepoImpl) Insert(ctx context.Context, params sqlc.InsertAgentDefinitionToolParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, adtRepoTracer, "repository.agent_definition_tool.insert")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.InsertAgentDefinitionTool(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentDefinitionToolRepoImpl) DeleteByAgentID(ctx context.Context, agentDefinitionID string) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, adtRepoTracer, "repository.agent_definition_tool.delete_by_agent_id")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.DeleteAgentDefinitionToolsByAgentID(ctx, agentDefinitionID)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentDefinitionToolRepoImpl) ListByAgentDefinitionID(ctx context.Context, agentDefinitionID string) ([]sqlc.ListToolsByAgentDefinitionIDRow, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, adtRepoTracer, "repository.agent_definition_tool.list_by_agent_definition_id")
	defer span.End()
	rows, err := r.queries.ListToolsByAgentDefinitionID(ctx, agentDefinitionID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}
