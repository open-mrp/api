package domain

import "github.com/open-mrp/api/shared/messaging"

// RepoFactory creates repository instances.
type RepoFactory interface {
	NewOutboxRepo() messaging.OutboxRepo
	NewAgentDefinitionRepo() AgentDefinitionRepo
	NewAgentConfigRepo() AgentConfigRepo
	NewAgentRunRepo() AgentRunRepo
	NewAgentActionRepo() AgentActionRepo
	NewAgentArtifactRepo() AgentArtifactRepo
	NewAgentMemoryRepo() AgentMemoryRepo
	NewAgentTokenUsageRepo() AgentTokenUsageRepo
	NewAgentDefinitionToolRepo() AgentDefinitionToolRepo
	NewAgentAccountStatusRepo() AgentAccountStatusRepo
	NewAgentRunEventRepo() AgentRunEventRepo
	NewIdempotencyKeyRepo() IdempotencyKeyRepo
	NewDeletedRecordRepo() DeletedRecordRepo
}
