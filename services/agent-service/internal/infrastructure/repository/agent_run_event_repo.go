package repository

import (
	"context"

	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var areRepoTracer = tracing.GetTracer("agent-service.agent_run_event_repository")

type agentRunEventRepoImpl struct {
	queries *sqlc.Queries
}

func NewAgentRunEventRepo(queries *sqlc.Queries) *agentRunEventRepoImpl {
	return &agentRunEventRepoImpl{queries: queries}
}

func (r *agentRunEventRepoImpl) Insert(ctx context.Context, params sqlc.InsertAgentRunEventParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, areRepoTracer, "repository.agent_run_event.insert")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.InsertAgentRunEvent(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentRunEventRepoImpl) ListByRunID(ctx context.Context, runID string) ([]sqlc.AgentRunEvent, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, areRepoTracer, "repository.agent_run_event.list_by_run_id")
	defer span.End()
	rows, err := r.queries.ListAgentRunEventsByRunID(ctx, runID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *agentRunEventRepoImpl) GetMaxSequence(ctx context.Context, runID string) (int32, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, areRepoTracer, "repository.agent_run_event.get_max_sequence")
	defer span.End()
	result, err := r.queries.GetMaxAgentRunEventSequence(ctx, runID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return result, nil
}
