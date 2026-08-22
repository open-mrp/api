package repository

import (
	"context"
	"time"

	agentdb "github.com/open-mrp/api/services/agent-service/internal/infrastructure/db"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var tokenRepoTracer = tracing.GetTracer("agent-service.agent_token_usage_repository")

type agentTokenUsageRepoImpl struct {
	queries *sqlc.Queries
}

func NewAgentTokenUsageRepo(queries *sqlc.Queries) *agentTokenUsageRepoImpl {
	return &agentTokenUsageRepoImpl{queries: queries}
}

func (r *agentTokenUsageRepoImpl) Upsert(ctx context.Context, params sqlc.UpsertAgentTokenUsageParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, tokenRepoTracer, "repository.agent_token_usage.upsert")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.UpsertAgentTokenUsage(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentTokenUsageRepoImpl) ListByAccount(ctx context.Context, params sqlc.ListAgentTokenUsageByAccountParams) ([]sqlc.AgentTokenUsage, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, tokenRepoTracer, "repository.agent_token_usage.list_by_account")
	defer span.End()
	rows, err := r.queries.ListAgentTokenUsageByAccount(ctx, params)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *agentTokenUsageRepoImpl) GetMonthlyUsage(ctx context.Context, accountID string, sinceDate time.Time) (int64, int64, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, tokenRepoTracer, "repository.agent_token_usage.get_monthly_usage")
	defer span.End()
	row, err := r.queries.GetMonthlyTokenUsageByAccount(ctx, sqlc.GetMonthlyTokenUsageByAccountParams{
		AccountID: accountID,
		Date:      agentdb.PgDate(sinceDate),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, 0, tracing.Trace(span, apiErr)
	}
	return row.InputTokens, row.OutputTokens, nil
}

func (r *agentTokenUsageRepoImpl) GetByAccountAndDate(ctx context.Context, accountID string, date time.Time) (*sqlc.AgentTokenUsage, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, tokenRepoTracer, "repository.agent_token_usage.get_by_account_and_date")
	defer span.End()
	row, err := r.queries.GetAgentTokenUsageByAccountAndDate(ctx, sqlc.GetAgentTokenUsageByAccountAndDateParams{
		AccountID: accountID,
		Date:      agentdb.PgDate(date),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &row, nil
}
