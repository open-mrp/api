package repository

import (
	"context"

	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var artifactRepoTracer = tracing.GetTracer("agent-service.agent_artifact_repository")

type agentArtifactRepoImpl struct {
	queries *sqlc.Queries
}

func NewAgentArtifactRepo(queries *sqlc.Queries) *agentArtifactRepoImpl {
	return &agentArtifactRepoImpl{queries: queries}
}

func (r *agentArtifactRepoImpl) Insert(ctx context.Context, params sqlc.InsertAgentArtifactParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, artifactRepoTracer, "repository.agent_artifact.insert")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.InsertAgentArtifact(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentArtifactRepoImpl) GetByID(ctx context.Context, id string) (*sqlc.AgentArtifact, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, artifactRepoTracer, "repository.agent_artifact.get_by_id")
	defer span.End()
	row, err := r.queries.GetAgentArtifactByID(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &row, nil
}

func (r *agentArtifactRepoImpl) ListByAction(ctx context.Context, actionID string) ([]sqlc.AgentArtifact, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, artifactRepoTracer, "repository.agent_artifact.list_by_action")
	defer span.End()
	rows, err := r.queries.ListAgentArtifactsByAction(ctx, actionID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}
