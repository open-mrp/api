package repository

import (
	"github.com/open-mrp/api/services/billing-service/internal/domain"
	"github.com/open-mrp/api/services/billing-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/messaging"
)

type repoFactoryImpl struct {
	queries *sqlc.Queries
}

func NewRepoFactory(queries *sqlc.Queries) domain.RepoFactory {
	return &repoFactoryImpl{queries: queries}
}

func (r *repoFactoryImpl) NewPricingPlanRepo() domain.PricingPlanRepo {
	return NewPricingPlanRepo(r.queries)
}

func (r *repoFactoryImpl) NewAccountUsageRepo() domain.AccountUsageRepo {
	return NewAccountUsageRepo(r.queries)
}

func (r *repoFactoryImpl) NewAgentTokenBillingRepo() domain.AgentTokenBillingRepo {
	return NewAgentTokenBillingRepo(r.queries)
}

func (r *repoFactoryImpl) NewIdempotencyKeyRepo() domain.IdempotencyKeyRepo {
	return NewIdempotencyKeyRepo(r.queries)
}

func (r *repoFactoryImpl) NewOutboxRepo() messaging.OutboxRepo {
	return NewOutboxRepo(r.queries)
}

func (r *repoFactoryImpl) NewInboxRepo() messaging.InboxRepo {
	return NewInboxRepo(r.queries)
}
