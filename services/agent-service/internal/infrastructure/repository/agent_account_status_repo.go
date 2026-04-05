package repository

import (
	"context"

	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var accountStatusRepoTracer = tracing.GetTracer("agent-service.agent_account_status_repository")

type agentAccountStatusRepoImpl struct {
	queries *sqlc.Queries
}

func NewAgentAccountStatusRepo(queries *sqlc.Queries) *agentAccountStatusRepoImpl {
	return &agentAccountStatusRepoImpl{queries: queries}
}

func (r *agentAccountStatusRepoImpl) Upsert(ctx context.Context, params sqlc.UpsertAgentAccountStatusParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, accountStatusRepoTracer, "repository.agent_account_status.upsert")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.UpsertAgentAccountStatus(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentAccountStatusRepoImpl) GetByAccountAndDefinition(ctx context.Context, accountID, agentDefinitionID string) (*sqlc.AgentAccountStatus, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, accountStatusRepoTracer, "repository.agent_account_status.get_by_account_and_definition")
	defer span.End()
	row, err := r.queries.GetAgentAccountStatus(ctx, sqlc.GetAgentAccountStatusParams{
		AccountID:         accountID,
		AgentDefinitionID: agentDefinitionID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &row, nil
}

func (r *agentAccountStatusRepoImpl) ListByAccount(ctx context.Context, accountID string) ([]sqlc.AgentAccountStatus, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, accountStatusRepoTracer, "repository.agent_account_status.list_by_account")
	defer span.End()
	rows, err := r.queries.ListAgentAccountStatusesByAccount(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *agentAccountStatusRepoImpl) DeleteByAccountAndDefinition(ctx context.Context, accountID, agentDefinitionID string) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, accountStatusRepoTracer, "repository.agent_account_status.delete_by_account_and_definition")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.DeleteAgentAccountStatus(ctx, sqlc.DeleteAgentAccountStatusParams{
		AccountID:         accountID,
		AgentDefinitionID: agentDefinitionID,
	})); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
