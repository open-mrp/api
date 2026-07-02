package repository

import (
	"context"

	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var actionRepoTracer = tracing.GetTracer("agent-service.agent_action_repository")

type agentActionRepoImpl struct {
	queries *sqlc.Queries
}

func NewAgentActionRepo(queries *sqlc.Queries) *agentActionRepoImpl {
	return &agentActionRepoImpl{queries: queries}
}

func (r *agentActionRepoImpl) Insert(ctx context.Context, params sqlc.InsertAgentActionParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, actionRepoTracer, "repository.agent_action.insert")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.InsertAgentAction(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentActionRepoImpl) GetByID(ctx context.Context, id string) (*sqlc.AgentAction, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, actionRepoTracer, "repository.agent_action.get_by_id")
	defer span.End()
	row, err := r.queries.GetAgentActionByID(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &row, nil
}

func (r *agentActionRepoImpl) ListByRun(ctx context.Context, runID string) ([]sqlc.AgentAction, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, actionRepoTracer, "repository.agent_action.list_by_run")
	defer span.End()
	rows, err := r.queries.ListAgentActionsByRun(ctx, runID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *agentActionRepoImpl) UpdateStatus(ctx context.Context, params sqlc.UpdateAgentActionStatusParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, actionRepoTracer, "repository.agent_action.update_status")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.UpdateAgentActionStatus(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentActionRepoImpl) MarkReviewed(ctx context.Context, params sqlc.MarkAgentActionReviewedParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, actionRepoTracer, "repository.agent_action.mark_reviewed")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.MarkAgentActionReviewed(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
