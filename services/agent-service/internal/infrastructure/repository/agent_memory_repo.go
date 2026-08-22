package repository

import (
	"context"

	agentdb "github.com/open-mrp/api/services/agent-service/internal/infrastructure/db"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var memoryRepoTracer = tracing.GetTracer("agent-service.agent_memory_repository")

type agentMemoryRepoImpl struct {
	queries *sqlc.Queries
}

func NewAgentMemoryRepo(queries *sqlc.Queries) *agentMemoryRepoImpl {
	return &agentMemoryRepoImpl{queries: queries}
}

func (r *agentMemoryRepoImpl) Insert(ctx context.Context, params sqlc.InsertAgentMemoryParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, memoryRepoTracer, "repository.agent_memory.insert")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.InsertAgentMemory(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentMemoryRepoImpl) GetByID(ctx context.Context, id string) (*sqlc.AgentMemory, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, memoryRepoTracer, "repository.agent_memory.get_by_id")
	defer span.End()
	row, err := r.queries.GetAgentMemoryByID(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &row, nil
}

func (r *agentMemoryRepoImpl) ListByAccount(ctx context.Context, accountID string, limit int32) ([]sqlc.AgentMemory, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, memoryRepoTracer, "repository.agent_memory.list_by_account")
	defer span.End()
	rows, err := r.queries.ListAgentMemoriesByAccount(ctx, sqlc.ListAgentMemoriesByAccountParams{
		AccountID: accountID,
		Limit:     limit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *agentMemoryRepoImpl) ListByEntity(ctx context.Context, accountID, entityType, entityID string, limit int32) ([]sqlc.AgentMemory, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, memoryRepoTracer, "repository.agent_memory.list_by_entity")
	defer span.End()
	rows, err := r.queries.ListMemoriesByEntity(ctx, sqlc.ListMemoriesByEntityParams{
		AccountID:  accountID,
		EntityType: agentdb.PgText(entityType),
		EntityID:   agentdb.PgText(entityID),
		Limit:      limit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *agentMemoryRepoImpl) ListAccountMemories(ctx context.Context, accountID, entityID string, limit int32) ([]sqlc.AgentMemory, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, memoryRepoTracer, "repository.agent_memory.list_account_memories")
	defer span.End()
	rows, err := r.queries.ListAccountMemories(ctx, sqlc.ListAccountMemoriesParams{
		AccountID: accountID,
		EntityID:  agentdb.PgText(entityID),
		Limit:     limit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *agentMemoryRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]sqlc.AgentMemory, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, memoryRepoTracer, "repository.agent_memory.get_by_ids")
	defer span.End()
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetAgentMemoriesByIDs(ctx, sqlc.GetAgentMemoriesByIDsParams{
		Ids:       ids,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *agentMemoryRepoImpl) Update(ctx context.Context, params sqlc.UpdateAgentMemoryParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, memoryRepoTracer, "repository.agent_memory.update")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.UpdateAgentMemory(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentMemoryRepoImpl) Delete(ctx context.Context, id, accountID string) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, memoryRepoTracer, "repository.agent_memory.delete")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.DeleteAgentMemory(ctx, sqlc.DeleteAgentMemoryParams{
		ID:        id,
		AccountID: accountID,
	})); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentMemoryRepoImpl) ListByAccountCursor(ctx context.Context, params sqlc.ListAgentMemoriesByAccountCursorParams) ([]sqlc.AgentMemory, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, memoryRepoTracer, "repository.agent_memory.list_by_account_cursor")
	defer span.End()
	rows, err := r.queries.ListAgentMemoriesByAccountCursor(ctx, params)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}
