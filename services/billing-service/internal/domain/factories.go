package domain

import "github.com/open-mrp/api/shared/messaging"

type RepoFactory interface {
	NewPricingPlanRepo() PricingPlanRepo
	NewAccountUsageRepo() AccountUsageRepo
	NewAgentTokenBillingRepo() AgentTokenBillingRepo
	NewIdempotencyKeyRepo() IdempotencyKeyRepo
	NewOutboxRepo() messaging.OutboxRepo
	// NewInboxRepo exposes the inbox on the transaction-scoped factory so a consumer can commit its recovery point inside the same transaction as its work. See messaging.InboxRepo.Complete.
	NewInboxRepo() messaging.InboxRepo
}
