package domain

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

// RunnerSvc orchestrates agent run execution.
type RunnerSvc interface {
	ExecuteRun(ctx context.Context, runID, configID, accountID, triggerType string) error
	ContinueRun(ctx context.Context, runID, accountID, message string, approvedToolSlugs []string, allowedToolSlugs []string, actorID, actorType, actorName string) error
}

// SchedulerSvc manages periodic agent scheduling.
type SchedulerSvc interface {
	Start(ctx context.Context) error
	Stop()
}

// AgentDefinitionSvc handles business logic for agent definitions, including CRUD operations with idempotency and tool management.
type AgentDefinitionSvc interface {
	// CreateCustomAgent creates a new custom agent definition with optional tool links.
	//
	// Preconditions:
	//   - All referenced tool IDs must exist.
	//
	// Side effects:
	//   - Persists a new agent_definition row and associated agent_definition_tool rows.
	//   - Caches the response in the service idempotency key.
	CreateCustomAgent(ctx context.Context, params CreateCustomAgentParams) (*AgentDefinitionInfo, *apierror.APIError)

	// UpdateCustomAgent updates a custom agent definition and replaces its tool links.
	//
	// Preconditions:
	//   - The definition must exist, be of type "custom", and belong to the caller's account.
	//   - All referenced tool IDs must exist.
	//
	// Side effects:
	//   - Updates the agent_definition row, deletes existing tool links, and re-creates them from the provided list.
	//   - Caches the response in the service idempotency key.
	UpdateCustomAgent(ctx context.Context, params UpdateCustomAgentParams) (*AgentDefinitionInfo, *apierror.APIError)

	// DeleteCustomAgent soft-deletes a custom agent definition.
	//
	// Preconditions:
	//   - The definition must exist, be of type "custom", and belong to the caller's account.
	//
	// Side effects:
	//   - Sets is_active = false on the agent_definition row.
	//   - Caches the response in the service idempotency key.
	DeleteCustomAgent(ctx context.Context, params DeleteCustomAgentParams) *apierror.APIError

	// GetAgentDefinition returns a single agent definition with its tools. System definitions are visible to all accounts; custom definitions are only visible to their owner.
	GetAgentDefinition(ctx context.Context, agentDefinitionID string, includes []string) (*AgentDefinitionInfo, *apierror.APIError)

	// ListAgentDefinitions returns all active agent definitions visible to the given account (system definitions plus the account's custom ones).
	ListAgentDefinitions(ctx context.Context, params ListAgentDefinitionsParams) (*ListAgentDefinitionsResult, *apierror.APIError)

	// ListAvailableTools returns platform tool definitions that can be attached to agent definitions, along with tool groups. Results are filtered by query and paginated by cursor/limit when provided.
	ListAvailableTools(ctx context.Context, params ListAvailableToolsParams) ([]AvailableToolInfo, []ToolGroupInfo, *apierror.APIError)

	// UpdateAgentAccountStatus upserts the per-account status for an agent definition.
	UpdateAgentAccountStatus(ctx context.Context, params UpdateAgentAccountStatusParams) (*AgentAccountStatusInfo, *apierror.APIError)

	// TriggerRun creates an agent run and publishes an outbox message to execute it.
	TriggerRun(ctx context.Context, params TriggerRunParams) (string, *apierror.APIError)

	// CancelRun cancels a pending or running agent run.
	CancelRun(ctx context.Context, params CancelRunParams) *apierror.APIError

	// ContinueRun continues an agent run that is awaiting input and publishes
	// an outbox message to resume execution.
	ContinueRun(ctx context.Context, params ContinueRunParams) (string, *apierror.APIError)

	// CreateAgentMemory creates a new agent memory record.
	CreateAgentMemory(ctx context.Context, params CreateAgentMemoryParams) (*AgentMemoryInfo, *apierror.APIError)

	// UpdateAgentMemory updates an existing agent memory record.
	UpdateAgentMemory(ctx context.Context, params UpdateAgentMemoryParams) (*AgentMemoryInfo, *apierror.APIError)

	// DeleteAgentMemory deletes an agent memory record.
	DeleteAgentMemory(ctx context.Context, params DeleteAgentMemoryParams) *apierror.APIError

	// AcknowledgeAgentAlert acknowledges an agent alert.
	AcknowledgeAgentAlert(ctx context.Context, params AcknowledgeAgentAlertParams) (*AgentAlertInfo, *apierror.APIError)
}
