package repository

import (
	"github.com/augno/api/services/agent-service/internal/domain"
	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/messaging"
)

type repoFactoryImpl struct {
	queries *sqlc.Queries
}

func NewRepoFactory(queries *sqlc.Queries) domain.RepoFactory {
	return &repoFactoryImpl{queries: queries}
}

func (r *repoFactoryImpl) NewOutboxRepo() messaging.OutboxRepo {
	return NewOutboxRepo(r.queries)
}

func (r *repoFactoryImpl) NewAgentDefinitionRepo() domain.AgentDefinitionRepo {
	return NewAgentDefinitionRepo(r.queries)
}

func (r *repoFactoryImpl) NewAgentConfigRepo() domain.AgentConfigRepo {
	return NewAgentConfigRepo(r.queries)
}

func (r *repoFactoryImpl) NewAgentRunRepo() domain.AgentRunRepo {
	return NewAgentRunRepo(r.queries)
}

func (r *repoFactoryImpl) NewAgentActionRepo() domain.AgentActionRepo {
	return NewAgentActionRepo(r.queries)
}

func (r *repoFactoryImpl) NewAgentArtifactRepo() domain.AgentArtifactRepo {
	return NewAgentArtifactRepo(r.queries)
}

func (r *repoFactoryImpl) NewAgentMemoryRepo() domain.AgentMemoryRepo {
	return NewAgentMemoryRepo(r.queries)
}

func (r *repoFactoryImpl) NewAgentAlertRepo() domain.AgentAlertRepo {
	return NewAgentAlertRepo(r.queries)
}

func (r *repoFactoryImpl) NewAgentTokenUsageRepo() domain.AgentTokenUsageRepo {
	return NewAgentTokenUsageRepo(r.queries)
}

func (r *repoFactoryImpl) NewToolDefinitionRepo() domain.ToolDefinitionRepo {
	return NewToolDefinitionRepo(r.queries)
}

func (r *repoFactoryImpl) NewAgentDefinitionToolRepo() domain.AgentDefinitionToolRepo {
	return NewAgentDefinitionToolRepo(r.queries)
}

func (r *repoFactoryImpl) NewAgentAccountStatusRepo() domain.AgentAccountStatusRepo {
	return NewAgentAccountStatusRepo(r.queries)
}

func (r *repoFactoryImpl) NewAgentRunEventRepo() domain.AgentRunEventRepo {
	return NewAgentRunEventRepo(r.queries)
}

func (r *repoFactoryImpl) NewIdempotencyKeyRepo() domain.IdempotencyKeyRepo {
	return NewIdempotencyKeyRepo(r.queries)
}

func (r *repoFactoryImpl) NewDeletedRecordRepo() domain.DeletedRecordRepo {
	return NewDeletedRecordRepo(r.queries)
}
