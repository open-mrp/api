package repository

import (
	"context"

	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var alertRepoTracer = tracing.GetTracer("agent-service.agent_alert_repository")

type agentAlertRepoImpl struct {
	queries *sqlc.Queries
}

func NewAgentAlertRepo(queries *sqlc.Queries) *agentAlertRepoImpl {
	return &agentAlertRepoImpl{queries: queries}
}

func (r *agentAlertRepoImpl) Insert(ctx context.Context, params sqlc.InsertAgentAlertParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, alertRepoTracer, "repository.agent_alert.insert")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.InsertAgentAlert(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentAlertRepoImpl) GetByID(ctx context.Context, id string) (*sqlc.GetAgentAlertByIDRow, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, alertRepoTracer, "repository.agent_alert.get_by_id")
	defer span.End()
	row, err := r.queries.GetAgentAlertByID(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &row, nil
}

func (r *agentAlertRepoImpl) ListByAccount(ctx context.Context, accountID string, limit int32) ([]sqlc.AgentAlert, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, alertRepoTracer, "repository.agent_alert.list_by_account")
	defer span.End()
	rows, err := r.queries.ListAgentAlertsByAccount(ctx, sqlc.ListAgentAlertsByAccountParams{
		AccountID: accountID,
		Limit:     limit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *agentAlertRepoImpl) ListByAccountCursor(ctx context.Context, params sqlc.ListAgentAlertsByAccountCursorParams) ([]sqlc.ListAgentAlertsByAccountCursorRow, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, alertRepoTracer, "repository.agent_alert.list_by_account_cursor")
	defer span.End()
	rows, err := r.queries.ListAgentAlertsByAccountCursor(ctx, params)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *agentAlertRepoImpl) Acknowledge(ctx context.Context, params sqlc.AcknowledgeAgentAlertParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, alertRepoTracer, "repository.agent_alert.acknowledge")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.AcknowledgeAgentAlert(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
