package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type AgentDefinitionRepo interface {
	GetByID(ctx context.Context, id string) (*sqlc.AgentDefinition, *apierror.APIError)
	GetBySlug(ctx context.Context, slug string) (*sqlc.AgentDefinition, *apierror.APIError)
	ListActive(ctx context.Context) ([]sqlc.AgentDefinition, *apierror.APIError)
	Insert(ctx context.Context, params sqlc.InsertAgentDefinitionParams) *apierror.APIError
	Update(ctx context.Context, params sqlc.UpdateAgentDefinitionParams) *apierror.APIError
	SoftDelete(ctx context.Context, id, accountID string) *apierror.APIError
	ListByAccount(ctx context.Context, accountID string) ([]sqlc.AgentDefinition, *apierror.APIError)
	ListByAccountFiltered(ctx context.Context, accountID string, definitionTypes, triggerTypes []string) ([]sqlc.AgentDefinition, *apierror.APIError)
	ListByAccountCursor(ctx context.Context, params sqlc.ListAgentDefinitionsByAccountCursorParams) ([]sqlc.AgentDefinition, *apierror.APIError)
	GetByAccountAndSlug(ctx context.Context, slug, accountID string) (*sqlc.AgentDefinition, *apierror.APIError)
}

type ToolDefinitionRepo interface {
	GetByID(ctx context.Context, id string) (*sqlc.ToolDefinition, *apierror.APIError)
	ListAll(ctx context.Context) ([]sqlc.ListToolDefinitionsRow, *apierror.APIError)
	ListToolGroups(ctx context.Context) ([]sqlc.ToolGroup, *apierror.APIError)
}

type AgentDefinitionToolRepo interface {
	Insert(ctx context.Context, params sqlc.InsertAgentDefinitionToolParams) *apierror.APIError
	DeleteByAgentID(ctx context.Context, agentDefinitionID string) *apierror.APIError
	ListByAgentDefinitionID(ctx context.Context, agentDefinitionID string) ([]sqlc.ListToolsByAgentDefinitionIDRow, *apierror.APIError)
}

type AgentConfigRepo interface {
	GetByID(ctx context.Context, id string) (*sqlc.AgentConfig, *apierror.APIError)
	Insert(ctx context.Context, params sqlc.InsertAgentConfigParams) *apierror.APIError
	ListByAccount(ctx context.Context, accountID string) ([]sqlc.AgentConfig, *apierror.APIError)
	ListEnabledWithSchedule(ctx context.Context) ([]sqlc.ListEnabledConfigsWithScheduleRow, *apierror.APIError)
	GetByAccountAndDefinition(ctx context.Context, accountID, definitionID string) (*sqlc.AgentConfig, *apierror.APIError)
}

type AgentRunRepo interface {
	Insert(ctx context.Context, params sqlc.InsertAgentRunParams) *apierror.APIError
	GetByID(ctx context.Context, id string) (*sqlc.AgentRun, *apierror.APIError)
	ListByAccountFiltered(ctx context.Context, params sqlc.ListAgentRunsByAccountFilteredParams) ([]sqlc.AgentRun, *apierror.APIError)
	UpdateStatus(ctx context.Context, id, status string) *apierror.APIError
	UpdateStarted(ctx context.Context, id string) *apierror.APIError
	UpdateCompleted(ctx context.Context, params sqlc.UpdateAgentRunCompletedParams) *apierror.APIError
	UpdateFailed(ctx context.Context, params sqlc.UpdateAgentRunFailedParams) *apierror.APIError
	UpdateAllowedToolSlugs(ctx context.Context, id string, slugsJSON []byte) *apierror.APIError
	GetLastByConfigID(ctx context.Context, configID string) (*sqlc.AgentRun, *apierror.APIError)
}

type AgentActionRepo interface {
	Insert(ctx context.Context, params sqlc.InsertAgentActionParams) *apierror.APIError
	GetByID(ctx context.Context, id string) (*sqlc.AgentAction, *apierror.APIError)
	ListByRun(ctx context.Context, runID string) ([]sqlc.AgentAction, *apierror.APIError)
	UpdateStatus(ctx context.Context, params sqlc.UpdateAgentActionStatusParams) *apierror.APIError
}

type AgentArtifactRepo interface {
	Insert(ctx context.Context, params sqlc.InsertAgentArtifactParams) *apierror.APIError
	GetByID(ctx context.Context, id string) (*sqlc.AgentArtifact, *apierror.APIError)
	ListByAction(ctx context.Context, actionID string) ([]sqlc.AgentArtifact, *apierror.APIError)
}

