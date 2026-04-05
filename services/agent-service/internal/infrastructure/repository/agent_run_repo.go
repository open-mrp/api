package repository

import (
	"context"

	agentdb "github.com/augno/api/services/agent-service/internal/infrastructure/db"
	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var runRepoTracer = tracing.GetTracer("agent-service.agent_run_repository")

type agentRunRepoImpl struct {
	queries *sqlc.Queries
}

func NewAgentRunRepo(queries *sqlc.Queries) *agentRunRepoImpl {
	return &agentRunRepoImpl{queries: queries}
}

func (r *agentRunRepoImpl) Insert(ctx context.Context, params sqlc.InsertAgentRunParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.insert")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.InsertAgentRun(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentRunRepoImpl) GetByID(ctx context.Context, id string) (*sqlc.AgentRun, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.get_by_id")
	defer span.End()
	row, err := r.queries.GetAgentRunByID(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &row, nil
}

func (r *agentRunRepoImpl) ListByAccountFiltered(ctx context.Context, params sqlc.ListAgentRunsByAccountFilteredParams) ([]sqlc.AgentRun, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.list_by_account_filtered")
	defer span.End()
	rows, err := r.queries.ListAgentRunsByAccountFiltered(ctx, params)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *agentRunRepoImpl) UpdateStatus(ctx context.Context, id, status string) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.update_status")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.UpdateAgentRunStatus(ctx, sqlc.UpdateAgentRunStatusParams{
		StatusCode: status,
		ID:         id,
	})); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentRunRepoImpl) UpdateStarted(ctx context.Context, id string) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.update_started")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.UpdateAgentRunStarted(ctx, id)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentRunRepoImpl) UpdateCompleted(ctx context.Context, params sqlc.UpdateAgentRunCompletedParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.update_completed")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.UpdateAgentRunCompleted(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentRunRepoImpl) UpdateFailed(ctx context.Context, params sqlc.UpdateAgentRunFailedParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.update_failed")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.UpdateAgentRunFailed(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentRunRepoImpl) UpdateAllowedToolSlugs(ctx context.Context, id string, slugsJSON []byte) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.update_allowed_tool_slugs")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.UpdateAgentRunAllowedToolSlugs(ctx, sqlc.UpdateAgentRunAllowedToolSlugsParams{
		AllowedToolSlugs: slugsJSON,
		ID:               id,
	})); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentRunRepoImpl) GetLastByConfigID(ctx context.Context, configID string) (*sqlc.AgentRun, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.get_last_by_config_id")
	defer span.End()
	row, err := r.queries.GetLastRunByConfigID(ctx, agentdb.PgText(configID))
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &row, nil
}
