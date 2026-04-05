package domain

import "github.com/augno/api/shared/messaging"

type RepoFactory interface {
	NewPricingPlanRepo() PricingPlanRepo
	NewAccountUsageRepo() AccountUsageRepo
	NewAgentTokenBillingRepo() AgentTokenBillingRepo
	NewIdempotencyKeyRepo() IdempotencyKeyRepo
	NewOutboxRepo() messaging.OutboxRepo
}
