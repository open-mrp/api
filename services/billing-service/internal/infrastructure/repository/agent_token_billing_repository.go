package repository

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/billing-service/internal/domain"
	"github.com/open-mrp/api/services/billing-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var agentTokenBillingRepoTracer = tracing.GetTracer("billing-service.agent_token_billing_repository")

type agentTokenBillingRepoImpl struct {
	queries *sqlc.Queries
}

func NewAgentTokenBillingRepo(queries *sqlc.Queries) domain.AgentTokenBillingRepo {
	return &agentTokenBillingRepoImpl{queries: queries}
}

func (r *agentTokenBillingRepoImpl) UpsertAgentTokenBilling(ctx context.Context, params domain.UpsertAgentTokenBillingParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, agentTokenBillingRepoTracer, "repository.agent_token_billing.upsert")
	defer span.End()

	err := r.queries.UpsertAgentTokenBilling(ctx, sqlc.UpsertAgentTokenBillingParams{
		ID:                params.ID,
		AccountID:         params.AccountID,
		PeriodStart:       params.PeriodStart,
		PeriodEnd:         params.PeriodEnd,
		TotalInputTokens:  params.InputTokens,
		TotalOutputTokens: params.OutputTokens,
		TotalTokens:       params.TotalTokens,
	})
	if err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *agentTokenBillingRepoImpl) GetByAccountAndPeriod(ctx context.Context, accountID string, periodStart time.Time) (*domain.AgentTokenBilling, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, agentTokenBillingRepoTracer, "repository.agent_token_billing.get_by_account_and_period")
	defer span.End()

	row, err := r.queries.GetAgentTokenBillingByAccountAndPeriod(ctx, sqlc.GetAgentTokenBillingByAccountAndPeriodParams{
		AccountID:   accountID,
		PeriodStart: periodStart,
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	var stripeMeteredItemID *string
	if row.StripeMeteredItemID.Valid {
		stripeMeteredItemID = &row.StripeMeteredItemID.String
	}

	return &domain.AgentTokenBilling{
		ID:                     row.ID,
		AccountID:              row.AccountID,
		PeriodStart:            row.PeriodStart,
		PeriodEnd:              row.PeriodEnd,
		TotalInputTokens:       row.TotalInputTokens,
		TotalOutputTokens:      row.TotalOutputTokens,
		TotalTokens:            row.TotalTokens,
		TokensReportedToStripe: row.TokensReportedToStripe,
		StripeMeteredItemID:    stripeMeteredItemID,
		RunCount:               int(row.RunCount),
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}, nil
}

func (r *agentTokenBillingRepoImpl) GetUsageSummary(ctx context.Context, accountID string, periodStart time.Time) (int64, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, agentTokenBillingRepoTracer, "repository.agent_token_billing.get_usage_summary")
	defer span.End()

	totalTokens, err := r.queries.GetAgentTokenUsageSummary(ctx, sqlc.GetAgentTokenUsageSummaryParams{
		AccountID:   accountID,
		PeriodStart: periodStart,
	})
	if err != nil {
		return 0, tracing.Trace(span, db.MapSQLError(err))
	}
	return totalTokens, nil
}

func (r *agentTokenBillingRepoImpl) GetCompletedTokensByAccount(ctx context.Context, accountID string) (int64, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, agentTokenBillingRepoTracer, "repository.agent_token_billing.get_completed_tokens_by_account")
	defer span.End()

	totalTokens, err := r.queries.GetCompletedTokensByAccount(ctx, accountID)
	if err != nil {
		return 0, tracing.Trace(span, db.MapSQLError(err))
	}
	return totalTokens, nil
}

var _ domain.AgentTokenBillingRepo = (*agentTokenBillingRepoImpl)(nil)
