package domain

import (
	"context"

	apierror "github.com/open-mrp/api/shared/errors"
)

// RunnerSvc orchestrates agent run execution.
type RunnerSvc interface {
	ExecuteRun(ctx context.Context, runID, configID, accountID, triggerType string) *apierror.APIError
	ContinueRun(ctx context.Context, runID, accountID, message string, approvedToolSlugs []string, approveAllPending bool, rejectedToolSlugs []string, approvedToolCallIDs, rejectedToolCallIDs []string, actorID, actorType, actorName, replyToMessageID string) *apierror.APIError
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

	// CreateChatRun creates a chat-linked agent run (conversation_id + trigger_message_id, trigger_type=chat, input=Message) for an agent definition and publishes the execute command.
	// Service-internal — called by the chat-run consumer when notification-service signals that an agent participant's trigger fired.
	CreateChatRun(ctx context.Context, in ChatRunInput) *apierror.APIError

	// CancelRun cancels a pending or running agent run.
	CancelRun(ctx context.Context, params CancelRunParams) *apierror.APIError

	// ContinueRun continues an agent run that is awaiting input, with idempotency support.
	//
	// 1. Validate the run exists, belongs to the account, and is awaiting input.
	// 2. Update status to running and create an outbox message atomically.
	// 3. Cache the success response for idempotent replay.
	ContinueRun(ctx context.Context, params ContinueRunParams) (string, *apierror.APIError)

	// RetryRun re-attempts a failed run by resuming its existing transcript — no new user message is added, so the agent picks up with full knowledge of what it already did (including any tool results), minimizing duplicate side effects vs. a fresh re-run. The atomic status→running transition (guarded on status='failed' and bounded by retry_count) is the source of truth that prevents double-retry races.
	RetryRun(ctx context.Context, params RetryRunParams) (string, *apierror.APIError)

	// CreateAgentMemory creates a new agent memory record.
	CreateAgentMemory(ctx context.Context, params CreateAgentMemoryParams) (*AgentMemoryInfo, *apierror.APIError)

	// UpdateAgentMemory updates an existing agent memory record.
	UpdateAgentMemory(ctx context.Context, params UpdateAgentMemoryParams) (*AgentMemoryInfo, *apierror.APIError)

	// DeleteAgentMemory deletes an agent memory record.
	DeleteAgentMemory(ctx context.Context, params DeleteAgentMemoryParams) *apierror.APIError
}