type AgentMemoryRepo interface {
	Insert(ctx context.Context, params sqlc.InsertAgentMemoryParams) *apierror.APIError
	GetByID(ctx context.Context, id string) (*sqlc.AgentMemory, *apierror.APIError)
	ListByAccount(ctx context.Context, accountID string, limit int32) ([]sqlc.AgentMemory, *apierror.APIError)
	ListByEntity(ctx context.Context, accountID, entityType, entityID string, limit int32) ([]sqlc.AgentMemory, *apierror.APIError)
	ListAccountMemories(ctx context.Context, accountID, entityID string, limit int32) ([]sqlc.AgentMemory, *apierror.APIError)
	Update(ctx context.Context, params sqlc.UpdateAgentMemoryParams) *apierror.APIError
	Delete(ctx context.Context, id, accountID string) *apierror.APIError
	ListByAccountCursor(ctx context.Context, params sqlc.ListAgentMemoriesByAccountCursorParams) ([]sqlc.AgentMemory, *apierror.APIError)
}

type AgentAlertRepo interface {
	Insert(ctx context.Context, params sqlc.InsertAgentAlertParams) *apierror.APIError
	GetByID(ctx context.Context, id string) (*sqlc.GetAgentAlertByIDRow, *apierror.APIError)
	ListByAccount(ctx context.Context, accountID string, limit int32) ([]sqlc.AgentAlert, *apierror.APIError)
	ListByAccountCursor(ctx context.Context, params sqlc.ListAgentAlertsByAccountCursorParams) ([]sqlc.ListAgentAlertsByAccountCursorRow, *apierror.APIError)
	Acknowledge(ctx context.Context, params sqlc.AcknowledgeAgentAlertParams) *apierror.APIError
}

type AgentTokenUsageRepo interface {
	Upsert(ctx context.Context, params sqlc.UpsertAgentTokenUsageParams) *apierror.APIError
	GetByAccountAndDate(ctx context.Context, accountID string, date time.Time) (*sqlc.AgentTokenUsage, *apierror.APIError)
	ListByAccount(ctx context.Context, params sqlc.ListAgentTokenUsageByAccountParams) ([]sqlc.AgentTokenUsage, *apierror.APIError)
	GetMonthlyUsage(ctx context.Context, accountID string, sinceDate time.Time) (inputTokens, outputTokens int64, apiErr *apierror.APIError)
}

type AgentAccountStatusRepo interface {
	Upsert(ctx context.Context, params sqlc.UpsertAgentAccountStatusParams) *apierror.APIError
	GetByAccountAndDefinition(ctx context.Context, accountID, agentDefinitionID string) (*sqlc.AgentAccountStatus, *apierror.APIError)
	ListByAccount(ctx context.Context, accountID string) ([]sqlc.AgentAccountStatus, *apierror.APIError)
	DeleteByAccountAndDefinition(ctx context.Context, accountID, agentDefinitionID string) *apierror.APIError
}

type AgentRunEventRepo interface {
	Insert(ctx context.Context, params sqlc.InsertAgentRunEventParams) *apierror.APIError
	ListByRunID(ctx context.Context, runID string) ([]sqlc.AgentRunEvent, *apierror.APIError)
	GetMaxSequence(ctx context.Context, runID string) (int32, *apierror.APIError)
}

type DeletedRecordRepo interface {
	Create(ctx context.Context, resourceType constants.DeletedRecordResourceType, resourceID string, data any) *apierror.APIError
	Exists(ctx context.Context, resourceType constants.DeletedRecordResourceType, resourceID string) (bool, *apierror.APIError)
}

// IdempotencyKeyRepo manages service-level idempotency keys.
type IdempotencyKeyRepo interface {
	GetByScopeHash(ctx context.Context, scopeHash string) (*IdempotencyKey, *apierror.APIError)
	Create(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, *apierror.APIError)
	AdvanceRecoveryPoint(ctx context.Context, typeID string, recoveryPoint RecoveryPoint) *apierror.APIError
	GetRecoveryPoint(ctx context.Context, typeID string) (RecoveryPoint, *apierror.APIError)
	SetResponse(ctx context.Context, typeID string, code int, body json.RawMessage, recoveryPoint RecoveryPoint) *apierror.APIError
}
