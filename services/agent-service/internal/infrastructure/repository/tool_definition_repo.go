package repository

import (
	"context"

	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var toolDefRepoTracer = tracing.GetTracer("agent-service.tool_definition_repository")

type toolDefinitionRepoImpl struct {
	queries *sqlc.Queries
}

func NewToolDefinitionRepo(queries *sqlc.Queries) *toolDefinitionRepoImpl {
	return &toolDefinitionRepoImpl{queries: queries}
}

func (r *toolDefinitionRepoImpl) GetByID(ctx context.Context, id string) (*sqlc.ToolDefinition, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, toolDefRepoTracer, "repository.tool_definition.get_by_id")
	defer span.End()
	row, err := r.queries.GetToolDefinitionByID(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &row, nil
}

func (r *toolDefinitionRepoImpl) ListAll(ctx context.Context) ([]sqlc.ListToolDefinitionsRow, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, toolDefRepoTracer, "repository.tool_definition.list_all")
	defer span.End()
	rows, err := r.queries.ListToolDefinitions(ctx)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *toolDefinitionRepoImpl) ListToolGroups(ctx context.Context) ([]sqlc.ToolGroup, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, toolDefRepoTracer, "repository.tool_definition.list_tool_groups")
	defer span.End()
	rows, err := r.queries.ListToolGroups(ctx)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}
